package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/swarm-ai/swarm/internal/config"
)

func TestDefaultServerConfig(t *testing.T) {
	config := DefaultServerConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "0.0.0.0", config.Host)
	assert.Equal(t, 8081, config.Port)
	assert.Equal(t, "/ws", config.Path)
	assert.True(t, config.EnableAuth)
	assert.Equal(t, 256, config.BufferSize)
	assert.Equal(t, 4, config.BroadcastWorkers)
}

func TestNewServer(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, serverConfig, server.config)
	assert.Equal(t, appConfig, server.appConfig)
	assert.NotNil(t, server.hub)
	assert.NotNil(t, server.broadcastMgr)
	assert.NotNil(t, server.router)
	assert.False(t, server.IsRunning())
}

func TestNewServerWithNilConfig(t *testing.T) {
	appConfig := &config.Config{}
	server, err := NewServer(nil, appConfig, nil)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, server.config)
}

func TestServerStartStop(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	assert.NoError(t, err)
	assert.True(t, server.IsRunning())

	err = server.Stop()
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestServerStartTwice(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)

	err = server.Start()
	assert.Error(t, err)

	server.Stop()
}

func TestServerStopNotRunning(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Stop()
	assert.NoError(t, err)
}

func TestServerAddr(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)

	addr := server.Addr()
	assert.NotEmpty(t, addr)

	server.Stop()
}

func TestServerContext(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)

	ctx := server.Context()
	assert.NotNil(t, ctx)

	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled")
	default:
	}

	server.Stop()

	select {
	case <-ctx.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("context should be cancelled after stop")
	}
}

func TestServerHub(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	hub := server.Hub()
	assert.NotNil(t, hub)
	assert.Equal(t, server.hub, hub)
}

func TestServerBroadcastManager(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	bm := server.BroadcastManager()
	assert.NotNil(t, bm)
	assert.Equal(t, server.broadcastMgr, bm)
}

func TestServerGetStats(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	stats := server.GetStats()
	assert.NotNil(t, stats)

	assert.Equal(t, false, stats["running"])
	assert.NotNil(t, stats["hub"])
	assert.NotNil(t, stats["broadcast"])
	assert.NotNil(t, stats["config"])
}

func TestServerGetStatsRunning(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)

	stats := server.GetStats()
	assert.NotNil(t, stats)

	assert.Equal(t, true, stats["running"])
	assert.NotEmpty(t, stats["address"])

	server.Stop()
}

func TestServerInfo(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	info := server.Info()
	assert.NotNil(t, info)
	assert.False(t, info.Running)
	assert.Equal(t, "/ws", info.Path)
}

func TestServerBroadcastAgentUpdate(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	err = server.BroadcastAgentUpdate(nil, nil, nil)
	assert.NoError(t, err)
}

func TestServerBroadcastTaskEvent(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	err = server.BroadcastTaskEvent(nil, "test", nil, nil)
	assert.NoError(t, err)
}

func TestServerBroadcastLog(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	err = server.BroadcastLog("info", "test message", "agent-1", "task-1")
	assert.NoError(t, err)
}

func TestServerBroadcastToUser(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err = server.BroadcastToUser(msg, "user-1")
	assert.NoError(t, err)
}

func TestServerBroadcastToAgent(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err = server.BroadcastToAgent(msg, "agent-1")
	assert.NoError(t, err)
}

func TestServerBroadcastToTask(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err = server.BroadcastToTask(msg, "task-1")
	assert.NoError(t, err)
}

func TestServerHandleAgentUpdate(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	server.HandleAgentUpdate(nil, nil, nil)
}

func TestServerHandleTaskEvent(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	server.HandleTaskEvent(nil, "test", nil, nil)
}

func TestServerHandleLog(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	server.HandleLog("info", "test message", "agent-1", "task-1")
}

func TestServerGracefulShutdown(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)

	err = server.GracefulShutdown(5 * time.Second)
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestServerCleanupStaleClients(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	cleaned := server.CleanupStaleClients(1 * time.Minute)
	assert.Equal(t, 0, cleaned)
}

func TestServerWebSocketUpgrade(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	serverConfig.EnableAuth = false
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	addr := server.Addr()

	dialer := websocket.Dialer{}
	wsURL := "ws://" + addr + "/ws"

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close()

	assert.NoError(t, err)
}

func TestServerHandleWebSocket(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	serverConfig.EnableAuth = false
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	server.handleWebSocket(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServerRegisterMiddleware(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	called := false
	server.RegisterMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	assert.False(t, called)
}

func TestServerContextCancellation(t *testing.T) {
	serverConfig := DefaultServerConfig()
	serverConfig.Port = 0
	appConfig := &config.Config{}
	server, err := NewServer(serverConfig, appConfig, nil)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)

	ctx := server.Context()
	assert.NotNil(t, ctx)

	go func() {
		time.Sleep(100 * time.Millisecond)
		server.Stop()
	}()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context should be cancelled")
	}
}
