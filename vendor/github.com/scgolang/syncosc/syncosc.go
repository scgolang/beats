// Package syncosc exists to define constants used in the oscsync protocol.
// See http://github.com/scgolang/oscsync/README.md
package syncosc

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
)

// OSC addresses.
const (
	AddressPulse       = "/sync/pulse"
	AddressSlaveAdd    = "/sync/slave/add"
	AddressSlaveList   = "/sync/slave/list"
	AddressSlaveRemove = "/sync/slave/remove"
	AddressTempo       = "/sync/tempo"
)

// MasterPort is the listening port for the oscsync master.
const MasterPort = 5776

// PulsesPerBar is the number of pulses in a bar (measure).
const PulsesPerBar = 96

// GetPulseDuration converts the tempo in bpm to a time.Duration
// callers are responsible for making concurrent access safe.
func GetPulseDuration(tempo float32) time.Duration {
	if tempo == 0 {
		return time.Duration(0)
	}
	return time.Duration(float32(int64(24e10)/PulsesPerBar) / tempo)
}

// Pulse represents the arguments in a /sync/pulse message.
type Pulse struct {
	Tempo float32
	Count int32
}

// PulseFromMessage gets a Pulse from an OSC message.
func PulseFromMessage(m osc.Message) (Pulse, error) {
	p := Pulse{}
	if expected, got := 2, len(m.Arguments); expected != got {
		return p, errors.Errorf("expected %d arguments, got %d", expected, got)
	}
	tempo, err := m.Arguments[0].ReadFloat32()
	if err != nil {
		return p, errors.Wrap(err, "reading tempo")
	}
	count, err := m.Arguments[1].ReadInt32()
	if err != nil {
		return p, errors.Wrap(err, "reading counter")
	}
	p.Tempo = tempo
	p.Count = count
	return p, nil
}

// Slave is any type that can sync to an oscsync master.
// The slave's Pulse method will be invoked every time a new pulse is received
// from the oscsync master.
type Slave interface {
	Pulse(Pulse) error
}

// ConnectorFunc connects a slave to an oscsync server.
type ConnectorFunc func(ctx context.Context, slave Slave, host string) error
