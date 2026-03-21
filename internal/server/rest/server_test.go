package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/swarm-ai/swarm/internal/config"
	"github.com/swarm-ai/swarm/internal/security/auth"
)

func TestServer_Health(t *testing.T) {
	jwtAuth, _ := auth.NewJWTAuthenticator(auth.DefaultJWTConfig())
	appConfig := &config.Config{
		Version: "1.0.0",
		Project: config.ProjectConfig{
			Name: "test",
			Root: "/test",
		},
	}
	serverConfig := DefaultConfig()
	serverConfig.EnableAuth = false

	server, err := NewServer(serverConfig, appConfig, jwtAuth)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var health map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if health["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %v", health["status"])
	}
}

func TestServer_Agents(t *testing.T) {
	jwtAuth, _ := auth.NewJWTAuthenticator(auth.DefaultJWTConfig())
	appConfig := &config.Config{
		Version: "1.0.0",
		Project: config.ProjectConfig{
			Name: "test",
			Root: "/test",
		},
	}
	serverConfig := DefaultConfig()
	serverConfig.EnableAuth = false

	server, err := NewServer(serverConfig, appConfig, jwtAuth)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	t.Run("ListAgents - empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["count"].(float64) != 0 {
			t.Errorf("Expected count 0, got %v", response["count"])
		}
	})

	t.Run("CreateAgent", func(t *testing.T) {
		agentReq := AgentRequest{
			Name:   "test-agent",
			Type:   "coder",
			Model:  "gpt-4",
			Skills: []string{"git", "filesystem"},
		}
		body, _ := json.Marshal(agentReq)
		req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var response AgentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Name != "test-agent" {
			t.Errorf("Expected name test-agent, got %v", response.Name)
		}

		if response.Type != "coder" {
			t.Errorf("Expected type coder, got %v", response.Type)
		}
	})

	t.Run("GetAgent", func(t *testing.T) {
		agentReq := AgentRequest{
			Name: "get-test-agent",
			Type: "architect",
		}
		body, _ := json.Marshal(agentReq)
		createReq := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp AgentResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("GET", "/api/v1/agents/"+createResp.ID, nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response AgentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.ID != createResp.ID {
			t.Errorf("Expected ID %s, got %s", createResp.ID, response.ID)
		}
	})

	t.Run("UpdateAgent", func(t *testing.T) {
		agentReq := AgentRequest{
			Name: "update-test-agent",
			Type: "reviewer",
		}
		body, _ := json.Marshal(agentReq)
		createReq := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp AgentResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		updateReq := AgentRequest{
			Name: "updated-agent-name",
		}
		updateBody, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/v1/agents/"+createResp.ID, bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response AgentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Name != "updated-agent-name" {
			t.Errorf("Expected name updated-agent-name, got %v", response.Name)
		}
	})

	t.Run("StartAgent", func(t *testing.T) {
		agentReq := AgentRequest{
			Name: "start-test-agent",
			Type: "tester",
		}
		body, _ := json.Marshal(agentReq)
		createReq := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp AgentResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("POST", "/api/v1/agents/"+createResp.ID+"/start", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response AgentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Status != "running" {
			t.Errorf("Expected status running, got %v", response.Status)
		}
	})

	t.Run("DeleteAgent", func(t *testing.T) {
		agentReq := AgentRequest{
			Name: "delete-test-agent",
			Type: "researcher",
		}
		body, _ := json.Marshal(agentReq)
		createReq := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp AgentResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("DELETE", "/api/v1/agents/"+createResp.ID, nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		getReq := httptest.NewRequest("GET", "/api/v1/agents/"+createResp.ID, nil)
		getW := httptest.NewRecorder()
		server.router.ServeHTTP(getW, getReq)

		if getW.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 after delete, got %d", getW.Code)
		}
	})
}

func TestServer_Tasks(t *testing.T) {
	jwtAuth, _ := auth.NewJWTAuthenticator(auth.DefaultJWTConfig())
	appConfig := &config.Config{
		Version: "1.0.0",
		Project: config.ProjectConfig{
			Name: "test",
			Root: "/test",
		},
	}
	serverConfig := DefaultConfig()
	serverConfig.EnableAuth = false

	server, err := NewServer(serverConfig, appConfig, jwtAuth)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	t.Run("ListTasks - empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["count"].(float64) != 0 {
			t.Errorf("Expected count 0, got %v", response["count"])
		}
	})

	t.Run("CreateTask", func(t *testing.T) {
		taskReq := TaskRequest{
			Type:       "code_review",
			Prompt:     "Review the following code",
			AgentType:  "reviewer",
			Priority:   1,
			MaxRetries: 3,
		}
		body, _ := json.Marshal(taskReq)
		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var response TaskResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Prompt != "Review the following code" {
			t.Errorf("Expected prompt 'Review the following code', got %v", response.Prompt)
		}

		if response.Status != "pending" {
			t.Errorf("Expected status pending, got %v", response.Status)
		}
	})

	t.Run("GetTask", func(t *testing.T) {
		taskReq := TaskRequest{
			Type:      "test_execution",
			Prompt:    "Run unit tests",
			AgentType: "tester",
		}
		body, _ := json.Marshal(taskReq)
		createReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp TaskResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("GET", "/api/v1/tasks/"+createResp.ID, nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response TaskResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.ID != createResp.ID {
			t.Errorf("Expected ID %s, got %s", createResp.ID, response.ID)
		}
	})

	t.Run("StartTask", func(t *testing.T) {
		taskReq := TaskRequest{
			Type:      "code_generation",
			Prompt:    "Generate a REST API handler",
			AgentType: "coder",
		}
		body, _ := json.Marshal(taskReq)
		createReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp TaskResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("POST", "/api/v1/tasks/"+createResp.ID+"/start", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response TaskResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Status != "running" {
			t.Errorf("Expected status running, got %v", response.Status)
		}

		if response.StartedAt == nil {
			t.Error("Expected started_at to be set")
		}
	})

	t.Run("CancelTask", func(t *testing.T) {
		taskReq := TaskRequest{
			Type:      "analysis",
			Prompt:    "Analyze the architecture",
			AgentType: "architect",
		}
		body, _ := json.Marshal(taskReq)
		createReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp TaskResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("POST", "/api/v1/tasks/"+createResp.ID+"/cancel", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response TaskResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Status != "cancelled" {
			t.Errorf("Expected status cancelled, got %v", response.Status)
		}
	})

	t.Run("DeleteTask", func(t *testing.T) {
		taskReq := TaskRequest{
			Type:      "cleanup",
			Prompt:    "Clean up temporary files",
			AgentType: "coder",
		}
		body, _ := json.Marshal(taskReq)
		createReq := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp TaskResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("DELETE", "/api/v1/tasks/"+createResp.ID, nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		getReq := httptest.NewRequest("GET", "/api/v1/tasks/"+createResp.ID, nil)
		getW := httptest.NewRecorder()
		server.router.ServeHTTP(getW, getReq)

		if getW.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 after delete, got %d", getW.Code)
		}
	})
}

func TestServer_Skills(t *testing.T) {
	jwtAuth, _ := auth.NewJWTAuthenticator(auth.DefaultJWTConfig())
	appConfig := &config.Config{
		Version: "1.0.0",
		Project: config.ProjectConfig{
			Name: "test",
			Root: "/test",
		},
	}
	serverConfig := DefaultConfig()
	serverConfig.EnableAuth = false

	server, err := NewServer(serverConfig, appConfig, jwtAuth)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	t.Run("ListSkills - empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/skills", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["count"].(float64) != 0 {
			t.Errorf("Expected count 0, got %v", response["count"])
		}
	})

	t.Run("LoadSkill", func(t *testing.T) {
		skillReq := SkillRequest{
			Name:    "test-skill",
			Version: "1.0.0",
			Config: map[string]interface{}{
				"option1": "value1",
			},
		}
		body, _ := json.Marshal(skillReq)
		req := httptest.NewRequest("POST", "/api/v1/skills", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var response SkillResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Name != "test-skill" {
			t.Errorf("Expected name test-skill, got %v", response.Name)
		}

		if response.Status != "loaded" {
			t.Errorf("Expected status loaded, got %v", response.Status)
		}
	})

	t.Run("GetSkill", func(t *testing.T) {
		skillReq := SkillRequest{
			Name:    "get-test-skill",
			Version: "2.0.0",
		}
		body, _ := json.Marshal(skillReq)
		createReq := httptest.NewRequest("POST", "/api/v1/skills", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp SkillResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("GET", "/api/v1/skills/"+createResp.ID, nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response SkillResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.ID != createResp.ID {
			t.Errorf("Expected ID %s, got %s", createResp.ID, response.ID)
		}
	})

	t.Run("UnloadSkill", func(t *testing.T) {
		skillReq := SkillRequest{
			Name:    "unload-test-skill",
			Version: "1.0.0",
		}
		body, _ := json.Marshal(skillReq)
		createReq := httptest.NewRequest("POST", "/api/v1/skills", bytes.NewReader(body))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		server.router.ServeHTTP(createW, createReq)

		var createResp SkillResponse
		json.Unmarshal(createW.Body.Bytes(), &createResp)

		req := httptest.NewRequest("DELETE", "/api/v1/skills/"+createResp.ID, nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		getReq := httptest.NewRequest("GET", "/api/v1/skills/"+createResp.ID, nil)
		getW := httptest.NewRecorder()
		server.router.ServeHTTP(getW, getReq)

		if getW.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 after delete, got %d", getW.Code)
		}
	})
}

func TestServer_Config(t *testing.T) {
	jwtAuth, _ := auth.NewJWTAuthenticator(auth.DefaultJWTConfig())
	appConfig := &config.Config{
		Version: "1.0.0",
		Project: config.ProjectConfig{
			Name: "test-project",
			Root: "/test/root",
		},
		LLM: config.LLMConfig{
			DefaultProvider: "openai",
			DefaultModel:    "gpt-4",
			Providers: map[string]config.ProviderConfig{
				"openai": {
					APIKey: "test-key",
				},
			},
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Timeout:    config.Duration{Duration: time.Minute * 5},
				MaxRetries: 3,
			},
			Roles: map[string]config.Role{
				"coder": {
					Model: "gpt-4",
				},
			},
		},
	}
	serverConfig := DefaultConfig()
	serverConfig.EnableAuth = false

	server, err := NewServer(serverConfig, appConfig, jwtAuth)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	t.Run("GetConfig", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/config", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ConfigResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Version != "1.0.0" {
			t.Errorf("Expected version 1.0.0, got %v", response.Version)
		}

		if response.Project["name"] != "test-project" {
			t.Errorf("Expected project name test-project, got %v", response.Project["name"])
		}
	})

	t.Run("ValidateConfig", func(t *testing.T) {
		validConfig := map[string]interface{}{
			"version": "1.0.0",
			"project": map[string]interface{}{
				"name": "validation-test",
				"root": "/test",
			},
		}
		body, _ := json.Marshal(validConfig)
		req := httptest.NewRequest("POST", "/api/v1/config/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// The partial config will fail validation (missing LLM defaults, etc.)
		// so we expect 422 Unprocessable Entity with structured errors.
		if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusOK {
			t.Errorf("Expected status 200 or 422, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Must contain "valid" boolean field
		if _, ok := response["valid"]; !ok {
			t.Error("Expected response to contain 'valid' field")
		}

		// Must contain "errors" array field
		if _, ok := response["errors"]; !ok {
			t.Error("Expected response to contain 'errors' field")
		}
	})
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(10, 20)

	if limiter == nil {
		t.Fatal("Expected limiter to be created")
	}

	if limiter.requestsPerSecond != 10 {
		t.Errorf("Expected requestsPerSecond 10, got %d", limiter.requestsPerSecond)
	}

	if limiter.burstSize != 20 {
		t.Errorf("Expected burstSize 20, got %d", limiter.burstSize)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	limiter := NewRateLimiter(100, 200)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanupInterval := 50 * time.Millisecond
	go limiter.Cleanup(ctx, cleanupInterval)

	clientLimiter := limiter.getClientLimiter("127.0.0.1")
	if clientLimiter == nil {
		t.Fatal("Expected client limiter to be created")
	}

	time.Sleep(cleanupInterval * 4)

	limiter.mu.RLock()
	exists := false
	for ip := range limiter.clients {
		if ip == "127.0.0.1" {
			exists = true
			break
		}
	}
	limiter.mu.RUnlock()

	if exists {
		t.Error("Expected client limiter to be cleaned up after timeout")
	}
}

func TestLogger(t *testing.T) {
	config := &LoggingConfig{
		Enable:       true,
		LogLevel:     "info",
		LogRequestID: true,
		LogUser:      true,
		LogLatency:   true,
	}

	logger := NewLogger(config)
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if logger.config != config {
		t.Error("Expected logger config to match")
	}
}

func TestMiddleware_RateLimit(t *testing.T) {
	limiter := NewRateLimiter(5, 10)
	middleware := limiter.Middleware

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	successCount := 0
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	if successCount > 10 {
		t.Errorf("Expected at most 10 successful requests (burst), got %d", successCount)
	}
}

func TestMiddleware_Logging(t *testing.T) {
	config := &LoggingConfig{
		Enable:       true,
		LogLevel:     "debug",
		LogLatency:   true,
		LogRequestID: true,
	}

	logger := NewLogger(config)
	middleware := logger.Middleware

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}
