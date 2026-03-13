package stockfish

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGoCommand(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		params SearchParams
	}{
		{
			name:   "empty params",
			params: SearchParams{},
			want:   "go",
		},
		{
			name:   "depth only",
			params: SearchParams{Depth: 10},
			want:   "go depth 10",
		},
		{
			name:   "infinite",
			params: SearchParams{Infinite: true},
			want:   "go infinite",
		},
		{
			name:   "movetime",
			params: SearchParams{MoveTime: 500 * time.Millisecond},
			want:   "go movetime 500",
		},
		{
			name:   "nodes",
			params: SearchParams{Nodes: 1000},
			want:   "go nodes 1000",
		},
		{
			name:   "mate",
			params: SearchParams{Mate: 3},
			want:   "go mate 3",
		},
		{
			name:   "perft",
			params: SearchParams{Perft: 5},
			want:   "go perft 5",
		},
		{
			name:   "ponder",
			params: SearchParams{Ponder: true},
			want:   "go ponder",
		},
		{
			name: "time controls",
			params: SearchParams{
				WTime:     5 * time.Minute,
				BTime:     4 * time.Minute,
				WInc:      2 * time.Second,
				BInc:      2 * time.Second,
				MovesToGo: 20,
			},
			want: "go wtime 300000 btime 240000 winc 2000 binc 2000 movestogo 20",
		},
		{
			name: "searchmoves",
			params: SearchParams{
				SearchMoves: []string{"e2e4", "d2d4"},
				Depth:       5,
			},
			want: "go searchmoves e2e4 d2d4 depth 5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildGoCommand(&tc.params)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseInfoLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    SearchInfo
		wantErr bool
	}{
		{
			name: "full info line",
			line: "info depth 10 seldepth 13 multipv 1 score cp 25 nodes 20978 nps 209780 hashfull 10 tbhits 0 time 100 pv e2e4 c7c5",
			want: SearchInfo{
				Depth:    10,
				SelDepth: 13,
				MultiPV:  1,
				Score:    Score{Type: ScoreTypeCentipawns, Value: 25},
				Nodes:    20978,
				NPS:      209780,
				HashFull: 10,
				TBHits:   0,
				Time:     100,
				PV:       []string{"e2e4", "c7c5"},
			},
		},
		{
			name: "mate score",
			line: "info depth 1 seldepth 1 multipv 1 score mate 1 nodes 31 nps 10333 hashfull 0 tbhits 0 time 3 pv d8h4",
			want: SearchInfo{
				Depth:    1,
				SelDepth: 1,
				MultiPV:  1,
				Score:    Score{Type: ScoreTypeMate, Value: 1},
				Nodes:    31,
				NPS:      10333,
				Time:     3,
				PV:       []string{"d8h4"},
			},
		},
		{
			name: "wdl present",
			line: "info depth 1 seldepth 1 multipv 1 score cp 18 wdl 22 974 4 nodes 20 nps 10000 hashfull 0 tbhits 0 time 2 pv e2e4",
			want: SearchInfo{
				Depth:    1,
				SelDepth: 1,
				MultiPV:  1,
				Score:    Score{Type: ScoreTypeCentipawns, Value: 18},
				WDL:      &WDL{Win: 22, Draw: 974, Loss: 4},
				Nodes:    20,
				NPS:      10000,
				Time:     2,
				PV:       []string{"e2e4"},
			},
		},
		{
			name: "upperbound",
			line: "info depth 7 seldepth 10 multipv 1 score cp 54 upperbound nodes 1003 nps 38576 hashfull 0 tbhits 0 time 26 pv e2e4",
			want: SearchInfo{
				Depth:    7,
				SelDepth: 10,
				MultiPV:  1,
				Score:    Score{Type: ScoreTypeUpperBound, Value: 54},
				Nodes:    1003,
				NPS:      38576,
				Time:     26,
				PV:       []string{"e2e4"},
			},
		},
		{
			name: "string line skipped",
			line: "info string NNUE evaluation using nn-abc.nnue enabled",
			want: SearchInfo{},
		},
		{
			name:    "not an info line",
			line:    "bestmove e2e4",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseInfoLine(tc.line)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseBestMoveLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    SearchInfo
		wantErr bool
	}{
		{
			name: "bestmove with ponder",
			line: "bestmove e2e4 ponder e7e6",
			want: SearchInfo{
				IsBestMove: true,
				BestMove:   "e2e4",
				PonderMove: "e7e6",
			},
		},
		{
			name: "bestmove no ponder",
			line: "bestmove d2d4",
			want: SearchInfo{
				IsBestMove: true,
				BestMove:   "d2d4",
			},
		},
		{
			name:    "invalid line",
			line:    "info depth 5",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseBestMoveLine(tc.line)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSearchState(t *testing.T) {
	t.Run("start and finish", func(t *testing.T) {
		s := newSearchState()

		assert.False(t, s.isActive())

		ctx := t.Context()
		ch, _, err := s.start(ctx)

		require.NoError(t, err)
		assert.NotNil(t, ch)
		assert.True(t, s.isActive())

		s.finish()
		assert.False(t, s.isActive())

		// Channel should be closed.
		_, ok := <-ch
		assert.False(t, ok)
	})

	t.Run("double start returns error", func(t *testing.T) {
		s := newSearchState()
		ctx := t.Context()

		_, _, err := s.start(ctx)
		require.NoError(t, err)

		_, _, err2 := s.start(ctx)
		assert.ErrorIs(t, err2, ErrSearchInProgress)

		s.finish()
	})

	t.Run("double finish is safe", func(t *testing.T) {
		s := newSearchState()
		ctx := t.Context()

		_, _, err := s.start(ctx)
		require.NoError(t, err)

		s.finish()
		assert.NotPanics(t, s.finish)
	})
}
