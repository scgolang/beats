package main

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/launchpad"
	"golang.org/x/sync/errgroup"
)

const (
	// NumTracks is the number of tracks the sequencer has.
	NumTracks = 8

	NumBanks = 8

	// Xmax is the index of the maximum x step.
	Xmax = 8

	// Ymax is the index of the maximum y step.
	Ymax = 8

	// MaxSteps is the maximum number of steps per track.
	MaxSteps = Xmax * Ymax
)

// Launchpad wraps a *launchpad.Launchpad with methods
// that are relevant to controlling a sequencer.
type Launchpad struct {
	*launchpad.Launchpad
	*errgroup.Group

	ctx          context.Context
	currentBank  int // 8 sample banks
	currentTrack int
	initialTempo float64
	periodChan   chan time.Duration
	samplesChan  chan int
	tickChan     chan *Pos
	tracks       [NumBanks][NumTracks][MaxSteps]int
}

// OpenLaunchpad opens a connection to a launchpad.
func OpenLaunchpad(ctx context.Context, samplesChan chan int, initialTempo float64) (*Launchpad, error) {
	padBase, err := launchpad.Open()
	if err != nil {
		return nil, err
	}
	g, gctx := errgroup.WithContext(ctx)

	pad := &Launchpad{
		Launchpad:    padBase,
		Group:        g,
		ctx:          gctx,
		initialTempo: initialTempo,
		periodChan:   make(chan time.Duration, 1),
		samplesChan:  samplesChan,
		tickChan:     make(chan *Pos),
	}
	if err := pad.Reset(); err != nil {
		return nil, errors.Wrap(err, "resetting pad")
	}
	if err := pad.Select(pad.currentBank, pad.currentTrack, false); err != nil {
		return nil, errors.Wrap(err, "selecting current track")
	}
	return pad, nil
}

func (pad *Launchpad) LightCurrentTrack() error {
	pos := &Pos{}

	for i := 0; i < MaxSteps; i++ {
		color := pad.tracks[pad.currentBank][pad.currentTrack][i]
		if err := pad.Light(pos.X, pos.Y, color, 0); err != nil {
			return err
		}
		pos.Increment()
	}
	return nil
}

// listen is an infinite loop that listens for touch events on the launchpad.
func (pad *Launchpad) listen() error {
HitLoop:
	for hits := range pad.Listen() {
		for _, hit := range hits {
			x, y := hit.X, hit.Y

			if y == Ymax {
				// Top row is the pattern switcher.
				if err := pad.Select(pad.currentBank, x, true); err != nil {
					return errors.Wrap(err, "selecting new track")
				}
				continue HitLoop
			}
			if x == Xmax {
				// TODO: what to do with buttons A - H
				if err := pad.Select(y, pad.currentTrack, true); err != nil {
					return errors.Wrap(err, "selecting new track")
				}
				continue HitLoop
			}
			if err := pad.toggle(x, y); err != nil {
				return err
			}
		}
	}
	return nil
}

func (pad *Launchpad) lit(step int) bool {
	return pad.tracks[pad.currentBank][pad.currentTrack][step] > 0
}

// loop controls which step the sequencer is playing.
func (pad *Launchpad) loop() error {
TickerLoop:
	for pos := range pad.tickChan {
		var (
			prevX, prevY = pos.PrevX, pos.PrevY
			x, y         = pos.X, pos.Y
		)
		// Light the current button.
		if err := pad.Light(x, y, 0, 2); err != nil {
			return errors.Wrap(err, "lighting pad")
		}
		if x == prevX && y == prevY {
			// We've just started, so we don't have to do anything
			// with the previous position.
			continue TickerLoop
		}
		var (
			prevStep  = makeStep(prevX, prevY)
			prevColor = pad.tracks[pad.currentBank][pad.currentTrack][prevStep]
		)
		// Turn off the previous button.
		if err := pad.Light(prevX, prevY, prevColor, 0); err != nil {
			return errors.Wrap(err, "lighting pad")
		}
	}
	return nil
}

// Main is the main loop of the launchpad.
func (pad *Launchpad) Main() error {
	pad.Go(pad.listen)
	pad.Go(pad.loop)
	pad.Go(pad.ticker)
	return pad.Wait()
}

func (pad *Launchpad) sampleNum() int {
	return (NumTracks * pad.currentBank) + pad.currentTrack
}

// Select selects a track.
func (pad *Launchpad) Select(bank, track int, trigger bool) error {
	pad.currentBank = bank
	pad.currentTrack = track

	if trigger {
		pad.samplesChan <- pad.sampleNum()
	}
	if err := pad.Light(pad.currentTrack, 8, 0, 0); err != nil {
		return errors.Wrap(err, "lighting button")
	}
	if err := pad.Light(8, pad.currentBank, 0, 0); err != nil {
		return errors.Wrap(err, "lighting button")
	}
	if err := pad.Reset(); err != nil {
		return errors.Wrap(err, "resetting pad")
	}

	// TODO: change out the pattern displayed on the grid
	if err := pad.LightCurrentTrack(); err != nil {
		return errors.Wrap(err, "lightning current track")
	}
	if err := pad.Light(track, 8, 0, 3); err != nil {
		return errors.Wrap(err, "lighting button")
	}
	if err := pad.Light(8, bank, 0, 3); err != nil {
		return errors.Wrap(err, "lighting button")
	}
	return nil
}

// ticker runs the ticker.
func (pad *Launchpad) ticker() error {
	var (
		pos    = &Pos{}
		tempo  = pad.initialTempo
		period = bpmToDuration(tempo)
	)

	// Send initial position.
	pad.tickChan <- pos

	for {
		for i := 0; i < NumBanks; i++ {
			for j := 0; j < NumTracks; j++ {
				if pad.tracks[i][j][makeStep(pos.X, pos.Y)] > 0 {
					// Play the first sample.
					pad.samplesChan <- (NumTracks * i) + j
				}
			}
		}
		// Send a tick.
		select {
		default:
		case pad.tickChan <- pos:
		}
		// Increment the position and sleep.
		pos.Increment()
		time.Sleep(period)
	}
	return nil
}

// toggle toggles a step for the current track.
func (pad *Launchpad) toggle(x, y int) error {
	step := makeStep(x, y)

	if pad.lit(step) {
		if err := pad.Light(x, y, 0, 0); err != nil {
			return errors.Wrap(err, "lighting pad")
		}
		pad.tracks[pad.currentBank][pad.currentTrack][step] = 0
	} else {
		if err := pad.Light(x, y, 3, 0); err != nil {
			return errors.Wrap(err, "lighting pad")
		}
		pad.tracks[pad.currentBank][pad.currentTrack][step] = 3
	}
	return nil
}

// makeStep returns the step for a given (x, y) position
func makeStep(x, y int) int {
	return (y * Ymax) + x
}

// nsPerMinute is the number of nanoseconds in a minute.
const nsPerMinute = float64(60e9)

// bpmToDuration returns a time.Duration that represents the duration
// of a quarter note at the specified bpm.
func bpmToDuration(bpm float64) time.Duration {
	return time.Duration(nsPerMinute / bpm)
}

// Pos describes the x, y position on the launchpad.
type Pos struct {
	PrevX int
	PrevY int
	X     int
	Y     int
}

// Increment increments the position.
func (pos *Pos) Increment() {
	pos.PrevX, pos.PrevY = pos.X, pos.Y
	if pos.X+1 == Xmax {
		pos.X = 0
		if pos.Y+1 == Ymax {
			pos.Y = 0
			return
		}
		pos.Y++
		return
	}
	pos.X++
}
