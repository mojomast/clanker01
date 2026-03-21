package main

import (
	"fmt"
	"os"

	"github.com/swarm-ai/swarm/cmd/swarm/internal"
)

var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
)

func main() {
	if err := internal.Execute(version, commit, builtAt); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
