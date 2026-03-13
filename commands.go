package stockfish

import (
	"fmt"
	"strings"
)

// BenchParams configures the "bench" non-standard UCI command.
type BenchParams struct {
	FENFile   string
	LimitType string
	TTSize    int
	Threads   int
	Limit     int
}

// buildBenchCommand constructs the UCI "bench" command string from BenchParams,
// filling in Stockfish defaults for any zero values.
func buildBenchCommand(p BenchParams) string {
	ttSize := p.TTSize
	if ttSize == 0 {
		ttSize = 16
	}

	threads := p.Threads
	if threads == 0 {
		threads = 1
	}

	limit := p.Limit
	if limit == 0 {
		limit = 13
	}

	fenFile := p.FENFile
	if fenFile == "" {
		fenFile = "default"
	}

	limitType := p.LimitType
	if limitType == "" {
		limitType = "depth"
	}

	return fmt.Sprintf("bench %d %d %d %s %s", ttSize, threads, limit, fenFile, limitType)
}

// benchTerminator returns true when the bench output has reached its summary
// section (the line of "=" characters).
func benchTerminator(line string) bool {
	return strings.HasPrefix(line, "===")
}
