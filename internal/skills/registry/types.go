package registry

import (
	"time"
)

type SkillMetadata struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	DisplayName string   `yaml:"displayName,omitempty" json:"displayName,omitempty"`
	Description string   `yaml:"description" json:"description"`
	Author      string   `yaml:"author,omitempty" json:"author,omitempty"`
	License     string   `yaml:"license,omitempty" json:"license,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Icon        string   `yaml:"icon,omitempty" json:"icon,omitempty"`
	Homepage    string   `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	Repository  string   `yaml:"repository,omitempty" json:"repository,omitempty"`
	Deprecated  bool     `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
}

type Trigger struct {
	Type       string   `yaml:"type" json:"type"`
	Patterns   []string `yaml:"patterns,omitempty" json:"patterns,omitempty"`
	Confidence float64  `yaml:"confidence,omitempty" json:"confidence,omitempty"`
	Events     []string `yaml:"events,omitempty" json:"events,omitempty"`
}

type ToolDef struct {
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description" json:"description"`
	Parameters  map[string]any `yaml:"parameters" json:"parameters"`
	Returns     map[string]any `yaml:"returns,omitempty" json:"returns,omitempty"`
}

type Prompts struct {
	System           string            `yaml:"system,omitempty" json:"system,omitempty"`
	Examples         []map[string]any  `yaml:"examples,omitempty" json:"examples,omitempty"`
	ContextInjection map[string]string `yaml:"context_injection,omitempty" json:"context_injection,omitempty"`
}

type Dependency struct {
	Name     string `yaml:"name" json:"name"`
	Version  string `yaml:"version,omitempty" json:"version,omitempty"`
	Source   string `yaml:"source,omitempty" json:"source,omitempty"`
	Optional bool   `yaml:"optional,omitempty" json:"optional,omitempty"`
}

type Resources struct {
	CPU         string        `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory      string        `yaml:"memory,omitempty" json:"memory,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	TempStorage string        `yaml:"temp_storage,omitempty" json:"temp_storage,omitempty"`
}

type FilesystemPermissions struct {
	Read   []string `yaml:"read,omitempty" json:"read,omitempty"`
	Write  []string `yaml:"write,omitempty" json:"write,omitempty"`
	Delete []string `yaml:"delete,omitempty" json:"delete,omitempty"`
}

type NetworkPermissions struct {
	Allow        bool     `yaml:"allow" json:"allow"`
	AllowedHosts []string `yaml:"allowed_hosts,omitempty" json:"allowed_hosts,omitempty"`
}

type EnvironmentPermissions struct {
	Allow []string `yaml:"allow,omitempty" json:"allow,omitempty"`
}

type Permissions struct {
	Filesystem  *FilesystemPermissions  `yaml:"filesystem,omitempty" json:"filesystem,omitempty"`
	Network     *NetworkPermissions     `yaml:"network,omitempty" json:"network,omitempty"`
	Environment *EnvironmentPermissions `yaml:"environment,omitempty" json:"environment,omitempty"`
}

type Enhancement struct {
	Skill      string `yaml:"skill" json:"skill"`
	MinVersion string `yaml:"min_version,omitempty" json:"min_version,omitempty"`
}

type Composition struct {
	CompatibleWith []string      `yaml:"compatible_with,omitempty" json:"compatible_with,omitempty"`
	ConflictsWith  []string      `yaml:"conflicts_with,omitempty" json:"conflicts_with,omitempty"`
	Enhances       []Enhancement `yaml:"enhances,omitempty" json:"enhances,omitempty"`
}

type SkillConfig struct {
	Schema map[string]any `yaml:"schema,omitempty" json:"schema,omitempty"`
	UI     map[string]any `yaml:"ui,omitempty" json:"ui,omitempty"`
}

type SkillSpec struct {
	Runtime      string       `yaml:"runtime" json:"runtime"`
	Entrypoint   string       `yaml:"entrypoint" json:"entrypoint"`
	Triggers     []Trigger    `yaml:"triggers,omitempty" json:"triggers,omitempty"`
	Tools        []ToolDef    `yaml:"tools,omitempty" json:"tools,omitempty"`
	Prompts      *Prompts     `yaml:"prompts,omitempty" json:"prompts,omitempty"`
	Dependencies []Dependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Resources    *Resources   `yaml:"resources,omitempty" json:"resources,omitempty"`
	Permissions  *Permissions `yaml:"permissions,omitempty" json:"permissions,omitempty"`
	Composition  *Composition `yaml:"composition,omitempty" json:"composition,omitempty"`
	Config       *SkillConfig `yaml:"config,omitempty" json:"config,omitempty"`
}

type SkillManifest struct {
	APIVersion string        `yaml:"apiVersion" json:"apiVersion"`
	Kind       string        `yaml:"kind" json:"kind"`
	Metadata   SkillMetadata `yaml:"metadata" json:"metadata"`
	Spec       SkillSpec     `yaml:"spec" json:"spec"`
	Source     string        `json:"source,omitempty"`
	FilePath   string        `json:"filePath,omitempty"`
}

func (m *SkillManifest) ID() string {
	return m.Metadata.Name + "@" + m.Metadata.Version
}

type SkillMatch struct {
	SkillID string         `json:"skillId"`
	Score   float64        `json:"score"`
	Context map[string]any `json:"context,omitempty"`
}

type Unsubscribe func()
