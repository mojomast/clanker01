package providers

import (
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

// CostTracker tracks and calculates LLM costs
type CostTracker struct {
	mu         sync.RWMutex
	byModel    map[string]*ModelCost
	byProvider map[string]*ProviderCost
}

// ModelCost tracks costs for a specific model
type ModelCost struct {
	ModelID          string
	TotalRequests    int64
	TotalTokens      int64
	PromptTokens     int64
	CompletionTokens int64
	TotalCost        float64
	LastUpdated      time.Time
}

// ProviderCost tracks costs for a specific provider
type ProviderCost struct {
	ProviderName     string
	TotalRequests    int64
	TotalTokens      int64
	PromptTokens     int64
	CompletionTokens int64
	TotalCost        float64
	Models           map[string]*ModelCost
	LastUpdated      time.Time
}

// NewCostTracker creates a new cost tracker
func NewCostTracker() *CostTracker {
	return &CostTracker{
		byModel:    make(map[string]*ModelCost),
		byProvider: make(map[string]*ProviderCost),
	}
}

// RecordCost records cost for a request
func (c *CostTracker) RecordCost(provider string, model string, usage *api.Usage, inputPrice, outputPrice float64) {
	if usage == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	promptCost := (float64(usage.PromptTokens) / 1000) * inputPrice
	outputCost := (float64(usage.CompletionTokens) / 1000) * outputPrice
	totalCost := promptCost + outputCost

	modelCost := c.byModel[model]
	if modelCost == nil {
		modelCost = &ModelCost{
			ModelID: model,
		}
		c.byModel[model] = modelCost
	}

	modelCost.TotalRequests++
	modelCost.TotalTokens += int64(usage.TotalTokens)
	modelCost.PromptTokens += int64(usage.PromptTokens)
	modelCost.CompletionTokens += int64(usage.CompletionTokens)
	modelCost.TotalCost += totalCost
	modelCost.LastUpdated = time.Now()

	providerCost := c.byProvider[provider]
	if providerCost == nil {
		providerCost = &ProviderCost{
			ProviderName: provider,
			Models:       make(map[string]*ModelCost),
		}
		c.byProvider[provider] = providerCost
	}

	providerCost.TotalRequests++
	providerCost.TotalTokens += int64(usage.TotalTokens)
	providerCost.PromptTokens += int64(usage.PromptTokens)
	providerCost.CompletionTokens += int64(usage.CompletionTokens)
	providerCost.TotalCost += totalCost
	providerCost.LastUpdated = time.Now()

	pModelCost := providerCost.Models[model]
	if pModelCost == nil {
		pModelCost = &ModelCost{ModelID: model}
		providerCost.Models[model] = pModelCost
	}

	pModelCost.TotalRequests++
	pModelCost.TotalTokens += int64(usage.TotalTokens)
	pModelCost.PromptTokens += int64(usage.PromptTokens)
	pModelCost.CompletionTokens += int64(usage.CompletionTokens)
	pModelCost.TotalCost += totalCost
	pModelCost.LastUpdated = time.Now()
}

// GetModelCost returns cost for a specific model
func (c *CostTracker) GetModelCost(model string) (*ModelCost, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cost, ok := c.byModel[model]
	return cost, ok
}

// GetProviderCost returns cost for a specific provider
func (c *CostTracker) GetProviderCost(provider string) (*ProviderCost, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cost, ok := c.byProvider[provider]
	return cost, ok
}

// GetAllModelCosts returns all model costs
func (c *CostTracker) GetAllModelCosts() []*ModelCost {
	c.mu.RLock()
	defer c.mu.RUnlock()

	costs := make([]*ModelCost, 0, len(c.byModel))
	for _, cost := range c.byModel {
		costs = append(costs, cost)
	}

	return costs
}

// GetAllProviderCosts returns all provider costs
func (c *CostTracker) GetAllProviderCosts() []*ProviderCost {
	c.mu.RLock()
	defer c.mu.RUnlock()

	costs := make([]*ProviderCost, 0, len(c.byProvider))
	for _, cost := range c.byProvider {
		costs = append(costs, cost)
	}

	return costs
}

// GetTotalCost returns the total cost across all providers and models
func (c *CostTracker) GetTotalCost() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var total float64
	for _, cost := range c.byProvider {
		total += cost.TotalCost
	}

	return total
}

// Reset clears all cost tracking
func (c *CostTracker) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.byModel = make(map[string]*ModelCost)
	c.byProvider = make(map[string]*ProviderCost)
}

// CalculateCost calculates cost for a request
func CalculateCost(provider *BaseProvider, model string, usage *api.Usage) float64 {
	if usage == nil {
		return 0
	}

	var inputPrice, outputPrice float64
	for _, m := range provider.models {
		if m.ID == model || m.Alias == model {
			inputPrice = m.InputPricePer1K
			outputPrice = m.OutputPricePer1K
			break
		}
	}

	promptCost := (float64(usage.PromptTokens) / 1000) * inputPrice
	outputCost := (float64(usage.CompletionTokens) / 1000) * outputPrice

	return promptCost + outputCost
}

// CostSummary generates a cost summary report
func (c *CostTracker) CostSummary() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalCost float64
	var totalTokens int64
	var totalRequests int64

	summary := "Cost Summary\n"
	summary += "============\n\n"

	for providerName, cost := range c.byProvider {
		summary += fmt.Sprintf("Provider: %s\n", providerName)
		summary += fmt.Sprintf("  Total Cost: $%.4f\n", cost.TotalCost)
		summary += fmt.Sprintf("  Total Tokens: %d\n", cost.TotalTokens)
		summary += fmt.Sprintf("  Total Requests: %d\n", cost.TotalRequests)
		summary += fmt.Sprintf("  Avg Cost per 1K Tokens: $%.4f\n", c.avgCostPer1K(cost.TotalCost, cost.TotalTokens))
		summary += "\n"

		totalCost += cost.TotalCost
		totalTokens += cost.TotalTokens
		totalRequests += cost.TotalRequests
	}

	summary += "Grand Totals\n"
	summary += "============\n"
	summary += fmt.Sprintf("Total Cost: $%.4f\n", totalCost)
	summary += fmt.Sprintf("Total Tokens: %d\n", totalTokens)
	summary += fmt.Sprintf("Total Requests: %d\n", totalRequests)
	summary += fmt.Sprintf("Avg Cost per 1K Tokens: $%.4f\n", c.avgCostPer1K(totalCost, totalTokens))

	return summary
}

func (c *CostTracker) avgCostPer1K(totalCost float64, totalTokens int64) float64 {
	if totalTokens == 0 {
		return 0
	}
	return (totalCost / float64(totalTokens)) * 1000
}
