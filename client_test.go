package stockfish

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nopWriteCloser wraps io.Discard as an io.WriteCloser for test stubs.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// fakeProcess simulates the engine process for unit tests by providing
// pre-scripted responses to commands.
type fakeProcess struct {
	reader   *bufio.Reader
	pw       *strings.Builder
	output   []string
	received []string
}

func newFakeProcess(outputLines []string) *fakeProcess {
	combined := strings.Join(outputLines, "\n") + "\n"
	fp := &fakeProcess{
		output:   outputLines,
		received: nil,
		reader:   bufio.NewReader(strings.NewReader(combined)),
		pw:       &strings.Builder{},
	}

	return fp
}

// buildTestEngine creates a ready-to-use engine backed by a fakeProcess.
func buildTestEngine(outputLines []string) *engine {
	proc := newFakeProcess(outputLines)

	eng := &engine{
		lineCh: make(chan string, 64),
		errCh:  make(chan error, 1),
		done:   make(chan struct{}),
	}

	// Feed the lines directly into lineCh (bypasses the real read loop).
	go func() {
		defer close(eng.lineCh)
		defer close(eng.done)

		for _, line := range proc.output {
			if line != "" {
				eng.lineCh <- line
			}
		}
	}()

	// We still need a proc for writeLine so attach a stub with a no-op stdin.
	eng.proc = &process{stdin: nopWriteCloser{io.Discard}, reader: proc.reader}

	return eng
}

func TestNew_InvalidPath(t *testing.T) {
	_, err := New("/nonexistent/stockfish")
	assert.Error(t, err)
}

func TestClient_OptionsAndMeta(t *testing.T) {
	uciOutput := []string{
		"id name Stockfish 16",
		"id author the Stockfish developers",
		"option name Threads type spin default 1 min 1 max 1024",
		"option name Hash type spin default 16 min 1 max 33554432",
		"option name Ponder type check default false",
		"uciok",
		"readyok",
	}

	eng := buildTestEngine(uciOutput)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	err := c.initialize()
	require.NoError(t, err)

	assert.Equal(t, "Stockfish 16", c.Name())
	assert.Equal(t, "the Stockfish developers", c.Author())

	opts := c.Options()
	assert.Len(t, opts, 3)

	threads, ok := opts["Threads"]
	require.True(t, ok)
	assert.Equal(t, OptionTypeSpin, threads.Type)
	assert.Equal(t, 1, threads.Min)
	assert.Equal(t, 1024, threads.Max)
}

func TestClient_SetPosition_Error(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	err := c.SetPosition(Position{})
	assert.Error(t, err)
}

func TestClient_SetPosition_BothStartPosAndFEN(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	err := c.SetPosition(Position{StartPos: true, FEN: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"})

	var posErr *ErrInvalidPosition

	assert.ErrorAs(t, err, &posErr)
}

func TestClient_Stop_NoSearch(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
	}

	err := c.Stop()
	assert.ErrorIs(t, err, ErrNoSearchInProgress)
}

func TestClient_PonderHit_NoSearch(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
	}

	err := c.PonderHit()
	assert.ErrorIs(t, err, ErrNoSearchInProgress)
}

func TestClient_Go_StreamsResults(t *testing.T) {
	searchLines := []string{
		"info depth 1 seldepth 1 multipv 1 score cp 17 nodes 20 nps 20000 hashfull 0 tbhits 0 time 1 pv e2e4",
		"info depth 2 seldepth 3 multipv 1 score cp 34 nodes 45 nps 22500 hashfull 0 tbhits 0 time 2 pv e2e4",
		"bestmove e2e4 ponder d7d6",
	}

	eng := buildTestEngine(searchLines)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	ctx := context.Background()
	ch, err := c.Go(ctx, &SearchParams{Depth: 2})

	require.NoError(t, err)

	var results []SearchInfo

	for info := range ch {
		results = append(results, info)
	}

	require.Len(t, results, 3)

	assert.False(t, results[0].IsBestMove)
	assert.Equal(t, 1, results[0].Depth)
	assert.Equal(t, []string{"e2e4"}, results[0].PV)

	assert.False(t, results[1].IsBestMove)
	assert.Equal(t, 2, results[1].Depth)

	assert.True(t, results[2].IsBestMove)
	assert.Equal(t, "e2e4", results[2].BestMove)
	assert.Equal(t, "d7d6", results[2].PonderMove)
}

func TestClient_Go_NilParams(t *testing.T) {
	searchLines := []string{
		"bestmove e2e4",
	}

	eng := buildTestEngine(searchLines)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	ctx := context.Background()
	ch, err := c.Go(ctx, nil)

	require.NoError(t, err)

	var results []SearchInfo

	for info := range ch {
		results = append(results, info)
	}

	require.Len(t, results, 1)
	assert.True(t, results[0].IsBestMove)
}

func TestClient_Go_DoubleStart(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	ctx := context.Background()

	// Prime search state to active manually.
	c.search.mu.Lock()
	c.search.active = true
	c.search.mu.Unlock()

	_, err := c.Go(ctx, &SearchParams{Depth: 1})
	assert.ErrorIs(t, err, ErrSearchInProgress)
}

func TestClient_SetOption_InvalidOption(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	strVal := "4"
	err := c.SetOption("NonExistentOption", &strVal)

	var optErr *ErrInvalidOption
	require.ErrorAs(t, err, &optErr)
	assert.Equal(t, "NonExistentOption", optErr.Name)
}

func TestClient_SetOption_ValidOption(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
		options: map[string]Option{
			"Threads": {Name: "Threads", Type: OptionTypeSpin},
		},
	}

	strVal := "4"
	err := c.SetOption("Threads", &strVal)
	assert.NoError(t, err)
}

func TestClient_SetOption_ButtonType(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
		options: map[string]Option{
			"Clear Hash": {Name: "Clear Hash", Type: OptionTypeButton},
		},
	}

	// nil value = button
	err := c.SetOption("Clear Hash", nil)
	assert.NoError(t, err)
}

func TestClient_Close_Idempotent(t *testing.T) {
	// Build a client with a fake engine that has a no-op close.
	eng := buildTestEngine(nil)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
		closed:  false,
	}

	// Mark as already closed to simulate second call.
	c.closed = true

	// Second close should be a no-op.
	err := c.Close()
	assert.NoError(t, err)
}

func TestClient_IsReady_SearchInProgress(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
	}

	c.search.mu.Lock()
	c.search.active = true
	c.search.mu.Unlock()

	err := c.IsReady()
	assert.ErrorIs(t, err, ErrSearchInProgress)
}

func TestClient_NewGame_SearchInProgress(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
	}

	c.search.mu.Lock()
	c.search.active = true
	c.search.mu.Unlock()

	err := c.NewGame()
	assert.ErrorIs(t, err, ErrSearchInProgress)
}

func TestClient_Bench_SearchInProgress(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
	}

	c.search.mu.Lock()
	c.search.active = true
	c.search.mu.Unlock()

	_, err := c.Bench(BenchParams{})
	assert.ErrorIs(t, err, ErrSearchInProgress)
}

func TestClient_Eval_SearchInProgress(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
	}

	c.search.mu.Lock()
	c.search.active = true
	c.search.mu.Unlock()

	_, err := c.Eval()
	assert.ErrorIs(t, err, ErrSearchInProgress)
}

// helpers to build a closed client for ErrEngineNotRunning tests.
func buildClosedClient() *Client {
	eng := buildTestEngine(nil)

	return &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
		closed:  true,
	}
}

func TestClient_IsReady_AfterClose(t *testing.T) {
	c := buildClosedClient()
	err := c.IsReady()
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_NewGame_AfterClose(t *testing.T) {
	c := buildClosedClient()
	err := c.NewGame()
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_SetOption_AfterClose(t *testing.T) {
	c := buildClosedClient()
	strVal := "4"
	err := c.SetOption("Threads", &strVal)
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_SetPosition_AfterClose(t *testing.T) {
	c := buildClosedClient()
	err := c.SetPosition(StartPosition())
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_Go_AfterClose(t *testing.T) {
	c := buildClosedClient()
	_, err := c.Go(context.Background(), nil)
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_Bench_AfterClose(t *testing.T) {
	c := buildClosedClient()
	_, err := c.Bench(BenchParams{})
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_Eval_AfterClose(t *testing.T) {
	c := buildClosedClient()
	_, err := c.Eval()
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_Display_AfterClose(t *testing.T) {
	c := buildClosedClient()
	_, err := c.Display()
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_Flip_AfterClose(t *testing.T) {
	c := buildClosedClient()
	err := c.Flip()
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_Compiler_AfterClose(t *testing.T) {
	c := buildClosedClient()
	_, err := c.Compiler()
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_ExportNet_AfterClose(t *testing.T) {
	c := buildClosedClient()
	err := c.ExportNet("", "")
	assert.ErrorIs(t, err, ErrEngineNotRunning)
}

func TestClient_Flip_SearchInProgress(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
	}

	c.search.mu.Lock()
	c.search.active = true
	c.search.mu.Unlock()

	err := c.Flip()
	assert.ErrorIs(t, err, ErrSearchInProgress)
}

func TestClient_ExportNet_SearchInProgress(t *testing.T) {
	eng := buildTestEngine(nil)

	c := &Client{
		eng:    eng,
		search: newSearchState(),
	}

	c.search.mu.Lock()
	c.search.active = true
	c.search.mu.Unlock()

	err := c.ExportNet("", "")
	assert.ErrorIs(t, err, ErrSearchInProgress)
}

// TestClient_Go_SlowConsumer verifies that cancelling the context while the
// caller is not reading from the result channel does not deadlock runSearch.
// After cancellation the search must become inactive so the client can accept
// new commands.
func TestClient_Go_SlowConsumer(t *testing.T) {
	// Many info lines followed by bestmove — more than the channel buffer (32).
	lines := make([]string, 0, 40)
	for i := 1; i <= 38; i++ {
		lines = append(lines, "info depth 1 seldepth 1 multipv 1 score cp 17 nodes 20 nps 20000 hashfull 0 tbhits 0 time 1 pv e2e4")
	}

	lines = append(lines, "bestmove e2e4 ponder d7d6")

	eng := buildTestEngine(lines)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := c.Go(ctx, &SearchParams{Depth: 1})
	require.NoError(t, err)

	// Cancel immediately without reading any values from the channel.
	cancel()

	// The channel must close within a reasonable time (no deadlock).
	deadline := time.After(2 * time.Second)
	drained := make(chan struct{})

	go func() {
		//nolint:revive // drain to unblock any pending sends
		for range ch {
		}

		close(drained)
	}()

	select {
	case <-drained:
		// Good: channel closed cleanly.
	case <-deadline:
		t.Fatal("timeout: runSearch goroutine appears to be deadlocked")
	}

	// After cancellation, search must be inactive so the client can be reused.
	assert.False(t, c.search.isActive(), "search should be inactive after context cancel")
}

// TestClient_Go_FullBuffer verifies that when the channel buffer fills up with
// info lines, the bestmove line is still delivered and the channel closes
// without deadlocking.
func TestClient_Go_FullBuffer(t *testing.T) {
	// Exactly fill the buffer (32) with info lines, then send bestmove.
	lines := make([]string, 0, 33)
	for i := 1; i <= 32; i++ {
		lines = append(lines, "info depth 1 seldepth 1 multipv 1 score cp 17 nodes 20 nps 20000 hashfull 0 tbhits 0 time 1 pv e2e4")
	}

	lines = append(lines, "bestmove e2e4 ponder d7d6")

	eng := buildTestEngine(lines)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	ch, err := c.Go(context.Background(), &SearchParams{Depth: 1})
	require.NoError(t, err)

	// Drain the channel slowly, giving the producer time to try sending more.
	var results []SearchInfo

	deadline := time.After(2 * time.Second)

	for {
		select {
		case info, ok := <-ch:
			if !ok {
				// Channel closed: verify bestmove was delivered.
				last := results[len(results)-1]
				assert.True(t, last.IsBestMove, "last result should be bestmove")
				assert.Equal(t, "e2e4", last.BestMove)
				assert.False(t, c.search.isActive(), "search should be inactive after bestmove")

				return
			}

			results = append(results, info)
		case <-deadline:
			t.Fatal("timeout: channel did not close — possible deadlock when buffer was full")
		}
	}
}
