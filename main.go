package main

import (
	"context"
	"flag"
	"log"
	"net"

	"github.com/pkg/errors"
	"github.com/scgolang/launchpad"
	"github.com/scgolang/osc"
	"github.com/scgolang/syncclient"
)

func main() {
	var (
		addr         string
		initialTempo float64
		mode         string
		resolution   string
		samplesDir   string
		scsynthAddr  string
	)
	flag.Float64Var(&initialTempo, "t", float64(120), "tempo")
	flag.StringVar(&addr, "addr", "127.0.0.1:8347", "listening address for commands")
	flag.StringVar(&mode, "mode", "", "sequencer mode (pattern, mutes)")
	flag.StringVar(&resolution, "r", "16th", "sequencer clock resolution (e.g. 16th, 32nd)")
	flag.StringVar(&samplesDir, "samples", "samples", "samples directory")
	flag.StringVar(&scsynthAddr, "scsynth", "127.0.0.1:57120", "scsynth UDP listening address")
	flag.Parse()

	if mode != "" {
		if err := setMode(mode, addr); err != nil {
			log.Fatal(err)
		}
		return
	}

	sampler, err := NewSamples(samplesDir, scsynthAddr)
	if err != nil {
		log.Fatal(err)
	}
	pad, err := launchpad.Open()
	if err != nil {
		log.Fatal(err)
	}
	if err := pad.Reset(); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = pad.Close() }() // Best effort.

	seq := pad.NewSequencer(syncclient.Connect, "127.0.0.1")

	if err := seq.SetResolution(resolution); err != nil {
		log.Fatal(err)
	}
	seq.AddTrigger(sampler)

	go commands(addr, seq)

	if err := seq.Main(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// commands controls the sequencer via OSC commands.
func commands(addr string, seq *launchpad.Sequencer) error {
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := osc.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	return conn.Serve(2, osc.Dispatcher{
		"/beats/sequencer/mode": osc.Method(func(m osc.Message) error {
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
		}),
	})
}

func setMode(mode, addr string) error {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := osc.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}
	return conn.Send(osc.Message{
		Address: "/beats/sequencer/mode",
		Arguments: osc.Arguments{
			osc.String(mode),
		},
	})
}

var seqModeMap = map[string]launchpad.Mode{
	"pattern": launchpad.ModePattern,
	"mutes":   launchpad.ModeMutes,
}
