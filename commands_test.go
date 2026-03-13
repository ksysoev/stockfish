package stockfish

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildBenchCommand(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		params BenchParams
	}{
		{
			name:   "defaults",
			params: BenchParams{},
			want:   "bench 16 1 13 default depth",
		},
		{
			name: "custom values",
			params: BenchParams{
				TTSize:    4096,
				Threads:   8,
				Limit:     1,
				FENFile:   "current",
				LimitType: "perft",
			},
			want: "bench 4096 8 1 current perft",
		},
		{
			name: "partial override",
			params: BenchParams{
				TTSize: 64,
			},
			want: "bench 64 1 13 default depth",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildBenchCommand(tc.params)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBenchTerminator(t *testing.T) {
	assert.True(t, benchTerminator("==========================="))
	assert.True(t, benchTerminator("==="))
	assert.False(t, benchTerminator("Total time (ms) : 2"))
	assert.False(t, benchTerminator(""))
}
