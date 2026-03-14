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

func TestErrOptionNotFound(t *testing.T) {
	err := &ErrOptionNotFound{Name: "BadOption"}
	assert.EqualError(t, err, `option not found: "BadOption"`)
}

func TestErrOptionTypeMismatch(t *testing.T) {
	err := &ErrOptionTypeMismatch{Name: "Threads", Expected: OptionTypeSpin, Got: OptionTypeCheck}
	assert.EqualError(t, err, `option "Threads" type mismatch: expected spin, got check`)
}

func TestErrOptionOutOfRange(t *testing.T) {
	err := &ErrOptionOutOfRange{Name: "Threads", Value: 9999, Min: 1, Max: 1024}
	assert.EqualError(t, err, `option "Threads" value 9999 out of range [1, 1024]`)
}

func TestErrOptionInvalidValue(t *testing.T) {
	err := &ErrOptionInvalidValue{Name: "NumaPolicy", Value: "bogus", Allowed: []string{"auto", "none"}}
	assert.EqualError(t, err, `option "NumaPolicy" value "bogus" not in allowed set [auto none]`)
}

func TestErrOptionInvalidCharacters(t *testing.T) {
	err := &ErrOptionInvalidCharacters{Name: "SyzygyPath", Value: "/tb\n/evil"}
	assert.EqualError(t, err, `option "SyzygyPath" value contains invalid characters`)
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
