package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/swarm-ai/swarm/internal/config"
	"github.com/swarm-ai/swarm/pkg/api"
)

type AgentRequest struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Model  string                 `json:"model"`
	Skills []string               `json:"skills"`
	Config map[string]interface{} `json:"config"`
}

type AgentResponse struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Config    map[string]interface{} `json:"config"`
	Metrics   map[string]interface{} `json:"metrics"`
}

type TaskRequest struct {
	Type       string                 `json:"type"`
	Prompt     string                 `json:"prompt"`
	AgentType  api.AgentType          `json:"agent_type"`
	AgentID    string                 `json:"agent_id,omitempty"`
	Priority   int                    `json:"priority"`
	MaxRetries int                    `json:"max_retries"`
	Timeout    string                 `json:"timeout"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type TaskResponse struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Prompt        string                 `json:"prompt"`
	Status        string                 `json:"status"`
	AgentType     string                 `json:"agent_type"`
	AssignedAgent string                 `json:"assigned_agent"`
	CreatedAt     time.Time              `json:"created_at"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Result        interface{}            `json:"result,omitempty"`
	Error         string                 `json:"error,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type SkillRequest struct {
	Name    string                 `json:"name"`
	Version string                 `json:"version"`
	Config  map[string]interface{} `json:"config"`
	Enable  bool                   `json:"enable"`
}

type SkillResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	LoadedAt    time.Time              `json:"loaded_at"`
	Config      map[string]interface{} `json:"config"`
	Tools       []ToolInfo             `json:"tools"`
}

type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Returns     map[string]interface{} `json:"returns"`
}

type ConfigResponse struct {
	Version  string                 `json:"version"`
	Project  map[string]interface{} `json:"project"`
	LLM      map[string]interface{} `json:"llm"`
	Agents   map[string]interface{} `json:"agents"`
	Skills   map[string]interface{} `json:"skills"`
	Server   map[string]interface{} `json:"server"`
	Security map[string]interface{} `json:"security"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   s.appConfig.Version,
	}

	if startTime, ok := s.ctx.Value(startTimeKey).(time.Time); ok {
		health["uptime"] = time.Since(startTime).String()
	} else {
		health["uptime"] = "unknown"
	}

	s.respondJSON(w, http.StatusOK, health)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var agents []*AgentInfo
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}

	response := map[string]interface{}{
		"count":  len(agents),
		"agents": agents,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "Agent name is required", nil)
		return
	}

	if req.Type == "" {
		req.Type = string(api.AgentTypeCoder)
	}

	agentID := fmt.Sprintf("agent_%d", time.Now().UnixNano())
	now := time.Now()

	config := req.Config
	if config == nil {
		config = make(map[string]interface{})
	}

	if req.Model != "" {
		config["model"] = req.Model
	}
	if len(req.Skills) > 0 {
		config["skills"] = req.Skills
	}

	agentInfo := &AgentInfo{
		ID:        agentID,
		Type:      req.Type,
		Name:      req.Name,
		Status:    string(api.AgentStatusCreated),
		CreatedAt: now,
		UpdatedAt: now,
		Config:    config,
		Metrics:   make(map[string]interface{}),
	}

	s.mu.Lock()
	s.agents[agentID] = agentInfo
	s.mu.Unlock()

	response := &AgentResponse{
		ID:        agentInfo.ID,
		Type:      agentInfo.Type,
		Name:      agentInfo.Name,
		Status:    agentInfo.Status,
		CreatedAt: agentInfo.CreatedAt,
		UpdatedAt: agentInfo.UpdatedAt,
		Config:    agentInfo.Config,
		Metrics:   agentInfo.Metrics,
	}

	s.respondJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	s.mu.RLock()
	agentInfo, exists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists {
		s.respondError(w, http.StatusNotFound, "Agent not found", nil)
		return
	}

	response := &AgentResponse{
		ID:        agentInfo.ID,
		Type:      agentInfo.Type,
		Name:      agentInfo.Name,
		Status:    agentInfo.Status,
		CreatedAt: agentInfo.CreatedAt,
		UpdatedAt: agentInfo.UpdatedAt,
		Config:    agentInfo.Config,
		Metrics:   agentInfo.Metrics,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	s.mu.Lock()
	agentInfo, exists := s.agents[agentID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Agent not found", nil)
		return
	}

	if req.Name != "" {
		agentInfo.Name = req.Name
	}
	if req.Type != "" {
		agentInfo.Type = req.Type
	}
	if req.Config != nil {
		for k, v := range req.Config {
			agentInfo.Config[k] = v
		}
	}
	agentInfo.UpdatedAt = time.Now()
	s.mu.Unlock()

	response := &AgentResponse{
		ID:        agentInfo.ID,
		Type:      agentInfo.Type,
		Name:      agentInfo.Name,
		Status:    agentInfo.Status,
		CreatedAt: agentInfo.CreatedAt,
		UpdatedAt: agentInfo.UpdatedAt,
		Config:    agentInfo.Config,
		Metrics:   agentInfo.Metrics,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	s.mu.Lock()
	_, exists := s.agents[agentID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Agent not found", nil)
		return
	}

	delete(s.agents, agentID)
	s.mu.Unlock()

	s.respondJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleStartAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	s.mu.Lock()
	agentInfo, exists := s.agents[agentID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Agent not found", nil)
		return
	}

	agentInfo.Status = string(api.AgentStatusRunning)
	agentInfo.UpdatedAt = time.Now()
	s.mu.Unlock()

	response := &AgentResponse{
		ID:        agentInfo.ID,
		Type:      agentInfo.Type,
		Name:      agentInfo.Name,
		Status:    agentInfo.Status,
		CreatedAt: agentInfo.CreatedAt,
		UpdatedAt: agentInfo.UpdatedAt,
		Config:    agentInfo.Config,
		Metrics:   agentInfo.Metrics,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleStopAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	s.mu.Lock()
	agentInfo, exists := s.agents[agentID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Agent not found", nil)
		return
	}

	agentInfo.Status = string(api.AgentStatusTerminated)
	agentInfo.UpdatedAt = time.Now()
	s.mu.Unlock()

	response := &AgentResponse{
		ID:        agentInfo.ID,
		Type:      agentInfo.Type,
		Name:      agentInfo.Name,
		Status:    agentInfo.Status,
		CreatedAt: agentInfo.CreatedAt,
		UpdatedAt: agentInfo.UpdatedAt,
		Config:    agentInfo.Config,
		Metrics:   agentInfo.Metrics,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handlePauseAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	s.mu.Lock()
	agentInfo, exists := s.agents[agentID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Agent not found", nil)
		return
	}

	agentInfo.Status = string(api.AgentStatusPaused)
	agentInfo.UpdatedAt = time.Now()
	s.mu.Unlock()

	response := &AgentResponse{
		ID:        agentInfo.ID,
		Type:      agentInfo.Type,
		Name:      agentInfo.Name,
		Status:    agentInfo.Status,
		CreatedAt: agentInfo.CreatedAt,
		UpdatedAt: agentInfo.UpdatedAt,
		Config:    agentInfo.Config,
		Metrics:   agentInfo.Metrics,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleResumeAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	s.mu.Lock()
	agentInfo, exists := s.agents[agentID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Agent not found", nil)
		return
	}

	agentInfo.Status = string(api.AgentStatusReady)
	agentInfo.UpdatedAt = time.Now()
	s.mu.Unlock()

	response := &AgentResponse{
		ID:        agentInfo.ID,
		Type:      agentInfo.Type,
		Name:      agentInfo.Name,
		Status:    agentInfo.Status,
		CreatedAt: agentInfo.CreatedAt,
		UpdatedAt: agentInfo.UpdatedAt,
		Config:    agentInfo.Config,
		Metrics:   agentInfo.Metrics,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetAgentMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	s.mu.RLock()
	agentInfo, exists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists {
		s.respondError(w, http.StatusNotFound, "Agent not found", nil)
		return
	}

	s.respondJSON(w, http.StatusOK, agentInfo.Metrics)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*TaskInfo
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	response := map[string]interface{}{
		"count": len(tasks),
		"tasks": tasks,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Prompt == "" {
		s.respondError(w, http.StatusBadRequest, "Task prompt is required", nil)
		return
	}

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	now := time.Now()

	taskInfo := &TaskInfo{
		ID:            taskID,
		Type:          req.Type,
		Prompt:        req.Prompt,
		Status:        string(api.TaskStatusPending),
		AgentType:     string(req.AgentType),
		AssignedAgent: req.AgentID,
		CreatedAt:     now,
		RetryCount:    0,
		Metadata:      req.Metadata,
	}

	s.mu.Lock()
	s.tasks[taskID] = taskInfo
	s.mu.Unlock()

	response := &TaskResponse{
		ID:            taskInfo.ID,
		Type:          taskInfo.Type,
		Prompt:        taskInfo.Prompt,
		Status:        taskInfo.Status,
		AgentType:     taskInfo.AgentType,
		AssignedAgent: taskInfo.AssignedAgent,
		CreatedAt:     taskInfo.CreatedAt,
		RetryCount:    taskInfo.RetryCount,
		Metadata:      taskInfo.Metadata,
	}

	s.respondJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	s.mu.RLock()
	taskInfo, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		s.respondError(w, http.StatusNotFound, "Task not found", nil)
		return
	}

	response := &TaskResponse{
		ID:            taskInfo.ID,
		Type:          taskInfo.Type,
		Prompt:        taskInfo.Prompt,
		Status:        taskInfo.Status,
		AgentType:     taskInfo.AgentType,
		AssignedAgent: taskInfo.AssignedAgent,
		CreatedAt:     taskInfo.CreatedAt,
		StartedAt:     taskInfo.StartedAt,
		CompletedAt:   taskInfo.CompletedAt,
		Result:        taskInfo.Result,
		Error:         taskInfo.Error,
		RetryCount:    taskInfo.RetryCount,
		Metadata:      taskInfo.Metadata,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	s.mu.Lock()
	taskInfo, exists := s.tasks[taskID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Task not found", nil)
		return
	}

	if req.Prompt != "" {
		taskInfo.Prompt = req.Prompt
	}
	if req.Type != "" {
		taskInfo.Type = req.Type
	}
	if req.AgentID != "" {
		taskInfo.AssignedAgent = req.AgentID
	}
	if req.Metadata != nil {
		taskInfo.Metadata = req.Metadata
	}
	s.mu.Unlock()

	response := &TaskResponse{
		ID:            taskInfo.ID,
		Type:          taskInfo.Type,
		Prompt:        taskInfo.Prompt,
		Status:        taskInfo.Status,
		AgentType:     taskInfo.AgentType,
		AssignedAgent: taskInfo.AssignedAgent,
		CreatedAt:     taskInfo.CreatedAt,
		StartedAt:     taskInfo.StartedAt,
		CompletedAt:   taskInfo.CompletedAt,
		Result:        taskInfo.Result,
		Error:         taskInfo.Error,
		RetryCount:    taskInfo.RetryCount,
		Metadata:      taskInfo.Metadata,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	s.mu.Lock()
	_, exists := s.tasks[taskID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Task not found", nil)
		return
	}

	delete(s.tasks, taskID)
	s.mu.Unlock()

	s.respondJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleStartTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	s.mu.Lock()
	taskInfo, exists := s.tasks[taskID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Task not found", nil)
		return
	}

	now := time.Now()
	taskInfo.Status = string(api.TaskStatusRunning)
	taskInfo.StartedAt = &now
	s.mu.Unlock()

	response := &TaskResponse{
		ID:            taskInfo.ID,
		Type:          taskInfo.Type,
		Prompt:        taskInfo.Prompt,
		Status:        taskInfo.Status,
		AgentType:     taskInfo.AgentType,
		AssignedAgent: taskInfo.AssignedAgent,
		CreatedAt:     taskInfo.CreatedAt,
		StartedAt:     taskInfo.StartedAt,
		RetryCount:    taskInfo.RetryCount,
		Metadata:      taskInfo.Metadata,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	s.mu.Lock()
	taskInfo, exists := s.tasks[taskID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Task not found", nil)
		return
	}

	taskInfo.Status = string(api.TaskStatusCancelled)
	s.mu.Unlock()

	response := &TaskResponse{
		ID:            taskInfo.ID,
		Type:          taskInfo.Type,
		Prompt:        taskInfo.Prompt,
		Status:        taskInfo.Status,
		AgentType:     taskInfo.AgentType,
		AssignedAgent: taskInfo.AssignedAgent,
		CreatedAt:     taskInfo.CreatedAt,
		StartedAt:     taskInfo.StartedAt,
		RetryCount:    taskInfo.RetryCount,
		Metadata:      taskInfo.Metadata,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var skills []*SkillInfo
	for _, skill := range s.skills {
		skills = append(skills, skill)
	}

	response := map[string]interface{}{
		"count":  len(skills),
		"skills": skills,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleLoadSkill(w http.ResponseWriter, r *http.Request) {
	var req SkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "Skill name is required", nil)
		return
	}

	skillID := fmt.Sprintf("skill_%s", req.Name)
	if req.Version != "" {
		skillID = fmt.Sprintf("%s_%s", skillID, req.Version)
	}
	skillID = fmt.Sprintf("%d_%s", time.Now().UnixNano(), skillID)

	now := time.Now()

	skillInfo := &SkillInfo{
		ID:          skillID,
		Name:        req.Name,
		Version:     req.Version,
		Description: fmt.Sprintf("Skill: %s", req.Name),
		Status:      "loaded",
		LoadedAt:    now,
		Config:      req.Config,
	}

	s.mu.Lock()
	s.skills[skillID] = skillInfo
	s.mu.Unlock()

	response := &SkillResponse{
		ID:          skillInfo.ID,
		Name:        skillInfo.Name,
		Version:     skillInfo.Version,
		Description: skillInfo.Description,
		Status:      skillInfo.Status,
		LoadedAt:    skillInfo.LoadedAt,
		Config:      skillInfo.Config,
		Tools:       []ToolInfo{},
	}

	s.respondJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetSkill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["id"]

	s.mu.RLock()
	skillInfo, exists := s.skills[skillID]
	s.mu.RUnlock()

	if !exists {
		s.respondError(w, http.StatusNotFound, "Skill not found", nil)
		return
	}

	response := &SkillResponse{
		ID:          skillInfo.ID,
		Name:        skillInfo.Name,
		Version:     skillInfo.Version,
		Description: skillInfo.Description,
		Status:      skillInfo.Status,
		LoadedAt:    skillInfo.LoadedAt,
		Config:      skillInfo.Config,
		Tools:       []ToolInfo{},
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateSkill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["id"]

	var req SkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	s.mu.Lock()
	skillInfo, exists := s.skills[skillID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Skill not found", nil)
		return
	}

	if req.Config != nil {
		skillInfo.Config = req.Config
	}
	s.mu.Unlock()

	response := &SkillResponse{
		ID:          skillInfo.ID,
		Name:        skillInfo.Name,
		Version:     skillInfo.Version,
		Description: skillInfo.Description,
		Status:      skillInfo.Status,
		LoadedAt:    skillInfo.LoadedAt,
		Config:      skillInfo.Config,
		Tools:       []ToolInfo{},
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleUnloadSkill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["id"]

	s.mu.Lock()
	_, exists := s.skills[skillID]
	if !exists {
		s.mu.Unlock()
		s.respondError(w, http.StatusNotFound, "Skill not found", nil)
		return
	}

	delete(s.skills, skillID)
	s.mu.Unlock()

	s.respondJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleListSkillTools(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["id"]

	s.mu.RLock()
	skillInfo, exists := s.skills[skillID]
	s.mu.RUnlock()

	if !exists {
		s.respondError(w, http.StatusNotFound, "Skill not found", nil)
		return
	}

	response := &SkillResponse{
		ID:          skillInfo.ID,
		Name:        skillInfo.Name,
		Version:     skillInfo.Version,
		Description: skillInfo.Description,
		Status:      skillInfo.Status,
		LoadedAt:    skillInfo.LoadedAt,
		Config:      skillInfo.Config,
		Tools:       []ToolInfo{},
	}

	s.respondJSON(w, http.StatusOK, response.Tools)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.getConfig()
	redacted := redactConfig(cfg)
	s.respondJSON(w, http.StatusOK, redacted)
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if s.configMgr == nil {
		s.respondError(w, http.StatusNotImplemented, "config manager not available", nil)
		return
	}

	// Decode partial update from request body into a generic map first
	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Apply changes via the config manager's transactional Update
	if err := s.configMgr.Update(func(cfg *config.Config) error {
		// Re-serialize the current config to JSON, merge the patch, then
		// deserialize back. This supports partial updates without requiring
		// the client to send the full config.
		current, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to serialize current config: %w", err)
		}

		var merged map[string]json.RawMessage
		if err := json.Unmarshal(current, &merged); err != nil {
			return fmt.Errorf("failed to parse current config: %w", err)
		}

		for k, v := range patch {
			merged[k] = v
		}

		mergedBytes, err := json.Marshal(merged)
		if err != nil {
			return fmt.Errorf("failed to serialize merged config: %w", err)
		}

		if err := json.Unmarshal(mergedBytes, cfg); err != nil {
			return fmt.Errorf("failed to apply merged config: %w", err)
		}

		return nil
	}); err != nil {
		s.respondError(w, http.StatusBadRequest, "config update failed", err)
		return
	}

	// Persist the updated config to disk (best-effort; report errors)
	if err := s.configMgr.Save(); err != nil {
		s.respondError(w, http.StatusInternalServerError, "config updated in memory but failed to persist", err)
		return
	}

	// Return the updated (redacted) config
	cfg := s.configMgr.Get()
	redacted := redactConfig(cfg)
	s.respondJSON(w, http.StatusOK, redacted)
}

func (s *Server) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	result := config.ValidateWithResult(&cfg)

	// Build a serializable response
	type validationErrorResp struct {
		Field   string      `json:"field"`
		Message string      `json:"message"`
		Value   interface{} `json:"value,omitempty"`
	}

	errors := make([]validationErrorResp, 0, len(result.Errors))
	for _, e := range result.Errors {
		errors = append(errors, validationErrorResp{
			Field:   e.Field,
			Message: e.Message,
			Value:   e.Value,
		})
	}

	resp := map[string]interface{}{
		"valid":    result.Valid,
		"errors":   errors,
		"warnings": result.Warnings,
	}

	status := http.StatusOK
	if !result.Valid {
		status = http.StatusUnprocessableEntity
	}
	s.respondJSON(w, status, resp)
}

func (s *Server) handleGetConfigSchema(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, configSchemaDefinition)
}

// redactConfig creates a ConfigResponse with sensitive values masked.
func redactConfig(cfg *config.Config) *ConfigResponse {
	// Build providers map with redacted API keys
	providers := make(map[string]interface{})
	for name, p := range cfg.LLM.Providers {
		apiKey := ""
		if p.APIKey != "" {
			apiKey = "****"
		}
		providers[name] = map[string]interface{}{
			"api_key":  apiKey,
			"base_url": p.BaseURL,
			"models":   p.Models,
			"options":  p.Options,
		}
	}

	// Redact JWT secret
	jwtSecret := ""
	if cfg.Server.Auth.JWTSecret != "" {
		jwtSecret = "****"
	}

	// Redact env vars containing sensitive keywords in MCP server configs
	mcpServers := make(map[string]interface{})
	for name, srv := range cfg.MCP.Servers {
		env := make(map[string]string)
		for k, v := range srv.Env {
			upper := strings.ToUpper(k)
			if strings.Contains(upper, "KEY") || strings.Contains(upper, "SECRET") ||
				strings.Contains(upper, "TOKEN") || strings.Contains(upper, "PASSWORD") {
				if v != "" {
					env[k] = "****"
				} else {
					env[k] = ""
				}
			} else {
				env[k] = v
			}
		}
		mcpServers[name] = map[string]interface{}{
			"type":    srv.Type,
			"command": srv.Cmd,
			"args":    srv.Args,
			"env":     env,
			"url":     srv.URL,
		}
	}

	return &ConfigResponse{
		Version: cfg.Version,
		Project: map[string]interface{}{
			"name": cfg.Project.Name,
			"root": cfg.Project.Root,
		},
		LLM: map[string]interface{}{
			"default_provider":    cfg.LLM.DefaultProvider,
			"default_model":       cfg.LLM.DefaultModel,
			"providers":           providers,
			"agent_model_mapping": cfg.LLM.AgentModelMapping,
		},
		Agents: map[string]interface{}{
			"defaults": cfg.Agents.Defaults,
			"roles":    cfg.Agents.Roles,
		},
		Skills: map[string]interface{}{
			"builtin":  cfg.Skills.Builtin,
			"external": cfg.Skills.External,
		},
		Server: map[string]interface{}{
			"enabled": cfg.Server.Enabled,
			"grpc":    cfg.Server.GRPC,
			"http":    cfg.Server.HTTP,
			"auth": map[string]interface{}{
				"enabled":    cfg.Server.Auth.Enabled,
				"jwt_secret": jwtSecret,
			},
		},
		Security: map[string]interface{}{
			"sandbox": cfg.Security.Sandbox,
			"audit":   cfg.Security.Audit,
			"secrets": cfg.Security.Secrets,
		},
	}
}
