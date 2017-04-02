package sampler_test

import (
	"testing"

	"github.com/scgolang/mockscsynth"
	"github.com/scgolang/sampler"
)

func TestAdd(t *testing.T) {
	const scsynthAddr = "127.0.0.1:57120"

	_ = mockscsynth.New(t, scsynthAddr)

	samps := newTestSampler(t, scsynthAddr)
	if err := samps.Add("MD16_Cow_01.wav", 0); err != nil {
		t.Fatal(err)
	}
}

func newTestSampler(t *testing.T, scsynthAddr string) *sampler.Sampler {
	s, err := sampler.New(scsynthAddr)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
