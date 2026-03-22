package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Connection holds the active connection state to a remote SWARM server.
type Connection struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

var (
	activeConnection *Connection
	connectionMu     sync.RWMutex
)

// SetConnection stores the active connection for reuse across commands.
func SetConnection(conn *Connection) {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	activeConnection = conn
}

// GetConnection returns the active connection, or nil if not connected.
func GetConnection() *Connection {
	connectionMu.RLock()
	defer connectionMu.RUnlock()
	return activeConnection
}

// GetClient returns a Client for the active connection, or an error if not
// connected to a remote server.
func GetClient() (*Client, error) {
	conn := GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("not connected to a server — run 'swarm connect --url <URL>' first")
	}
	client, err := NewClient(conn.URL, conn.Token)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// SaveConnection persists the connection to ~/.config/swarm/connection.json
// so it can be reloaded in future CLI sessions.
func SaveConnection(conn *Connection) error {
	path, err := connectionFilePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(conn, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal connection: %w", err)
	}

	// Write atomically: write to temp file then rename to avoid partial writes.
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write connection file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // clean up
		return fmt.Errorf("failed to commit connection file: %w", err)
	}

	return nil
}

// LoadConnection loads a previously saved connection from disk.
// Returns (nil, nil) if no saved connection exists.
func LoadConnection() (*Connection, error) {
	path, err := connectionFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read connection file: %w", err)
	}

	var conn Connection
	if err := json.Unmarshal(data, &conn); err != nil {
		return nil, fmt.Errorf("failed to parse connection file: %w", err)
	}

	if conn.URL == "" {
		return nil, nil
	}

	return &conn, nil
}

// connectionFilePath returns the path to the saved connection file:
// ~/.config/swarm/connection.json
func connectionFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "swarm", "connection.json"), nil
}
