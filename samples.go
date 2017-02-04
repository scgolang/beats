package main

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/sc"
	"golang.org/x/sync/errgroup"
)

// Samples plays back samples.
type Samples struct {
	*errgroup.Group

	LoadedChan chan struct{}
	SampleChan chan int

	ctx context.Context
	dir string
	sc  *sc.Client
	scg *sc.Group
}

// NewSamples creates a new sample player.
func NewSamples(ctx context.Context, dir string) (*Samples, error) {
	client, err := sc.NewClient("udp", "127.0.0.1:0", "127.0.0.1:57120", 5*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "creating SuperCollider client")
	}
	scgroup, err := client.AddDefaultGroup()
	if err != nil {
		return nil, errors.Wrap(err, "adding default group")
	}
	g, gctx := errgroup.WithContext(ctx)

	s := &Samples{
		Group:      g,
		LoadedChan: make(chan struct{}),
		SampleChan: make(chan int, NumTracks*NumTracks),
		ctx:        gctx,
		dir:        dir,
		sc:         client,
		scg:        scgroup,
	}
	if err := s.sc.SendDef(def); err != nil {
		return nil, errors.Wrap(err, "sending synthdef")
	}
	log.Println("sent synthdef")

	if err := s.loadSamples(); err != nil {
		return nil, errors.Wrap(err, "loading samples")
	}
	log.Println("loaded samples")

	return s, nil
}

// load loads a single sample
func (s *Samples) load(bufnum int32, sample string) error {
	samplePath, err := filepath.Abs(sample)
	if err != nil {
		return errors.Wrap(err, "making path absolute")
	}
	if _, err := s.sc.ReadBuffer(samplePath, bufnum); err != nil {
		return errors.Wrap(err, "reading buffer")
	}
	log.Printf("loaded %s\n", samplePath)

	return nil
}

// loadSamples loads all the wav files in the current directory.
func (s *Samples) loadSamples() error {
	log.Printf("loading samples from dir %s\n", s.dir)
	glob := filepath.Join(s.dir, "*.wav")
	log.Printf("loading samples with glob %s\n", glob)
	samples, err := filepath.Glob(glob)
	if err != nil {
		return errors.Wrap(err, "listing files with wildcard pattern")
	}
	for i, sample := range samples {
		if i >= NumTracks*NumBanks {
			break
		}
		if err := s.load(int32(i), sample); err != nil {
			return err
		}
	}
	close(s.LoadedChan)
	return nil
}

func (s *Samples) Main() error {
	s.Go(s.play)
	return s.Wait()
}

// play is an infinite loop that plays samples
func (s *Samples) play() error {
	for bufnum := range s.SampleChan {
		var (
			action = sc.AddToTail
			ctls   = map[string]float32{
				"bufnum": float32(bufnum),
			}
			sid = s.sc.NextSynthID()
		)
		if _, err := s.scg.Synth("beats_def", sid, action, ctls); err != nil {
			return err
		}
	}
	return nil
}

var def = sc.NewSynthdef("beats_def", func(params sc.Params) sc.Ugen {
	sig := sc.PlayBuf{
		NumChannels: 1,
		BufNum:      params.Add("bufnum", 0),
		Done:        sc.FreeEnclosing,
	}.Rate(sc.AR)

	return sc.Out{
		Bus:      sc.C(0),
		Channels: sc.Multi(sig, sig),
	}.Rate(sc.AR)
})
