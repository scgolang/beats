// Package syncclient allows programs to easily sync to an oscsync master.
package syncclient

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
	"github.com/scgolang/syncosc"
	"golang.org/x/sync/errgroup"
)

// Connect connects a slave to an oscsync master.
// This func blocks forever.
func Connect(ctx context.Context, slave syncosc.Slave, host string) error {
	local, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return errors.Wrap(err, "creating listening address")
	}
	remote, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(syncosc.MasterPort)))
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
		Address: syncosc.AddressSlaveAdd,
		Arguments: osc.Arguments{
			osc.String("127.0.0.1"),
			osc.Int(lport),
		},
	}); err != nil {
		return errors.Wrap(err, "sending add-slave message")
	}
	return g.Wait()
}

func receivePulses(conn osc.Conn, slave syncosc.Slave) error {
	// Arbitrary number of worker routines.
	return conn.Serve(8, osc.Dispatcher{
		syncosc.AddressPulse: osc.Method(func(m osc.Message) error {
			pulse, err := syncosc.PulseFromMessage(m)
			if err != nil {
				return errors.Wrap(err, "getting pulse from message")
			}
			return slave.Pulse(pulse)
		}),
	})
}
