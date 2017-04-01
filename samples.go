package main

import (
	"log"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/scgolang/launchpad"
	"github.com/scgolang/sampler"
)

const numSlots = 64

// Samples plays back samples.
type Samples struct {
	*sampler.Sampler
}

// NewSamples creates a new sample player.
func NewSamples(dir, scsynthAddr string) (*Samples, error) {
	samp, err := sampler.New(scsynthAddr)
	if err != nil {
		return nil, err
	}
	s := &Samples{
		Sampler: samp,
	}
	if err := s.loadSamples(dir); err != nil {
		return nil, errors.Wrap(err, "loading samples")
	}
	log.Println("loaded samples")

	return s, nil
}

// loadSamples loads all the wav files in the current directory.
func (s *Samples) loadSamples(dir string) error {
	log.Printf("loading samples from dir %s\n", dir)
	glob := filepath.Join(dir, "*.wav")
	log.Printf("loading samples with glob %s\n", glob)
	samples, err := filepath.Glob(glob)
	if err != nil {
		return errors.Wrap(err, "listing files with wildcard pattern")
	}
	for i, sample := range samples {
		if i >= numSlots {
			break
		}
		if err := s.Add(i, sample); err != nil {
			return errors.Wrap(err, "adding sample")
		}
	}
	return nil
}

// Trigger triggers sample playback from the launchpad sequencer.
func (s *Samples) Trigger(trig launchpad.Trig) error {
	return s.Play(int(trig.Track), nil)
}
