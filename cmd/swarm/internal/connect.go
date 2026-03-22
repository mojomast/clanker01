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
		if connectToken != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Using authentication token: ****\n")
		}
	}

	// Create a temporary client and test connectivity via the health endpoint.
	client, err := NewClient(connectURL, connectToken)
	if err != nil {
		return err
	}
	client.httpClient.Timeout = connectTimeout

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to %s: %w", connectURL, err)
	}

	// Store the active connection for use by subsequent commands.
	conn := &Connection{URL: connectURL, Token: connectToken}
	SetConnection(conn)

	// Persist connection to disk so future CLI invocations can reuse it.
	if err := SaveConnection(conn); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not save connection: %v\n", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Connected to %s\n", connectURL)
	return nil
}
