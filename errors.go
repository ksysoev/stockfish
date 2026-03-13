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

// ErrInvalidOption is returned when an unknown or invalid option name is
// provided to SetOption.
type ErrInvalidOption struct {
	Name string
}

// Error implements the error interface.
func (e *ErrInvalidOption) Error() string {
	return fmt.Sprintf("invalid option: %q", e.Name)
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
