package main

import (
	"net"

	"github.com/pkg/errors"
	"github.com/scgolang/launchpad"
	"github.com/scgolang/osc"
)

// Sequencer is a sample sequencer that is controlled by a launchpad and a CLI.
type Sequencer struct {
	*launchpad.Sequencer
}

// handleMode handles an OSC message that sets the mode of the sequencer.
func (seq *Sequencer) handleMode(m osc.Message) error {
	if expected, got := 1, len(m.Arguments); expected != got {
		return errors.Errorf("expected %d arguments, got %d", expected, got)
	}
	mode, err := m.Arguments[0].ReadString()
	if err != nil {
		return err
	}
	modeNum, ok := seqModeMap[mode]
	if !ok {
		return errors.Errorf("unrecognized mode: %s", mode)
	}
	seq.SetMode(modeNum)
	return nil
}

// handleReset resets the drum machine's sequencer.
func (seq *Sequencer) handleReset(m osc.Message) error {
	return seq.Reset()
}

// Serve controls the sequencer via OSC commands.
func (seq *Sequencer) Serve(addr string) error {
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := osc.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	return conn.Serve(2, osc.Dispatcher{
		"/beats/sequencer/mode":  osc.Method(seq.handleMode),
		"/beats/sequencer/reset": osc.Method(seq.handleReset),
	})
}
