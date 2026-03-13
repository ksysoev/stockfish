package stockfish

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	readyOK          = "readyok"
	defaultHandshake = 30 * time.Second
)

// Client is a high-level UCI client that manages the lifecycle of a Stockfish
// engine process and provides methods for all standard and non-standard UCI
// commands.
//
// A Client must be created with New and must be closed with Close when no
// longer needed to release the underlying process resources.
type Client struct {
	eng     *engine
	search  *searchState
	options map[string]Option
	name    string
	author  string
	mu      sync.Mutex
	closed  bool
}

// New launches the Stockfish binary at path, performs the UCI handshake, and
// returns a ready-to-use Client. The caller must call Close to release
// resources.
func New(path string) (*Client, error) {
	proc, err := newProcess(path)
	if err != nil {
		return nil, fmt.Errorf("launch engine: %w", err)
	}

	eng := newEngine(proc)

	c := &Client{
		eng:     eng,
		search:  newSearchState(),
		options: make(map[string]Option),
	}

	if err = c.initialize(); err != nil {
		_ = proc.close()
		return nil, fmt.Errorf("initialize UCI: %w", err)
	}

	return c, nil
}

// initialize sends "uci" and collects "id"/"option" lines until "uciok".
func (c *Client) initialize() error {
	if err := c.eng.send("uci"); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultHandshake)
	defer cancel()

	lines, err := c.eng.readUntil(ctx, func(line string) bool {
		return line == "uciok"
	})
	if err != nil {
		return fmt.Errorf("waiting for uciok: %w", err)
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "id name "):
			c.name = strings.TrimPrefix(line, "id name ")
		case strings.HasPrefix(line, "id author "):
			c.author = strings.TrimPrefix(line, "id author ")
		case strings.HasPrefix(line, "option "):
			opt, parseErr := parseOption(line)
			if parseErr == nil {
				c.options[opt.Name] = opt
			}
		}
	}

	return nil
}

// Name returns the engine name as reported by the "id name" UCI response.
func (c *Client) Name() string {
	return c.name
}

// Author returns the engine author as reported by the "id author" UCI response.
func (c *Client) Author() string {
	return c.author
}

// Options returns a copy of the UCI options map discovered during
// initialisation. Keys are option names.
func (c *Client) Options() map[string]Option {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make(map[string]Option, len(c.options))

	for k, v := range c.options {
		out[k] = v
	}

	return out
}

// IsReady sends "isready" and waits for "readyok". This synchronises the
// client with the engine after any potentially slow operation.
func (c *Client) IsReady() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.search.isActive() {
		return ErrSearchInProgress
	}

	return c.isReady()
}

// isReady is the unlocked internal implementation.
func (c *Client) isReady() error {
	if err := c.eng.send("isready"); err != nil {
		return fmt.Errorf("send isready: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultHandshake)
	defer cancel()

	_, err := c.eng.readUntil(ctx, func(line string) bool {
		return line == readyOK
	})
	if err != nil {
		return fmt.Errorf("waiting for readyok: %w", err)
	}

	return nil
}

// NewGame sends "ucinewgame" followed by "isready" to prepare the engine for
// a new game. This clears the engine's transposition table and search history.
func (c *Client) NewGame() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.search.isActive() {
		return ErrSearchInProgress
	}

	if err := c.eng.send("ucinewgame"); err != nil {
		return fmt.Errorf("send ucinewgame: %w", err)
	}

	return c.isReady()
}

// SetOption sends a "setoption" command to the engine. For button-type options
// pass nil as value. For string-type options that should be set to the empty
// string, pass a pointer to an empty string.
//
// Returns ErrInvalidOption if name is not among the options reported by the
// engine during initialisation.
func (c *Client) SetOption(name string, value *string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.options[name]; !ok {
		return &ErrInvalidOption{Name: name}
	}

	cmd := buildSetOption(name, value)

	if err := c.eng.send(cmd); err != nil {
		return fmt.Errorf("send setoption: %w", err)
	}

	return nil
}

// SetPosition sends a "position" command to the engine.
func (c *Client) SetPosition(pos Position) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd, err := buildPositionCommand(pos)
	if err != nil {
		return err
	}

	if err = c.eng.send(cmd); err != nil {
		return fmt.Errorf("send position: %w", err)
	}

	return nil
}

// Go starts a search using the given parameters. It returns a read-only channel
// that streams SearchInfo values as the engine emits "info" lines, ending with
// a final SearchInfo where IsBestMove is true (containing BestMove and
// optionally PonderMove). The channel is closed after the bestmove line.
//
// params may be nil, in which case a default (unlimited) search is started.
//
// The search can be cancelled early by cancelling ctx, which will cause Stop to
// be sent automatically.
func (c *Client) Go(ctx context.Context, params *SearchParams) (<-chan SearchInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if params == nil {
		params = &SearchParams{}
	}

	ch, searchCtx, err := c.search.start(ctx)
	if err != nil {
		return nil, err
	}

	cmd := buildGoCommand(params)
	if err = c.eng.send(cmd); err != nil {
		c.search.finish()
		return nil, fmt.Errorf("send go: %w", err)
	}

	go c.runSearch(searchCtx, ch)

	return ch, nil
}

// runSearch reads engine output lines and forwards them as SearchInfo values on
// ch until a bestmove line is received or the context is cancelled.
func (c *Client) runSearch(ctx context.Context, ch chan SearchInfo) {
	defer c.search.finish()

	for {
		select {
		case <-ctx.Done():
			// Only send stop if the search is still active — it may have already
			// received bestmove and called finish() on the other branch.
			if c.search.isActive() {
				_ = c.eng.send("stop")
			}
			// Drain until bestmove to keep the engine in sync.
			c.drainUntilBestMove(ch)

			return
		case line, ok := <-c.eng.lineCh:
			if !ok {
				return
			}

			if strings.HasPrefix(line, "bestmove") {
				info, err := parseBestMoveLine(line)
				if err == nil {
					ch <- info
				}

				return
			}

			if strings.HasPrefix(line, "info") {
				info, err := parseInfoLine(line)
				if err == nil && (info.Depth > 0 || info.CurrMove != "") {
					ch <- info
				}
			}
		}
	}
}

// drainUntilBestMove reads and forwards lines until bestmove, used after a
// stop is sent.
func (c *Client) drainUntilBestMove(ch chan SearchInfo) {
	for line := range c.eng.lineCh {
		if strings.HasPrefix(line, "bestmove") {
			info, err := parseBestMoveLine(line)
			if err == nil {
				ch <- info
			}

			return
		}

		if strings.HasPrefix(line, "info") {
			info, err := parseInfoLine(line)
			if err == nil && (info.Depth > 0 || info.CurrMove != "") {
				ch <- info
			}
		}
	}
}

// Stop sends the "stop" command to the engine, ending the current search. The
// search channel returned by Go will receive the final bestmove and then be
// closed.
func (c *Client) Stop() error {
	if !c.search.isActive() {
		return ErrNoSearchInProgress
	}

	if err := c.eng.send("stop"); err != nil {
		return fmt.Errorf("send stop: %w", err)
	}

	return nil
}

// PonderHit sends the "ponderhit" command, switching the engine from pondering
// to normal search mode.
func (c *Client) PonderHit() error {
	if !c.search.isActive() {
		return ErrNoSearchInProgress
	}

	if err := c.eng.send("ponderhit"); err != nil {
		return fmt.Errorf("send ponderhit: %w", err)
	}

	return nil
}

// Bench runs the non-standard "bench" command and returns the raw output lines.
// It blocks until the "===" summary terminator line is received.
func (c *Client) Bench(params BenchParams) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.search.isActive() {
		return nil, ErrSearchInProgress
	}

	cmd := buildBenchCommand(params)

	if err := c.eng.send(cmd); err != nil {
		return nil, fmt.Errorf("send bench: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultHandshake)
	defer cancel()

	lines, err := c.eng.readUntil(ctx, benchTerminator)
	if err != nil {
		return nil, fmt.Errorf("read bench output: %w", err)
	}

	return lines, nil
}

// sendAndReadUntilReady sends cmd, then sends "isready" and collects all lines
// until "readyok", stripping the terminal readyok. It is used by non-standard
// commands whose output has no well-defined terminator of its own.
func (c *Client) sendAndReadUntilReady(cmd, sendErrMsg, readErrMsg string) (string, error) {
	if c.search.isActive() {
		return "", ErrSearchInProgress
	}

	if err := c.eng.send(cmd); err != nil {
		return "", fmt.Errorf("%s: %w", sendErrMsg, err)
	}

	if err := c.eng.send("isready"); err != nil {
		return "", fmt.Errorf("send isready after %s: %w", cmd, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultHandshake)
	defer cancel()

	lines, err := c.eng.readUntil(ctx, func(line string) bool {
		return line == readyOK
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", readErrMsg, err)
	}

	if len(lines) > 0 && lines[len(lines)-1] == readyOK {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n"), nil
}

// Eval sends the non-standard "eval" command and returns the full output as a
// single string.
func (c *Client) Eval() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.sendAndReadUntilReady("eval", "send eval", "read eval output")
}

// Display sends the non-standard "d" command and returns the board display
// output as a single string.
func (c *Client) Display() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.sendAndReadUntilReady("d", "send d", "read d output")
}

// Flip sends the non-standard "flip" command which flips the side to move.
func (c *Client) Flip() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.eng.send("flip"); err != nil {
		return fmt.Errorf("send flip: %w", err)
	}

	return nil
}

// Compiler sends the non-standard "compiler" command and returns the compiler
// information string.
func (c *Client) Compiler() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.sendAndReadUntilReady("compiler", "send compiler", "read compiler output")
}

// ExportNet sends the non-standard "export_net" command. Pass empty strings for
// bigNet and smallNet to export the embedded networks with their default names.
func (c *Client) ExportNet(bigNet, smallNet string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var cmd string

	switch {
	case bigNet != "" && smallNet != "":
		cmd = fmt.Sprintf("export_net %s %s", bigNet, smallNet)
	case bigNet != "":
		cmd = fmt.Sprintf("export_net %s", bigNet)
	default:
		cmd = "export_net"
	}

	if err := c.eng.send(cmd); err != nil {
		return fmt.Errorf("send export_net: %w", err)
	}

	return nil
}

// Close sends "quit" to the engine and waits for the process to exit, releasing
// all resources. It is safe to call Close multiple times; subsequent calls
// return nil immediately.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if err := c.eng.proc.close(); err != nil {
		return fmt.Errorf("close engine: %w", err)
	}

	return nil
}
