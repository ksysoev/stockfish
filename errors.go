// Package stockfish provides a Go wrapper for the Stockfish chess engine,
// implementing the Universal Chess Interface (UCI) protocol.
package stockfish

import (
	"errors"
	"fmt"
)

// ErrEngineNotRunning is returned when an operation is attempted on an engine
// that has not been started or has already been stopped.
var ErrEngineNotRunning = errors.New("engine is not running")

// ErrEngineTimeout is returned when the engine does not respond within the
// expected time frame.
var ErrEngineTimeout = errors.New("engine response timed out")

// ErrSearchInProgress is returned when a new search command is issued while
// another search is already running.
var ErrSearchInProgress = errors.New("search already in progress")

// ErrNoSearchInProgress is returned when Stop or PonderHit is called but no
// search is currently running.
var ErrNoSearchInProgress = errors.New("no search in progress")

// ErrUnexpectedResponse is returned when the engine returns output that does
// not conform to the expected UCI format.
type ErrUnexpectedResponse struct {
	Line string
}

// Error implements the error interface.
func (e *ErrUnexpectedResponse) Error() string {
	return fmt.Sprintf("unexpected engine response: %q", e.Line)
}

// ErrOptionNotFound is returned when an option name is not among the options
// reported by the engine during initialisation.
type ErrOptionNotFound struct {
	Name string
}

// Error implements the error interface.
func (e *ErrOptionNotFound) Error() string {
	return fmt.Sprintf("option not found: %q", e.Name)
}

// ErrOptionTypeMismatch is returned when a typed option constructor is applied
// to an option whose UCI type differs from the expected type.
type ErrOptionTypeMismatch struct {
	Name     string
	Expected OptionType
	Got      OptionType
}

// Error implements the error interface.
func (e *ErrOptionTypeMismatch) Error() string {
	return fmt.Sprintf("option %q type mismatch: expected %s, got %s", e.Name, e.Expected, e.Got)
}

// ErrOptionOutOfRange is returned when a spin option value falls outside the
// [Min, Max] bounds reported by the engine.
type ErrOptionOutOfRange struct {
	Name  string
	Value int
	Min   int
	Max   int
}

// Error implements the error interface.
func (e *ErrOptionOutOfRange) Error() string {
	return fmt.Sprintf("option %q value %d out of range [%d, %d]", e.Name, e.Value, e.Min, e.Max)
}

// ErrOptionInvalidValue is returned when a combo option value is not among the
// allowed variants reported by the engine.
type ErrOptionInvalidValue struct {
	Name    string
	Value   string
	Allowed []string
}

// Error implements the error interface.
func (e *ErrOptionInvalidValue) Error() string {
	return fmt.Sprintf("option %q value %q not in allowed set %v", e.Name, e.Value, e.Allowed)
}

// ErrInvalidPosition is returned when a malformed FEN string or invalid move
// is provided to SetPosition.
type ErrInvalidPosition struct {
	Detail string
}

// Error implements the error interface.
func (e *ErrInvalidPosition) Error() string {
	return fmt.Sprintf("invalid position: %s", e.Detail)
}
