package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/internal/security/auth"
)

type RateLimiter struct {
	requestsPerSecond int
	burstSize         int
	clients           map[string]*ClientLimiter
	mu                sync.RWMutex
}

type ClientLimiter struct {
	tokens     chan struct{}
	lastAccess time.Time
}

func NewRateLimiter(requestsPerSecond, burstSize int) *RateLimiter {
	return &RateLimiter{
		requestsPerSecond: requestsPerSecond,
		burstSize:         burstSize,
		clients:           make(map[string]*ClientLimiter),
	}
}

func (rl *RateLimiter) getClientLimiter(clientIP string) *ClientLimiter {
	rl.mu.RLock()
	limiter, exists := rl.clients[clientIP]
	rl.mu.RUnlock()

	if exists {
		rl.mu.Lock()
		limiter.lastAccess = time.Now()
		rl.mu.Unlock()
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limiter, exists := rl.clients[clientIP]; exists {
		return limiter
	}

	limiter = &ClientLimiter{
		tokens:     make(chan struct{}, rl.burstSize),
		lastAccess: time.Now(),
	}

	for i := 0; i < rl.burstSize; i++ {
		limiter.tokens <- struct{}{}
	}

	rl.clients[clientIP] = limiter
	return limiter
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		limiter := rl.getClientLimiter(clientIP)

		select {
		case <-limiter.tokens:
			go func() {
				time.Sleep(time.Second / time.Duration(rl.requestsPerSecond))
				limiter.tokens <- struct{}{}
			}()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":       "rate limit exceeded",
				"retry_after": "1s",
			})
		}
	})
}

func (rl *RateLimiter) Cleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, limiter := range rl.clients {
				if now.Sub(limiter.lastAccess) > interval*3 {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

type Logger struct {
	config *LoggingConfig
	logger *log.Logger
}

func NewLogger(config *LoggingConfig) *Logger {
	writer := os.Stdout
	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			writer = file
		}
	}

	return &Logger{
		config: config,
		logger: log.New(writer, "", 0),
	}
}

func (l *Logger) logRequest(r *http.Request, statusCode int, duration time.Duration, requestID, userID string) {
	logEntry := map[string]interface{}{
		"timestamp":   time.Now().UTC(),
		"method":      r.Method,
		"path":        r.URL.Path,
		"query":       r.URL.RawQuery,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"remote_addr": r.RemoteAddr,
	}

	if requestID != "" {
		logEntry["request_id"] = requestID
	}

	if userID != "" {
		logEntry["user_id"] = userID
	}

	if l.config.LogLevel == "debug" {
		logEntry["user_agent"] = r.Header.Get("User-Agent")
	}

	line, _ := json.Marshal(logEntry)
	l.logger.Println(string(line))
}

func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.config.Enable {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" && l.config.LogRequestID {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
			w.Header().Set("X-Request-ID", requestID)
		}

		var userID string
		if l.config.LogUser {
			if user, ok := auth.GetUserFromContext(r.Context()); ok {
				userID = user.ID
			}
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		if l.config.IncludeBody {
			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(io.NewSectionReader(
				NewBytesReader(body),
				0, int64(len(body)),
			))
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		if l.config.LogLatency {
			l.logRequest(r, rw.statusCode, duration, requestID, userID)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

type byteBuffer struct {
	data []byte
	pos  int
}

func (b *byteBuffer) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

func (b *byteBuffer) ReadByte() (byte, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	c := b.data[b.pos]
	b.pos++
	return c, nil
}

func NewBytesReader(data []byte) *byteBuffer {
	return &byteBuffer{data: data}
}

func (b *byteBuffer) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	if off < 0 {
		return 0, fmt.Errorf("negative offset")
	}
	b.pos = int(off)
	return b.Read(p)
}

func (b *byteBuffer) Seek(offset int64, whence int) (int64, error) {
	var newPos int
	switch whence {
	case io.SeekStart:
		newPos = int(offset)
	case io.SeekCurrent:
		newPos = b.pos + int(offset)
	case io.SeekEnd:
		newPos = len(b.data) + int(offset)
	default:
		return 0, fmt.Errorf("invalid whence")
	}

	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}

	b.pos = newPos
	return int64(b.pos), nil
}

func (b *byteBuffer) Len() int {
	return len(b.data)
}

func (b *byteBuffer) Size() int64 {
	return int64(len(b.data))
}
