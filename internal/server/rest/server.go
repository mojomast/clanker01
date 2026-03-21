package rest

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/swarm-ai/swarm/internal/config"
	"github.com/swarm-ai/swarm/internal/security/auth"
)

type ServerConfig struct {
	Host           string
	Port           int
	EnableTLS      bool
	CertFile       string
	KeyFile        string
	EnableAuth     bool
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
	RateLimit      *RateLimitConfig
	Logging        *LoggingConfig
}

type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize         int
	CleanupInterval   time.Duration
}

type LoggingConfig struct {
	Enable       bool
	LogLevel     string
	LogFile      string
	LogRequestID bool
	LogUser      bool
	LogLatency   bool
	IncludeBody  bool
}

type Server struct {
	config      *ServerConfig
	appConfig   *config.Config
	router      *mux.Router
	httpSrv     *http.Server
	jwtAuth     *auth.JWTAuthenticator
	authMW      *auth.AuthMiddleware
	rateLimiter *RateLimiter
	logger      *Logger

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	agents map[string]*AgentInfo
	tasks  map[string]*TaskInfo
	skills map[string]*SkillInfo
}

type AgentInfo struct {
	ID        string
	Type      string
	Name      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Config    map[string]interface{}
	Metrics   map[string]interface{}
}

type TaskInfo struct {
	ID            string
	Type          string
	Prompt        string
	Status        string
	AgentType     string
	AssignedAgent string
	CreatedAt     time.Time
	StartedAt     *time.Time
	CompletedAt   *time.Time
	Result        interface{}
	Error         string
	RetryCount    int
	Metadata      map[string]interface{}
}

type SkillInfo struct {
	ID          string
	Name        string
	Version     string
	Description string
	Status      string
	LoadedAt    time.Time
	Config      map[string]interface{}
}

func NewServer(serverConfig *ServerConfig, appConfig *config.Config, jwtAuth *auth.JWTAuthenticator) (*Server, error) {
	if serverConfig == nil {
		serverConfig = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "startTime", time.Now())

	s := &Server{
		config:    serverConfig,
		appConfig: appConfig,
		router:    mux.NewRouter(),
		jwtAuth:   jwtAuth,
		ctx:       ctx,
		cancel:    cancel,
		agents:    make(map[string]*AgentInfo),
		tasks:     make(map[string]*TaskInfo),
		skills:    make(map[string]*SkillInfo),
	}

	if err := s.initAuth(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	if err := s.initRateLimiter(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize rate limiter: %w", err)
	}

	if err := s.initLogger(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	if err := s.setupRoutes(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	if err := s.initHTTPServer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	return s, nil
}

func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Host:           "0.0.0.0",
		Port:           8080,
		EnableTLS:      false,
		EnableAuth:     true,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
		RateLimit: &RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         200,
			CleanupInterval:   5 * time.Minute,
		},
		Logging: &LoggingConfig{
			Enable:       true,
			LogLevel:     "info",
			LogRequestID: true,
			LogUser:      true,
			LogLatency:   true,
			IncludeBody:  false,
		},
	}
}

func (s *Server) initAuth() error {
	if s.config.EnableAuth && s.jwtAuth != nil {
		sessionMgr := auth.NewMemorySessionManager(nil)
		s.authMW = auth.NewAuthMiddleware(s.jwtAuth, sessionMgr, nil, false)
	}
	return nil
}

func (s *Server) initRateLimiter() error {
	if s.config.RateLimit != nil {
		s.rateLimiter = NewRateLimiter(s.config.RateLimit.RequestsPerSecond, s.config.RateLimit.BurstSize)
		go s.rateLimiter.Cleanup(s.ctx, s.config.RateLimit.CleanupInterval)
	}
	return nil
}

func (s *Server) initLogger() error {
	if s.config.Logging != nil && s.config.Logging.Enable {
		s.logger = NewLogger(s.config.Logging)
	}
	return nil
}

func (s *Server) setupRoutes() error {
	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()

	if s.config.EnableAuth && s.authMW != nil {
		apiRouter.Use(s.authMW.Middleware)
	}

	if s.rateLimiter != nil {
		apiRouter.Use(s.rateLimiter.Middleware)
	}

	if s.logger != nil {
		apiRouter.Use(s.logger.Middleware)
	}

	apiRouter.HandleFunc("/health", s.handleHealth).Methods("GET")
	apiRouter.HandleFunc("/docs", s.handleDocs).Methods("GET")
	apiRouter.HandleFunc("/swagger.json", s.handleSwaggerJSON).Methods("GET")

	agentsRouter := apiRouter.PathPrefix("/agents").Subrouter()
	agentsRouter.HandleFunc("", s.handleListAgents).Methods("GET")
	agentsRouter.HandleFunc("", s.handleCreateAgent).Methods("POST")
	agentsRouter.HandleFunc("/{id}", s.handleGetAgent).Methods("GET")
	agentsRouter.HandleFunc("/{id}", s.handleUpdateAgent).Methods("PUT")
	agentsRouter.HandleFunc("/{id}", s.handleDeleteAgent).Methods("DELETE")
	agentsRouter.HandleFunc("/{id}/start", s.handleStartAgent).Methods("POST")
	agentsRouter.HandleFunc("/{id}/stop", s.handleStopAgent).Methods("POST")
	agentsRouter.HandleFunc("/{id}/pause", s.handlePauseAgent).Methods("POST")
	agentsRouter.HandleFunc("/{id}/resume", s.handleResumeAgent).Methods("POST")
	agentsRouter.HandleFunc("/{id}/metrics", s.handleGetAgentMetrics).Methods("GET")

	tasksRouter := apiRouter.PathPrefix("/tasks").Subrouter()
	tasksRouter.HandleFunc("", s.handleListTasks).Methods("GET")
	tasksRouter.HandleFunc("", s.handleCreateTask).Methods("POST")
	tasksRouter.HandleFunc("/{id}", s.handleGetTask).Methods("GET")
	tasksRouter.HandleFunc("/{id}", s.handleUpdateTask).Methods("PUT")
	tasksRouter.HandleFunc("/{id}", s.handleDeleteTask).Methods("DELETE")
	tasksRouter.HandleFunc("/{id}/start", s.handleStartTask).Methods("POST")
	tasksRouter.HandleFunc("/{id}/cancel", s.handleCancelTask).Methods("POST")

	skillsRouter := apiRouter.PathPrefix("/skills").Subrouter()
	skillsRouter.HandleFunc("", s.handleListSkills).Methods("GET")
	skillsRouter.HandleFunc("", s.handleLoadSkill).Methods("POST")
	skillsRouter.HandleFunc("/{id}", s.handleGetSkill).Methods("GET")
	skillsRouter.HandleFunc("/{id}", s.handleUpdateSkill).Methods("PUT")
	skillsRouter.HandleFunc("/{id}", s.handleUnloadSkill).Methods("DELETE")
	skillsRouter.HandleFunc("/{id}/tools", s.handleListSkillTools).Methods("GET")

	configRouter := apiRouter.PathPrefix("/config").Subrouter()
	configRouter.HandleFunc("", s.handleGetConfig).Methods("GET")
	configRouter.HandleFunc("", s.handleUpdateConfig).Methods("PUT")
	configRouter.HandleFunc("/validate", s.handleValidateConfig).Methods("POST")

	s.router.PathPrefix("/").Handler(s.handleSwaggerUI())

	return nil
}

func (s *Server) initHTTPServer() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpSrv = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
		BaseContext: func(_ net.Listener) context.Context {
			return s.ctx
		},
	}

	return nil
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	listener, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.httpSrv.Addr, err)
	}

	go func() {
		if s.config.EnableTLS && s.config.CertFile != "" && s.config.KeyFile != "" {
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
			s.httpSrv.TLSConfig = tlsConfig

			if err := s.httpSrv.ServeTLS(listener, s.config.CertFile, s.config.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Printf("TLS server error: %v", err)
			}
		} else {
			if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}

func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.httpSrv.Addr
}

func (s *Server) Context() context.Context {
	return s.ctx
}

func (s *Server) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Failed to encode JSON response: %v", err)
		}
	}
}

func (s *Server) respondError(w http.ResponseWriter, statusCode int, message string, err error) {
	errorResponse := map[string]interface{}{
		"error": message,
		"code":  statusCode,
	}
	if err != nil {
		errorResponse["details"] = err.Error()
	}
	s.respondJSON(w, statusCode, errorResponse)
}

func (s *Server) getUserFromRequest(r *http.Request) *auth.User {
	user, _ := auth.GetUserFromContext(r.Context())
	return user
}

func (s *Server) getRequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return ""
}

func (s *Server) handleSwaggerUI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/docs" {
			s.handleDocs(w, r)
			return
		}
		if r.URL.Path == "/api/v1/swagger.json" {
			s.handleSwaggerJSON(w, r)
			return
		}

		http.ServeFile(w, r, "/data/projects/swarm/internal/server/rest/swagger-ui.html")
	})
}
