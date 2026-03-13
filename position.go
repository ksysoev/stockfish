package stockfish

import (
	"strings"
)

// Position represents a chess position that can be sent to the engine using the
// "position" UCI command. Either StartPos or FEN must be set, but not both.
type Position struct {
	// FEN is the Forsyth-Edwards Notation string for the position.
	// Leave empty when StartPos is true.
	FEN string
	// Moves is the list of moves played from the base position in long
	// algebraic notation (e.g. "e2e4", "g1f3").
	Moves []string
	// StartPos indicates that the position is the standard chess starting
	// position. When true, FEN is ignored.
	StartPos bool
}

// StartPosition returns a Position set to the standard chess starting position
// with no moves applied.
func StartPosition() Position {
	return Position{StartPos: true}
}

// FENPosition returns a Position set to the given FEN string with no moves
// applied.
func FENPosition(fen string) Position {
	return Position{FEN: fen}
}

// WithMoves returns a copy of the position with the given moves appended.
func (p Position) WithMoves(moves ...string) Position {
	combined := make([]string, len(p.Moves)+len(moves))
	copy(combined, p.Moves)
	copy(combined[len(p.Moves):], moves)

	return Position{
		StartPos: p.StartPos,
		FEN:      p.FEN,
		Moves:    combined,
	}
}

// buildPositionCommand converts the Position into a UCI "position" command
// string.
func buildPositionCommand(pos Position) (string, error) {
	var sb strings.Builder

	sb.WriteString("position ")

	switch {
	case pos.StartPos:
		sb.WriteString("startpos")
	case pos.FEN != "":
		sb.WriteString("fen ")
		sb.WriteString(pos.FEN)
	default:
		return "", &ErrInvalidPosition{Detail: "either StartPos must be true or FEN must be non-empty"}
	}

	if len(pos.Moves) > 0 {
		sb.WriteString(" moves ")
		sb.WriteString(strings.Join(pos.Moves, " "))
	}

	return sb.String(), nil
}
