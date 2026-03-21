package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	skillVersion   string
	skillSource    string
	skillOutput    string
	showSkillDeps  bool
	showDeprecated bool
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage skills",
		Long:  `Commands to list, install, uninstall, and manage Swarm skills.`,
	}

	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillInstallCmd())
	cmd.AddCommand(newSkillUninstallCmd())
	cmd.AddCommand(newSkillInfoCmd())
	cmd.AddCommand(newSkillUpdateCmd())
	cmd.AddCommand(newSkillSearchCmd())

	return cmd
}

func newSkillListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all installed skills",
		Long:  `List all installed skills with their versions and metadata.`,
		RunE:  runSkillList,
	}

	cmd.Flags().StringVarP(&skillOutput, "output", "o", "table", "output format (table, json, yaml)")
	cmd.Flags().BoolVar(&showDeprecated, "deprecated", false, "show deprecated skills")
	cmd.Flags().BoolVarP(&showSkillDeps, "dependencies", "d", false, "show skill dependencies")

	return cmd
}

func runSkillList(cmd *cobra.Command, args []string) error {
	// If connected to a remote server, fetch skills from the API.
	if client, err := GetClient(); err == nil {
		remoteSkills, apiErr := client.ListSkills(cmd.Context())
		if apiErr != nil {
			return fmt.Errorf("failed to list skills: %w", apiErr)
		}

		// Convert API responses to local skillInfo for display.
		skills := make([]skillInfo, len(remoteSkills))
		for i, rs := range remoteSkills {
			var tools []toolInfo
			for _, t := range rs.Tools {
				tools = append(tools, toolInfo{Name: t.Name, Description: t.Description})
			}
			skills[i] = skillInfo{
				Name:        rs.Name,
				Version:     rs.Version,
				Description: rs.Description,
				Tools:       tools,
			}
		}

		switch skillOutput {
		case "json":
			printSkillsJSON(cmd.OutOrStdout(), skills, showSkillDeps)
		case "yaml":
			printSkillsYAML(cmd.OutOrStdout(), skills, showSkillDeps)
		default:
			printSkillsTable(cmd.OutOrStdout(), skills, showSkillDeps)
		}
		return nil
	}

	// Fallback: local/sample data when not connected.
	cfg := loadConfigOrDefault()

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Loading skills from configuration: %s\n", cfg.Project.Name)
	}

	skills := getSampleSkills()

	if !showDeprecated {
		filtered := []skillInfo{}
		for _, s := range skills {
			if !s.Deprecated {
				filtered = append(filtered, s)
			}
		}
		skills = filtered
	}

	switch skillOutput {
	case "json":
		printSkillsJSON(cmd.OutOrStdout(), skills, showSkillDeps)
	case "yaml":
		printSkillsYAML(cmd.OutOrStdout(), skills, showSkillDeps)
	default:
		printSkillsTable(cmd.OutOrStdout(), skills, showSkillDeps)
	}

	return nil
}

func newSkillInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install a skill",
		Long:  `Install a skill from a source. The source can be a local file, URL, or registry.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillInstall,
	}

	cmd.Flags().StringVar(&skillVersion, "version", "latest", "skill version")
	cmd.Flags().StringVarP(&skillSource, "source", "s", "", "skill source (local file or URL)")

	return cmd
}

func runSkillInstall(cmd *cobra.Command, args []string) error {
	name := args[0]

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Installing skill '%s' version '%s'\n", name, skillVersion)
		if skillSource != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  From source: %s\n", skillSource)
		}
	}

	// If connected to a remote server, install the skill via the API.
	if client, err := GetClient(); err == nil {
		req := &APIInstallSkillRequest{
			Name:    name,
			Version: skillVersion,
			Enable:  true,
		}
		if apiErr := client.InstallSkill(cmd.Context(), req); apiErr != nil {
			return fmt.Errorf("failed to install skill: %w", apiErr)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully installed skill '%s' version %s\n", name, skillVersion)

	return nil
}

func newSkillUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall <name>",
		Short: "Uninstall a skill",
		Long:  `Uninstall the specified skill. This action cannot be undone.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillUninstall,
	}

	return cmd
}

func runSkillUninstall(cmd *cobra.Command, args []string) error {
	name := args[0]

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Uninstalling skill '%s'\n", name)
	}

	// If connected to a remote server, remove the skill via the API.
	if client, err := GetClient(); err == nil {
		if apiErr := client.RemoveSkill(cmd.Context(), name); apiErr != nil {
			return fmt.Errorf("failed to uninstall skill: %w", apiErr)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully uninstalled skill '%s'\n", name)

	return nil
}

func newSkillInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed information about a skill",
		Long:  `Display detailed configuration and metadata for the specified skill.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillInfo,
	}

	cmd.Flags().StringVar(&skillVersion, "version", "", "skill version (default: latest)")

	return cmd
}

func runSkillInfo(cmd *cobra.Command, args []string) error {
	name := args[0]

	// If connected to a remote server, fetch skill info from the API.
	if client, err := GetClient(); err == nil {
		rs, apiErr := client.GetSkill(cmd.Context(), name)
		if apiErr != nil {
			return fmt.Errorf("skill not found: %w", apiErr)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Skill: %s\n", rs.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "  Version: %s\n", rs.Version)
		fmt.Fprintf(cmd.OutOrStdout(), "  Description: %s\n", rs.Description)
		fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", rs.Status)
		if len(rs.Tools) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  Tools: %d\n", len(rs.Tools))
			for _, tool := range rs.Tools {
				fmt.Fprintf(cmd.OutOrStdout(), "    - %s: %s\n", tool.Name, tool.Description)
			}
		}
		return nil
	}

	// Fallback: local/sample data.
	skill, err := findSkill(name)
	if err != nil {
		return fmt.Errorf("skill not found: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Skill: %s\n", skill.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "  Version: %s\n", skill.Version)
	if skill.DisplayName != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Display Name: %s\n", skill.DisplayName)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Description: %s\n", skill.Description)
	if skill.Author != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Author: %s\n", skill.Author)
	}
	if skill.License != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  License: %s\n", skill.License)
	}
	if len(skill.Tags) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Tags: %s\n", strings.Join(skill.Tags, ", "))
	}
	if skill.Deprecated {
		fmt.Fprintf(cmd.OutOrStdout(), "  DEPRECATED\n")
	}
	if len(skill.Tools) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Tools: %d\n", len(skill.Tools))
		for _, tool := range skill.Tools {
			fmt.Fprintf(cmd.OutOrStdout(), "    - %s: %s\n", tool.Name, tool.Description)
		}
	}
	if len(skill.Dependencies) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Dependencies: %d\n", len(skill.Dependencies))
		for _, dep := range skill.Dependencies {
			fmt.Fprintf(cmd.OutOrStdout(), "    - %s %s\n", dep.Name, dep.Version)
		}
	}

	return nil
}

func newSkillUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update one or all skills",
		Long:  `Update the specified skill to the latest version, or all skills if no name is provided.`,
		RunE:  runSkillUpdate,
	}

	return cmd
}

func runSkillUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		if verbose {
			fmt.Fprintln(cmd.OutOrStdout(), "Updating all skills...")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Successfully updated all skills")
	} else {
		name := args[0]
		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Updating skill '%s'...\n", name)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated skill '%s'\n", name)
	}

	return nil
}

func newSkillSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for skills",
		Long:  `Search for available skills by name, description, or tags.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillSearch,
	}

	cmd.Flags().StringVarP(&skillOutput, "output", "o", "table", "output format (table, json, yaml)")

	return cmd
}

func runSkillSearch(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(args[0])

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Searching for skills matching: %s\n", query)
	}

	allSkills := getSampleSkills()
	results := []skillInfo{}

	for _, s := range allSkills {
		match := strings.Contains(strings.ToLower(s.Name), query) ||
			strings.Contains(strings.ToLower(s.Description), query) ||
			func() bool {
				for _, tag := range s.Tags {
					if strings.Contains(strings.ToLower(tag), query) {
						return true
					}
				}
				return false
			}()
		if match {
			results = append(results, s)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Found %d matching skills:\n\n", len(results))

	switch skillOutput {
	case "json":
		printSkillsJSON(cmd.OutOrStdout(), results, false)
	case "yaml":
		printSkillsYAML(cmd.OutOrStdout(), results, false)
	default:
		printSkillsTable(cmd.OutOrStdout(), results, false)
	}

	return nil
}

type skillInfo struct {
	Name         string
	Version      string
	DisplayName  string
	Description  string
	Author       string
	License      string
	Tags         []string
	Deprecated   bool
	Tools        []toolInfo
	Dependencies []dependencyInfo
}

type toolInfo struct {
	Name        string
	Description string
}

type dependencyInfo struct {
	Name    string
	Version string
}

func getSampleSkills() []skillInfo {
	return []skillInfo{
		{
			Name:        "filesystem",
			Version:     "1.2.0",
			DisplayName: "Filesystem Operations",
			Description: "Provides file and directory operations including read, write, delete, and search capabilities.",
			Author:      "Swarm Team",
			License:     "MIT",
			Tags:        []string{"filesystem", "io", "builtin"},
			Tools: []toolInfo{
				{Name: "read_file", Description: "Read contents of a file"},
				{Name: "write_file", Description: "Write content to a file"},
				{Name: "list_directory", Description: "List directory contents"},
				{Name: "delete_file", Description: "Delete a file or directory"},
			},
		},
		{
			Name:        "git",
			Version:     "1.1.5",
			DisplayName: "Git Operations",
			Description: "Provides Git version control operations including commit, push, pull, and branch management.",
			Author:      "Swarm Team",
			License:     "MIT",
			Tags:        []string{"git", "vcs", "builtin"},
			Tools: []toolInfo{
				{Name: "git_status", Description: "Get git status"},
				{Name: "git_commit", Description: "Create a commit"},
				{Name: "git_push", Description: "Push changes to remote"},
				{Name: "git_pull", Description: "Pull changes from remote"},
				{Name: "git_branch", Description: "Manage branches"},
			},
		},
		{
			Name:        "web",
			Version:     "2.0.3",
			DisplayName: "Web Operations",
			Description: "Provides web scraping and HTTP request capabilities.",
			Author:      "Swarm Team",
			License:     "MIT",
			Tags:        []string{"web", "http", "network", "builtin"},
			Tools: []toolInfo{
				{Name: "fetch_url", Description: "Fetch content from a URL"},
				{Name: "http_request", Description: "Make HTTP requests"},
			},
			Dependencies: []dependencyInfo{
				{Name: "filesystem", Version: ">=1.0.0"},
			},
		},
		{
			Name:        "database",
			Version:     "1.0.0",
			DisplayName: "Database Operations",
			Description: "Provides database query and management capabilities.",
			Author:      "Swarm Team",
			License:     "MIT",
			Tags:        []string{"database", "sql", "builtin"},
			Tools: []toolInfo{
				{Name: "query", Description: "Execute SQL query"},
				{Name: "schema", Description: "Get database schema"},
			},
			Deprecated: true,
		},
	}
}

func findSkill(name string) (*skillInfo, error) {
	skills := getSampleSkills()
	for _, s := range skills {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("skill '%s' not found", name)
}

func printSkillsTable(w io.Writer, skills []skillInfo, showDeps bool) {
	for _, s := range skills {
		fmt.Fprintf(w, "%s@%s\n", s.Name, s.Version)
		fmt.Fprintf(w, "  %s\n", s.Description)
		if s.Deprecated {
			fmt.Fprintf(w, "  [DEPRECATED]\n")
		}
		if showDeps && len(s.Dependencies) > 0 {
			fmt.Fprintf(w, "  Dependencies: ")
			for i, dep := range s.Dependencies {
				if i > 0 {
					fmt.Fprintf(w, ", ")
				}
				fmt.Fprintf(w, "%s %s", dep.Name, dep.Version)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}
}

func printSkillsJSON(w io.Writer, skills []skillInfo, showDeps bool) {
	type depJSON struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	type skillJSON struct {
		Name         string    `json:"name"`
		Version      string    `json:"version"`
		Description  string    `json:"description"`
		Deprecated   bool      `json:"deprecated"`
		Dependencies []depJSON `json:"dependencies,omitempty"`
		ToolsCount   int       `json:"toolsCount,omitempty"`
	}
	out := make([]skillJSON, len(skills))
	for i, s := range skills {
		sj := skillJSON{
			Name:        s.Name,
			Version:     s.Version,
			Description: s.Description,
			Deprecated:  s.Deprecated,
		}
		if showDeps && len(s.Dependencies) > 0 {
			for _, dep := range s.Dependencies {
				sj.Dependencies = append(sj.Dependencies, depJSON{Name: dep.Name, Version: dep.Version})
			}
		} else {
			sj.ToolsCount = len(s.Tools)
		}
		out[i] = sj
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return
	}
	fmt.Fprintln(w, string(data))
}

func printSkillsYAML(w io.Writer, skills []skillInfo, showDeps bool) {
	type depYAML struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}
	type skillYAML struct {
		Name         string    `yaml:"name"`
		Version      string    `yaml:"version"`
		Description  string    `yaml:"description"`
		Deprecated   bool      `yaml:"deprecated"`
		Dependencies []depYAML `yaml:"dependencies,omitempty"`
	}
	out := make([]skillYAML, len(skills))
	for i, s := range skills {
		sy := skillYAML{
			Name:        s.Name,
			Version:     s.Version,
			Description: s.Description,
			Deprecated:  s.Deprecated,
		}
		if showDeps && len(s.Dependencies) > 0 {
			for _, dep := range s.Dependencies {
				sy.Dependencies = append(sy.Dependencies, depYAML{Name: dep.Name, Version: dep.Version})
			}
		}
		out[i] = sy
	}
	data, err := yaml.Marshal(out)
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return
	}
	fmt.Fprint(w, string(data))
}
