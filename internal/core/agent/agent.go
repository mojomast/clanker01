package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type BaseAgent struct {
	id           string
	agentType    api.AgentType
	config       *api.AgentConfig
	status       api.AgentStatus
	currentTask  *api.Task
	provider     api.LLMProvider
	skills       *SkillManager
	mcp          *MCPConnector
	inbox        chan *api.AgentMessage
	outbox       chan *api.AgentMessage
	mu           sync.RWMutex
	metrics      api.AgentMetrics
	health       api.AgentHealth
	ctx          context.Context
	cancel       context.CancelFunc
	stateMachine *StateMachine
}

func NewBaseAgent(id string, agentType api.AgentType, config *api.AgentConfig, provider api.LLMProvider, skills *SkillManager, mcp *MCPConnector) *BaseAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &BaseAgent{
		id:        id,
		agentType: agentType,
		config:    config,
		status:    api.AgentStatusCreated,
		provider:  provider,
		skills:    skills,
		mcp:       mcp,
		inbox:     make(chan *api.AgentMessage, 100),
		outbox:    make(chan *api.AgentMessage, 100),
		metrics: api.AgentMetrics{
			LastActivity: time.Now(),
		},
		health: api.AgentHealth{
			Status:        "created",
			LastHeartbeat: time.Now(),
		},
		ctx:          ctx,
		cancel:       cancel,
		stateMachine: NewStateMachine(),
	}
}

func (a *BaseAgent) ID() string {
	return a.id
}

func (a *BaseAgent) Type() api.AgentType {
	return a.agentType
}

func (a *BaseAgent) Name() string {
	return a.config.Name
}

func (a *BaseAgent) Initialize(ctx context.Context, config *api.AgentConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.stateMachine.Transition(api.AgentStatusCreated, api.AgentStatusReady); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	a.config = config
	a.status = api.AgentStatusReady
	a.health.Status = "ready"

	for _, skillName := range config.Skills {
		if err := a.skills.Load(ctx, skillName); err != nil {
			return fmt.Errorf("load skill %s: %w", skillName, err)
		}
	}

	return nil
}

func (a *BaseAgent) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.status != api.AgentStatusReady {
		a.mu.Unlock()
		return fmt.Errorf("agent not ready: %s", a.status)
	}
	a.mu.Unlock()

	go a.messageLoop(ctx)
	go a.heartbeat(ctx)

	return nil
}

func (a *BaseAgent) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentTask != nil {
		taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		<-taskCtx.Done()
	}

	if err := a.stateMachine.Transition(a.status, api.AgentStatusTerminated); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	a.status = api.AgentStatusTerminated
	a.health.Status = "terminated"
	a.cancel()
	close(a.inbox)
	close(a.outbox)

	return nil
}

func (a *BaseAgent) Pause(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.stateMachine.Transition(api.AgentStatusRunning, api.AgentStatusPaused); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	a.status = api.AgentStatusPaused
	a.health.Status = "paused"
	return nil
}

func (a *BaseAgent) Resume(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.stateMachine.Transition(api.AgentStatusPaused, api.AgentStatusReady); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	a.status = api.AgentStatusReady
	a.health.Status = "ready"
	return nil
}

func (a *BaseAgent) Execute(ctx context.Context, task *api.Task) (*api.AgentResult, error) {
	a.mu.Lock()
	if a.status != api.AgentStatusReady && a.status != api.AgentStatusRunning {
		a.mu.Unlock()
		return nil, fmt.Errorf("agent not ready: current status %s", a.status)
	}

	if err := a.stateMachine.Transition(a.status, api.AgentStatusRunning); err != nil {
		a.mu.Unlock()
		return nil, fmt.Errorf("invalid state transition: %w", err)
	}

	a.status = api.AgentStatusRunning
	a.currentTask = task
	a.health.Status = "running"
	a.health.ActiveRequests++
	a.metrics.LastActivity = time.Now()
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.status = api.AgentStatusReady
		a.currentTask = nil
		a.health.Status = "ready"
		a.health.ActiveRequests--
		a.mu.Unlock()
	}()

	taskCtx := a.buildTaskContext(task)
	messages := a.buildMessages(task, taskCtx)

	req := &api.ChatRequest{
		Model:        a.config.Model,
		Messages:     messages,
		Tools:        a.getAvailableTools(),
		MaxTokens:    a.config.ResourceLimits.MaxTokensPerTask,
		SystemPrompt: a.config.SystemPrompt,
	}

	result := &api.AgentResult{
		TaskID: task.ID,
	}

	resp, err := a.executeWithTools(ctx, req)
	if err != nil {
		result.Success = false
		result.Error = err

		a.mu.Lock()
		a.metrics.TasksFailed++
		a.mu.Unlock()

		a.handleError(ctx, err)
		return result, err
	}

	result.Success = true
	result.Output = resp.Choices[0].Message.Content
	result.CompletedAt = time.Now()

	a.mu.Lock()
	a.metrics.TasksCompleted++
	a.metrics.TotalTokensUsed += int64(resp.Usage.TotalTokens)
	duration := time.Since(task.StartedAt)
	a.updateAvgDuration(duration)
	a.mu.Unlock()

	return result, nil
}

func (a *BaseAgent) SendMessage(ctx context.Context, msg *api.AgentMessage) error {
	a.outbox <- msg
	return nil
}

func (a *BaseAgent) ReceiveMessage() <-chan *api.AgentMessage {
	return a.inbox
}

func (a *BaseAgent) Broadcast(ctx context.Context, msg *api.AgentMessage) error {
	return a.SendMessage(ctx, msg)
}

func (a *BaseAgent) Status() api.AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *BaseAgent) CurrentTask() *api.Task {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentTask
}

func (a *BaseAgent) Metrics() *api.AgentMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	metricsCopy := a.metrics
	return &metricsCopy
}

func (a *BaseAgent) Health() *api.AgentHealth {
	a.mu.RLock()
	defer a.mu.RUnlock()
	healthCopy := a.health
	return &healthCopy
}

func (a *BaseAgent) messageLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-a.inbox:
			if !ok {
				return
			}
			a.handleMessage(ctx, msg)
		}
	}
}

func (a *BaseAgent) handleMessage(ctx context.Context, msg *api.AgentMessage) {
	switch msg.Type {
	case api.MessageTypeTaskAssignment:
		a.handleTaskAssignment(ctx, msg)
	case api.MessageTypeContextShare:
		a.handleContextShare(ctx, msg)
	case api.MessageTypeAssistanceRequest:
		a.handleAssistanceRequest(ctx, msg)
	case api.MessageTypeConsensusRequest:
		a.handleConsensusRequest(ctx, msg)
	case api.MessageTypeHeartbeat:
		a.handleHeartbeat(ctx, msg)
	}
}

func (a *BaseAgent) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			a.health.LastHeartbeat = time.Now()
			a.health.CPUUsage = a.calculateCPUUsage()
			a.health.MemoryUsageMB = a.calculateMemoryUsage()
			a.mu.Unlock()
		}
	}
}

func (a *BaseAgent) buildTaskContext(task *api.Task) map[string]any {
	return map[string]any{
		"task_id":    task.ID,
		"task_type":  task.AgentType,
		"agent_type": a.agentType,
		"agent_id":   a.id,
		"priority":   task.Priority,
		"context":    task.Description,
	}
}

func (a *BaseAgent) buildMessages(task *api.Task, taskCtx map[string]any) []api.Message {
	return []api.Message{
		{
			Role:    "system",
			Content: a.config.SystemPrompt,
		},
		{
			Role:    "user",
			Content: task.Prompt,
		},
	}
}

func (a *BaseAgent) getAvailableTools() []api.Tool {
	if a.mcp == nil {
		return []api.Tool{}
	}
	return a.mcp.GetAvailableTools()
}

func (a *BaseAgent) executeWithTools(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	maxIterations := 10
	var messages []api.Message = req.Messages

	for i := 0; i < maxIterations; i++ {
		currentReq := *req
		currentReq.Messages = messages

		resp, err := a.provider.Chat(ctx, &currentReq)
		if err != nil {
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("no choices in response")
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			return resp, nil
		}

		messages = append(messages, msg)

		for _, toolCall := range msg.ToolCalls {
			result, err := a.executeToolCall(ctx, toolCall)
			if err != nil {
				return nil, fmt.Errorf("tool execution failed: %w", err)
			}

			messages = append(messages, api.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: toolCall.ID,
			})
		}
	}

	return nil, fmt.Errorf("max tool iterations exceeded")
}

func (a *BaseAgent) executeToolCall(ctx context.Context, toolCall api.ToolCall) (string, error) {
	if a.mcp == nil {
		return "", fmt.Errorf("MCP connector not available")
	}
	return a.mcp.ExecuteTool(ctx, toolCall)
}

func (a *BaseAgent) handleError(ctx context.Context, err error) {
	a.mu.Lock()
	a.health.ErrorCount++
	a.status = api.AgentStatusError
	a.health.Status = "error"
	a.mu.Unlock()
}

func (a *BaseAgent) handleTaskAssignment(ctx context.Context, msg *api.AgentMessage) {
}

func (a *BaseAgent) handleContextShare(ctx context.Context, msg *api.AgentMessage) {
}

func (a *BaseAgent) handleAssistanceRequest(ctx context.Context, msg *api.AgentMessage) {
}

func (a *BaseAgent) handleConsensusRequest(ctx context.Context, msg *api.AgentMessage) {
}

func (a *BaseAgent) handleHeartbeat(ctx context.Context, msg *api.AgentMessage) {
}

func (a *BaseAgent) calculateCPUUsage() float64 {
	return 0.0
}

func (a *BaseAgent) calculateMemoryUsage() int {
	return 0
}

func (a *BaseAgent) updateAvgDuration(duration time.Duration) {
	if a.metrics.TasksCompleted == 1 {
		a.metrics.AvgTaskDuration = duration
		return
	}

	total := a.metrics.AvgTaskDuration * time.Duration(a.metrics.TasksCompleted-1)
	a.metrics.AvgTaskDuration = (total + duration) / time.Duration(a.metrics.TasksCompleted)
}
