package launchpad_test

import (
	"context"
	"testing"
	"time"

	"github.com/scgolang/launchpad"
)

func TestSequencer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		if err := seq.Main(ctx); err != nil && err != context.DeadlineExceeded {
			t.Fatal(err)
		}
		close(done)
	}()
	time.Sleep(20 * time.Second)
	seq.SetMode(launchpad.ModeMutes)

	time.Sleep(20 * time.Second)
	seq.SetMode(launchpad.ModePattern)

	time.Sleep(20 * time.Second)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
