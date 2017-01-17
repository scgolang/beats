package main

import (
	"context"
	"flag"
	"log"
)

func main() {
	var (
		ctx          = context.Background()
		initialTempo float64
		dir          string
	)
	flag.Float64Var(&initialTempo, "t", float64(120), "tempo")
	flag.StringVar(&dir, "d", "samples", "samples directory")
	flag.Parse()

	samples, err := NewSamples(ctx, dir)
	if err != nil {
		log.Fatal(err)
	}
	<-samples.LoadedChan

	pad, err := OpenLaunchpad(ctx, samples.SampleChan, initialTempo)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = pad.Close() }() // Best effort.

	go func() {
		if err := samples.Main(); err != nil {
			log.Fatal(err)
		}
	}()

	if err := pad.Main(); err != nil {
		log.Fatal(err)
	}
}
