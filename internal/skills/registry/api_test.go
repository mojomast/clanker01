package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPAPI(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)

	require.NotNil(t, api)
	assert.NotNil(t, api.registry)
}

func TestHTTPAPI_HandleHealth(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, float64(0), response["skills"])
}

func TestHTTPAPI_HandleList_Empty(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	req := httptest.NewRequest("GET", "/v1/skills", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["count"])
	assert.Empty(t, response["skills"])
}

func TestHTTPAPI_HandleList_WithSkills(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

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

	require.NoError(t, reg.Register(manifest))

	req := httptest.NewRequest("GET", "/v1/skills", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, float64(1), response["count"])
	skills, ok := response["skills"].([]any)
	require.True(t, ok)
	assert.Len(t, skills, 1)
}

func TestHTTPAPI_HandleList_WithTag(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	manifest := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "1.0.0",
			Description: "A test skill",
			Tags:        []string{"test", "development"},
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "./bin/skill",
		},
	}

	require.NoError(t, reg.Register(manifest))

	req := httptest.NewRequest("GET", "/v1/skills?tag=test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, float64(1), response["count"])
}

func TestHTTPAPI_HandleList_WithSearch(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	manifest := &SkillManifest{
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
	}

	require.NoError(t, reg.Register(manifest))

	req := httptest.NewRequest("GET", "/v1/skills?search=review", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, float64(1), response["count"])
}

func TestHTTPAPI_HandleList_WithRuntime(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

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

	req := httptest.NewRequest("GET", "/v1/skills?runtime=go", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, float64(1), response["count"])
}

func TestHTTPAPI_HandleList_WithLimit(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	for i := 0; i < 5; i++ {
		manifest := &SkillManifest{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:        "skill-" + string(rune('a'+i)),
				Version:     "1.0.0",
				Description: "Skill",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/skill",
			},
		}
		require.NoError(t, reg.Register(manifest))
	}

	req := httptest.NewRequest("GET", "/v1/skills?limit=3", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, float64(3), response["count"])
}

func TestHTTPAPI_HandleGet_Success(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

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

	require.NoError(t, reg.Register(manifest))

	req := httptest.NewRequest("GET", "/v1/skills/test-skill", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "test-skill", response["name"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Equal(t, "A test skill", response["description"])
}

func TestHTTPAPI_HandleGet_WithVersion(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	manifest1 := &SkillManifest{
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
	}

	manifest2 := &SkillManifest{
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
	}

	require.NoError(t, reg.Register(manifest1))
	require.NoError(t, reg.Register(manifest2))

	req := httptest.NewRequest("GET", "/v1/skills/test-skill?version=1.0.0", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", response["version"])
	assert.Equal(t, "v1", response["description"])
}

func TestHTTPAPI_HandleGet_NotFound(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	req := httptest.NewRequest("GET", "/v1/skills/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["error"])
}

func TestHTTPAPI_HandlePublish_Success(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	manifest := map[string]any{
		"apiVersion": "swarm.ai/v1",
		"kind":       "Skill",
		"metadata": map[string]any{
			"name":        "test-skill",
			"version":     "1.0.0",
			"description": "A test skill",
		},
		"spec": map[string]any{
			"runtime":    "go",
			"entrypoint": "./bin/skill",
		},
	}

	body, _ := json.Marshal(manifest)
	req := httptest.NewRequest("POST", "/v1/skills", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "skill registered successfully", response["message"])
	assert.Equal(t, "test-skill@1.0.0", response["id"])
}

func TestHTTPAPI_HandlePublish_Duplicate(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

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

	body, _ := json.Marshal(manifest)
	req := httptest.NewRequest("POST", "/v1/skills", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHTTPAPI_HandleVersions(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	manifests := []*SkillManifest{
		{
			APIVersion: "swarm.ai/v1",
			Kind:       "Skill",
			Metadata: SkillMetadata{
				Name:    "test-skill",
				Version: "1.0.0",
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
				Name:    "test-skill",
				Version: "2.0.0",
			},
			Spec: SkillSpec{
				Runtime:    "go",
				Entrypoint: "./bin/v2",
			},
		},
	}

	for _, m := range manifests {
		require.NoError(t, reg.Register(m))
	}

	req := httptest.NewRequest("GET", "/v1/skills/test-skill/versions", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "test-skill", response["name"])
	versions, ok := response["versions"].([]any)
	require.True(t, ok)
	assert.Len(t, versions, 2)
}

func TestHTTPAPI_HandleDownload(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

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

	req := httptest.NewRequest("GET", "/v1/skills/test-skill/1.0.0/download", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "test-skill@1.0.0.json")

	var downloadedManifest SkillManifest
	err := json.NewDecoder(w.Body).Decode(&downloadedManifest)
	require.NoError(t, err)

	assert.Equal(t, "test-skill", downloadedManifest.Metadata.Name)
	assert.Equal(t, "1.0.0", downloadedManifest.Metadata.Version)
}

func TestHTTPAPI_HandleSearch(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	manifest := &SkillManifest{
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
	}

	require.NoError(t, reg.Register(manifest))

	body, _ := json.Marshal(map[string]any{
		"query": "review",
		"limit": 10,
	})

	req := httptest.NewRequest("POST", "/v1/skills/search", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "review", response["query"])
	assert.Equal(t, float64(1), response["count"])
}

func TestHTTPAPI_HandleDiscover(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	body, _ := json.Marshal(map[string]any{
		"sources": []string{},
	})

	req := httptest.NewRequest("POST", "/v1/skills/discover", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "discovery completed", response["message"])
}

func TestHTTPAPI_Handler_NotFound(t *testing.T) {
	reg := NewSkillRegistry()
	api := NewHTTPAPI(reg)
	handler := api.Handler()

	req := httptest.NewRequest("GET", "/v1/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestParseSkillID_Valid(t *testing.T) {
	name, version, err := ParseSkillID("test-skill@1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "test-skill", name)
	assert.Equal(t, "1.0.0", version)
}

func TestParseSkillID_Invalid(t *testing.T) {
	_, _, err := ParseSkillID("invalid-format")
	assert.Error(t, err)

	_, _, err = ParseSkillID("test-skill")
	assert.Error(t, err)
}

func TestWriteManifestResponse(t *testing.T) {
	w := httptest.NewRecorder()

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

	WriteManifestResponse(w, manifest, 1.0)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]any
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "test-skill", response["name"])
	assert.Equal(t, "1.0.0", response["version"])
}
