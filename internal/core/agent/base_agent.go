package agent

import (
	"context"

	"github.com/swarm-ai/swarm/pkg/api"
)

type ArchitectAgent struct {
	*BaseAgent
}

func NewArchitectAgent(id string, config *api.AgentConfig, provider api.LLMProvider, skills *SkillManager, mcp *MCPConnector) *ArchitectAgent {
	base := NewBaseAgent(id, api.AgentTypeArchitect, config, provider, skills, mcp)
	return &ArchitectAgent{
		BaseAgent: base,
	}
}

func (a *ArchitectAgent) Execute(ctx context.Context, task *api.Task) (*api.AgentResult, error) {
	result, err := a.BaseAgent.Execute(ctx, task)
	if err != nil {
		return result, err
	}

	return result, nil
}

type CoderAgent struct {
	*BaseAgent
}

func NewCoderAgent(id string, config *api.AgentConfig, provider api.LLMProvider, skills *SkillManager, mcp *MCPConnector) *CoderAgent {
	base := NewBaseAgent(id, api.AgentTypeCoder, config, provider, skills, mcp)
	return &CoderAgent{
		BaseAgent: base,
	}
}

func (a *CoderAgent) Execute(ctx context.Context, task *api.Task) (*api.AgentResult, error) {
	result, err := a.BaseAgent.Execute(ctx, task)
	if err != nil {
		return result, err
	}

	return result, nil
}

type TesterAgent struct {
	*BaseAgent
}

func NewTesterAgent(id string, config *api.AgentConfig, provider api.LLMProvider, skills *SkillManager, mcp *MCPConnector) *TesterAgent {
	base := NewBaseAgent(id, api.AgentTypeTester, config, provider, skills, mcp)
	return &TesterAgent{
		BaseAgent: base,
	}
}

func (a *TesterAgent) Execute(ctx context.Context, task *api.Task) (*api.AgentResult, error) {
	result, err := a.BaseAgent.Execute(ctx, task)
	if err != nil {
		return result, err
	}

	return result, nil
}

type ReviewerAgent struct {
	*BaseAgent
}

func NewReviewerAgent(id string, config *api.AgentConfig, provider api.LLMProvider, skills *SkillManager, mcp *MCPConnector) *ReviewerAgent {
	base := NewBaseAgent(id, api.AgentTypeReviewer, config, provider, skills, mcp)
	return &ReviewerAgent{
		BaseAgent: base,
	}
}

func (a *ReviewerAgent) Execute(ctx context.Context, task *api.Task) (*api.AgentResult, error) {
	result, err := a.BaseAgent.Execute(ctx, task)
	if err != nil {
		return result, err
	}

	return result, nil
}

type ResearcherAgent struct {
	*BaseAgent
}

func NewResearcherAgent(id string, config *api.AgentConfig, provider api.LLMProvider, skills *SkillManager, mcp *MCPConnector) *ResearcherAgent {
	base := NewBaseAgent(id, api.AgentTypeResearcher, config, provider, skills, mcp)
	return &ResearcherAgent{
		BaseAgent: base,
	}
}

func (a *ResearcherAgent) Execute(ctx context.Context, task *api.Task) (*api.AgentResult, error) {
	result, err := a.BaseAgent.Execute(ctx, task)
	if err != nil {
		return result, err
	}

	return result, nil
}

type CoordinatorAgent struct {
	*BaseAgent
}

func NewCoordinatorAgent(id string, config *api.AgentConfig, provider api.LLMProvider, skills *SkillManager, mcp *MCPConnector) *CoordinatorAgent {
	base := NewBaseAgent(id, api.AgentTypeCoordinator, config, provider, skills, mcp)
	return &CoordinatorAgent{
		BaseAgent: base,
	}
}

func (a *CoordinatorAgent) Execute(ctx context.Context, task *api.Task) (*api.AgentResult, error) {
	result, err := a.BaseAgent.Execute(ctx, task)
	if err != nil {
		return result, err
	}

	return result, nil
}
