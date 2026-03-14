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

func TestCollectUntilKeyword(t *testing.T) {
	tokens := []string{"nn-abc.nnue", "min", "1"}
	result := collectUntilKeyword(tokens, []string{"min", "max"})
	assert.Equal(t, []string{"nn-abc.nnue"}, result)
}

// buildTestClientWithOptions returns a Client with a fake engine and a
// pre-populated options map, suitable for testing option constructors.
func buildTestClientWithOptions(opts map[string]OptionInfo) *Client {
	eng := buildTestEngine(nil)

	return &Client{
		eng:     eng,
		search:  newSearchState(),
		options: opts,
	}
}

// buildCapturingClientWithOptions returns a Client whose engine captures sent
// commands, together with the writer so tests can assert exact UCI strings.
func buildCapturingClientWithOptions(opts map[string]OptionInfo) (*Client, *capturingWriter) {
	eng, cw := buildCapturingEngine(nil)

	return &Client{
		eng:     eng,
		search:  newSearchState(),
		options: opts,
	}, cw
}

// --- WithSpinOption ---

func TestWithSpinOption_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Threads": {Name: "Threads", Type: OptionTypeSpin, Min: 1, Max: 1024},
	})

	err := WithSpinOption("Threads", 4)(c)
	require.NoError(t, err)
	assert.Equal(t, []string{"setoption name Threads value 4"}, cw.sentLines())
}

func TestWithSpinOption_NotFound(t *testing.T) {
	c := buildTestClientWithOptions(nil)

	err := WithSpinOption("Threads", 4)(c)

	var notFound *ErrOptionNotFound
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, "Threads", notFound.Name)
}

func TestWithSpinOption_TypeMismatch(t *testing.T) {
	c := buildTestClientWithOptions(map[string]OptionInfo{
		"Ponder": {Name: "Ponder", Type: OptionTypeCheck},
	})

	err := WithSpinOption("Ponder", 1)(c)

	var mismatch *ErrOptionTypeMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, OptionTypeSpin, mismatch.Expected)
	assert.Equal(t, OptionTypeCheck, mismatch.Got)
}

func TestWithSpinOption_OutOfRange(t *testing.T) {
	c := buildTestClientWithOptions(map[string]OptionInfo{
		"Threads": {Name: "Threads", Type: OptionTypeSpin, Min: 1, Max: 8},
	})

	err := WithSpinOption("Threads", 100)(c)

	var oor *ErrOptionOutOfRange
	require.ErrorAs(t, err, &oor)
	assert.Equal(t, 100, oor.Value)
	assert.Equal(t, 1, oor.Min)
	assert.Equal(t, 8, oor.Max)
}

// --- WithCheckOption ---

func TestWithCheckOption_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Ponder": {Name: "Ponder", Type: OptionTypeCheck},
	})

	err := WithCheckOption("Ponder", true)(c)
	require.NoError(t, err)
	assert.Equal(t, []string{"setoption name Ponder value true"}, cw.sentLines())
}

func TestWithCheckOption_NotFound(t *testing.T) {
	c := buildTestClientWithOptions(nil)

	err := WithCheckOption("Ponder", true)(c)

	var notFound *ErrOptionNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestWithCheckOption_TypeMismatch(t *testing.T) {
	c := buildTestClientWithOptions(map[string]OptionInfo{
		"Threads": {Name: "Threads", Type: OptionTypeSpin, Min: 1, Max: 1024},
	})

	err := WithCheckOption("Threads", true)(c)

	var mismatch *ErrOptionTypeMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, OptionTypeCheck, mismatch.Expected)
	assert.Equal(t, OptionTypeSpin, mismatch.Got)
}

// --- WithComboOption ---

func TestWithComboOption_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"NumaPolicy": {Name: "NumaPolicy", Type: OptionTypeCombo, Vars: []string{"auto", "none"}},
	})

	err := WithComboOption("NumaPolicy", "auto")(c)
	require.NoError(t, err)
	assert.Equal(t, []string{"setoption name NumaPolicy value auto"}, cw.sentLines())
}

func TestWithComboOption_InvalidValue(t *testing.T) {
	c := buildTestClientWithOptions(map[string]OptionInfo{
		"NumaPolicy": {Name: "NumaPolicy", Type: OptionTypeCombo, Vars: []string{"auto", "none"}},
	})

	err := WithComboOption("NumaPolicy", "bogus")(c)

	var invalid *ErrOptionInvalidValue
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "bogus", invalid.Value)
	assert.Equal(t, []string{"auto", "none"}, invalid.Allowed)
}

func TestWithComboOption_TypeMismatch(t *testing.T) {
	c := buildTestClientWithOptions(map[string]OptionInfo{
		"Ponder": {Name: "Ponder", Type: OptionTypeCheck},
	})

	err := WithComboOption("Ponder", "auto")(c)

	var mismatch *ErrOptionTypeMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, OptionTypeCombo, mismatch.Expected)
}

// --- WithStringOption ---

func TestWithStringOption_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"SyzygyPath": {Name: "SyzygyPath", Type: OptionTypeString},
	})

	err := WithStringOption("SyzygyPath", "/tb")(c)
	require.NoError(t, err)
	assert.Equal(t, []string{"setoption name SyzygyPath value /tb"}, cw.sentLines())
}

func TestWithStringOption_TypeMismatch(t *testing.T) {
	c := buildTestClientWithOptions(map[string]OptionInfo{
		"Threads": {Name: "Threads", Type: OptionTypeSpin, Min: 1, Max: 1024},
	})

	err := WithStringOption("Threads", "4")(c)

	var mismatch *ErrOptionTypeMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, OptionTypeString, mismatch.Expected)
}

func TestWithStringOption_NewlineRejected(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"LF in value", "/tb\n/evil"},
		{"CR in value", "/tb\r/evil"},
		{"CRLF in value", "/tb\r\n/evil"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := buildTestClientWithOptions(map[string]OptionInfo{
				"SyzygyPath": {Name: "SyzygyPath", Type: OptionTypeString},
			})

			err := WithStringOption("SyzygyPath", tc.value)(c)
			assert.Error(t, err)
			assert.ErrorContains(t, err, "invalid characters")
		})
	}
}

// --- WithButtonOption ---

func TestWithButtonOption_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Clear Hash": {Name: "Clear Hash", Type: OptionTypeButton},
	})

	err := WithButtonOption("Clear Hash")(c)
	require.NoError(t, err)
	assert.Equal(t, []string{"setoption name Clear Hash"}, cw.sentLines())
}

func TestWithButtonOption_TypeMismatch(t *testing.T) {
	c := buildTestClientWithOptions(map[string]OptionInfo{
		"Threads": {Name: "Threads", Type: OptionTypeSpin, Min: 1, Max: 1024},
	})

	err := WithButtonOption("Threads")(c)

	var mismatch *ErrOptionTypeMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, OptionTypeButton, mismatch.Expected)
}

// --- Named helpers delegate to generic constructors ---

func TestWithThreads_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Threads": {Name: "Threads", Type: OptionTypeSpin, Min: 1, Max: 1024},
	})
	require.NoError(t, WithThreads(4)(c))
	assert.Equal(t, []string{"setoption name Threads value 4"}, cw.sentLines())
}

func TestWithHash_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Hash": {Name: "Hash", Type: OptionTypeSpin, Min: 1, Max: 33554432},
	})
	require.NoError(t, WithHash(256)(c))
	assert.Equal(t, []string{"setoption name Hash value 256"}, cw.sentLines())
}

func TestWithPonder_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Ponder": {Name: "Ponder", Type: OptionTypeCheck},
	})
	require.NoError(t, WithPonder(true)(c))
	assert.Equal(t, []string{"setoption name Ponder value true"}, cw.sentLines())
}

func TestWithMultiPV_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"MultiPV": {Name: "MultiPV", Type: OptionTypeSpin, Min: 1, Max: 500},
	})
	require.NoError(t, WithMultiPV(3)(c))
	assert.Equal(t, []string{"setoption name MultiPV value 3"}, cw.sentLines())
}

func TestWithSkillLevel_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Skill Level": {Name: "Skill Level", Type: OptionTypeSpin, Min: 0, Max: 20},
	})
	require.NoError(t, WithSkillLevel(10)(c))
	assert.Equal(t, []string{"setoption name Skill Level value 10"}, cw.sentLines())
}

func TestWithMoveOverhead_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Move Overhead": {Name: "Move Overhead", Type: OptionTypeSpin, Min: 0, Max: 5000},
	})
	require.NoError(t, WithMoveOverhead(50)(c))
	assert.Equal(t, []string{"setoption name Move Overhead value 50"}, cw.sentLines())
}

func TestWithClearHash_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"Clear Hash": {Name: "Clear Hash", Type: OptionTypeButton},
	})
	require.NoError(t, WithClearHash()(c))
	assert.Equal(t, []string{"setoption name Clear Hash"}, cw.sentLines())
}

func TestWithUCIChess960_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"UCI_Chess960": {Name: "UCI_Chess960", Type: OptionTypeCheck},
	})
	require.NoError(t, WithUCIChess960(true)(c))
	assert.Equal(t, []string{"setoption name UCI_Chess960 value true"}, cw.sentLines())
}

func TestWithSyzygyPath_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"SyzygyPath": {Name: "SyzygyPath", Type: OptionTypeString},
	})
	require.NoError(t, WithSyzygyPath("/tb")(c))
	assert.Equal(t, []string{"setoption name SyzygyPath value /tb"}, cw.sentLines())
}

func TestWithUCIAnalyseMode_Valid(t *testing.T) {
	c, cw := buildCapturingClientWithOptions(map[string]OptionInfo{
		"UCI_AnalyseMode": {Name: "UCI_AnalyseMode", Type: OptionTypeCheck},
	})
	require.NoError(t, WithUCIAnalyseMode(true)(c))
	assert.Equal(t, []string{"setoption name UCI_AnalyseMode value true"}, cw.sentLines())
}
