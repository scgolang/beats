package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/scgolang/sampler"
)

// Samples plays back samples.
type Samples struct {
	*sampler.Sampler

	LoadedChan chan struct{}
	SampleChan chan int

	dir string
}

// NewSamples creates a new sample player.
func NewSamples(dir, scsynthAddr string) (*Samples, error) {
	samp, err := sampler.New(scsynthAddr)
	if err != nil {
		return nil, err
	}
	s := &Samples{
		Sampler:    samp,
		LoadedChan: make(chan struct{}),
		SampleChan: make(chan int),
		dir:        dir,
	}
	if err := s.loadSamples(); err != nil {
		return nil, errors.Wrap(err, "loading samples")
	}
	log.Println("loaded samples")

	return s, nil
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
		if err := s.Add(i, sample); err != nil {
			return errors.Wrap(err, "adding sample")
		}
	}
	return nil
}

// Main is the main loop of the sample player.
func (s *Samples) Main(ctx context.Context) error {
	for bufnum := range s.SampleChan {
		if err := s.Play(bufnum, nil); err != nil {
			return err
		}
	}
	return nil
}
