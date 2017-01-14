package main

import (
	"context"
	"time"

	"github.com/scgolang/launchpad"
	"golang.org/x/sync/errgroup"
)

const (
	// NumTracks is the number of tracks the sequencer has.
	NumTracks = 8

	// MaxSteps is the maximum number of steps per track.
	MaxSteps = 64

	Xmax = 7
	Ymax = 7
)

// Launchpad wraps a *launchpad.Launchpad with methods
// that are relevant to controlling a sequencer.
type Launchpad struct {
	*launchpad.Launchpad
	*errgroup.Group

	ctx context.Context
}

// OpenLaunchpad opens a connection to a launchpad.
func OpenLaunchpad(ctx context.Context) (*Launchpad, error) {
	lpad, err := launchpad.Open()
	if err != nil {
		return nil, err
	}
	g, gctx := errgroup.WithContext(ctx)
	return &Launchpad{
		Launchpad: lpad,
		Group:     g,
		ctx:       gctx,
	}, nil
}

func (pad *Launchpad) Main() {
	pad.Go(pad.listen)
	pad.Go(pad.loop)
}

// loop is the main loop for the launchpad sequencer.
func (pad *Launchpad) loop() error {
	prevX, prevY := 0, 0
	x, y := 0, 0

	for _ = range time.NewTicker(200 * time.Millisecond).C {
		// Light the current button.
		pad.Light(x, y, 0, 3)

		if x == prevX && y == prevY {
			// We've just started, so the next button is (1, 0).
			x += 1
			continue
		}

		// Turn off the previous button.
		pad.Light(prevX, prevY, 0, 0)

		// Store then increment the position.
		prevX, prevY = x, y
		if x == Xmax {
			x = 0
			if y == Ymax {
				y = 0
			} else {
				y++
			}
		} else {
			x++
		}
	}
	return nil
}

// listen is an infinite loop that listens for touch events on the launchpad.
func (pad *Launchpad) listen() error {
HitLoop:
	for hit := range pad.Listen() {
		x, y := hit.X, hit.Y

		if y == 8 {
			// Top row is the pattern switcher.
			pad.Select(x)
			continue HitLoop
		}
	}
	return nil
}

// Select selects a pattern.
func (pad *Launchpad) Select(pattern int) {
}
