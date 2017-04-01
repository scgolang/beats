package main

import (
	"context"
	"flag"
	"log"

	"github.com/scgolang/launchpad"
	"github.com/scgolang/syncosc"
)

func main() {
	var (
		initialTempo float64
		resolution   string
		samplesDir   string
		scsynthAddr  string
	)
	flag.Float64Var(&initialTempo, "t", float64(120), "tempo")
	flag.StringVar(&resolution, "r", "16th", "sequencer clock resolution (e.g. 16th, 32nd)")
	flag.StringVar(&samplesDir, "samples", "samples", "samples directory")
	flag.StringVar(&scsynthAddr, "scsynth", "127.0.0.1:57120", "scsynth UDP listening address")
	flag.Parse()

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

	seq := pad.NewSequencer(syncosc.Ticker, "127.0.0.1")

	if err := seq.SetResolution(resolution); err != nil {
		log.Fatal(err)
	}
	seq.AddTrigger(sampler)

	if err := seq.Main(context.Background()); err != nil {
		log.Fatal(err)
	}
}
