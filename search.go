package stockfish

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ScoreType indicates whether the score is in centipawns or mate distance.
type ScoreType string

const (
	// ScoreTypeCentipawns means the score is in centipawns.
	ScoreTypeCentipawns ScoreType = "cp"
	// ScoreTypeMate means the score is mate-in-N (negative = being mated).
	ScoreTypeMate ScoreType = "mate"
)

// ScoreBound indicates whether the score is an exact value or a bound.
type ScoreBound string

const (
	// ScoreBoundExact means the score is the exact minimax value.
	ScoreBoundExact ScoreBound = ""
	// ScoreBoundLower means the score is a lower bound (fail-high) on the true value.
	ScoreBoundLower ScoreBound = "lowerbound"
	// ScoreBoundUpper means the score is an upper bound (fail-low) on the true value.
	ScoreBoundUpper ScoreBound = "upperbound"
)

// Score holds the evaluation score from a search info line.
type Score struct {
	// Type indicates whether Value is in centipawns or mate-in-N.
	Type ScoreType
	// Bound indicates whether the score is exact, a lower bound, or an upper bound.
	// ScoreBoundExact (empty string) means the score is the exact minimax value.
	Bound ScoreBound
	// Value is the numeric score (centipawns or mate-in-N).
	Value int
}

// WDL holds the Win/Draw/Loss statistics from a search info line (requires
// UCI_ShowWDL to be enabled).
type WDL struct {
	Win  int
	Draw int
	Loss int
}

// SearchInfo represents a single info line (or the final bestmove line) emitted
// by the engine during a search.
type SearchInfo struct {
	WDL            *WDL
	BestMove       string
	PonderMove     string
	CurrMove       string
	Score          Score
	PV             []string
	NPS            int64
	Nodes          int64
	HashFull       int64
	TBHits         int64
	Time           int64
	Depth          int
	SelDepth       int
	MultiPV        int
	CurrMoveNumber int
	IsBestMove     bool
}

// SearchParams configures a "go" UCI command. At least one limiting parameter
// should be set; if none are set the engine will search with depth 245 (its
// internal default for unlimited search).
type SearchParams struct {
	// SearchMoves restricts the search to only these moves (long algebraic notation).
	SearchMoves []string
	// WTime is the time White has remaining on the clock.
	WTime time.Duration
	// BTime is the time Black has remaining on the clock.
	BTime time.Duration
	// WInc is White's increment per move.
	WInc time.Duration
	// BInc is Black's increment per move.
	BInc time.Duration
	// MoveTime limits search to this duration.
	MoveTime time.Duration
	// MovesToGo is the number of moves remaining until the next time control.
	MovesToGo int
	// Depth limits search to this ply depth.
	Depth int
	// Nodes limits search to approximately this many nodes.
	Nodes int
	// Mate stops the search when a mate in this many moves is found.
	Mate int
	// Perft runs perft to this depth instead of a normal search.
	Perft int
	// Ponder starts the search in pondering mode.
	Ponder bool
	// Infinite starts an unbounded search; only stops on an explicit Stop().
	Infinite bool
}

// buildGoCommand converts SearchParams into a UCI "go" command string.
func buildGoCommand(p *SearchParams) string {
	var sb strings.Builder

	sb.WriteString("go")

	if len(p.SearchMoves) > 0 {
		sb.WriteString(" searchmoves ")
		sb.WriteString(strings.Join(p.SearchMoves, " "))
	}

	if p.Ponder {
		sb.WriteString(" ponder")
	}

	if p.WTime > 0 {
		fmt.Fprintf(&sb, " wtime %d", p.WTime.Milliseconds())
	}

	if p.BTime > 0 {
		fmt.Fprintf(&sb, " btime %d", p.BTime.Milliseconds())
	}

	if p.WInc > 0 {
		fmt.Fprintf(&sb, " winc %d", p.WInc.Milliseconds())
	}

	if p.BInc > 0 {
		fmt.Fprintf(&sb, " binc %d", p.BInc.Milliseconds())
	}

	if p.MovesToGo > 0 {
		fmt.Fprintf(&sb, " movestogo %d", p.MovesToGo)
	}

	if p.Depth > 0 {
		fmt.Fprintf(&sb, " depth %d", p.Depth)
	}

	if p.Nodes > 0 {
		fmt.Fprintf(&sb, " nodes %d", p.Nodes)
	}

	if p.Mate > 0 {
		fmt.Fprintf(&sb, " mate %d", p.Mate)
	}

	if p.MoveTime > 0 {
		fmt.Fprintf(&sb, " movetime %d", p.MoveTime.Milliseconds())
	}

	if p.Infinite {
		sb.WriteString(" infinite")
	}

	if p.Perft > 0 {
		fmt.Fprintf(&sb, " perft %d", p.Perft)
	}

	return sb.String()
}

// parseInfoLine parses a UCI "info …" line into a SearchInfo struct.
// Fields that are absent in the line are left at their zero values.
func parseInfoLine(line string) (SearchInfo, error) {
	rest, ok := strings.CutPrefix(line, "info ")
	if !ok {
		return SearchInfo{}, &ErrUnexpectedResponse{Line: line}
	}

	// Skip pure string info lines (e.g. "info string NNUE evaluation …")
	if strings.HasPrefix(rest, "string ") {
		return SearchInfo{}, nil
	}

	fields := strings.Fields(rest)

	var info SearchInfo

	for i := 0; i < len(fields); i++ {
		var err error

		i, err = parseInfoField(fields, i, &info)
		if err != nil {
			return info, err
		}
	}

	return info, nil
}

// parseInfoField handles a single keyword token at position i within fields,
// updating info in place. It returns the new index after consuming any
// associated value tokens.
func parseInfoField(fields []string, i int, info *SearchInfo) (int, error) {
	switch fields[i] {
	case "depth":
		return parseInfoInt(fields, i, func(v int) { info.Depth = v })
	case "seldepth":
		return parseInfoInt(fields, i, func(v int) { info.SelDepth = v })
	case "multipv":
		return parseInfoInt(fields, i, func(v int) { info.MultiPV = v })
	case "currmove":
		if i+1 < len(fields) {
			info.CurrMove = fields[i+1]
			return i + 1, nil
		}
	case "currmovenumber":
		return parseInfoInt(fields, i, func(v int) { info.CurrMoveNumber = v })
	case "nodes":
		return parseInfoInt64(fields, i, func(v int64) { info.Nodes = v })
	case "nps":
		return parseInfoInt64(fields, i, func(v int64) { info.NPS = v })
	case "hashfull":
		return parseInfoInt64(fields, i, func(v int64) { info.HashFull = v })
	case "tbhits":
		return parseInfoInt64(fields, i, func(v int64) { info.TBHits = v })
	case "time":
		return parseInfoInt64(fields, i, func(v int64) { info.Time = v })
	case "score":
		parsed, consumed, err := parseScore(fields[i+1:])
		if err != nil {
			return i, fmt.Errorf("parse score: %w", err)
		}

		info.Score = parsed

		return i + consumed, nil
	case "wdl":
		return parseInfoWDL(fields, i, info)
	case "pv":
		// PV runs to end of line.
		info.PV = fields[i+1:]

		return len(fields), nil
	}

	return i, nil
}

// parseInfoInt reads an integer value at fields[i+1], calls set, and returns
// the advanced index. Used for int-typed info fields.
func parseInfoInt(fields []string, i int, set func(int)) (int, error) {
	if i+1 >= len(fields) {
		return i, nil
	}

	v, err := strconv.Atoi(fields[i+1])
	if err != nil {
		return i, fmt.Errorf("parse %s: %w", fields[i], err)
	}

	set(v)

	return i + 1, nil
}

// parseInfoInt64 reads an int64 value at fields[i+1], calls set, and returns
// the advanced index. Used for int64-typed info fields.
func parseInfoInt64(fields []string, i int, set func(int64)) (int, error) {
	if i+1 >= len(fields) {
		return i, nil
	}

	v, err := strconv.ParseInt(fields[i+1], 10, 64)
	if err != nil {
		return i, fmt.Errorf("parse %s: %w", fields[i], err)
	}

	set(v)

	return i + 1, nil
}

// parseInfoWDL reads three integer tokens after a "wdl" keyword and populates
// info.WDL. Returns the advanced index.
func parseInfoWDL(fields []string, i int, info *SearchInfo) (int, error) {
	if i+3 >= len(fields) {
		return i, nil
	}

	w, err := strconv.Atoi(fields[i+1])
	if err != nil {
		return i, fmt.Errorf("parse wdl win: %w", err)
	}

	d, err := strconv.Atoi(fields[i+2])
	if err != nil {
		return i, fmt.Errorf("parse wdl draw: %w", err)
	}

	l, err := strconv.Atoi(fields[i+3])
	if err != nil {
		return i, fmt.Errorf("parse wdl loss: %w", err)
	}

	info.WDL = &WDL{Win: w, Draw: d, Loss: l}

	return i + 3, nil
}

// parseScore parses the tokens following "score" in a UCI info line. Returns
// the Score and the number of tokens consumed (not counting the "score" token
// itself).
func parseScore(tokens []string) (Score, int, error) {
	if len(tokens) == 0 {
		return Score{}, 0, fmt.Errorf("empty score tokens")
	}

	var s Score

	consumed := 0

	switch tokens[0] {
	case "cp", "mate":
		s.Type = ScoreType(tokens[0])
		consumed++

		if len(tokens) > 1 {
			v, err := strconv.Atoi(tokens[1])
			if err != nil {
				return s, consumed, fmt.Errorf("parse score value %q: %w", tokens[1], err)
			}

			s.Value = v
			consumed++
		}
	default:
		return Score{}, 0, fmt.Errorf("unknown score type %q", tokens[0])
	}

	// Optional bound qualifiers.
	for consumed < len(tokens) {
		switch tokens[consumed] {
		case "lowerbound":
			s.Bound = ScoreBoundLower
			consumed++

			continue
		case "upperbound":
			s.Bound = ScoreBoundUpper
			consumed++

			continue
		}

		break
	}

	return s, consumed, nil
}

// parseBestMoveLine parses a "bestmove <move> [ponder <move>]" line and returns
// a SearchInfo with IsBestMove=true.
func parseBestMoveLine(line string) (SearchInfo, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 || fields[0] != "bestmove" {
		return SearchInfo{}, &ErrUnexpectedResponse{Line: line}
	}

	info := SearchInfo{
		IsBestMove: true,
		BestMove:   fields[1],
	}

	if len(fields) >= 4 && fields[2] == "ponder" {
		info.PonderMove = fields[3]
	}

	return info, nil
}

// searchState manages the active search goroutine and its result channel.
type searchState struct {
	ch     chan SearchInfo
	cancel context.CancelFunc
	mu     sync.Mutex
	active bool
}

func newSearchState() *searchState {
	return &searchState{}
}

// start marks a search as active and returns the channel to send results on and
// a cancel function. Returns ErrSearchInProgress if already active.
func (s *searchState) start(ctx context.Context) (chan SearchInfo, context.Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return nil, nil, ErrSearchInProgress
	}

	ch := make(chan SearchInfo, 32)
	ctx, cancel := context.WithCancel(ctx)
	s.ch = ch
	s.cancel = cancel
	s.active = true

	return ch, ctx, nil
}

// finish marks the search as complete and closes the result channel.
func (s *searchState) finish() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false

	close(s.ch)

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// isActive reports whether a search is currently running.
func (s *searchState) isActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.active
}
