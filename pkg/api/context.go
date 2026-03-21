package api

import (
	"context"
	"time"
)

type ContextLayer string

const (
	ContextLayerSession   ContextLayer = "session"
	ContextLayerAgent     ContextLayer = "agent"
	ContextLayerProject   ContextLayer = "project"
	ContextLayerKnowledge ContextLayer = "knowledge"
)

type ContextStore interface {
	Get(ctx context.Context, key string, layer ContextLayer) (*ContextEntry, error)
	Set(ctx context.Context, entry *ContextEntry) error
	Delete(ctx context.Context, key string, layer ContextLayer) error
	Query(ctx context.Context, q *ContextQuery) ([]*ContextEntry, error)
	SemanticSearch(ctx context.Context, query string, opts *SearchOptions) ([]*ContextEntry, error)
	Subscribe(pattern string, callback ContextCallback) (Unsubscribe, error)
	Export(ctx context.Context, layer ContextLayer) (*ContextSnapshot, error)
}

type ContextEntry struct {
	ID         string
	Key        string
	Layer      ContextLayer
	Content    any
	Embedding  []float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	AccessedAt time.Time
	TTL        time.Duration
	Metadata   map[string]any
}

type ContextQuery struct {
	Layer        ContextLayer
	KeyPattern   string
	Tags         []string
	CreatedAfter time.Time
	Limit        int
}

type SearchOptions struct {
	TopK     int
	MinScore float64
	Filters  map[string]string
	Layer    ContextLayer
}

type ContextCallback func(event *ContextEvent)

type Unsubscribe func() error

type ContextEvent struct {
	Type  string
	Key   string
	Layer ContextLayer
	Entry *ContextEntry
	Delta *ContextDelta
}

type ContextDelta struct {
	Field    string
	OldValue any
	NewValue any
}

type ContextSnapshot struct {
	Layer      ContextLayer
	Entries    []*ContextEntry
	Compressed []byte
	Checksum   string
	TakenAt    time.Time
}
