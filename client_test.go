package stockfish

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"

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
