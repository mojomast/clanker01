package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillIndex_New(t *testing.T) {
	idx := NewSkillIndex()
	require.NotNil(t, idx)
	assert.Equal(t, 0, idx.Count())
}

func TestSkillIndex_Add(t *testing.T) {
	idx := NewSkillIndex()

	manifest := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "1.0.0",
			Description: "A test skill",
			Tags:        []string{"test", "sample"},
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/skill",
			Triggers: []Trigger{
				{
					Type:     "intent",
					Patterns: []string{"test.*pattern"},
				},
			},
		},
	}

	idx.Add(manifest)

	assert.Equal(t, 1, idx.Count())
	assert.True(t, idx.Has("test-skill@1.0.0"))
}

func TestSkillIndex_Add_Duplicate(t *testing.T) {
	idx := NewSkillIndex()

	manifest := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "1.0.0",
			Description: "A test skill",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/skill",
		},
	}

	idx.Add(manifest)
	idx.Add(manifest)

	assert.Equal(t, 1, idx.Count())
}

func TestSkillIndex_Remove(t *testing.T) {
	idx := NewSkillIndex()

	manifest := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "1.0.0",
			Description: "A test skill",
			Tags:        []string{"test"},
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/skill",
		},
	}

	idx.Add(manifest)
	assert.Equal(t, 1, idx.Count())

	idx.Remove("test-skill@1.0.0")
	assert.Equal(t, 0, idx.Count())
	assert.False(t, idx.Has("test-skill@1.0.0"))
}

func TestSkillIndex_GetByName(t *testing.T) {
	idx := NewSkillIndex()

	manifest1 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-a",
			Version:     "1.0.0",
			Description: "Skill A",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/a",
		},
	}

	manifest2 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-a",
			Version:     "2.0.0",
			Description: "Skill A v2",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/a",
		},
	}

	manifest3 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-b",
			Version:     "1.0.0",
			Description: "Skill B",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/b",
		},
	}

	idx.Add(manifest1)
	idx.Add(manifest2)
	idx.Add(manifest3)

	ids := idx.GetByName("skill-a")
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, "skill-a@1.0.0")
	assert.Contains(t, ids, "skill-a@2.0.0")
}

func TestSkillIndex_GetByTag(t *testing.T) {
	idx := NewSkillIndex()

	manifest1 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-a",
			Version:     "1.0.0",
			Description: "Skill A",
			Tags:        []string{"test", "development"},
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/a",
		},
	}

	manifest2 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-b",
			Version:     "1.0.0",
			Description: "Skill B",
			Tags:        []string{"test", "production"},
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/b",
		},
	}

	idx.Add(manifest1)
	idx.Add(manifest2)

	ids := idx.GetByTag("test")
	assert.Len(t, ids, 2)

	ids = idx.GetByTag("development")
	assert.Len(t, ids, 1)
	assert.Equal(t, "skill-a@1.0.0", ids[0])
}

func TestSkillIndex_GetByTrigger(t *testing.T) {
	idx := NewSkillIndex()

	manifest := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "1.0.0",
			Description: "A test skill",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/skill",
			Triggers: []Trigger{
				{
					Type:     "intent",
					Patterns: []string{"test.*pattern", "another.*pattern"},
				},
			},
		},
	}

	idx.Add(manifest)

	ids := idx.GetByTrigger("test.*pattern")
	assert.Len(t, ids, 1)
	assert.Equal(t, "test-skill@1.0.0", ids[0])

	ids = idx.GetByTrigger("another.*pattern")
	assert.Len(t, ids, 1)
}

func TestSkillIndex_GetBySource(t *testing.T) {
	idx := NewSkillIndex()

	manifest1 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-a",
			Version:     "1.0.0",
			Description: "Skill A",
		},
		Spec:   SkillSpec{Runtime: "go", Entrypoint: "./bin/a"},
		Source: "local",
	}

	manifest2 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-b",
			Version:     "1.0.0",
			Description: "Skill B",
		},
		Spec:   SkillSpec{Runtime: "go", Entrypoint: "./bin/b"},
		Source: "https://registry.example.com",
	}

	idx.Add(manifest1)
	idx.Add(manifest2)

	ids := idx.GetBySource("local")
	assert.Len(t, ids, 1)
	assert.Equal(t, "skill-a@1.0.0", ids[0])
}

func TestSkillIndex_GetByRuntime(t *testing.T) {
	idx := NewSkillIndex()

	manifest1 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-a",
			Version:     "1.0.0",
			Description: "Skill A",
		},
		Spec: SkillSpec{Runtime: "go", Entrypoint: "./bin/a"},
	}

	manifest2 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-b",
			Version:     "1.0.0",
			Description: "Skill B",
		},
		Spec: SkillSpec{Runtime: "python", Entrypoint: "./bin/b"},
	}

	idx.Add(manifest1)
	idx.Add(manifest2)

	ids := idx.GetByRuntime("go")
	assert.Len(t, ids, 1)
	assert.Equal(t, "skill-a@1.0.0", ids[0])

	ids = idx.GetByRuntime("python")
	assert.Len(t, ids, 1)
	assert.Equal(t, "skill-b@1.0.0", ids[0])
}

func TestSkillIndex_ListAll(t *testing.T) {
	idx := NewSkillIndex()

	manifest1 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-a",
			Version:     "1.0.0",
			Description: "Skill A",
		},
		Spec: SkillSpec{Runtime: "go", Entrypoint: "./bin/a"},
	}

	manifest2 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "skill-b",
			Version:     "1.0.0",
			Description: "Skill B",
		},
		Spec: SkillSpec{Runtime: "python", Entrypoint: "./bin/b"},
	}

	idx.Add(manifest1)
	idx.Add(manifest2)

	ids := idx.ListAll()
	assert.Len(t, ids, 2)
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		query  string
		target string
		want   float64
	}{
		{"test", "test", 1.0},
		{"test", "test-skill", 0.9},
		{"test", "my-test-skill", 0.9},
		{"test skill", "test-skill", 0.8},
		{"code reviewer", "code-reviewer", 0.8},
		{"code", "reviewer", 0.0},
		{"", "test", 0.0},
		{"test", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := fuzzyMatch(tt.query, tt.target)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

func TestRegexMatch(t *testing.T) {
	tests := []struct {
		input   string
		pattern string
		want    bool
	}{
		{"review code", "review.*code", true},
		{"Review my code", "review", true},
		{"check quality", "review", false},
		{"test", "test.*pattern", false},
		{"test pattern", "test.*pattern", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := regexMatch(tt.input, tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompareVersions(t *testing.T) {
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
		{"1.0", "1.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.v1+" vs "+tt.v2, func(t *testing.T) {
			got := compareVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.want, got)
		})
	}
}
