package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/swarm-ai/swarm/internal/config"
	"github.com/swarm-ai/swarm/internal/core/orchestrator"
	"github.com/swarm-ai/swarm/internal/core/task"
	"github.com/swarm-ai/swarm/internal/providers"
	"github.com/swarm-ai/swarm/internal/security/auth"
	grpcServer "github.com/swarm-ai/swarm/internal/server/grpc"
	restServer "github.com/swarm-ai/swarm/internal/server/rest"
)

var (
	serveHTTPPort   int
	serveGRPCPort   int
	serveDataDir    string
	serveEnableAuth bool
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Swarm server",
		Long: `Start the Swarm server, which boots the REST and gRPC APIs, initializes
configured LLM providers, creates the task orchestrator, and begins
accepting client connections.`,
		RunE: runServe,
	}

	cmd.Flags().IntVar(&serveHTTPPort, "http-port", 0, "HTTP/REST port (overrides config)")
	cmd.Flags().IntVar(&serveGRPCPort, "grpc-port", 0, "gRPC port (overrides config)")
	cmd.Flags().StringVar(&serveDataDir, "data-dir", "", "data directory for orchestrator state")
	cmd.Flags().BoolVar(&serveEnableAuth, "auth", true, "enable authentication")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	// 1. Load configuration
	cfg, err := loadConfig()
	if err != nil {
		// Fall back to a minimal default config if no config file is present.
		cfg = &config.Config{
			Version: "dev",
			Project: config.ProjectConfig{Name: "swarm", Root: "."},
			LLM: config.LLMConfig{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-sonnet-4-20250514",
				Providers:       make(map[string]config.ProviderConfig),
			},
			Server: config.ServerConfig{
				Enabled: true,
				HTTP:    config.HTTPConfig{Port: 8080},
				GRPC:    config.GRPCConfig{Port: 50051},
				Auth:    config.AuthConfig{Enabled: serveEnableAuth},
			},
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: no config file found, using defaults: %v\n", err)
	}

	// Apply CLI flag overrides
	if serveHTTPPort > 0 {
		cfg.Server.HTTP.Port = serveHTTPPort
	}
	if serveGRPCPort > 0 {
		cfg.Server.GRPC.Port = serveGRPCPort
	}

	httpPort := cfg.Server.HTTP.Port
	if httpPort == 0 {
		httpPort = 8080
	}
	grpcPort := cfg.Server.GRPC.Port
	if grpcPort == 0 {
		grpcPort = 50051
	}

	// 2. Initialize LLM providers from config (may be empty if no providers configured)
	var providerRegistry *providers.ProviderRegistry
	if len(cfg.LLM.Providers) > 0 {
		providerRegistry, err = providers.InitializeProviders(cfg.LLM)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to initialize some providers: %v\n", err)
			// Continue without providers — the server can still accept REST/gRPC requests.
		}
	}
	if providerRegistry == nil {
		providerRegistry = providers.GlobalRegistry()
	}
	_ = providerRegistry // Used by agents when they are registered.

	// 3. Set up JWT authenticator
	var jwtAuth *auth.JWTAuthenticator
	if cfg.Server.Auth.Enabled && serveEnableAuth {
		jwtCfg := auth.DefaultJWTConfig()
		if cfg.Server.Auth.JWTSecret != "" {
			jwtCfg.SecretKey = cfg.Server.Auth.JWTSecret
		}
		jwtAuth, err = auth.NewJWTAuthenticator(jwtCfg)
		if err != nil {
			return fmt.Errorf("initialize JWT authenticator: %w", err)
		}
	}

	// 4. Create the task orchestrator
	dataDir := serveDataDir
	if dataDir == "" {
		dataDir = ".swarm/state"
	}

	orch, err := orchestrator.NewOrchestrator(&orchestrator.OrchestratorConfig{
		DataDir:            dataDir,
		ScheduleInterval:   100 * time.Millisecond,
		MaxConcurrent:      10,
		MaxRetries:         3,
		PlannerTemplates:   make(map[string]task.TaskTemplate),
		PlannerConstraints: task.Constraints{},
	})
	if err != nil {
		return fmt.Errorf("create orchestrator: %w", err)
	}

	// 5. Create and start the REST server
	restCfg := &restServer.ServerConfig{
		Host:           "0.0.0.0",
		Port:           httpPort,
		EnableTLS:      false,
		EnableAuth:     cfg.Server.Auth.Enabled && serveEnableAuth,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
		RateLimit: &restServer.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         200,
			CleanupInterval:   5 * time.Minute,
		},
		Logging: &restServer.LoggingConfig{
			Enable:       true,
			LogLevel:     "info",
			LogRequestID: true,
			LogUser:      true,
			LogLatency:   true,
			IncludeBody:  false,
		},
	}

	restSrv, err := restServer.NewServer(restCfg, cfg, jwtAuth)
	if err != nil {
		return fmt.Errorf("create REST server: %w", err)
	}

	if err := restSrv.Start(); err != nil {
		return fmt.Errorf("start REST server: %w", err)
	}

	// 6. Create and start the gRPC server
	grpcCfg := &grpcServer.Config{
		Host:             "0.0.0.0",
		Port:             grpcPort,
		EnableTLS:        false,
		EnableAuth:       cfg.Server.Auth.Enabled && serveEnableAuth,
		EnableReflection: true,
		MaxRecvMsgSize:   1024 * 1024 * 16,
		MaxSendMsgSize:   1024 * 1024 * 16,
		KeepAlive: &grpcServer.KeepAliveConfig{
			MaxConnectionIdle: 5 * time.Minute,
			MaxConnectionAge:  5 * time.Minute,
			Time:              10 * time.Second,
			Timeout:           3 * time.Second,
		},
	}

	grpcSrv, err := grpcServer.NewServer(grpcCfg, jwtAuth)
	if err != nil {
		return fmt.Errorf("create gRPC server: %w", err)
	}

	if err := grpcSrv.Start(); err != nil {
		_ = restSrv.Stop()
		return fmt.Errorf("start gRPC server: %w", err)
	}

	// 7. Start the orchestrator
	orchCtx, orchCancel := context.WithCancel(context.Background())
	defer orchCancel()

	if err := orch.Start(orchCtx); err != nil {
		_ = restSrv.Stop()
		_ = grpcSrv.Stop()
		return fmt.Errorf("start orchestrator: %w", err)
	}

	// 8. Print startup banner
	fmt.Fprintf(cmd.OutOrStdout(), "Swarm server started\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  REST API: http://0.0.0.0:%d/api/v1\n", httpPort)
	fmt.Fprintf(cmd.OutOrStdout(), "  gRPC:     0.0.0.0:%d\n", grpcPort)
	fmt.Fprintf(cmd.OutOrStdout(), "  Auth:     %v\n", cfg.Server.Auth.Enabled && serveEnableAuth)
	fmt.Fprintf(cmd.OutOrStdout(), "\nPress Ctrl+C to stop.\n")

	// 9. Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Fprintf(cmd.OutOrStdout(), "\nShutting down...\n")

	// 10. Graceful shutdown
	orchCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := orch.Stop(shutdownCtx); err != nil {
		log.Printf("Warning: orchestrator stop: %v", err)
	}
	if err := grpcSrv.Stop(); err != nil {
		log.Printf("Warning: gRPC server stop: %v", err)
	}
	if err := restSrv.Stop(); err != nil {
		log.Printf("Warning: REST server stop: %v", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Server stopped.\n")
	return nil
}
