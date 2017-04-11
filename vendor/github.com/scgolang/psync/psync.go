// Package psync provides ways to synchronize Go processes.
package psync

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
	"golang.org/x/sync/errgroup"
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

// Synchronizer connects a slave to an oscsync server.
type Synchronizer interface {
	Synchronize(ctx context.Context, slave Slave) error
}

// OSC synchronizes processes via OSC messages.
type OSC struct {
	Host string
}

// Synchronize connects a slave to an oscsync master.
// This func blocks forever.
func (o OSC) Synchronize(ctx context.Context, slave Slave) error {
	local, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return errors.Wrap(err, "creating listening address")
	}
	remote, err := net.ResolveUDPAddr("udp", net.JoinHostPort(o.Host, strconv.Itoa(MasterPort)))
	if err != nil {
		return errors.Wrap(err, "creating listening address")
	}
	g, gctx := errgroup.WithContext(ctx)

	conn, err := osc.DialUDPContext(gctx, "udp", local, remote)
	if err != nil {
		return errors.Wrap(err, "connecting to master")
	}
	// Start the OSC server so we receive the master's messages.
	g.Go(func() error {
		return receivePulses(conn, slave)
	})
	// Announce the slave to the master.
	portStr := strings.Split(conn.LocalAddr().String(), ":")[1]
	lport, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "parsing int from %s", portStr)
	}
	if err := conn.Send(osc.Message{
		Address: AddressSlaveAdd,
		Arguments: osc.Arguments{
			osc.String("127.0.0.1"),
			osc.Int(lport),
		},
	}); err != nil {
		return errors.Wrap(err, "sending add-slave message")
	}
	return g.Wait()
}

func receivePulses(conn osc.Conn, slave Slave) error {
	// Arbitrary number of worker routines.
	return conn.Serve(8, osc.Dispatcher{
		AddressPulse: osc.Method(func(m osc.Message) error {
			pulse, err := PulseFromMessage(m)
			if err != nil {
				return errors.Wrap(err, "getting pulse from message")
			}
			return slave.Pulse(pulse)
		}),
	})
}

// Ticker runs a ticker that triggers the slave.
type Ticker struct{}

// Synchronize runs the ticker.
// This func blocks forever.
func (t Ticker) Synchronize(ctx context.Context, slave Slave) error {
	var (
		count = int32(0)
		tempo = float32(120)
		tk    = time.NewTicker(GetPulseDuration(tempo))
	)
	for {
		select {
		case <-tk.C:
			if err := slave.Pulse(Pulse{
				Count: count,
				Tempo: tempo,
			}); err != nil {
				return err
			}
			count++
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
