package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type Transport interface {
	Send(ctx context.Context, msg json.RawMessage) error
	Receive(ctx context.Context) (json.RawMessage, error)
	Close() error
}

type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.Reader
	mu     sync.Mutex
}

func NewStdioTransport(command string, args []string, env map[string]string) *StdioTransport {
	cmd := exec.Command(command, args...)

	envList := make([]string, 0, len(env)+len(os.Environ()))
	envList = append(envList, os.Environ()...)
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envList

	return &StdioTransport{cmd: cmd}
}

func (t *StdioTransport) Start(ctx context.Context) error {
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	t.stdout = bufio.NewReader(stdout)

	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	t.stderr = stderr

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	return nil
}

func (t *StdioTransport) Send(ctx context.Context, msg json.RawMessage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.stdin.Write(append(msg, '\n')); err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}
	return nil
}

func (t *StdioTransport) Receive(ctx context.Context) (json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	line, err := t.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdout: %w", err)
	}

	if len(line) == 0 {
		return nil, io.EOF
	}

	if line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	return json.RawMessage(line), nil
}

func (t *StdioTransport) Close() error {
	var errs []error

	if t.stdin != nil {
		if err := t.stdin.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if t.cmd.Process != nil {
		if err := t.cmd.Process.Kill(); err != nil {
			errs = append(errs, err)
		}
		if err := t.cmd.Wait(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors closing transport: %v", errs)
	}
	return nil
}

func (t *StdioTransport) Stderr() io.Reader {
	return t.stderr
}

type HTTPTransport struct {
	endpoint string
	headers  map[string]string
	client   httpClient

	eventCh chan json.RawMessage
	mu      sync.Mutex
}

func NewHTTPTransport(endpoint string, headers map[string]string) *HTTPTransport {
	return &HTTPTransport{
		endpoint: endpoint,
		headers:  headers,
		client:   newHTTPClient(),
		eventCh:  make(chan json.RawMessage, 100),
	}
}

func (t *HTTPTransport) Send(ctx context.Context, msg json.RawMessage) error {
	return t.client.post(ctx, t.endpoint, t.headers, msg)
}

func (t *HTTPTransport) Receive(ctx context.Context) (json.RawMessage, error) {
	select {
	case msg := <-t.eventCh:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *HTTPTransport) Close() error {
	close(t.eventCh)
	return nil
}

type httpClient interface {
	post(ctx context.Context, url string, headers map[string]string, data json.RawMessage) error
}

type defaultHTTPClient struct{}

func newHTTPClient() httpClient {
	return &defaultHTTPClient{}
}

func (c *defaultHTTPClient) post(ctx context.Context, url string, headers map[string]string, data json.RawMessage) error {
	return fmt.Errorf("HTTP transport requires event-stream implementation for full bidirectional communication")
}

type WebSocketTransport struct {
	conn   wsConn
	mu     sync.Mutex
	closed bool
}

type wsConn interface {
	Write(ctx context.Context, msgType int, data []byte) error
	Read(ctx context.Context) (int, []byte, error)
	Close(status int, reason string) error
}

func NewWebSocketTransport(url string, headers map[string]string) (*WebSocketTransport, error) {
	return &WebSocketTransport{}, fmt.Errorf("WebSocket transport requires websocket dependency")
}

func (t *WebSocketTransport) Send(ctx context.Context, msg json.RawMessage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		return fmt.Errorf("connection not established")
	}
	return t.conn.Write(ctx, 1, msg)
}

func (t *WebSocketTransport) Receive(ctx context.Context) (json.RawMessage, error) {
	if t.conn == nil {
		return nil, fmt.Errorf("connection not established")
	}

	typ, data, err := t.conn.Read(ctx)
	if err != nil {
		return nil, err
	}

	if typ != 1 {
		return nil, fmt.Errorf("expected text message, got type %d", typ)
	}

	return json.RawMessage(data), nil
}

func (t *WebSocketTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || t.conn == nil {
		return nil
	}
	t.closed = true

	return t.conn.Close(1000, "normal closure")
}
