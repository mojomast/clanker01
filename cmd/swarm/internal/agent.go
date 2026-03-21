package internal

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/swarm-ai/swarm/pkg/api"
)

var (
	agentType      string
	agentModel     string
	agentName      string
	agentOutput    string
	showAgentStats bool
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
		Long:  `Commands to create, list, delete, and manage Swarm agents.`,
	}

	cmd.AddCommand(newAgentListCmd())
	cmd.AddCommand(newAgentCreateCmd())
	cmd.AddCommand(newAgentDeleteCmd())
	cmd.AddCommand(newAgentInfoCmd())
	cmd.AddCommand(newAgentStatsCmd())

	return cmd
}

func newAgentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		Long:  `List all registered agents with their status and configuration.`,
		RunE:  runAgentList,
	}

	cmd.Flags().StringVarP(&agentType, "type", "t", "", "filter by agent type")
	cmd.Flags().StringVarP(&agentOutput, "output", "o", "table", "output format (table, json, yaml)")

	return cmd
}

func runAgentList(cmd *cobra.Command, args []string) error {
	cfg := loadConfigOrDefault()

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Loading agent configuration from: %s\n", cfg.Project.Name)
	}

	agents := getSampleAgents()

	if agentType != "" {
		filtered := []agentInfo{}
		for _, a := range agents {
			if string(a.Type) == agentType {
				filtered = append(filtered, a)
			}
		}
		agents = filtered
	}

	switch agentOutput {
	case "json":
		printAgentsJSON(cmd.OutOrStdout(), agents)
	case "yaml":
		printAgentsYAML(cmd.OutOrStdout(), agents)
	default:
		printAgentsTable(cmd.OutOrStdout(), agents)
	}

	return nil
}

func newAgentCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new agent",
		Long:  `Create a new agent with the specified name and type.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runAgentCreate,
	}

	cmd.Flags().StringVarP(&agentType, "type", "t", string(api.AgentTypeCoder), "agent type (architect, coder, tester, reviewer, researcher, coordinator)")
	cmd.Flags().StringVar(&agentModel, "model", "", "model to use for the agent")

	return cmd
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if agentType == "" {
		agentType = string(api.AgentTypeCoder)
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Creating agent '%s' of type '%s'\n", name, agentType)
	}

	cfg := loadConfigOrDefault()

	if agentModel == "" {
		model, err := cfg.LLM.AgentModelMapping[agentType]
		if !err {
			agentModel = cfg.LLM.DefaultModel
		} else {
			agentModel = model
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully created agent '%s'\n", name)
	fmt.Fprintf(cmd.OutOrStdout(), "  Type: %s\n", agentType)
	fmt.Fprintf(cmd.OutOrStdout(), "  Model: %s\n", agentModel)

	return nil
}

func newAgentDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an agent",
		Long:  `Delete the specified agent. This action cannot be undone.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runAgentDelete,
	}

	return cmd
}

func runAgentDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Deleting agent '%s'\n", name)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted agent '%s'\n", name)

	return nil
}

func newAgentInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed information about an agent",
		Long:  `Display detailed configuration and status information for the specified agent.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runAgentInfo,
	}

	return cmd
}

func runAgentInfo(cmd *cobra.Command, args []string) error {
	name := args[0]

	agent, err := findAgent(name)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Agent: %s\n", agent.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "  ID: %s\n", agent.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "  Type: %s\n", agent.Type)
	fmt.Fprintf(cmd.OutOrStdout(), "  Model: %s\n", agent.Model)
	fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", agent.Status)
	fmt.Fprintf(cmd.OutOrStdout(), "  Tasks Completed: %d\n", agent.TasksCompleted)
	fmt.Fprintf(cmd.OutOrStdout(), "  Tasks Failed: %d\n", agent.TasksFailed)
	fmt.Fprintf(cmd.OutOrStdout(), "  Last Activity: %s\n", agent.LastActivity)

	return nil
}

func newAgentStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show agent statistics",
		Long:  `Display aggregated statistics for all agents or a specific agent type.`,
		RunE:  runAgentStats,
	}

	cmd.Flags().StringVarP(&agentType, "type", "t", "", "filter by agent type")

	return cmd
}

func runAgentStats(cmd *cobra.Command, args []string) error {
	agents := getSampleAgents()

	if agentType != "" {
		filtered := []agentInfo{}
		for _, a := range agents {
			if string(a.Type) == agentType {
				filtered = append(filtered, a)
			}
		}
		agents = filtered
	}

	totalTasks := 0
	failedTasks := 0
	for _, a := range agents {
		totalTasks += a.TasksCompleted + a.TasksFailed
		failedTasks += a.TasksFailed
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Agent Statistics")
	fmt.Fprintf(cmd.OutOrStdout(), "  Total Agents: %d\n", len(agents))
	fmt.Fprintf(cmd.OutOrStdout(), "  Total Tasks: %d\n", totalTasks)
	fmt.Fprintf(cmd.OutOrStdout(), "  Successful Tasks: %d\n", totalTasks-failedTasks)
	fmt.Fprintf(cmd.OutOrStdout(), "  Failed Tasks: %d\n", failedTasks)
	if totalTasks > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Success Rate: %.2f%%\n", float64(totalTasks-failedTasks)/float64(totalTasks)*100)
	}

	return nil
}

type agentInfo struct {
	ID             string
	Name           string
	Type           api.AgentType
	Model          string
	Status         api.AgentStatus
	TasksCompleted int
	TasksFailed    int
	LastActivity   string
}

func getSampleAgents() []agentInfo {
	return []agentInfo{
		{
			ID:             "agent-001",
			Name:           "architect-1",
			Type:           api.AgentTypeArchitect,
			Model:          "gpt-4",
			Status:         api.AgentStatusReady,
			TasksCompleted: 42,
			TasksFailed:    3,
			LastActivity:   "2024-03-21 15:30:00",
		},
		{
			ID:             "agent-002",
			Name:           "coder-1",
			Type:           api.AgentTypeCoder,
			Model:          "gpt-3.5-turbo",
			Status:         api.AgentStatusRunning,
			TasksCompleted: 128,
			TasksFailed:    12,
			LastActivity:   "2024-03-21 16:45:00",
		},
		{
			ID:             "agent-003",
			Name:           "tester-1",
			Type:           api.AgentTypeTester,
			Model:          "gpt-3.5-turbo",
			Status:         api.AgentStatusReady,
			TasksCompleted: 87,
			TasksFailed:    5,
			LastActivity:   "2024-03-21 16:30:00",
		},
		{
			ID:             "agent-004",
			Name:           "reviewer-1",
			Type:           api.AgentTypeReviewer,
			Model:          "gpt-4",
			Status:         api.AgentStatusReady,
			TasksCompleted: 65,
			TasksFailed:    2,
			LastActivity:   "2024-03-21 16:20:00",
		},
	}
}

func findAgent(name string) (*agentInfo, error) {
	agents := getSampleAgents()
	for _, a := range agents {
		if a.Name == name {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("agent '%s' not found", name)
}

func printAgentsTable(w io.Writer, agents []agentInfo) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tMODEL\tSTATUS\tTASKS\tLAST ACTIVITY")
	for _, a := range agents {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d/%d\t%s\n",
			a.Name, a.Type, a.Model, a.Status, a.TasksCompleted, a.TasksCompleted+a.TasksFailed, a.LastActivity)
	}
	tw.Flush()
}

func printAgentsJSON(w io.Writer, agents []agentInfo) {
	fmt.Fprintln(w, "[")
	for i, a := range agents {
		if i > 0 {
			fmt.Fprintln(w, ",")
		}
		fmt.Fprintf(w, "  {\n")
		fmt.Fprintf(w, "    \"name\": \"%s\",\n", a.Name)
		fmt.Fprintf(w, "    \"type\": \"%s\",\n", a.Type)
		fmt.Fprintf(w, "    \"model\": \"%s\",\n", a.Model)
		fmt.Fprintf(w, "    \"status\": \"%s\",\n", a.Status)
		fmt.Fprintf(w, "    \"tasksCompleted\": %d,\n", a.TasksCompleted)
		fmt.Fprintf(w, "    \"tasksFailed\": %d\n", a.TasksFailed)
		fmt.Fprintf(w, "  }")
	}
	fmt.Fprintln(w, "\n]")
}

func printAgentsYAML(w io.Writer, agents []agentInfo) {
	for _, a := range agents {
		fmt.Fprintf(w, "- name: %s\n", a.Name)
		fmt.Fprintf(w, "  type: %s\n", a.Type)
		fmt.Fprintf(w, "  model: %s\n", a.Model)
		fmt.Fprintf(w, "  status: %s\n", a.Status)
		fmt.Fprintf(w, "  tasksCompleted: %d\n", a.TasksCompleted)
		fmt.Fprintf(w, "  tasksFailed: %d\n", a.TasksFailed)
		fmt.Fprintln(w)
	}
}
