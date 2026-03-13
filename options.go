package stockfish

import (
	"fmt"
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

// Option describes a single UCI engine option as reported during initialisation.
type Option struct {
	Name    string
	Type    OptionType
	Default string
	Vars    []string
	Min     int
	Max     int
}

// parseOption parses a single "option name … type …" line from the engine's UCI
// output and returns the corresponding Option, or an error if the line is
// malformed.
func parseOption(line string) (Option, error) {
	// Strip the leading "option " prefix.
	rest, ok := strings.CutPrefix(line, "option ")
	if !ok {
		return Option{}, &ErrUnexpectedResponse{Line: line}
	}

	var opt Option

	// Extract "name <id>"
	nameIdx := strings.Index(rest, "name ")
	typeIdx := strings.Index(rest, " type ")

	if nameIdx < 0 || typeIdx < 0 {
		return Option{}, &ErrUnexpectedResponse{Line: line}
	}

	opt.Name = strings.TrimSpace(rest[nameIdx+5 : typeIdx])

	// Everything after " type "
	after := rest[typeIdx+6:]
	fields := strings.Fields(after)

	if len(fields) == 0 {
		return Option{}, &ErrUnexpectedResponse{Line: line}
	}

	opt.Type = OptionType(fields[0])

	// Parse remaining key-value tokens.
	for i := 1; i < len(fields); i++ {
		switch fields[i] {
		case "default":
			// Default value may be multi-word (e.g. empty string is just "default" with nothing after)
			defaultParts := collectUntilKeyword(fields[i+1:], []string{"min", "max", "var"})
			opt.Default = strings.Join(defaultParts, " ")
			i += len(defaultParts)

		case "min":
			if i+1 < len(fields) {
				v, err := strconv.Atoi(fields[i+1])
				if err != nil {
					return Option{}, fmt.Errorf("parse option min value %q: %w", fields[i+1], err)
				}

				opt.Min = v
				i++
			}

		case "max":
			if i+1 < len(fields) {
				v, err := strconv.Atoi(fields[i+1])
				if err != nil {
					return Option{}, fmt.Errorf("parse option max value %q: %w", fields[i+1], err)
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

// buildSetOption constructs the UCI "setoption" command string.
// For button-type options, value may be empty.
func buildSetOption(name, value string) string {
	if value == "" {
		return fmt.Sprintf("setoption name %s", name)
	}

	return fmt.Sprintf("setoption name %s value %s", name, value)
}
