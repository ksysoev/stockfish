package stockfish

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrUnexpectedResponse(t *testing.T) {
	err := &ErrUnexpectedResponse{Line: "foobar"}
	assert.EqualError(t, err, `unexpected engine response: "foobar"`)
}

func TestErrInvalidOption(t *testing.T) {
	err := &ErrInvalidOption{Name: "BadOption"}
	assert.EqualError(t, err, `invalid option: "BadOption"`)
}

func TestErrInvalidPosition(t *testing.T) {
	err := &ErrInvalidPosition{Detail: "missing FEN"}
	assert.EqualError(t, err, "invalid position: missing FEN")
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrEngineNotRunning, "engine is not running"},
		{ErrEngineTimeout, "engine response timed out"},
		{ErrSearchInProgress, "search already in progress"},
		{ErrNoSearchInProgress, "no search in progress"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			require.EqualError(t, tc.err, tc.want)
		})
	}
}
