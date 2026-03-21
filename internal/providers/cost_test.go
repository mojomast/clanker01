package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewCostTracker(t *testing.T) {
	tracker := NewCostTracker()
	assert.NotNil(t, tracker)
}

func TestCostTracker_RecordCost(t *testing.T) {
	tracker := NewCostTracker()

	usage := &api.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	tracker.RecordCost("anthropic", "claude-sonnet-4", usage, 3.0, 15.0)

	modelCost, ok := tracker.GetModelCost("claude-sonnet-4")
	assert.True(t, ok)
	assert.Equal(t, int64(1), modelCost.TotalRequests)
	assert.Equal(t, int64(1500), modelCost.TotalTokens)
	assert.Equal(t, int64(1000), modelCost.PromptTokens)
	assert.Equal(t, int64(500), modelCost.CompletionTokens)
	assert.Equal(t, 10.5, modelCost.TotalCost)
}

func TestCostTracker_RecordCost_NilUsage(t *testing.T) {
	tracker := NewCostTracker()

	tracker.RecordCost("anthropic", "claude-sonnet-4", nil, 3.0, 15.0)

	modelCost, ok := tracker.GetModelCost("claude-sonnet-4")
	assert.False(t, ok)
	assert.Nil(t, modelCost)
}

func TestCostTracker_GetProviderCost(t *testing.T) {
	tracker := NewCostTracker()

	usage := &api.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	tracker.RecordCost("openai", "gpt-4", usage, 2.5, 10.0)

	providerCost, ok := tracker.GetProviderCost("openai")
	assert.True(t, ok)
	assert.Equal(t, "openai", providerCost.ProviderName)
	assert.Equal(t, int64(1), providerCost.TotalRequests)
	assert.Equal(t, 7.5, providerCost.TotalCost)
}

func TestCostTracker_GetAllModelCosts(t *testing.T) {
	tracker := NewCostTracker()

	usage1 := &api.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}
	usage2 := &api.Usage{PromptTokens: 2000, CompletionTokens: 1000, TotalTokens: 3000}

	tracker.RecordCost("anthropic", "claude-sonnet-4", usage1, 3.0, 15.0)
	tracker.RecordCost("anthropic", "claude-haiku", usage2, 0.25, 1.25)

	costs := tracker.GetAllModelCosts()
	assert.Len(t, costs, 2)
}

func TestCostTracker_GetAllProviderCosts(t *testing.T) {
	tracker := NewCostTracker()

	usage1 := &api.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}
	usage2 := &api.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}

	tracker.RecordCost("anthropic", "claude-sonnet-4", usage1, 3.0, 15.0)
	tracker.RecordCost("openai", "gpt-4", usage2, 2.5, 10.0)

	costs := tracker.GetAllProviderCosts()
	assert.Len(t, costs, 2)
}

func TestCostTracker_GetTotalCost(t *testing.T) {
	tracker := NewCostTracker()

	usage1 := &api.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}
	usage2 := &api.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}

	tracker.RecordCost("anthropic", "claude-sonnet-4", usage1, 3.0, 15.0)
	tracker.RecordCost("openai", "gpt-4", usage2, 2.5, 10.0)

	totalCost := tracker.GetTotalCost()
	assert.Equal(t, 18.0, totalCost)
}

func TestCostTracker_Reset(t *testing.T) {
	tracker := NewCostTracker()

	usage := &api.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}
	tracker.RecordCost("anthropic", "claude-sonnet-4", usage, 3.0, 15.0)

	tracker.Reset()

	modelCost, ok := tracker.GetModelCost("claude-sonnet-4")
	assert.False(t, ok)
	assert.Nil(t, modelCost)

	providerCost, ok := tracker.GetProviderCost("anthropic")
	assert.False(t, ok)
	assert.Nil(t, providerCost)
}

func TestCostTracker_CostSummary(t *testing.T) {
	tracker := NewCostTracker()

	usage1 := &api.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}
	usage2 := &api.Usage{PromptTokens: 2000, CompletionTokens: 1000, TotalTokens: 3000}

	tracker.RecordCost("anthropic", "claude-sonnet-4", usage1, 3.0, 15.0)
	tracker.RecordCost("anthropic", "claude-haiku", usage2, 0.25, 1.25)

	summary := tracker.CostSummary()
	assert.Contains(t, summary, "Cost Summary")
	assert.Contains(t, summary, "anthropic")
	assert.Contains(t, summary, "Grand Totals")
}

func TestCalculateCost(t *testing.T) {
	provider := NewAnthropicProvider()

	usage := &api.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	cost := CalculateCost(provider.BaseProvider, "claude-sonnet-4", usage)
	assert.Equal(t, 10.5, cost)
}

func TestCalculateCost_NilUsage(t *testing.T) {
	provider := NewAnthropicProvider()

	cost := CalculateCost(provider.BaseProvider, "claude-sonnet-4", nil)
	assert.Equal(t, 0.0, cost)
}

func TestCalculateCost_UnknownModel(t *testing.T) {
	provider := NewAnthropicProvider()

	usage := &api.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	cost := CalculateCost(provider.BaseProvider, "unknown-model", usage)
	assert.Equal(t, 0.0, cost)
}
