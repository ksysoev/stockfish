package stockfish

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOption_Spin(t *testing.T) {
	line := "option name Threads type spin default 1 min 1 max 1024"
	opt, err := parseOption(line)

	require.NoError(t, err)
	assert.Equal(t, "Threads", opt.Name)
	assert.Equal(t, OptionTypeSpin, opt.Type)
	assert.Equal(t, "1", opt.Default)
	assert.Equal(t, 1, opt.Min)
	assert.Equal(t, 1024, opt.Max)
}

func TestParseOption_Check(t *testing.T) {
	line := "option name Ponder type check default false"
	opt, err := parseOption(line)

	require.NoError(t, err)
	assert.Equal(t, "Ponder", opt.Name)
	assert.Equal(t, OptionTypeCheck, opt.Type)
	assert.Equal(t, "false", opt.Default)
}

func TestParseOption_Button(t *testing.T) {
	line := "option name Clear Hash type button"
	opt, err := parseOption(line)

	require.NoError(t, err)
	assert.Equal(t, "Clear Hash", opt.Name)
	assert.Equal(t, OptionTypeButton, opt.Type)
	assert.Empty(t, opt.Default)
}

func TestParseOption_String(t *testing.T) {
	line := "option name EvalFile type string default nn-b1a57edbea57.nnue"
	opt, err := parseOption(line)

	require.NoError(t, err)
	assert.Equal(t, "EvalFile", opt.Name)
	assert.Equal(t, OptionTypeString, opt.Type)
	assert.Equal(t, "nn-b1a57edbea57.nnue", opt.Default)
}

func TestParseOption_StringEmpty(t *testing.T) {
	line := "option name Debug Log File type string default"
	opt, err := parseOption(line)

	require.NoError(t, err)
	assert.Equal(t, "Debug Log File", opt.Name)
	assert.Equal(t, OptionTypeString, opt.Type)
	assert.Empty(t, opt.Default)
}

func TestParseOption_Combo(t *testing.T) {
	line := "option name NumaPolicy type combo default auto var none var system var auto var hardware"
	opt, err := parseOption(line)

	require.NoError(t, err)
	assert.Equal(t, "NumaPolicy", opt.Name)
	assert.Equal(t, OptionTypeCombo, opt.Type)
	assert.Equal(t, "auto", opt.Default)
	assert.Equal(t, []string{"none", "system", "auto", "hardware"}, opt.Vars)
}

func TestParseOption_Invalid(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"no option prefix", "id name Stockfish"},
		{"missing type", "option name Foo"},
		{"empty", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseOption(tc.line)
			assert.Error(t, err)
		})
	}
}

func TestBuildSetOption(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name  string
		value *string
		want  string
	}{
		{"Threads", strPtr("4"), "setoption name Threads value 4"},
		{"Clear Hash", nil, "setoption name Clear Hash"},
		{"SyzygyPath", strPtr("/tb"), "setoption name SyzygyPath value /tb"},
		{"Debug Log File", strPtr(""), "setoption name Debug Log File value "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSetOption(tc.name, tc.value)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCollectUntilKeyword(t *testing.T) {
	tokens := []string{"nn-abc.nnue", "min", "1"}
	result := collectUntilKeyword(tokens, []string{"min", "max"})
	assert.Equal(t, []string{"nn-abc.nnue"}, result)
}
