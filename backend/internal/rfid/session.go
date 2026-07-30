package rfid

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

const (
	proxmarkSessionStartupTimeout = 15 * time.Second
	proxmarkSessionPollTimeout    = 2 * time.Second
	proxmarkSessionWriteTimeout   = 15 * time.Second
	proxmarkReconnectMinBackoff   = time.Second
	proxmarkReconnectMaxBackoff   = 15 * time.Second
)

// PM3Session is a long-lived interactive Proxmark CLI connection.
type PM3Session interface {
	Run(ctx context.Context, command string) (stdout string, err error)
	Close() error
}

// PM3SessionFactory opens a new interactive session.
type PM3SessionFactory func(ctx context.Context) (PM3Session, error)

var pm3PromptPattern = regexp.MustCompile(`(?m)(?:\[[^\]]+\]\s*)?pm3\s*-->\s*$`)

type processSession struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	closed bool
}

func openProcessSession(ctx context.Context, cliPath, port string) (PM3Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	exe := resolveProxmarkExecutable(cliPath)
	args := []string{"-f", "--incognito"}
	if port != "" {
		args = append([]string{"-p", port}, args...)
	}
	// Process lifetime is independent of per-command deadlines; only startup
	// and Run use ctx timeouts. Killing the child on poll timeout would defeat
	// the persistent-session goal.
	// Always spawn the real .exe — .cmd wrappers do not forward Go stdin pipes.
	cmd := exec.Command(exe, args...)
	cmd.Env = proxmarkCommandEnv(exe)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	// Merge stderr into the same writer StdoutPipe installed on cmd.Stdout.
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}

	s := &processSession{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdoutPipe),
	}

	// Startup must not inherit short Poll deadlines (2s). Opening the CLI and
	// waiting for the first prompt routinely needs several seconds on Windows.
	startupCtx, cancel := context.WithTimeout(context.Background(), proxmarkSessionStartupTimeout)
	defer cancel()
	if _, err := s.readUntilPrompt(startupCtx); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("proxmark session startup: %w", err)
	}
	return s, nil
}

func (s *processSession) Run(ctx context.Context, command string) (string, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return "", fmt.Errorf("proxmark session closed")
	}
	stdin := s.stdin
	s.mu.Unlock()
	// Do not hold s.mu across readUntilPrompt — Close must be able to kill the
	// child while ArmScan is blocked in hf 14a reader -w.
	if _, err := io.WriteString(stdin, command+"\n"); err != nil {
		return "", fmt.Errorf("proxmark session write: %w", err)
	}
	return s.readUntilPrompt(ctx)
}

func (s *processSession) readUntilPrompt(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	tmp := make([]byte, 256)
	for {
		if err := ctx.Err(); err != nil {
			return buf.String(), err
		}
		// Interactive pm3 often prints "pm3 --> " without a trailing newline.
		// Read chunks (not lines) so we can detect the prompt at buffer end.
		type readResult struct {
			n   int
			err error
		}
		ch := make(chan readResult, 1)
		go func() {
			n, err := s.stdout.Read(tmp)
			ch <- readResult{n: n, err: err}
		}()
		select {
		case <-ctx.Done():
			return buf.String(), ctx.Err()
		case r := <-ch:
			if r.n > 0 {
				buf.Write(tmp[:r.n])
				if pm3PromptPattern.MatchString(buf.String()) {
					return buf.String(), nil
				}
			}
			if r.err != nil {
				return buf.String(), r.err
			}
		}
	}
}

func (s *processSession) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	stdin := s.stdin
	cmd := s.cmd
	s.mu.Unlock()
	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if cmd != nil {
		_ = cmd.Wait()
	}
	return nil
}

func defaultSessionFactory(cliPath, port string) PM3SessionFactory {
	return func(ctx context.Context) (PM3Session, error) {
		return openProcessSession(ctx, cliPath, port)
	}
}
