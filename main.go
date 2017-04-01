package main

import (
	"context"
	"flag"
	"log"

	"github.com/scgolang/launchpad"
	"github.com/scgolang/syncclient"
)

func main() {
	var (
		initialTempo float64
		samplesDir   string
		scsynthAddr  string
	)
	flag.Float64Var(&initialTempo, "t", float64(120), "tempo")
	flag.StringVar(&samplesDir, "samples", "samples", "samples directory")
	flag.StringVar(&scsynthAddr, "scsynth", "127.0.0.1:57120", "scsynth UDP listening address")
	flag.Parse()

	_, err := NewSamples(samplesDir, scsynthAddr)
	if err != nil {
		log.Fatal(err)
	}
	pad, err := launchpad.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = pad.Close() }() // Best effort.

	seq := pad.NewSequencer(syncclient.Connect, "127.0.0.1")

	if err := seq.Main(context.Background()); err != nil {
		log.Fatal(err)
	}
}
