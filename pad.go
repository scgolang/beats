package main

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/launchpad"
	"golang.org/x/sync/errgroup"
)

const (
	// NumTracks is the number of tracks the sequencer has.
	NumTracks = 8

	// NumBanks is the number of sample banks.
	NumBanks = 8

	// Xmax is the index of the maximum x step.
	Xmax = 8

	// Ymax is the index of the maximum y step.
	Ymax = 8

	// MaxSteps is the maximum number of steps per track.
	MaxSteps = Xmax * Ymax
)

type Command struct {
	Input string
	Done  chan struct{}
}

// Launchpad wraps a *launchpad.Launchpad with methods
// that are relevant to controlling a sequencer.
type Launchpad struct {
	*launchpad.Launchpad
	*errgroup.Group

	CommandChan chan Command
	Mode        int

	ctx          context.Context
	currentBank  uint8 // 8 sample banks
	currentTrack uint8
	initialTempo float64
	periodChan   chan time.Duration
	samplesChan  chan int
	tickChan     chan *Pos
	tracks       [NumBanks][NumTracks][MaxSteps]uint8
}

// OpenLaunchpad opens a connection to a launchpad.
func OpenLaunchpad(ctx context.Context, deviceID string, samplesChan chan int, initialTempo float64) (*Launchpad, error) {
	padBase, err := launchpad.Open(deviceID)
	if err != nil {
		return nil, err
	}
	g, gctx := errgroup.WithContext(ctx)

	pad := &Launchpad{
		Launchpad:    padBase,
		Group:        g,
		CommandChan:  make(chan Command),
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

func (pad *Launchpad) command(command Command) error {
	// TODO: handle command
	switch command.Input {
	case "live":
		pad.Mode = ModeLive
	case "edit":
		pad.Mode = ModeEdit
		if err := pad.LightCurrentTrack(); err != nil {
			return errors.Wrap(err, "lighting current track")
		}
	}
	close(command.Done)
	return nil
}

// displayEdit displays an edit mode in edit mode.
func (pad *Launchpad) displayEdit(x, y uint8, color launchpad.Color) error {
	// Skip displaying the pattern edits in live mode.
	if pad.Mode == ModeLive {
		return nil
	}
	return errors.Wrap(pad.Light(x, y, color), "lighting button")
}

// editPattern toggles a step in the current pattern.
func (pad *Launchpad) editPattern(x, y uint8) error {
	step := makeStep(x, y)

	if pad.lit(step) {
		pad.tracks[pad.currentBank][pad.currentTrack][step] = 0
	} else {
		pad.tracks[pad.currentBank][pad.currentTrack][step] = 3
	}
	return nil
}

// LightCurrentTrack lights the current track.
func (pad *Launchpad) LightCurrentTrack() error {
	pos := &Pos{}

	for i := 0; i < MaxSteps; i++ {
		color := pad.tracks[pad.currentBank][pad.currentTrack][i]

		if err := pad.Light(pos.X, pos.Y, launchpad.Color{Green: color, Red: 0}); err != nil {
			log.Printf("error lighting pad: %s\n", err)
			return err
		}
		pos.Increment()
	}
	return nil
}

// listen is an infinite loop that listens for touch events on the launchpad.
func (pad *Launchpad) listen() error {
	hits, err := pad.Hits()
	if err != nil {
		return err
	}

HitLoop:
	for hit := range hits {
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
		if err := pad.editPattern(x, y); err != nil {
			return errors.Wrap(err, "editing pattern")
		}
		// If we're in live mode we don't toggle anything on the grid.
		if pad.Mode == ModeLive {
			println("live mode")
			continue
		}
		// Toggle the button on the launchpad.
		if err := pad.updatePattern(x, y); err != nil {
			return err
		}
	}
	return nil
}

func (pad *Launchpad) lit(step uint8) bool {
	return pad.tracks[pad.currentBank][pad.currentTrack][step] > 0
}

// loop controls which step the sequencer is playing and receives commands that
// change the behavior of the launchpad.
func (pad *Launchpad) loop() error {
	for {
		select {
		case command := <-pad.CommandChan:
			if err := pad.command(command); err != nil {
				return err
			}
		case pos := <-pad.tickChan:
			if err := pad.tick(pos); err != nil {
				return err
			}
		}
	}
}

func (pad *Launchpad) liveRecording() {
}

// Main is the main loop of the launchpad.
func (pad *Launchpad) Main() error {
	pad.Go(pad.listen)
	pad.Go(pad.loop)
	pad.Go(pad.ticker)
	return pad.Wait()
}

func (pad *Launchpad) sampleNum() uint8 {
	return (NumTracks * pad.currentBank) + pad.currentTrack
}

// Select selects a track.
func (pad *Launchpad) Select(bank, track uint8, trigger bool) error {
	pad.currentBank = bank
	pad.currentTrack = track

	if trigger {
		pad.samplesChan <- int(pad.sampleNum())
	}
	if err := pad.Light(pad.currentTrack, 8, launchpad.Color{Green: 0, Red: 0}); err != nil {
		log.Printf("error lighting pad: %s\n", err)
		return errors.Wrap(err, "lighting button")
	}
	if err := pad.Light(8, pad.currentBank, launchpad.Color{Green: 0, Red: 0}); err != nil {
		log.Printf("error lighting pad: %s\n", err)
		return errors.Wrap(err, "lighting button")
	}
	if err := pad.Reset(); err != nil {
		return errors.Wrap(err, "resetting pad")
	}

	// TODO: change out the pattern displayed on the grid
	if err := pad.LightCurrentTrack(); err != nil {
		return errors.Wrap(err, "lightning current track")
	}
	if err := pad.Light(track, 8, launchpad.Color{Green: 0, Red: 3}); err != nil {
		log.Printf("error lighting pad: %s\n", err)
		return errors.Wrap(err, "lighting button")
	}
	if err := pad.Light(8, bank, launchpad.Color{Green: 0, Red: 3}); err != nil {
		log.Printf("error lighting pad: %s\n", err)
		return errors.Wrap(err, "lighting button")
	}
	return nil
}

func (pad *Launchpad) tick(pos *Pos) error {
	var (
		prevX, prevY = pos.PrevX, pos.PrevY
		x, y         = pos.X, pos.Y
	)
	// Light the current button.
	if err := pad.Light(x, y, launchpad.Color{
		Green: 0,
		Red:   2,
	}); err != nil {
		return errors.Wrap(err, "lighting pad")
	}
	if x == prevX && y == prevY {
		// We've just started, so we don't have to do anything
		// with the previous position.
		return nil
	}
	var (
		prevStep  = makeStep(prevX, prevY)
		prevColor = pad.tracks[pad.currentBank][pad.currentTrack][prevStep]
	)
	// Turn off the previous button.
	if err := pad.Light(prevX, prevY, launchpad.Color{
		Green: prevColor,
		Red:   0,
	}); err != nil {
		return errors.Wrap(err, "lighting pad")
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
}

// updatePattern updates the displayed pattern for the step specified as x, y coordinates.
func (pad *Launchpad) updatePattern(x, y uint8) error {
	step := makeStep(x, y)

	if pad.lit(step) {
		if err := pad.displayEdit(x, y, launchpad.Color{Green: 3, Red: 0}); err != nil {
			return errors.Wrap(err, "toggling button")
		}
	} else {
		if err := pad.displayEdit(x, y, launchpad.Color{Green: 0, Red: 0}); err != nil {
			return errors.Wrap(err, "toggling button")
		}
	}
	return nil
}

// makeStep returns the step for a given (x, y) position
func makeStep(x, y uint8) uint8 {
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
	PrevX uint8
	PrevY uint8
	X     uint8
	Y     uint8
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

const (
	ModeEdit = iota
	ModeLive
)
