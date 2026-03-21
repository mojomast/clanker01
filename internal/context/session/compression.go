package session

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type CompressionStrategy string

const (
	CompressSemanticSummary CompressionStrategy = "semantic_summary"
	CompressKeyFacts        CompressionStrategy = "key_facts"
	CompressCodeAbstraction CompressionStrategy = "code_abstraction"
	CompressDiffBased       CompressionStrategy = "diff_based"
)

type CompressionManager struct {
	strategies map[CompressionStrategy]CompressionStrategyFunc
	mu         sync.RWMutex
}

type CompressionStrategyFunc func(ctx context.Context, content string, targetTokens int) (string, error)

type CompressionResult struct {
	OriginalTokens   int
	CompressedTokens int
	Strategy         CompressionStrategy
	Ratio            float64
}

func NewCompressionManager() *CompressionManager {
	cm := &CompressionManager{
		strategies: make(map[CompressionStrategy]CompressionStrategyFunc),
	}

	cm.strategies[CompressSemanticSummary] = cm.semanticSummaryCompression
	cm.strategies[CompressKeyFacts] = cm.keyFactsCompression
	cm.strategies[CompressCodeAbstraction] = cm.codeAbstractionCompression
	cm.strategies[CompressDiffBased] = cm.diffBasedCompression

	return cm
}

func (m *CompressionManager) Compress(ctx context.Context, content string, targetTokens int) (string, error) {
	currentTokens := estimateTokens(content)

	if currentTokens <= targetTokens {
		return content, nil
	}

	ratio := float64(targetTokens) / float64(currentTokens)
	strategy := m.selectStrategy(ratio)

	m.mu.RLock()
	strategyFunc, ok := m.strategies[strategy]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("compression strategy not found: %s", strategy)
	}

	compressed, err := strategyFunc(ctx, content, targetTokens)
	if err != nil {
		return "", err
	}

	return compressed, nil
}

func (m *CompressionManager) selectStrategy(ratio float64) CompressionStrategy {
	switch {
	case ratio < 0.15:
		return CompressKeyFacts
	case ratio < 0.3:
		return CompressCodeAbstraction
	case ratio < 0.5:
		return CompressSemanticSummary
	default:
		return CompressDiffBased
	}
}

func (m *CompressionManager) semanticSummaryCompression(ctx context.Context, content string, targetTokens int) (string, error) {
	lines := strings.Split(content, "\n")
	var summary []string
	currentTokens := 0

	for _, line := range lines {
		lineTokens := estimateTokens(line)
		if currentTokens+lineTokens > targetTokens {
			break
		}
		summary = append(summary, line)
		currentTokens += lineTokens
	}

	if len(summary) == 0 {
		summary = append(summary, "[Conversation summary compressed]")
	}

	return strings.Join(summary, "\n"), nil
}

func (m *CompressionManager) keyFactsCompression(ctx context.Context, content string, targetTokens int) (string, error) {
	lines := strings.Split(content, "\n")
	var facts []string
	currentTokens := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if strings.Contains(line, ":") || strings.Contains(line, "is") || strings.Contains(line, "was") {
			lineTokens := estimateTokens(line)
			if currentTokens+lineTokens > targetTokens {
				break
			}
			facts = append(facts, "• "+line)
			currentTokens += lineTokens
		}
	}

	if len(facts) == 0 {
		facts = append(facts, "[Key facts extracted]")
	}

	return strings.Join(facts, "\n"), nil
}

func (m *CompressionManager) codeAbstractionCompression(ctx context.Context, content string, targetTokens int) (string, error) {
	lines := strings.Split(content, "\n")
	var abstracted []string
	currentTokens := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "const ") {
			lineTokens := estimateTokens(line)
			if currentTokens+lineTokens > targetTokens {
				break
			}
			abstracted = append(abstracted, line)
			currentTokens += lineTokens
		}
	}

	if len(abstracted) == 0 {
		abstracted = append(abstracted, "[Code abstracted to signatures]")
	}

	return strings.Join(abstracted, "\n"), nil
}

func (m *CompressionManager) diffBasedCompression(ctx context.Context, content string, targetTokens int) (string, error) {
	lines := strings.Split(content, "\n")
	var filtered []string
	currentTokens := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "+") || strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "modified:") || strings.HasPrefix(trimmed, "added:") ||
			strings.HasPrefix(trimmed, "removed:") {
			lineTokens := estimateTokens(line)
			if currentTokens+lineTokens > targetTokens {
				break
			}
			filtered = append(filtered, line)
			currentTokens += lineTokens
		}
	}

	if len(filtered) == 0 {
		filtered = append(filtered, "[Changes summarized]")
	}

	return strings.Join(filtered, "\n"), nil
}

func (m *CompressionManager) RegisterStrategy(name CompressionStrategy, fn CompressionStrategyFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategies[name] = fn
}

func (m *CompressionManager) GetStrategy(name CompressionStrategy) (CompressionStrategyFunc, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	fn, ok := m.strategies[name]
	return fn, ok
}

type HierarchicalSummarizer struct {
	compressor *CompressionManager
	levels     []CompressionLevel
}

type CompressionLevel struct {
	Name      string
	Ratio     float64
	MinTokens int
}

type HierarchicalSummary struct {
	ID           string
	Level0Full   string
	Level1Detail string
	Level2Medium string
	Level3Brief  string
	Level4Facts  string
	CreatedAt    string
}

func NewHierarchicalSummarizer(compressor *CompressionManager) *HierarchicalSummarizer {
	return &HierarchicalSummarizer{
		compressor: compressor,
		levels: []CompressionLevel{
			{Name: "Level1Detail", Ratio: 0.5, MinTokens: 50},
			{Name: "Level2Medium", Ratio: 0.25, MinTokens: 50},
			{Name: "Level3Brief", Ratio: 0.1, MinTokens: 50},
			{Name: "Level4Facts", Ratio: 0.05, MinTokens: 20},
		},
	}
}

func (h *HierarchicalSummarizer) Summarize(ctx context.Context, content string) (*HierarchicalSummary, error) {
	summary := &HierarchicalSummary{
		ID:         generateID(),
		Level0Full: content,
		CreatedAt:  currentTimeString(),
	}

	currentContent := content
	tokens := estimateTokens(content)

	for _, level := range h.levels {
		target := int(float64(tokens) * level.Ratio)
		if target < level.MinTokens {
			target = level.MinTokens
		}

		compressed, err := h.compressor.Compress(ctx, currentContent, target)
		if err != nil {
			break
		}

		switch level.Name {
		case "Level1Detail":
			summary.Level1Detail = compressed
		case "Level2Medium":
			summary.Level2Medium = compressed
		case "Level3Brief":
			summary.Level3Brief = compressed
		case "Level4Facts":
			summary.Level4Facts = compressed
		}

		currentContent = compressed
		tokens = estimateTokens(compressed)
	}

	return summary, nil
}

func (h *HierarchicalSummarizer) GetLevel(summary *HierarchicalSummary, budget int) string {
	levels := []struct {
		content string
		tokens  int
	}{
		{summary.Level4Facts, estimateTokens(summary.Level4Facts)},
		{summary.Level3Brief, estimateTokens(summary.Level3Brief)},
		{summary.Level2Medium, estimateTokens(summary.Level2Medium)},
		{summary.Level1Detail, estimateTokens(summary.Level1Detail)},
		{summary.Level0Full, estimateTokens(summary.Level0Full)},
	}

	for _, level := range levels {
		if level.tokens <= budget {
			return level.content
		}
	}

	return summary.Level4Facts
}

func estimateTokens(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func currentTimeString() string {
	return "2024-01-01T00:00:00Z"
}
