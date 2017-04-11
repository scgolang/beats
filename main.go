package main

import (
	"context"
	"flag"
	"log"
	"net"

	"github.com/scgolang/launchpad"
	"github.com/scgolang/osc"
	"github.com/scgolang/psync"
)

func main() {
	var (
		addr         string
		initialTempo float64
		mode         string
		reset        bool
		resolution   string
		samplesDir   string
		scsynthAddr  string
	)
	flag.Float64Var(&initialTempo, "t", float64(120), "tempo")
	flag.StringVar(&addr, "addr", "127.0.0.1:8347", "listening address for commands")
	flag.StringVar(&mode, "mode", "", "sequencer mode (pattern, mutes)")
	flag.BoolVar(&reset, "reset", false, "reset the sequencer")
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
	if reset {
		if err := resetSequencer(addr); err != nil {
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

	seq := &Sequencer{
		Sequencer: pad.NewSequencer(psync.OSC{Host: "127.0.0.1"}),
	}
	if err := seq.SetResolution(resolution); err != nil {
		log.Fatal(err)
	}
	seq.AddTrigger(sampler)

	go func() {
		if err := seq.Serve(addr); err != nil {
			log.Fatal(err)
		}
	}()

	if err := seq.Main(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func setMode(mode, addr string) error {
	return sendTo(addr, osc.Message{
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

// resetSequencer resets the drum machine's sequencer.
func resetSequencer(addr string) error {
	return sendTo(addr, osc.Message{
		Address: "/beats/sequencer/reset",
	})
}

func sendTo(addr string, m osc.Message) error {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := osc.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}
	return conn.Send(m)
}
