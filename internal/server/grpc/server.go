package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	grpcAuth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/swarm-ai/swarm/internal/security/auth"
	"github.com/swarm-ai/swarm/internal/server/grpc/proto"
)

type Config struct {
	Host             string
	Port             int
	EnableTLS        bool
	CertFile         string
	KeyFile          string
	EnableAuth       bool
	EnableReflection bool
	MaxRecvMsgSize   int
	MaxSendMsgSize   int
	KeepAlive        *KeepAliveConfig
}

type KeepAliveConfig struct {
	MaxConnectionIdle time.Duration
	MaxConnectionAge  time.Duration
	Time              time.Duration
	Timeout           time.Duration
}

type Server struct {
	config     *Config
	grpcServer *grpc.Server
	listener   net.Listener
	jwtAuth    *auth.JWTAuthenticator

	agentHandler *AgentHandler
	taskHandler  *TaskHandler
	skillHandler *SkillHandler

	streamManager *StreamManager

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func NewServer(config *Config, jwtAuth *auth.JWTAuthenticator) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		config:        config,
		jwtAuth:       jwtAuth,
		streamManager: NewStreamManager(),
		ctx:           ctx,
		cancel:        cancel,
	}

	if err := s.initHandlers(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	if err := s.initGRPCServer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize gRPC server: %w", err)
	}

	return s, nil
}

func DefaultConfig() *Config {
	return &Config{
		Host:             "0.0.0.0",
		Port:             50051,
		EnableTLS:        false,
		EnableAuth:       true,
		EnableReflection: true,
		MaxRecvMsgSize:   1024 * 1024 * 16,
		MaxSendMsgSize:   1024 * 1024 * 16,
		KeepAlive: &KeepAliveConfig{
			MaxConnectionIdle: 5 * time.Minute,
			MaxConnectionAge:  5 * time.Minute,
			Time:              10 * time.Second,
			Timeout:           3 * time.Second,
		},
	}
}

func (s *Server) initHandlers() error {
	s.agentHandler = NewAgentHandler(s)
	s.taskHandler = NewTaskHandler(s)
	s.skillHandler = NewSkillHandler(s)
	return nil
}

func (s *Server) initGRPCServer() error {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(s.config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.config.MaxSendMsgSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: s.config.KeepAlive.MaxConnectionIdle,
			MaxConnectionAge:  s.config.KeepAlive.MaxConnectionAge,
			Time:              s.config.KeepAlive.Time,
			Timeout:           s.config.KeepAlive.Timeout,
		}),
	}

	if s.config.EnableAuth {
		opts = append(opts,
			grpc.ChainUnaryInterceptor(
				grpcRecovery.UnaryServerInterceptor(),
				grpcAuth.UnaryServerInterceptor(s.authFunc),
				loggingInterceptor(),
			),
			grpc.ChainStreamInterceptor(
				grpcRecovery.StreamServerInterceptor(),
				grpcAuth.StreamServerInterceptor(s.authFunc),
				streamLoggingInterceptor(),
			),
		)
	} else {
		opts = append(opts,
			grpc.ChainUnaryInterceptor(
				grpcRecovery.UnaryServerInterceptor(),
				loggingInterceptor(),
			),
			grpc.ChainStreamInterceptor(
				grpcRecovery.StreamServerInterceptor(),
				streamLoggingInterceptor(),
			),
		)
	}

	creds := insecure.NewCredentials()
	if s.config.EnableTLS {
		cert, err := credentials.NewServerTLSFromFile(s.config.CertFile, s.config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		creds = cert
	}
	opts = append(opts, grpc.Creds(creds))

	s.grpcServer = grpc.NewServer(opts...)

	proto.RegisterAgentServiceServer(s.grpcServer, s.agentHandler)
	proto.RegisterTaskServiceServer(s.grpcServer, s.taskHandler)
	proto.RegisterSkillServiceServer(s.grpcServer, s.skillHandler)

	if s.config.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	return nil
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener

	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				status.Errorf(codes.Internal, "server error: %v", err)
			}
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancel()
	s.streamManager.CloseAll()

	if s.grpcServer != nil {
		stopped := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
		case <-time.After(10 * time.Second):
			s.grpcServer.Stop()
		}
	}

	if s.listener != nil {
		s.listener.Close()
	}

	return nil
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

func (s *Server) StreamManager() *StreamManager {
	return s.streamManager
}

func (s *Server) AuthFunc() func(ctx context.Context) (context.Context, error) {
	return s.authFunc
}

func (s *Server) authFunc(ctx context.Context) (context.Context, error) {
	if !s.config.EnableAuth {
		return ctx, nil
	}

	token, err := grpcAuth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	user, err := s.jwtAuth.ValidateToken(ctx, token)
	if err != nil {
		if err == auth.ErrExpiredToken {
			return nil, status.Errorf(codes.Unauthenticated, "token expired")
		}
		return nil, status.Errorf(codes.Unauthenticated, "invalid token")
	}

	newCtx := context.WithValue(ctx, userContextKey{}, user)
	return newCtx, nil
}

func loggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			_ = status.Code(err)
			_ = duration.String()
		}

		return resp, err
	}
}

func streamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		if err != nil {
			_ = status.Code(err)
		}
		return err
	}
}

type userContextKey struct{}

func GetUserFromContext(ctx context.Context) (*auth.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(*auth.User)
	return user, ok
}
