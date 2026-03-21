package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalDiscovery_Name(t *testing.T) {
	d := NewLocalDiscovery([]string{"/tmp/skills"})
	assert.Equal(t, "local", d.Name())
}

func TestLocalDiscovery_Discover(t *testing.T) {
	tempDir := t.TempDir()

	skillDir := filepath.Join(tempDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	manifestContent := `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: test-skill
  version: 1.0.0
  description: A test skill
  tags:
    - test

spec:
  runtime: go
  entrypoint: ./bin/skill
`

	manifestPath := filepath.Join(skillDir, "skill.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	d := NewLocalDiscovery([]string{tempDir})

	manifests, err := d.Discover(context.Background())
	require.NoError(t, err)
	assert.Len(t, manifests, 1)
	assert.Equal(t, "test-skill", manifests[0].Metadata.Name)
	assert.Equal(t, "1.0.0", manifests[0].Metadata.Version)
	assert.Equal(t, "local", manifests[0].Source)
}

func TestLocalDiscovery_Discover_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	d := NewLocalDiscovery([]string{tempDir})

	manifests, err := d.Discover(context.Background())
	require.NoError(t, err)
	assert.Len(t, manifests, 0)
}

func TestLocalDiscovery_Discover_MultipleSkills(t *testing.T) {
	tempDir := t.TempDir()

	skillContent := `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: %s
  version: 1.0.0
  description: A test skill

spec:
  runtime: go
  entrypoint: ./bin/skill
`

	for i := 0; i < 3; i++ {
		skillDir := filepath.Join(tempDir, "skill-"+string(rune('a'+i)))
		require.NoError(t, os.MkdirAll(skillDir, 0755))
		manifestPath := filepath.Join(skillDir, "skill.yaml")
		name := "skill-" + string(rune('a'+i))
		require.NoError(t, os.WriteFile(manifestPath, []byte(string(fmt.Sprintf(skillContent, name))), 0644))
	}

	d := NewLocalDiscovery([]string{tempDir})

	manifests, err := d.Discover(context.Background())
	require.NoError(t, err)
	assert.Len(t, manifests, 3)
}

func TestLoadManifest_Valid(t *testing.T) {
	tempDir := t.TempDir()

	manifestContent := `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: test-skill
  version: 1.0.0
  description: A test skill
  tags:
    - test

spec:
  runtime: go
  entrypoint: ./bin/skill
  triggers:
    - type: intent
      patterns:
        - test.*pattern
`

	manifestPath := filepath.Join(tempDir, "skill.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	manifest, err := loadManifest(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, "swarm.ai/v1", manifest.APIVersion)
	assert.Equal(t, "Skill", manifest.Kind)
	assert.Equal(t, "test-skill", manifest.Metadata.Name)
	assert.Equal(t, "1.0.0", manifest.Metadata.Version)
	assert.Equal(t, "go", manifest.Spec.Runtime)
	assert.Equal(t, "./bin/skill", manifest.Spec.Entrypoint)
	assert.Len(t, manifest.Metadata.Tags, 1)
	assert.Equal(t, "test", manifest.Metadata.Tags[0])
}

func TestLoadManifest_InvalidAPIVersion(t *testing.T) {
	tempDir := t.TempDir()

	manifestContent := `
apiVersion: invalid/v1
kind: Skill

metadata:
  name: test-skill
  version: 1.0.0
  description: A test skill

spec:
  runtime: go
  entrypoint: ./bin/skill
`

	manifestPath := filepath.Join(tempDir, "skill.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	_, err := loadManifest(manifestPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest")
}

func TestLoadManifest_MissingRequiredField(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "missing name",
			content: `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  version: 1.0.0
  description: A test skill

spec:
  runtime: go
  entrypoint: ./bin/skill
`,
			expected: "metadata.name",
		},
		{
			name: "missing version",
			content: `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: test-skill
  description: A test skill

spec:
  runtime: go
  entrypoint: ./bin/skill
`,
			expected: "metadata.version",
		},
		{
			name: "missing description",
			content: `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: test-skill
  version: 1.0.0

spec:
  runtime: go
  entrypoint: ./bin/skill
`,
			expected: "metadata.description",
		},
		{
			name: "missing runtime",
			content: `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: test-skill
  version: 1.0.0
  description: A test skill

spec:
  entrypoint: ./bin/skill
`,
			expected: "spec.runtime",
		},
		{
			name: "missing entrypoint",
			content: `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: test-skill
  version: 1.0.0
  description: A test skill

spec:
  runtime: go
`,
			expected: "spec.entrypoint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manifestPath := filepath.Join(tempDir, "skill-"+tc.name+".yaml")
			require.NoError(t, os.WriteFile(manifestPath, []byte(tc.content), 0644))

			_, err := loadManifest(manifestPath)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expected)
		})
	}
}

func TestRegistryDiscovery_Name(t *testing.T) {
	d := NewRegistryDiscovery("https://registry.example.com")
	assert.Equal(t, "registry", d.Name())
}

func TestGitDiscovery_Name(t *testing.T) {
	d := NewGitDiscovery([]GitRepoSource{{URL: "https://github.com/test/repo"}}, "/tmp/cache")
	assert.Equal(t, "git", d.Name())
}

func TestCompositeDiscovery_Name(t *testing.T) {
	local := NewLocalDiscovery([]string{"/tmp/skills"})
	composite := NewCompositeDiscovery([]DiscoverySource{local})
	assert.Equal(t, "composite", composite.Name())
}

func TestCompositeDiscovery_Discover(t *testing.T) {
	tempDir := t.TempDir()

	skillDir := filepath.Join(tempDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	manifestContent := `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: test-skill
  version: 1.0.0
  description: A test skill

spec:
  runtime: go
  entrypoint: ./bin/skill
`

	manifestPath := filepath.Join(skillDir, "skill.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	local1 := NewLocalDiscovery([]string{tempDir})
	local2 := NewLocalDiscovery([]string{"/nonexistent"})

	composite := NewCompositeDiscovery([]DiscoverySource{local1, local2})

	manifests, err := composite.Discover(context.Background())
	require.NoError(t, err)
	assert.Len(t, manifests, 1)
}

func TestCompareVersions_Semver(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"1.1.0", "1.2.0", -1},
		{"1.2.0", "1.1.0", 1},
		{"1.0", "1.0.0", 0},
		{"1", "1.0.0", 0},
		{"1.0.0", "1.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.v1+" vs "+tt.v2, func(t *testing.T) {
			got := compareVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompareVersions_Invalid(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		{"invalid", "1.0.0", -1},
		{"1.0.0", "invalid", 1},
		{"invalid", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.v1+" vs "+tt.v2, func(t *testing.T) {
			got := compareVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadManifest_Full(t *testing.T) {
	tempDir := t.TempDir()

	manifestContent := `
apiVersion: swarm.ai/v1
kind: Skill

metadata:
  name: code-reviewer
  version: 1.3.0
  displayName: Code Reviewer
  description: Automated code review with style checks
  author: SWARM Team
  license: Apache-2.0
  tags:
    - review
    - quality
  icon: icon.png
  homepage: https://example.com
  repository: https://github.com/test/repo
  deprecated: false

spec:
  runtime: go
  entrypoint: ./bin/code-reviewer
  
  triggers:
    - type: intent
      patterns:
        - "review.*code"
        - "check.*quality"
      confidence: 0.8
      
    - type: event
      events:
        - pull_request.opened
        - pull_request.synchronize
  
  tools:
    - name: review_file
      description: Review a single file
      parameters:
        type: object
        required: [file_path]
        properties:
          file_path:
            type: string
      returns:
        type: object
        properties:
          issues:
            type: array
  
  prompts:
    system: |
      You are an expert code reviewer.
    
    examples:
      - input: "Review the auth module"
        output: "Reviewing auth module..."
    
    context_injection:
      before: "Project root: {{workspace.root}}"
      after: "Review complete."

  dependencies:
    - name: github.com/swarm-ai/go-lint
      version: "^1.2.0"
      source: go

  resources:
    cpu: "500m"
    memory: "256Mi"
    timeout: 60s
    temp_storage: 100Mi

  permissions:
    filesystem:
      read: ["**/*.go"]
      write: []
      delete: []
    network:
      allow: false
      allowed_hosts: []
    environment:
      allow: ["GOLANGCI_*"]

  composition:
    compatible_with:
      - git-ops
      - test-runner
    conflicts_with: []
    enhances:
      - skill: ci-pipeline
        min_version: "2.0.0"
  
  config:
    schema:
      type: object
      properties:
        rules:
          type: array
    ui:
      rules:
        widget: multiselect
`

	manifestPath := filepath.Join(tempDir, "skill.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	manifest, err := loadManifest(manifestPath)
	require.NoError(t, err)

	assert.Equal(t, "code-reviewer", manifest.Metadata.Name)
	assert.Equal(t, "1.3.0", manifest.Metadata.Version)
	assert.Equal(t, "Code Reviewer", manifest.Metadata.DisplayName)
	assert.Equal(t, "SWARM Team", manifest.Metadata.Author)
	assert.Equal(t, "Apache-2.0", manifest.Metadata.License)
	assert.Len(t, manifest.Metadata.Tags, 2)
	assert.Equal(t, "review", manifest.Metadata.Tags[0])
	assert.Equal(t, "quality", manifest.Metadata.Tags[1])
	assert.False(t, manifest.Metadata.Deprecated)

	assert.Equal(t, "go", manifest.Spec.Runtime)
	assert.Equal(t, "./bin/code-reviewer", manifest.Spec.Entrypoint)
	assert.Len(t, manifest.Spec.Triggers, 2)
	assert.Len(t, manifest.Spec.Tools, 1)
	assert.NotNil(t, manifest.Spec.Prompts)
	assert.Len(t, manifest.Spec.Dependencies, 1)
	assert.NotNil(t, manifest.Spec.Resources)
	assert.NotNil(t, manifest.Spec.Permissions)
	assert.NotNil(t, manifest.Spec.Composition)
	assert.NotNil(t, manifest.Spec.Config)
}
