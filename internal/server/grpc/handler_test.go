package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/swarm-ai/swarm/internal/security/auth"
	"github.com/swarm-ai/swarm/internal/server/grpc/proto"
)

func setupTestServer(t *testing.T) *Server {
	jwtConfig := auth.DefaultJWTConfig()
	jwtAuth, err := auth.NewJWTAuthenticator(jwtConfig)
	require.NoError(t, err)

	config := DefaultConfig()
	config.EnableAuth = false
	server, err := NewServer(config, jwtAuth)
	require.NoError(t, err)

	return server
}

func TestAgentHandler_CreateAgent(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.CreateAgentRequest{
		Id:            "agent-1",
		Type:          proto.AgentType_AGENT_TYPE_CODER,
		Name:          "Test Agent",
		Model:         "gpt-4",
		SystemPrompt:  "You are a helpful assistant",
		Skills:        []string{"code_generation"},
		MaxConcurrent: 5,
		TimeoutMs:     30000,
		MaxRetries:    3,
		ResourceLimits: &proto.ResourceLimits{
			MaxMemoryMb:      1024,
			MaxCpuPercent:    80,
			MaxTokensPerTask: 4000,
			MaxTasksPerHour:  100,
		},
	}

	resp, err := handler.CreateAgent(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Agent)
	assert.Equal(t, req.Id, resp.Agent.Id)
	assert.Equal(t, req.Name, resp.Agent.Name)
	assert.Equal(t, req.Model, resp.Agent.Model)
	assert.Equal(t, proto.AgentStatus_AGENT_STATUS_CREATED, resp.Agent.Status)
	assert.NotNil(t, resp.Agent.CreatedAt)
}

func TestAgentHandler_GetAgent(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.GetAgentRequest{
		Id: "agent-1",
	}

	resp, err := handler.GetAgent(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Agent)
	assert.Equal(t, "agent-1", resp.Agent.Id)
	assert.Equal(t, "Test Agent", resp.Agent.Name)
}

func TestAgentHandler_GetAgent_InvalidID(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.GetAgentRequest{
		Id: "",
	}

	resp, err := handler.GetAgent(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestAgentHandler_ListAgents(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.ListAgentsRequest{}

	resp, err := handler.ListAgents(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Agents)
	assert.GreaterOrEqual(t, resp.TotalCount, int32(2))
}

func TestAgentHandler_UpdateAgent(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.UpdateAgentRequest{
		Id:           "agent-1",
		Name:         "Updated Agent",
		SystemPrompt: "Updated prompt",
		Skills:       []string{"code_generation", "testing"},
	}

	resp, err := handler.UpdateAgent(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Agent)
	assert.Equal(t, "Updated Agent", resp.Agent.Name)
}

func TestAgentHandler_DeleteAgent(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.DeleteAgentRequest{
		Id: "agent-1",
	}

	resp, err := handler.DeleteAgent(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestAgentHandler_StartAgent(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.StartAgentRequest{
		Id: "agent-1",
	}

	resp, err := handler.StartAgent(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Agent)
	assert.Equal(t, proto.AgentStatus_AGENT_STATUS_READY, resp.Agent.Status)
}

func TestAgentHandler_StopAgent(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.StopAgentRequest{
		Id: "agent-1",
	}

	resp, err := handler.StopAgent(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Agent)
	assert.Equal(t, proto.AgentStatus_AGENT_STATUS_TERMINATED, resp.Agent.Status)
}

func TestAgentHandler_PauseAgent(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.PauseAgentRequest{
		Id: "agent-1",
	}

	resp, err := handler.PauseAgent(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Agent)
	assert.Equal(t, proto.AgentStatus_AGENT_STATUS_PAUSED, resp.Agent.Status)
}

func TestAgentHandler_ResumeAgent(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.ResumeAgentRequest{
		Id: "agent-1",
	}

	resp, err := handler.ResumeAgent(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Agent)
	assert.Equal(t, proto.AgentStatus_AGENT_STATUS_READY, resp.Agent.Status)
}

func TestAgentHandler_GetAgentMetrics(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.GetAgentMetricsRequest{
		Id: "agent-1",
	}

	resp, err := handler.GetAgentMetrics(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Metrics)
	assert.Equal(t, int64(100), resp.Metrics.TasksCompleted)
	assert.Equal(t, int64(5), resp.Metrics.TasksFailed)
}

func TestAgentHandler_GetAgentHealth(t *testing.T) {
	server := setupTestServer(t)
	handler := server.agentHandler

	req := &proto.GetAgentHealthRequest{
		Id: "agent-1",
	}

	resp, err := handler.GetAgentHealth(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Health)
	assert.Equal(t, "healthy", resp.Health.Status)
	assert.Greater(t, resp.Health.CpuUsage, 0.0)
}

func TestTaskHandler_CreateTask(t *testing.T) {
	server := setupTestServer(t)
	handler := server.taskHandler

	req := &proto.CreateTaskRequest{
		Id:          "task-1",
		Name:        "Test Task",
		Description: "A test task",
		Prompt:      "Complete this task",
		Kind:        proto.TaskKind_TASK_KIND_COMPUTE,
		Priority:    2,
		AgentType:   proto.AgentType_AGENT_TYPE_CODER,
		TimeoutMs:   30000,
		MaxRetries:  3,
		Input:       map[string]string{"param1": "value1"},
	}

	resp, err := handler.CreateTask(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Task)
	assert.Equal(t, req.Id, resp.Task.Id)
	assert.Equal(t, req.Name, resp.Task.Name)
	assert.Equal(t, proto.TaskStatus_TASK_STATUS_PENDING, resp.Task.Status)
}

func TestTaskHandler_GetTask(t *testing.T) {
	server := setupTestServer(t)
	handler := server.taskHandler

	req := &proto.GetTaskRequest{
		Id: "task-1",
	}

	resp, err := handler.GetTask(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Task)
	assert.Equal(t, "task-1", resp.Task.Id)
	assert.Equal(t, "Test Task", resp.Task.Name)
}

func TestTaskHandler_ListTasks(t *testing.T) {
	server := setupTestServer(t)
	handler := server.taskHandler

	req := &proto.ListTasksRequest{}

	resp, err := handler.ListTasks(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Tasks)
	assert.GreaterOrEqual(t, resp.TotalCount, int32(2))
}

func TestTaskHandler_ExecuteTask(t *testing.T) {
	server := setupTestServer(t)
	handler := server.taskHandler

	req := &proto.ExecuteTaskRequest{
		TaskId:  "task-1",
		AgentId: "agent-1",
	}

	resp, err := handler.ExecuteTask(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Result)
	assert.True(t, resp.Result.Success)
	assert.Equal(t, "task-1", resp.Result.TaskId)
}

func TestTaskHandler_CancelTask(t *testing.T) {
	server := setupTestServer(t)
	handler := server.taskHandler

	req := &proto.CancelTaskRequest{
		Id: "task-1",
	}

	resp, err := handler.CancelTask(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Task)
	assert.Equal(t, proto.TaskStatus_TASK_STATUS_CANCELLED, resp.Task.Status)
}

func TestTaskHandler_RetryTask(t *testing.T) {
	server := setupTestServer(t)
	handler := server.taskHandler

	req := &proto.RetryTaskRequest{
		Id: "task-1",
	}

	resp, err := handler.RetryTask(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Task)
	assert.Equal(t, int32(1), resp.Task.RetryCount)
}

func TestTaskHandler_ListTaskDependencies(t *testing.T) {
	server := setupTestServer(t)
	handler := server.taskHandler

	req := &proto.ListTaskDependenciesRequest{
		TaskId: "task-1",
	}

	resp, err := handler.ListTaskDependencies(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Dependencies)
}

func TestSkillHandler_RegisterSkill(t *testing.T) {
	server := setupTestServer(t)
	handler := server.skillHandler

	req := &proto.RegisterSkillRequest{
		Manifest: &proto.SkillManifest{
			Metadata: &proto.SkillMetadata{
				Name:        "test_skill",
				Version:     "1.0.0",
				DisplayName: "Test Skill",
				Description: "A test skill",
				Author:      "Test Author",
				License:     "MIT",
				Tags:        []string{"test"},
			},
			Spec: &proto.SkillSpec{
				Runtime:    "go",
				Entrypoint: "main.go",
			},
		},
	}

	resp, err := handler.RegisterSkill(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Skill)
	assert.Equal(t, "test_skill@1.0.0", resp.Skill.Id)
	assert.Equal(t, "test_skill", resp.Skill.Name)
	assert.Equal(t, "1.0.0", resp.Skill.Version)
}

func TestSkillHandler_GetSkill(t *testing.T) {
	server := setupTestServer(t)
	handler := server.skillHandler

	req := &proto.GetSkillRequest{
		Name:    "code_generation",
		Version: "1.0.0",
	}

	resp, err := handler.GetSkill(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Skill)
	assert.Equal(t, "code_generation", resp.Skill.Name)
}

func TestSkillHandler_ListSkills(t *testing.T) {
	server := setupTestServer(t)
	handler := server.skillHandler

	req := &proto.ListSkillsRequest{}

	resp, err := handler.ListSkills(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Skills)
	assert.GreaterOrEqual(t, resp.TotalCount, int32(2))
}

func TestSkillHandler_UpdateSkill(t *testing.T) {
	server := setupTestServer(t)
	handler := server.skillHandler

	req := &proto.UpdateSkillRequest{
		Name:    "test_skill",
		Version: "1.0.0",
		Manifest: &proto.SkillManifest{
			Metadata: &proto.SkillMetadata{
				Name:        "test_skill",
				Version:     "1.0.0",
				DisplayName: "Updated Test Skill",
				Description: "Updated description",
			},
		},
	}

	resp, err := handler.UpdateSkill(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Skill)
	assert.Equal(t, "Updated Test Skill", resp.Skill.DisplayName)
}

func TestSkillHandler_DeleteSkill(t *testing.T) {
	server := setupTestServer(t)
	handler := server.skillHandler

	req := &proto.DeleteSkillRequest{
		Name:    "test_skill",
		Version: "1.0.0",
	}

	resp, err := handler.DeleteSkill(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestSkillHandler_DiscoverSkills(t *testing.T) {
	server := setupTestServer(t)
	handler := server.skillHandler

	req := &proto.DiscoverSkillsRequest{
		Query: "code generation",
		Tags:  []string{"code"},
		Limit: 10,
	}

	resp, err := handler.DiscoverSkills(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Matches)
}

func TestSkillHandler_GetSkillManifest(t *testing.T) {
	server := setupTestServer(t)
	handler := server.skillHandler

	req := &proto.GetSkillManifestRequest{
		Name:    "test_skill",
		Version: "1.0.0",
	}

	resp, err := handler.GetSkillManifest(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Manifest)
	assert.Equal(t, "test_skill", resp.Manifest.Metadata.Name)
}
