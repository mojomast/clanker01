package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSkillRegistry(t *testing.T) {
	reg := NewSkillRegistry()
	require.NotNil(t, reg)
	assert.Equal(t, 0, reg.Count())
	assert.Empty(t, reg.Sources())
}

func TestSkillRegistry_Register(t *testing.T) {
	reg := NewSkillRegistry()

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

	err := reg.Register(manifest)
	require.NoError(t, err)
	assert.Equal(t, 1, reg.Count())
	assert.True(t, reg.IsRegistered("test-skill@1.0.0"))
}

func TestSkillRegistry_Register_Duplicate(t *testing.T) {
	reg := NewSkillRegistry()

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

	err := reg.Register(manifest)
	require.NoError(t, err)

	err = reg.Register(manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestSkillRegistry_Register_Nil(t *testing.T) {
	reg := NewSkillRegistry()

	err := reg.Register(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot register nil")
}

func TestSkillRegistry_Unregister(t *testing.T) {
	reg := NewSkillRegistry()

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

	require.NoError(t, reg.Register(manifest))
	assert.Equal(t, 1, reg.Count())

	err := reg.Unregister("test-skill@1.0.0")
	require.NoError(t, err)
	assert.Equal(t, 0, reg.Count())
	assert.False(t, reg.IsRegistered("test-skill@1.0.0"))
}

func TestSkillRegistry_Unregister_NotFound(t *testing.T) {
	reg := NewSkillRegistry()

	err := reg.Unregister("test-skill@1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSkillRegistry_Get_ByNameAndVersion(t *testing.T) {
	reg := NewSkillRegistry()

	manifest1 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "1.0.0",
			Description: "Test skill v1",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/v1",
		},
	}

	manifest2 := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "2.0.0",
			Description: "Test skill v2",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/v2",
		},
	}

	require.NoError(t, reg.Register(manifest1))
	require.NoError(t, reg.Register(manifest2))

	got, err := reg.Get("test-skill", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", got.Metadata.Version)
	assert.Equal(t, "Test skill v1", got.Metadata.Description)

	got, err = reg.Get("test-skill", "2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", got.Metadata.Version)
	assert.Equal(t, "Test skill v2", got.Metadata.Description)
}

func TestSkillRegistry_Get_Latest(t *testing.T) {
	reg := NewSkillRegistry()

	manifests := []*SkillManifest{
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "test-skill",
				Version:     "1.0.0",
				Description: "v1",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/v1",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "test-skill",
				Version:     "2.0.0",
				Description: "v2",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/v2",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "test-skill",
				Version:     "1.5.0",
				Description: "v1.5",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/v1.5",
			},
		},
	}

	for _, m := range manifests {
		require.NoError(t, reg.Register(m))
	}

	got, err := reg.Get("test-skill", "")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", got.Metadata.Version)
}

func TestSkillRegistry_Get_NotFound(t *testing.T) {
	reg := NewSkillRegistry()

	_, err := reg.Get("nonexistent", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = reg.Get("nonexistent", "1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSkillRegistry_GetByID(t *testing.T) {
	reg := NewSkillRegistry()

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

	require.NoError(t, reg.Register(manifest))

	got, err := reg.GetByID("test-skill@1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "test-skill", got.Metadata.Name)
	assert.Equal(t, "1.0.0", got.Metadata.Version)

	_, err = reg.GetByID("nonexistent@1.0.0")
	assert.Error(t, err)
}

func TestSkillRegistry_List(t *testing.T) {
	reg := NewSkillRegistry()

	manifests := []*SkillManifest{
		{
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
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "skill-b",
				Version:     "1.0.0",
				Description: "Skill B",
			},
			Spec: SkillSpec{
				Runtime:    "python",
				Entrypoint: "./bin/b",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "skill-c",
				Version:     "1.0.0",
				Description: "Skill C",
			},
			Spec: SkillSpec{
				Runtime:    "nodejs",
				Entrypoint: "./bin/c",
			},
		},
	}

	for _, m := range manifests {
		require.NoError(t, reg.Register(m))
	}

	matches := reg.List(0)
	assert.Len(t, matches, 3)

	matches = reg.List(2)
	assert.Len(t, matches, 2)
}

func TestSkillRegistry_ListByTag(t *testing.T) {
	reg := NewSkillRegistry()

	manifests := []*SkillManifest{
		{
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
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "skill-b",
				Version:     "1.0.0",
				Description: "Skill B",
				Tags:        []string{"test", "production"},
			},
			Spec: SkillSpec{
				Runtime:    "python",
				Entrypoint: "./bin/b",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "skill-c",
				Version:     "1.0.0",
				Description: "Skill C",
				Tags:        []string{"review"},
			},
			Spec: SkillSpec{
				Runtime:    "nodejs",
				Entrypoint: "./bin/c",
			},
		},
	}

	for _, m := range manifests {
		require.NoError(t, reg.Register(m))
	}

	matches := reg.ListByTag("test", 0)
	assert.Len(t, matches, 2)

	matches = reg.ListByTag("review", 0)
	assert.Len(t, matches, 1)
	assert.Equal(t, "skill-c@1.0.0", matches[0].SkillID)

	matches = reg.ListByTag("nonexistent", 0)
	assert.Len(t, matches, 0)
}

func TestSkillRegistry_ListByRuntime(t *testing.T) {
	reg := NewSkillRegistry()

	manifests := []*SkillManifest{
		{
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
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "skill-b",
				Version:     "1.0.0",
				Description: "Skill B",
			},
			Spec: SkillSpec{
				Runtime:    "python",
				Entrypoint: "./bin/b",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "skill-c",
				Version:     "1.0.0",
				Description: "Skill C",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/c",
			},
		},
	}

	for _, m := range manifests {
		require.NoError(t, reg.Register(m))
	}

	matches := reg.ListByRuntime("go", 0)
	assert.Len(t, matches, 2)

	matches = reg.ListByRuntime("python", 0)
	assert.Len(t, matches, 1)

	matches = reg.ListByRuntime("nodejs", 0)
	assert.Len(t, matches, 0)
}

func TestSkillRegistry_Discover(t *testing.T) {
	reg := NewSkillRegistry()

	manifests := []*SkillManifest{
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "code-reviewer",
				Version:     "1.0.0",
				Description: "Automated code review",
				Tags:        []string{"review", "quality"},
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/review",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "test-runner",
				Version:     "1.0.0",
				Description: "Run tests automatically",
				Tags:        []string{"test", "ci"},
			},
			Spec: SkillSpec{
				Runtime:    "python",
				Entrypoint: "./bin/test",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "linter",
				Version:     "1.0.0",
				Description: "Code linter",
				Tags:        []string{"quality", "lint"},
			},
			Spec: SkillSpec{
				Runtime:    "nodejs",
				Entrypoint: "./bin/lint",
			},
		},
	}

	for _, m := range manifests {
		require.NoError(t, reg.Register(m))
	}

	t.Run("search by name exact", func(t *testing.T) {
		matches := reg.Discover("code-reviewer", 10)
		assert.Len(t, matches, 1)
		assert.Equal(t, "code-reviewer@1.0.0", matches[0].SkillID)
		assert.InDelta(t, 0.8, matches[0].Score, 0.01)
	})

	t.Run("search by tag", func(t *testing.T) {
		matches := reg.Discover("quality", 10)
		assert.Len(t, matches, 2)
	})

	t.Run("search by description", func(t *testing.T) {
		matches := reg.Discover("review", 10)
		assert.Len(t, matches, 1)
	})

	t.Run("search limit", func(t *testing.T) {
		matches := reg.Discover("skill", 2)
		assert.LessOrEqual(t, len(matches), 2)
	})

	t.Run("search no results", func(t *testing.T) {
		matches := reg.Discover("nonexistent", 10)
		assert.Len(t, matches, 0)
	})
}

func TestSkillRegistry_Versions(t *testing.T) {
	reg := NewSkillRegistry()

	manifests := []*SkillManifest{
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "test-skill",
				Version:     "1.0.0",
				Description: "v1",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/v1",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "test-skill",
				Version:     "2.0.0",
				Description: "v2",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/v2",
			},
		},
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "test-skill",
				Version:     "1.5.0",
				Description: "v1.5",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/v1.5",
			},
		},
	}

	for _, m := range manifests {
		require.NoError(t, reg.Register(m))
	}

	versions, err := reg.Versions("test-skill")
	require.NoError(t, err)
	assert.Len(t, versions, 3)
	assert.Equal(t, []string{"2.0.0", "1.5.0", "1.0.0"}, versions)

	_, err = reg.Versions("nonexistent")
	assert.Error(t, err)
}

func TestSkillRegistry_AddSource(t *testing.T) {
	reg := NewSkillRegistry()

	source := NewLocalDiscovery([]string{"/tmp/skills"})
	reg.AddSource(source)

	sources := reg.Sources()
	assert.Len(t, sources, 1)
	assert.Equal(t, "local", sources[0])
}

func TestSkillRegistry_AddSource_Duplicate(t *testing.T) {
	reg := NewSkillRegistry()

	source := NewLocalDiscovery([]string{"/tmp/skills"})
	reg.AddSource(source)
	reg.AddSource(source)

	sources := reg.Sources()
	assert.Len(t, sources, 1)
}

func TestSkillRegistry_RemoveSource(t *testing.T) {
	reg := NewSkillRegistry()

	source := NewLocalDiscovery([]string{"/tmp/skills"})
	reg.AddSource(source)

	reg.RemoveSource("local")
	sources := reg.Sources()
	assert.Len(t, sources, 0)
}

func TestSkillRegistry_IsRegistered(t *testing.T) {
	reg := NewSkillRegistry()

	assert.False(t, reg.IsRegistered("test-skill@1.0.0"))

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

	require.NoError(t, reg.Register(manifest))
	assert.True(t, reg.IsRegistered("test-skill@1.0.0"))
}
