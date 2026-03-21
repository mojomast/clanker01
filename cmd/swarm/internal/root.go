package internal

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var (
	version string
	commit  string
	builtAt string
)

var rootCmd = &cobra.Command{
	Use:   "swarm",
	Short: "Swarm AI Agent Platform CLI",
	Long: `Swarm is an AI coding platform that decomposes complex user requests 
into executable tasks, manages dependencies, and coordinates multiple agent 
roles to complete software engineering work.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	cobra.EnableCommandSorting = false

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.swarm.yaml or ./swarm.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(newConnectCmd())
	rootCmd.AddCommand(newAgentCmd())
	rootCmd.AddCommand(newSkillCmd())
	rootCmd.AddCommand(newVersionCmd())
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the version, commit, and build time of the Swarm CLI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Swarm CLI Version: %s\n", version)
			fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", commit)
			fmt.Fprintf(cmd.OutOrStdout(), "Built: %s\n", builtAt)
			return nil
		},
	}
	return cmd
}

func Execute(v, c, b string) error {
	version = v
	commit = c
	builtAt = b
	return rootCmd.Execute()
}

func GetRootCmd() *cobra.Command {
	return rootCmd
}
