package stockfish

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineReadUntil(t *testing.T) {
	lines := []string{
		"id name Stockfish 16",
		"option name Threads type spin default 1 min 1 max 1024",
		"uciok",
		"extra line should not be read",
	}

	eng := buildTestEngine(lines)

	collected, err := eng.readUntil(func(line string) bool {
		return line == "uciok"
	})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"id name Stockfish 16",
		"option name Threads type spin default 1 min 1 max 1024",
		"uciok",
	}, collected)
}

func TestEngineReadUntil_ChannelClosed(t *testing.T) {
	eng := buildTestEngine([]string{"one line"})

	// Drain the one line to force channel close.
	<-eng.lineCh

	// Wait for channel to close (buildTestEngine goroutine closes it after all lines sent).
	for range eng.lineCh { //nolint:revive // intentional drain
	}

	_, err := eng.readUntil(func(line string) bool {
		return line == "never"
	})

	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestIsClosedPipeError(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"broken pipe", true},
		{"read/write on closed pipe", true},
		{"file already closed", true},
		{"some other error", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.msg, func(t *testing.T) {
			var err error
			if tc.msg != "" {
				err = &mockError{tc.msg}
			}

			got := isClosedPipeError(err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// mockError is a simple error type for testing.
type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

func TestEngineReadUntil_PredicateNotMet(t *testing.T) {
	// Provide lines that never satisfy the predicate — channel closes first.
	eng := buildTestEngine([]string{"line1", "line2"})

	collected, err := eng.readUntil(func(line string) bool {
		return strings.HasPrefix(line, "NEVER")
	})

	assert.ErrorIs(t, err, ErrEngineNotRunning)
	assert.Contains(t, collected, "line1")
	assert.Contains(t, collected, "line2")
}
