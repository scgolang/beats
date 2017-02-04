package main

import (
	"context"
	"flag"
	"log"
)

func main() {
	var (
		ctx          = context.Background()
		deviceID     string
		initialTempo float64
		samplesDir   string
	)
	flag.Float64Var(&initialTempo, "t", float64(120), "tempo")
	flag.StringVar(&deviceID, "device", "hw:0,0,0", "System-specific MIDI device ID")
	flag.StringVar(&samplesDir, "samples", "samples", "samples directory")
	flag.Parse()

	samples, err := NewSamples(ctx, samplesDir)
	if err != nil {
		log.Fatal(err)
	}
	<-samples.LoadedChan

	pad, err := OpenLaunchpad(ctx, deviceID, samples.SampleChan, initialTempo)
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
