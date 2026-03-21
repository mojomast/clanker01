package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	connectTimeout time.Duration
	connectURL     string
	connectToken   string
)

func newConnectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to a Swarm server",
		Long:  `Connect to a remote Swarm server to interact with agents and skills.`,
		RunE:  runConnect,
	}

	cmd.Flags().DurationVar(&connectTimeout, "timeout", 30*time.Second, "connection timeout")
	cmd.Flags().StringVar(&connectURL, "url", "", "server URL (e.g., http://localhost:8080)")
	cmd.Flags().StringVar(&connectToken, "token", "", "authentication token")

	return cmd
}

func runConnect(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), connectTimeout)
	defer cancel()

	if connectURL == "" {
		cfg := loadConfigOrDefault()
		if cfg.Server.Enabled {
			connectURL = fmt.Sprintf("http://localhost:%d", cfg.Server.HTTP.Port)
		} else {
			return fmt.Errorf("no server URL provided and server not configured")
		}
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Connecting to: %s\n", connectURL)
	}

	// TODO: Use ctx for the actual network call when connection logic is implemented.
	// TODO: Use connectToken for authentication when connection logic is implemented.
	_ = ctx

	return fmt.Errorf("not yet implemented: connection to %s", connectURL)
}
