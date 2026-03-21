package knowledge

import (
	"fmt"
	"time"
)

type GraphNode struct {
	ID         string
	Type       string
	Name       string
	Labels     []string
	Properties map[string]any
	Embedding  []float64
}

type GraphEdge struct {
	ID         string
	From       string
	To         string
	Type       string
	Weight     float64
	Properties map[string]any
}

type KnowledgeGraph struct {
	store GraphStore
}

type GraphStore interface {
	CreateNode(node *GraphNode) error
	CreateEdge(edge *GraphEdge) error
	GetNode(id string) (*GraphNode, error)
	GetNeighbors(id string, edgeType string, depth int) ([]*GraphNode, error)
	Query(pattern *GraphQuery) ([]*GraphNode, error)
	QueryEdges(pattern *EdgeQuery) ([]*GraphEdge, error)
	DeleteNode(id string) error
	DeleteEdge(id string) error
}

type GraphQuery struct {
	NodeTypes  []string
	Labels     []string
	Properties map[string]any
	EdgeTypes  []string
	MaxDepth   int
}

type EdgeQuery struct {
	EdgeTypes  []string
	FromNodeID string
	ToNodeID   string
	MinWeight  float64
	MaxWeight  float64
	Properties map[string]any
}

func NewKnowledgeGraph(store GraphStore) *KnowledgeGraph {
	return &KnowledgeGraph{store: store}
}

func (g *KnowledgeGraph) AddCodeEntity(
	file string,
	symbol string,
	symbolType string,
	content string,
	embedding []float64,
) error {
	id := fmt.Sprintf("%s#%s", file, symbol)
	return g.store.CreateNode(&GraphNode{
		ID:     id,
		Type:   "CodeEntity",
		Name:   symbol,
		Labels: []string{symbolType, file},
		Properties: map[string]any{
			"file":    file,
			"symbol":  symbol,
			"type":    symbolType,
			"content": content,
		},
		Embedding: embedding,
	})
}

func (g *KnowledgeGraph) RecordDecision(
	summary string,
	rationale string,
	agentID string,
	affectedEntities []string,
) error {
	node := &GraphNode{
		ID:   generateID(),
		Type: "Decision",
		Name: summary,
		Properties: map[string]any{
			"rationale": rationale,
			"agent":     agentID,
			"timestamp": time.Now(),
		},
	}

	if err := g.store.CreateNode(node); err != nil {
		return err
	}

	for _, entity := range affectedEntities {
		if err := g.store.CreateEdge(&GraphEdge{
			ID:   generateID(),
			From: node.ID,
			To:   entity,
			Type: "AFFECTS",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (g *KnowledgeGraph) FindRelated(
	query string,
	embedding []float64,
	maxDepth int,
) ([]*GraphNode, error) {
	nodes, err := g.store.Query(&GraphQuery{
		EdgeTypes: []string{"SIMILAR_TO"},
		MaxDepth:  maxDepth,
	})
	if err != nil {
		return nil, err
	}

	var related []*GraphNode
	seen := make(map[string]bool)

	for _, node := range nodes {
		neighbors, err := g.store.GetNeighbors(node.ID, "", maxDepth)
		if err != nil {
			continue
		}
		for _, n := range neighbors {
			if !seen[n.ID] {
				seen[n.ID] = true
				related = append(related, n)
			}
		}
	}

	return related, nil
}

func (g *KnowledgeGraph) AddEdge(fromID, toID, edgeType string, weight float64) error {
	return g.store.CreateEdge(&GraphEdge{
		ID:     generateID(),
		From:   fromID,
		To:     toID,
		Type:   edgeType,
		Weight: weight,
	})
}

func (g *KnowledgeGraph) GetNode(id string) (*GraphNode, error) {
	return g.store.GetNode(id)
}

func (g *KnowledgeGraph) GetNeighbors(id string, edgeType string, depth int) ([]*GraphNode, error) {
	return g.store.GetNeighbors(id, edgeType, depth)
}

func (g *KnowledgeGraph) Query(pattern *GraphQuery) ([]*GraphNode, error) {
	return g.store.Query(pattern)
}

func (g *KnowledgeGraph) QueryEdges(pattern *EdgeQuery) ([]*GraphEdge, error) {
	return g.store.QueryEdges(pattern)
}

func (g *KnowledgeGraph) DeleteNode(id string) error {
	return g.store.DeleteNode(id)
}

func (g *KnowledgeGraph) DeleteEdge(id string) error {
	return g.store.DeleteEdge(id)
}

func (g *KnowledgeGraph) AddConcept(
	name string,
	description string,
	labels []string,
	embedding []float64,
) error {
	return g.store.CreateNode(&GraphNode{
		ID:     generateID(),
		Type:   "Concept",
		Name:   name,
		Labels: labels,
		Properties: map[string]any{
			"description": description,
		},
		Embedding: embedding,
	})
}

func (g *KnowledgeGraph) AddAgent(
	agentID string,
	role string,
	capabilities []string,
) error {
	return g.store.CreateNode(&GraphNode{
		ID:     agentID,
		Type:   "Agent",
		Name:   role,
		Labels: capabilities,
		Properties: map[string]any{
			"role": role,
		},
	})
}

func (g *KnowledgeGraph) AddSession(
	sessionID string,
	projectID string,
	startTime time.Time,
) error {
	return g.store.CreateNode(&GraphNode{
		ID:   sessionID,
		Type: "Session",
		Name: sessionID,
		Properties: map[string]any{
			"project_id": projectID,
			"start_time": startTime,
		},
	})
}

func generateID() string {
	return fmt.Sprintf("node_%d", time.Now().UnixNano())
}
