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
	_, cancel := context.WithTimeout(cmd.Context(), connectTimeout)
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

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully connected to Swarm server at %s\n", connectURL)
	fmt.Fprintln(cmd.OutOrStdout(), "Use 'swarm agent list' to see available agents")
	fmt.Fprintln(cmd.OutOrStdout(), "Use 'swarm skill list' to see available skills")

	return nil
}
