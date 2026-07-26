package rfid

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
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

	startupCtx, cancel := context.WithTimeout(ctx, proxmarkSessionStartupTimeout)
	defer cancel()
	if _, err := s.readUntilPrompt(startupCtx); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("proxmark session startup: %w", err)
	}
	return s, nil
}

func (s *processSession) Run(ctx context.Context, command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", fmt.Errorf("proxmark session closed")
	}
	if _, err := io.WriteString(s.stdin, command+"\n"); err != nil {
		return "", fmt.Errorf("proxmark session write: %w", err)
	}
	return s.readUntilPrompt(ctx)
}

func (s *processSession) readUntilPrompt(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	for {
		if err := ctx.Err(); err != nil {
			return buf.String(), err
		}
		// Bound each ReadSlice wait via deadline on the underlying pipe is hard;
		// poll with short SetReadDeadline when available, else use goroutine.
		lineCh := make(chan string, 1)
		errCh := make(chan error, 1)
		go func() {
			line, err := s.stdout.ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}
			lineCh <- line
		}()
		select {
		case <-ctx.Done():
			return buf.String(), ctx.Err()
		case err := <-errCh:
			return buf.String(), err
		case line := <-lineCh:
			buf.WriteString(line)
			if pm3PromptPattern.MatchString(strings.TrimRight(line, "\r\n")) ||
				pm3PromptPattern.MatchString(buf.String()) {
				return buf.String(), nil
			}
		}
	}
}

func (s *processSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	_ = s.stdin.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	_ = s.cmd.Wait()
	return nil
}

func defaultSessionFactory(cliPath, port string) PM3SessionFactory {
	return func(ctx context.Context) (PM3Session, error) {
		return openProcessSession(ctx, cliPath, port)
	}
}
