package psync_test

import (
	"testing"
	"time"

	"github.com/scgolang/psync"
)

func TestGetPulseDuration(t *testing.T) {
	for i, testcase := range []struct {
		input  float32
		output time.Duration
	}{
		{
			input:  120,
			output: time.Duration(int64(5e8) / 24),
		},
	} {
		var (
			got      = psync.GetPulseDuration(testcase.input)
			expected = testcase.output
		)
		if !sufficientlyClose(expected, got) {
			t.Fatalf("(test case %d) expected %s, got %s", i, expected, got)
		}
	}
}

func sufficientlyClose(d1, d2 time.Duration) bool {
	var (
		thresh = time.Duration(10)
		diff   = d1 - d2
	)
	if diff < 0 {
		diff = -1 * diff
	}
	return diff < thresh
}
