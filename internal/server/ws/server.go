package ws

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/swarm-ai/swarm/internal/config"
	"github.com/swarm-ai/swarm/internal/security/auth"
	"github.com/swarm-ai/swarm/pkg/api"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Allow non-browser clients with no Origin header
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return u.Host == r.Host
	},
}

type ServerConfig struct {
	Host             string
	Port             int
	Path             string
	EnableAuth       bool
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	PingTimeout      time.Duration
	PongTimeout      time.Duration
	MaxMessageSize   int64
	BufferSize       int
	BroadcastWorkers int
}

func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:             "0.0.0.0",
		Port:             8081,
		Path:             "/ws",
		EnableAuth:       true,
		ReadTimeout:      10 * time.Second,
		WriteTimeout:     10 * time.Second,
		PingTimeout:      60 * time.Second,
		PongTimeout:      60 * time.Second,
		MaxMessageSize:   1024 * 1024,
		BufferSize:       256,
		BroadcastWorkers: 4,
	}
}

type Server struct {
	config       *ServerConfig
	appConfig    *config.Config
	router       *mux.Router
	httpServer   *http.Server
	hub          *Hub
	broadcastMgr *BroadcastManager
	authMW       *auth.AuthMiddleware
	listener     net.Listener

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

func NewServer(config *ServerConfig, appConfig *config.Config, authMW *auth.AuthMiddleware) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	hub := NewHub()
	broadcastMgr := NewBroadcastManager(hub, config.BroadcastWorkers)

	s := &Server{
		config:       config,
		appConfig:    appConfig,
		router:       mux.NewRouter(),
		hub:          hub,
		broadcastMgr: broadcastMgr,
		authMW:       authMW,
		ctx:          ctx,
		cancel:       cancel,
		running:      false,
	}

	if err := s.setupRoutes(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	return s, nil
}

func (s *Server) setupRoutes() error {
	wsRouter := s.router.PathPrefix(s.config.Path).Subrouter()

	if s.config.EnableAuth && s.authMW != nil {
		wsRouter.Use(s.authMW.Middleware)
	}

	wsRouter.HandleFunc("", s.handleWebSocket).Methods("GET")

	return nil
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	go s.hub.Run()
	s.broadcastMgr.Start()

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.hub.Error(fmt.Errorf("server error: %w", err))
		}
	}()

	s.running = true
	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.cancel()

	s.broadcastMgr.Stop()
	s.hub.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.running = false
	return nil
}

func (s *Server) Hub() *Hub {
	return s.hub
}

func (s *Server) BroadcastManager() *BroadcastManager {
	return s.broadcastMgr
}

func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

func (s *Server) Context() context.Context {
	return s.ctx
}

func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.hub.Error(fmt.Errorf("failed to upgrade websocket: %w", err))
		return
	}

	userID := ExtractUserIDFromContext(r)
	if userID == "" {
		userID = ExtractUserIDFromRequest(r)
	}

	client := NewClient(s.hub, conn, userID)
	client.Connect()

	s.hub.Register(client)
}

func (s *Server) BroadcastAgentUpdate(agent interface{}, metrics interface{}, health interface{}) error {
	// Type-assert the parameters to the expected types before passing through
	typedAgent, _ := agent.(api.Agent)
	typedMetrics, _ := metrics.(*api.AgentMetrics)
	typedHealth, _ := health.(*api.AgentHealth)
	return s.broadcastMgr.BroadcastAgentUpdate(typedAgent, typedMetrics, typedHealth)
}

func (s *Server) BroadcastTaskEvent(task interface{}, eventType string, result interface{}, err error) error {
	// Type-assert the parameters to the expected types before passing through
	typedTask, _ := task.(*api.Task)
	typedResult, _ := result.(*api.TaskResult)
	return s.broadcastMgr.BroadcastTaskEvent(typedTask, eventType, typedResult, err)
}

func (s *Server) BroadcastLog(level, message, agentID, taskID string) error {
	return s.broadcastMgr.BroadcastLog(level, message, agentID, taskID)
}

func (s *Server) BroadcastToUser(msg *Message, userID string) error {
	return s.broadcastMgr.BroadcastToUser(msg, userID)
}

func (s *Server) BroadcastToAgent(msg *Message, agentID string) error {
	return s.broadcastMgr.BroadcastToAgent(msg, agentID)
}

func (s *Server) BroadcastToTask(msg *Message, taskID string) error {
	return s.broadcastMgr.BroadcastToTask(msg, taskID)
}

func (s *Server) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"running":   s.IsRunning(),
		"address":   s.Addr(),
		"hub":       s.hub.Stats(),
		"broadcast": s.broadcastMgr.Stats(),
		"config": map[string]interface{}{
			"host":              s.config.Host,
			"port":              s.config.Port,
			"path":              s.config.Path,
			"enable_auth":       s.config.EnableAuth,
			"buffer_size":       s.config.BufferSize,
			"broadcast_workers": s.config.BroadcastWorkers,
		},
	}
}

func (s *Server) HandleAgentUpdate(agent interface{}, metrics interface{}, health interface{}) {
	_ = s.BroadcastAgentUpdate(agent, metrics, health)
}

func (s *Server) HandleTaskEvent(task interface{}, eventType string, result interface{}, err error) {
	_ = s.BroadcastTaskEvent(task, eventType, result, err)
}

func (s *Server) HandleLog(level, message, agentID, taskID string) {
	_ = s.BroadcastLog(level, message, agentID, taskID)
}

type ServerInfo struct {
	Running bool   `json:"running"`
	Address string `json:"address"`
	Path    string `json:"path"`
}

func (s *Server) Info() *ServerInfo {
	return &ServerInfo{
		Running: s.IsRunning(),
		Address: s.Addr(),
		Path:    s.config.Path,
	}
}

func (s *Server) RegisterMiddleware(middleware mux.MiddlewareFunc) {
	s.router.Use(middleware)
}

func (s *Server) GracefulShutdown(timeout time.Duration) error {
	stopChan := make(chan struct{})
	go func() {
		_ = s.Stop()
		close(stopChan)
	}()

	select {
	case <-stopChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}

func (s *Server) CleanupStaleClients(timeout time.Duration) int {
	count := 0
	for _, client := range s.hub.GetAllClients() {
		if client.IsStale(timeout) {
			_ = client.Close()
			s.hub.Unregister(client)
			count++
		}
	}
	return count
}

func (s *Server) StartCleanupTask(interval time.Duration, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cleaned := s.CleanupStaleClients(timeout)
				if cleaned > 0 {
					s.hub.Error(fmt.Errorf("cleaned %d stale clients", cleaned))
				}
			case <-s.ctx.Done():
				return
			}
		}
	}()
}
