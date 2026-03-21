package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientID(t *testing.T) {
	hub := NewHub()
	userID := "user-1"

	client1 := NewClient(hub, nil, userID)
	client2 := NewClient(hub, nil, userID)

	assert.NotEmpty(t, client1.ID())
	assert.NotEmpty(t, client2.ID())
	assert.NotEqual(t, client1.ID(), client2.ID())
}

func TestClientUserID(t *testing.T) {
	hub := NewHub()
	userID := "user-1"

	client := NewClient(hub, nil, userID)

	assert.Equal(t, userID, client.UserID())
}

func TestClientSubscription(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "user-1")

	sub := client.Subscription()
	assert.NotNil(t, sub)

	sub.AddAgentID("agent-1")
	assert.True(t, sub.ShouldReceive(&Message{AgentID: "agent-1"}))
}

func TestClientSend(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "user-1")

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")

	err := client.Send(msg)
	assert.NoError(t, err)

	select {
	case sentMsg := <-client.send:
		assert.Equal(t, msg.Type, sentMsg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("message not received in send channel")
	}
}

func TestClientSendClosed(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "user-1")

	err := client.Disconnect()
	require.NoError(t, err)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err = client.Send(msg)
	assert.Error(t, err)
	assert.Equal(t, ErrClientClosed, err)
}

func TestClientDisconnect(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "user-1")

	err := client.Disconnect()
	assert.NoError(t, err)
	assert.True(t, client.IsClosed())
}

func TestClientDisconnectTwice(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "user-1")

	err := client.Disconnect()
	require.NoError(t, err)

	err = client.Disconnect()
	assert.NoError(t, err)
	assert.True(t, client.IsClosed())
}

func TestClientIsStale(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "user-1")

	assert.False(t, client.IsStale(1*time.Minute))

	client.mu.Lock()
	client.lastPong = time.Now().Add(-2 * time.Minute)
	client.mu.Unlock()

	assert.True(t, client.IsStale(1*time.Minute))
}

func TestClientPingPong(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "user-1")

	client.UpdatePing()
	assert.False(t, client.LastPing().IsZero())

	client.UpdatePong()
	assert.False(t, client.LastPong().IsZero())
}

func TestExtractUserIDFromContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)

	userID := ExtractUserIDFromContext(req)
	assert.Empty(t, userID)

	req = req.WithContext(context.WithValue(req.Context(), "user", &struct{ ID string }{ID: "user-1"}))
	userID = ExtractUserIDFromContext(req)
	assert.Empty(t, userID)
}

func TestExtractUserIDFromRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)

	userID := ExtractUserIDFromRequest(req)
	assert.Empty(t, userID)

	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
	userID = ExtractUserIDFromRequest(req)
	assert.Equal(t, "session-123", userID)

	req = httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer token-456")
	userID = ExtractUserIDFromRequest(req)
	assert.Equal(t, "token-456", userID)

	req = httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer")
	userID = ExtractUserIDFromRequest(req)
	assert.Empty(t, userID)
}

func TestGenerateClientID(t *testing.T) {
	id1 := generateClientID()
	id2 := generateClientID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "client-")
	assert.Contains(t, id2, "client-")
}

func TestClientReceive(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "user-1")

	receiveChan := client.Receive()
	assert.NotNil(t, receiveChan)
}
