package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/swarm-ai/swarm/internal/security/auth"
)

var (
	ErrClientClosed       = errors.New("client closed")
	ErrClientDisconnected = errors.New("client disconnected")
	ErrWriteFailed        = errors.New("write failed")
	ErrReadFailed         = errors.New("read failed")
)

type Client struct {
	id        string
	conn      *websocket.Conn
	hub       *Hub
	userID    string
	send      chan *Message
	receive   chan *Message
	sub       *Subscription
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	closed    bool
	lastPing  time.Time
	lastPong  time.Time
	closeOnce sync.Once
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		id:       generateClientID(),
		conn:     conn,
		hub:      hub,
		userID:   userID,
		send:     make(chan *Message, 256),
		receive:  make(chan *Message, 256),
		sub:      NewSubscription(),
		ctx:      ctx,
		cancel:   cancel,
		closed:   false,
		lastPing: time.Now(),
		lastPong: time.Now(),
	}
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) UserID() string {
	return c.userID
}

func (c *Client) Send(msg *Message) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClientClosed
	}
	c.mu.RUnlock()

	select {
	case c.send <- msg:
		return nil
	case <-c.ctx.Done():
		return ErrClientClosed
	default:
		return fmt.Errorf("send buffer full")
	}
}

func (c *Client) SendDirect(msg *Message) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClientClosed
	}
	c.mu.RUnlock()

	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := c.conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("%w: %v", ErrWriteFailed, err)
	}
	return nil
}

func (c *Client) Receive() <-chan *Message {
	return c.receive
}

func (c *Client) Subscription() *Subscription {
	return c.sub
}

func (c *Client) Connect() {
	go c.readPump()
	go c.writePump()
}

func (c *Client) Disconnect() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	c.cancel()

	c.closeOnce.Do(func() {
		close(c.send)
	})

	if c.conn != nil {
		if err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			return fmt.Errorf("failed to send close message: %w", err)
		}
		return c.conn.Close()
	}

	return nil
}

func (c *Client) Close() error {
	return c.Disconnect()
}

func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

func (c *Client) RemoteAddr() string {
	if c.conn == nil {
		return ""
	}
	return c.conn.RemoteAddr().String()
}

func (c *Client) LastPing() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastPing
}

func (c *Client) LastPong() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastPong
}

func (c *Client) UpdatePing() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastPing = time.Now()
}

func (c *Client) UpdatePong() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastPong = time.Now()
}

func (c *Client) IsStale(timeout time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return true
	}
	if time.Since(c.lastPong) > timeout {
		return true
	}
	return false
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Disconnect()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.UpdatePong()
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.Error(fmt.Errorf("read error: %w", err))
			}
			break
		}

		msg, err := ParseMessage(data)
		if err != nil {
			c.hub.Error(fmt.Errorf("parse message error: %w", err))
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *Client) writePump() {
	pingTicker := time.NewTicker(30 * time.Second)
	defer func() {
		pingTicker.Stop()
		c.Disconnect()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}

		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			c.UpdatePing()

		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MessageTypePing:
		pong := NewPong(time.Now())
		if err := c.SendDirect(pong); err != nil {
			c.hub.Error(err)
		}
	case MessageTypePong:
		c.UpdatePong()
	default:
		select {
		case c.receive <- msg:
		default:
		}
	}
}

func generateClientID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}

func ExtractUserIDFromContext(r *http.Request) string {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		return ""
	}
	return user.ID
}

func ExtractUserIDFromRequest(r *http.Request) string {
	sessionCookie, err := r.Cookie("session_id")
	if err == nil && sessionCookie.Value != "" {
		return sessionCookie.Value
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		const prefix = "Bearer "
		if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
			return authHeader[len(prefix):]
		}
	}

	return ""
}

type ClientInfo struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	RemoteAddr  string    `json:"remote_addr"`
	ConnectedAt time.Time `json:"connected_at"`
	LastActive  time.Time `json:"last_active"`
}

func (c *Client) Info() *ClientInfo {
	return &ClientInfo{
		ID:          c.id,
		UserID:      c.userID,
		RemoteAddr:  c.RemoteAddr(),
		ConnectedAt: c.lastPing,
		LastActive:  c.lastPong,
	}
}

func (c *Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Info())
}
