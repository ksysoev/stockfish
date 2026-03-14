package stockfish

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// OptionType represents the UCI option type as reported by the engine.
type OptionType string

const (
	// OptionTypeSpin represents a numeric option with min/max bounds.
	OptionTypeSpin OptionType = "spin"
	// OptionTypeCheck represents a boolean (true/false) option.
	OptionTypeCheck OptionType = "check"
	// OptionTypeString represents a free-form string option.
	OptionTypeString OptionType = "string"
	// OptionTypeButton represents a button option with no value.
	OptionTypeButton OptionType = "button"
	// OptionTypeCombo represents an option with a fixed set of allowed values.
	OptionTypeCombo OptionType = "combo"
)

// OptionInfo describes a single UCI engine option as reported during
// initialisation.
type OptionInfo struct {
	Name    string
	Type    OptionType
	Default string
	Vars    []string
	Min     int
	Max     int
}

// Option is a functional option that configures a Client by sending a
// "setoption" UCI command. It validates the value against the engine's
// reported metadata before sending.
type Option func(*Client) error

// WithThreads returns an Option that sets the "Threads" spin option.
// The value must be within the engine's reported [Min, Max] range.
func WithThreads(n int) Option {
	return WithSpinOption("Threads", n)
}

// WithHash returns an Option that sets the "Hash" spin option (hash table size
// in MB). The value must be within the engine's reported [Min, Max] range.
func WithHash(mb int) Option {
	return WithSpinOption("Hash", mb)
}

// WithPonder returns an Option that sets the "Ponder" check option.
func WithPonder(v bool) Option {
	return WithCheckOption("Ponder", v)
}

// WithMultiPV returns an Option that sets the "MultiPV" spin option (number of
// principal variations to search). The value must be within the engine's
// reported [Min, Max] range.
func WithMultiPV(n int) Option {
	return WithSpinOption("MultiPV", n)
}

// WithSkillLevel returns an Option that sets the "Skill Level" spin option.
// The value must be within the engine's reported [Min, Max] range.
func WithSkillLevel(n int) Option {
	return WithSpinOption("Skill Level", n)
}

// WithMoveOverhead returns an Option that sets the "Move Overhead" spin option
// (time in ms to reserve per move for network/GUI overhead). The value must be
// within the engine's reported [Min, Max] range.
func WithMoveOverhead(ms int) Option {
	return WithSpinOption("Move Overhead", ms)
}

// WithClearHash returns an Option that triggers the "Clear Hash" button option,
// clearing the engine's transposition table.
func WithClearHash() Option {
	return WithButtonOption("Clear Hash")
}

// WithUCIChess960 returns an Option that sets the "UCI_Chess960" check option.
func WithUCIChess960(v bool) Option {
	return WithCheckOption("UCI_Chess960", v)
}

// WithSyzygyPath returns an Option that sets the "SyzygyPath" string option.
func WithSyzygyPath(path string) Option {
	return WithStringOption("SyzygyPath", path)
}

// WithUCIAnalyseMode returns an Option that sets the "UCI_AnalyseMode" check
// option, enabling analysis mode in the engine.
func WithUCIAnalyseMode(v bool) Option {
	return WithCheckOption("UCI_AnalyseMode", v)
}

// containsNewline reports whether s contains a CR or LF character, which would
// allow injecting additional UCI commands into the engine's stdin stream.
func containsNewline(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

// WithSpinOption returns an Option that sets any spin-type UCI option by name.
// It validates that:
//   - the option exists on the engine,
//   - the option is of type spin,
//   - value is within the engine's [Min, Max] bounds.
func WithSpinOption(name string, value int) Option {
	return func(c *Client) error {
		info, ok := c.options[name]
		if !ok {
			return &ErrOptionNotFound{Name: name}
		}

		if info.Type != OptionTypeSpin {
			return &ErrOptionTypeMismatch{Name: name, Expected: OptionTypeSpin, Got: info.Type}
		}

		if value < info.Min || value > info.Max {
			return &ErrOptionOutOfRange{Name: name, Value: value, Min: info.Min, Max: info.Max}
		}

		cmd := fmt.Sprintf("setoption name %s value %d", name, value)

		if err := c.eng.send(cmd); err != nil {
			return fmt.Errorf("send setoption %q: %w", name, err)
		}

		return nil
	}
}

// WithCheckOption returns an Option that sets any check-type UCI option by
// name. It validates that the option exists and is of type check.
func WithCheckOption(name string, value bool) Option {
	return func(c *Client) error {
		info, ok := c.options[name]
		if !ok {
			return &ErrOptionNotFound{Name: name}
		}

		if info.Type != OptionTypeCheck {
			return &ErrOptionTypeMismatch{Name: name, Expected: OptionTypeCheck, Got: info.Type}
		}

		cmd := fmt.Sprintf("setoption name %s value %t", name, value)

		if err := c.eng.send(cmd); err != nil {
			return fmt.Errorf("send setoption %q: %w", name, err)
		}

		return nil
	}
}

// WithComboOption returns an Option that sets any combo-type UCI option by
// name. It validates that:
//   - the option exists on the engine,
//   - the option is of type combo,
//   - value is one of the engine's reported Vars.
func WithComboOption(name, value string) Option {
	return func(c *Client) error {
		info, ok := c.options[name]
		if !ok {
			return &ErrOptionNotFound{Name: name}
		}

		if info.Type != OptionTypeCombo {
			return &ErrOptionTypeMismatch{Name: name, Expected: OptionTypeCombo, Got: info.Type}
		}

		if !slices.Contains(info.Vars, value) {
			return &ErrOptionInvalidValue{Name: name, Value: value, Allowed: slices.Clone(info.Vars)}
		}

		cmd := fmt.Sprintf("setoption name %s value %s", name, value)

		if err := c.eng.send(cmd); err != nil {
			return fmt.Errorf("send setoption %q: %w", name, err)
		}

		return nil
	}
}

// WithStringOption returns an Option that sets any string-type UCI option by
// name. It validates that the option exists and is of type string.
//
// The value must not contain CR or LF characters, as these would allow
// injecting additional UCI commands into the engine's stdin stream.
func WithStringOption(name, value string) Option {
	return func(c *Client) error {
		if containsNewline(value) {
			return &ErrOptionInvalidCharacters{Name: name, Value: value}
		}

		info, ok := c.options[name]
		if !ok {
			return &ErrOptionNotFound{Name: name}
		}

		if info.Type != OptionTypeString {
			return &ErrOptionTypeMismatch{Name: name, Expected: OptionTypeString, Got: info.Type}
		}

		cmd := fmt.Sprintf("setoption name %s value %s", name, value)

		if err := c.eng.send(cmd); err != nil {
			return fmt.Errorf("send setoption %q: %w", name, err)
		}

		return nil
	}
}

// WithButtonOption returns an Option that triggers any button-type UCI option
// by name. It validates that the option exists and is of type button.
func WithButtonOption(name string) Option {
	return func(c *Client) error {
		info, ok := c.options[name]
		if !ok {
			return &ErrOptionNotFound{Name: name}
		}

		if info.Type != OptionTypeButton {
			return &ErrOptionTypeMismatch{Name: name, Expected: OptionTypeButton, Got: info.Type}
		}

		cmd := fmt.Sprintf("setoption name %s", name)

		if err := c.eng.send(cmd); err != nil {
			return fmt.Errorf("send setoption %q: %w", name, err)
		}

		return nil
	}
}

// parseOption parses a single "option name … type …" line from the engine's
// UCI output and returns the corresponding OptionInfo, or an error if the line
// is malformed.
func parseOption(line string) (OptionInfo, error) {
	// Strip the leading "option " prefix.
	rest, ok := strings.CutPrefix(line, "option ")
	if !ok {
		return OptionInfo{}, &ErrUnexpectedResponse{Line: line}
	}

	var opt OptionInfo

	// Extract "name <id>"
	nameIdx := strings.Index(rest, "name ")
	typeIdx := strings.Index(rest, " type ")

	if nameIdx < 0 || typeIdx < 0 {
		return OptionInfo{}, &ErrUnexpectedResponse{Line: line}
	}

	opt.Name = strings.TrimSpace(rest[nameIdx+5 : typeIdx])

	// Everything after " type "
	after := rest[typeIdx+6:]
	fields := strings.Fields(after)

	if len(fields) == 0 {
		return OptionInfo{}, &ErrUnexpectedResponse{Line: line}
	}

	opt.Type = OptionType(fields[0])

	// Parse remaining key-value tokens.
	for i := 1; i < len(fields); i++ {
		switch fields[i] {
		case "default":
			// Default value may be multi-word (e.g. empty string is just
			// "default" with nothing after).
			defaultParts := collectUntilKeyword(fields[i+1:], []string{"min", "max", "var"})
			opt.Default = strings.Join(defaultParts, " ")
			i += len(defaultParts)

		case "min":
			if i+1 < len(fields) {
				v, err := strconv.Atoi(fields[i+1])
				if err != nil {
					return OptionInfo{}, fmt.Errorf("parse option min value %q: %w", fields[i+1], err)
				}

				opt.Min = v
				i++
			}

		case "max":
			if i+1 < len(fields) {
				v, err := strconv.Atoi(fields[i+1])
				if err != nil {
					return OptionInfo{}, fmt.Errorf("parse option max value %q: %w", fields[i+1], err)
				}

				opt.Max = v
				i++
			}

		case "var":
			varParts := collectUntilKeyword(fields[i+1:], []string{"var"})
			opt.Vars = append(opt.Vars, strings.Join(varParts, " "))
			i += len(varParts)
		}
	}

	return opt, nil
}

// collectUntilKeyword returns the slice of tokens from tokens up to (but not
// including) the first occurrence of any keyword in stop.
func collectUntilKeyword(tokens, stop []string) []string {
	stopSet := make(map[string]struct{}, len(stop))

	for _, s := range stop {
		stopSet[s] = struct{}{}
	}

	var result []string

	for _, t := range tokens {
		if _, found := stopSet[t]; found {
			break
		}

		result = append(result, t)
	}

	return result
}
