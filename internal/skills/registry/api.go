package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type HTTPAPI struct {
	registry *SkillRegistry
}

func NewHTTPAPI(registry *SkillRegistry) *HTTPAPI {
	return &HTTPAPI{registry: registry}
}

func (api *HTTPAPI) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/skills", api.handleListOrPublish)
	mux.HandleFunc("/v1/skills/", api.handleSkillPath)
	mux.HandleFunc("/v1/skills/search", api.handleSearch)
	mux.HandleFunc("/v1/skills/discover", api.handleDiscover)
	mux.HandleFunc("/health", api.handleHealth)

	return mux
}

func (api *HTTPAPI) handleListOrPublish(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		api.handleList(w, req)
	case "POST":
		api.handlePublish(w, req)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *HTTPAPI) handleSkillPath(w http.ResponseWriter, req *http.Request) {
	pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")

	if len(pathParts) < 3 {
		http.NotFound(w, req)
		return
	}

	name := pathParts[2]

	switch {
	case len(pathParts) == 3:
		if req.Method == "GET" {
			version := req.URL.Query().Get("version")
			api.handleGetWithNameVersion(w, req, name, version)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case len(pathParts) == 4 && pathParts[3] == "versions":
		if req.Method == "GET" {
			api.handleVersions(w, req, name)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case len(pathParts) == 5 && pathParts[4] == "download":
		if req.Method == "GET" {
			version := pathParts[3]
			api.handleDownload(w, req, name, version)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, req)
	}
}

func (api *HTTPAPI) handleList(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	tags := query["tag"]
	search := query.Get("search")
	runtime := query.Get("runtime")
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit == 0 {
		limit = 20
	}

	var matches []*SkillMatch
	if search != "" {
		matches = api.registry.Discover(search, limit)
	} else if len(tags) > 0 {
		matches = api.discoverByTags(tags, limit)
	} else if runtime != "" {
		matches = api.registry.ListByRuntime(runtime, limit)
	} else {
		matches = api.registry.List(limit)
	}

	skillDetails := make([]map[string]any, 0, len(matches))
	for _, match := range matches {
		manifest, err := api.registry.GetByID(match.SkillID)
		if err != nil {
			continue
		}
		skillDetails = append(skillDetails, api.marshalManifest(manifest, match.Score))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"skills": skillDetails,
		"count":  len(skillDetails),
	})
}

func (api *HTTPAPI) handleGetWithNameVersion(w http.ResponseWriter, req *http.Request, name string, version string) {
	manifest, err := api.registry.Get(name, version)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, api.marshalManifest(manifest, 1.0))
}

func (api *HTTPAPI) handlePublish(w http.ResponseWriter, req *http.Request) {
	var manifest SkillManifest
	if err := json.NewDecoder(req.Body).Decode(&manifest); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if err := api.registry.Register(&manifest); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "skill registered successfully",
		"id":      manifest.ID(),
	})
}

func (api *HTTPAPI) handleVersions(w http.ResponseWriter, req *http.Request, name string) {
	versions, err := api.registry.Versions(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":     name,
		"versions": versions,
	})
}

func (api *HTTPAPI) handleDownload(w http.ResponseWriter, req *http.Request, name string, version string) {
	manifest, err := api.registry.Get(name, version)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("%s@%s.json", name, version)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (api *HTTPAPI) handleSearch(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if body.Limit == 0 {
		body.Limit = 10
	}

	matches := api.registry.Discover(body.Query, body.Limit)

	results := make([]map[string]any, 0, len(matches))
	for _, match := range matches {
		manifest, err := api.registry.GetByID(match.SkillID)
		if err != nil {
			continue
		}
		result := api.marshalManifest(manifest, match.Score)
		result["score"] = match.Score
		results = append(results, result)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   body.Query,
		"results": results,
		"count":   len(results),
	})
}

func (api *HTTPAPI) handleDiscover(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	manifests, err := api.registry.DiscoverAll(req.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	registeredCount := 0
	for _, manifest := range manifests {
		if err := api.registry.Register(manifest); err != nil {
			continue
		}
		registeredCount++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":          "discovery completed",
		"found":            len(manifests),
		"registered":       registeredCount,
		"total_registered": api.registry.Count(),
	})
}

func (api *HTTPAPI) handleHealth(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "healthy",
		"skills":  api.registry.Count(),
		"sources": api.registry.Sources(),
	})
}

func (api *HTTPAPI) discoverByTags(tags []string, limit int) []*SkillMatch {
	allIDs := make([]string, 0)
	seen := make(map[string]bool)

	for _, tag := range tags {
		matches := api.registry.ListByTag(tag, 0)
		for _, match := range matches {
			if !seen[match.SkillID] {
				seen[match.SkillID] = true
				allIDs = append(allIDs, match.SkillID)
			}
		}
	}

	if limit > 0 && len(allIDs) > limit {
		allIDs = allIDs[:limit]
	}

	matches := make([]*SkillMatch, len(allIDs))
	for i, id := range allIDs {
		matches[i] = &SkillMatch{SkillID: id, Score: 1.0}
	}

	return matches
}

func (api *HTTPAPI) marshalManifest(manifest *SkillManifest, score float64) map[string]any {
	return map[string]any{
		"id":          manifest.ID(),
		"name":        manifest.Metadata.Name,
		"version":     manifest.Metadata.Version,
		"displayName": manifest.Metadata.DisplayName,
		"description": manifest.Metadata.Description,
		"author":      manifest.Metadata.Author,
		"license":     manifest.Metadata.License,
		"tags":        manifest.Metadata.Tags,
		"runtime":     manifest.Spec.Runtime,
		"deprecated":  manifest.Metadata.Deprecated,
		"homepage":    manifest.Metadata.Homepage,
		"repository":  manifest.Metadata.Repository,
		"source":      manifest.Source,
		"score":       score,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": message,
	})
}

func WriteManifestResponse(w http.ResponseWriter, manifest *SkillManifest, score float64) {
	api := &HTTPAPI{}
	writeJSON(w, http.StatusOK, api.marshalManifest(manifest, score))
}

func ParseSkillID(id string) (name, version string, err error) {
	parts := strings.Split(id, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid skill ID format: %s", id)
	}
	return parts[0], parts[1], nil
}
