package stockfish

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// process wraps an os/exec.Cmd and provides line-level read/write access to the
// engine's stdin/stdout streams.
type process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	mu     sync.Mutex
}

// newProcess launches the engine binary at the given path and returns a process
// ready for communication.
func newProcess(path string) (*process, error) {
	cmd := exec.Command(path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start engine process: %w", err)
	}

	return &process{
		cmd:    cmd,
		stdin:  stdin,
		reader: bufio.NewReader(stdout),
	}, nil
}

// writeLine sends a single line to the engine's stdin, appending a newline.
func (p *process) writeLine(line string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := fmt.Fprintln(p.stdin, line)
	if err != nil {
		return fmt.Errorf("write to engine: %w", err)
	}

	return nil
}

// readLine reads and returns the next line from the engine's stdout, stripping
// the trailing newline.
func (p *process) readLine() (string, error) {
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read from engine: %w", err)
	}

	return strings.TrimRight(line, "\r\n"), nil
}

// close sends quit to stdin, closes the pipe, and waits for the process to
// exit.
func (p *process) close() error {
	if err := p.writeLine("quit"); err != nil {
		// Best effort — still try to close stdin.
		_ = p.stdin.Close()

		return err
	}

	if err := p.stdin.Close(); err != nil {
		return fmt.Errorf("close stdin: %w", err)
	}

	if err := p.cmd.Wait(); err != nil {
		return fmt.Errorf("wait for engine process: %w", err)
	}

	return nil
}

// engine manages the higher-level UCI communication layer on top of a process.
// It owns the read loop goroutine and routes output lines to the current active
// subscriber.
type engine struct {
	proc   *process
	lineCh chan string
	errCh  chan error
	done   chan struct{}
}

// newEngine wraps an existing process and starts the background read loop.
func newEngine(proc *process) *engine {
	e := &engine{
		proc:   proc,
		lineCh: make(chan string, 64),
		errCh:  make(chan error, 1),
		done:   make(chan struct{}),
	}

	go e.readLoop()

	return e
}

// readLoop continuously reads lines from the process stdout and forwards them
// to lineCh until EOF or error.
func (e *engine) readLoop() {
	defer close(e.lineCh)
	defer close(e.done)

	for {
		line, err := e.proc.readLine()
		if err != nil {
			// EOF means engine quit cleanly; any other error is unexpected.
			if !errors.Is(err, io.EOF) && !isClosedPipeError(err) {
				select {
				case e.errCh <- err:
				default:
				}
			}

			return
		}

		if line == "" {
			continue
		}

		e.lineCh <- line
	}
}

// send writes a command line to the engine.
func (e *engine) send(line string) error {
	return e.proc.writeLine(line)
}

// readUntil reads lines from the engine until a line satisfying the predicate
// is found, the context is cancelled, or the channel is closed. It returns all
// consumed lines (including the terminating line). If the context is cancelled
// before the predicate is satisfied, ErrEngineTimeout is returned.
func (e *engine) readUntil(ctx context.Context, pred func(string) bool) ([]string, error) {
	var lines []string

	for {
		select {
		case <-ctx.Done():
			return lines, ErrEngineTimeout

		case line, ok := <-e.lineCh:
			if !ok {
				return lines, ErrEngineNotRunning
			}

			lines = append(lines, line)

			if pred(line) {
				return lines, nil
			}

		case err := <-e.errCh:
			return lines, fmt.Errorf("engine read error: %w", err)
		}
	}
}

// isClosedPipeError reports whether the error is a broken/closed pipe — a
// normal condition when the engine process exits.
func isClosedPipeError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "closed pipe") ||
		strings.Contains(msg, "read/write on closed pipe") ||
		strings.Contains(msg, "file already closed") ||
		strings.Contains(msg, "use of closed network connection")
}
