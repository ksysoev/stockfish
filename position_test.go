package stockfish

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartPosition(t *testing.T) {
	pos := StartPosition()
	assert.True(t, pos.StartPos)
	assert.Empty(t, pos.FEN)
	assert.Empty(t, pos.Moves)
}

func TestFENPosition(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	pos := FENPosition(fen)
	assert.False(t, pos.StartPos)
	assert.Equal(t, fen, pos.FEN)
	assert.Empty(t, pos.Moves)
}

func TestPositionWithMoves(t *testing.T) {
	pos := StartPosition().WithMoves("e2e4", "e7e5")
	assert.True(t, pos.StartPos)
	assert.Equal(t, []string{"e2e4", "e7e5"}, pos.Moves)

	// WithMoves should not mutate the original.
	pos2 := pos.WithMoves("g1f3")
	assert.Equal(t, []string{"e2e4", "e7e5"}, pos.Moves)
	assert.Equal(t, []string{"e2e4", "e7e5", "g1f3"}, pos2.Moves)
}

func TestBuildPositionCommand(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		pos     Position
		wantErr bool
	}{
		{
			name: "startpos no moves",
			pos:  StartPosition(),
			want: "position startpos",
		},
		{
			name: "startpos with moves",
			pos:  StartPosition().WithMoves("e2e4", "e7e5"),
			want: "position startpos moves e2e4 e7e5",
		},
		{
			name: "FEN no moves",
			pos:  FENPosition("8/8/8/8/8/8/8/8 w - - 0 1"),
			want: "position fen 8/8/8/8/8/8/8/8 w - - 0 1",
		},
		{
			name: "FEN with moves",
			pos:  FENPosition("8/8/8/8/8/8/8/8 w - - 0 1").WithMoves("a1a2"),
			want: "position fen 8/8/8/8/8/8/8/8 w - - 0 1 moves a1a2",
		},
		{
			name:    "empty position",
			pos:     Position{},
			wantErr: true,
		},
		{
			name:    "both StartPos and FEN set",
			pos:     Position{StartPos: true, FEN: "8/8/8/8/8/8/8/8 w - - 0 1"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildPositionCommand(tc.pos)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
