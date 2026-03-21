package session

import (
	"context"
	"testing"
)

func TestNewCompressionManager(t *testing.T) {
	cm := NewCompressionManager()

	if cm == nil {
		t.Fatal("NewCompressionManager returned nil")
	}

	if len(cm.strategies) != 4 {
		t.Errorf("Expected 4 strategies, got %d", len(cm.strategies))
	}
}

func TestCompressionManager_Compress_NoCompressionNeeded(t *testing.T) {
	cm := NewCompressionManager()

	shortText := "Hello"
	compressed, err := cm.Compress(context.Background(), shortText, 100)

	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if compressed != shortText {
		t.Errorf("Expected '%s', got '%s'", shortText, compressed)
	}
}

func TestCompressionManager_Compress_WithCompression(t *testing.T) {
	cm := NewCompressionManager()

	longText := "This is a very long text that should be compressed because it has many words and exceeds the target token count significantly so compression should occur"
	compressed, err := cm.Compress(context.Background(), longText, 5)

	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if len(compressed) >= len(longText) {
		t.Errorf("Expected compressed text to be shorter than original")
	}
}

func TestCompressionManager_SelectStrategy(t *testing.T) {
	cm := NewCompressionManager()

	tests := []struct {
		ratio    float64
		expected CompressionStrategy
	}{
		{0.1, CompressKeyFacts},
		{0.2, CompressCodeAbstraction},
		{0.4, CompressSemanticSummary},
		{0.7, CompressDiffBased},
	}

	for _, tt := range tests {
		strategy := cm.selectStrategy(tt.ratio)
		if strategy != tt.expected {
			t.Errorf("Ratio %.2f: expected %s, got %s", tt.ratio, tt.expected, strategy)
		}
	}
}

func TestCompressionManager_SemanticSummaryCompression(t *testing.T) {
	cm := NewCompressionManager()

	text := "Line one\nLine two\nLine three\nLine four\nLine five"
	compressed, err := cm.semanticSummaryCompression(context.Background(), text, 10)

	if err != nil {
		t.Fatalf("semanticSummaryCompression failed: %v", err)
	}

	if compressed == "" {
		t.Error("Expected non-empty compressed text")
	}

	if estimateTokens(compressed) > 10 {
		t.Errorf("Expected <= 10 tokens, got %d", estimateTokens(compressed))
	}
}

func TestCompressionManager_KeyFactsCompression(t *testing.T) {
	cm := NewCompressionManager()

	text := "The system is: important\nConfiguration: value\nStatus: active\nMode: production"
	compressed, err := cm.keyFactsCompression(context.Background(), text, 20)

	if err != nil {
		t.Fatalf("keyFactsCompression failed: %v", err)
	}

	if compressed == "" {
		t.Error("Expected non-empty compressed text")
	}
}

func TestCompressionManager_CodeAbstractionCompression(t *testing.T) {
	cm := NewCompressionManager()

	text := "func test() {}\nvar x = 1\ntype MyStruct struct{}\nconst PI = 3.14"
	compressed, err := cm.codeAbstractionCompression(context.Background(), text, 20)

	if err != nil {
		t.Fatalf("codeAbstractionCompression failed: %v", err)
	}

	if compressed == "" {
		t.Error("Expected non-empty compressed text")
	}
}

func TestCompressionManager_DiffBasedCompression(t *testing.T) {
	cm := NewCompressionManager()

	text := "+ added line\n- removed line\nmodified: file.txt\nadded: newfile.txt"
	compressed, err := cm.diffBasedCompression(context.Background(), text, 20)

	if err != nil {
		t.Fatalf("diffBasedCompression failed: %v", err)
	}

	if compressed == "" {
		t.Error("Expected non-empty compressed text")
	}
}

func TestCompressionManager_RegisterStrategy(t *testing.T) {
	cm := NewCompressionManager()

	customStrategy := func(ctx context.Context, content string, targetTokens int) (string, error) {
		return "custom", nil
	}

	cm.RegisterStrategy("custom", customStrategy)

	fn, ok := cm.GetStrategy("custom")
	if !ok {
		t.Error("Expected strategy to be registered")
	}

	if fn == nil {
		t.Error("Expected non-nil strategy function")
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"Hello", 1},
		{"Hello world", 2},
		{"Hello world test", 3},
	}

	for _, tt := range tests {
		tokens := estimateTokens(tt.text)
		if tokens != tt.expected {
			t.Errorf("Text '%s': expected %d tokens, got %d", tt.text, tt.expected, tokens)
		}
	}
}

func TestNewHierarchicalSummarizer(t *testing.T) {
	cm := NewCompressionManager()
	hs := NewHierarchicalSummarizer(cm)

	if hs == nil {
		t.Fatal("NewHierarchicalSummarizer returned nil")
	}

	if hs.compressor != cm {
		t.Error("Expected compressor to be set")
	}

	if len(hs.levels) != 4 {
		t.Errorf("Expected 4 levels, got %d", len(hs.levels))
	}
}

func TestHierarchicalSummarizer_Summarize(t *testing.T) {
	cm := NewCompressionManager()
	hs := NewHierarchicalSummarizer(cm)

	content := "This is a long text that will be summarized at multiple levels of compression. " +
		"It contains various important pieces of information that should be preserved across " +
		"different compression ratios while maintaining the essential meaning and context."

	summary, err := hs.Summarize(context.Background(), content)
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	if summary.ID == "" {
		t.Error("Summary ID should not be empty")
	}

	if summary.Level0Full != content {
		t.Error("Level0Full should match original content")
	}

	if summary.Level1Detail == "" {
		t.Error("Level1Detail should not be empty")
	}
}

func TestHierarchicalSummarizer_GetLevel(t *testing.T) {
	cm := NewCompressionManager()
	hs := NewHierarchicalSummarizer(cm)

	content := "Long text content for testing hierarchical summarization"
	summary, _ := hs.Summarize(context.Background(), content)

	level := hs.GetLevel(summary, 1)
	if level == "" {
		t.Error("GetLevel should return non-empty content")
	}

	level = hs.GetLevel(summary, 10000)
	if level == "" {
		t.Error("GetLevel with large budget should return content")
	}
}
