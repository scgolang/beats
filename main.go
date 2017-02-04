package main

import (
	"context"
	"flag"
	"log"

	"github.com/chzyer/readline"
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

	samples, samplesErr := NewSamples(ctx, samplesDir)
	if samplesErr != nil {
		log.Fatal(samplesErr)
	}
	<-samples.LoadedChan

	pad, padErr := OpenLaunchpad(ctx, deviceID, samples.SampleChan, initialTempo)
	if padErr != nil {
		log.Fatal(padErr)
	}
	defer func() { _ = pad.Close() }() // Best effort.

	go func() {
		if err := samples.Main(); err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		if err := pad.Main(); err != nil {
			log.Fatal(err)
		}
	}()

	rl, err := readline.New("beats> ")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = rl.Close() }()

	for {
		line, err := rl.Readline()
		if err != nil {
			log.Fatal(err)
		}
		command := Command{
			Done:  make(chan struct{}),
			Input: line,
		}
		pad.CommandChan <- command
		<-command.Done
	}
}
