package knowledge

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type RetrievalConfig struct {
	MaxDepth       int
	MaxNodes       int
	MinSimilarity  float64
	BoostDecisions float64
	BoostCode      float64
	BoostConcepts  float64
}

type GraphRAGResult struct {
	Nodes       []*RetrievedNode
	Edges       []*RetrievedEdge
	Query       string
	Context     string
	TotalTokens int
}

type RetrievedNode struct {
	Node          *GraphNode
	Score         float64
	Relevance     string
	PathFromQuery []string
}

type RetrievedEdge struct {
	Edge  *GraphEdge
	Score float64
}

type GraphRAG struct {
	graph  *KnowledgeGraph
	config *RetrievalConfig
}

func NewGraphRAG(graph *KnowledgeGraph, config *RetrievalConfig) *GraphRAG {
	if config == nil {
		config = &RetrievalConfig{
			MaxDepth:       3,
			MaxNodes:       50,
			MinSimilarity:  0.1,
			BoostDecisions: 1.5,
			BoostCode:      1.2,
			BoostConcepts:  1.3,
		}
	}
	return &GraphRAG{
		graph:  graph,
		config: config,
	}
}

func (rag *GraphRAG) Retrieve(
	query string,
	queryEmbedding []float64,
) (*GraphRAGResult, error) {
	startNodes, err := rag.findStartNodes(queryEmbedding)
	if err != nil {
		return nil, err
	}

	var retrieved []*RetrievedNode
	seen := make(map[string]bool)

	for _, node := range startNodes {
		if seen[node.ID] {
			continue
		}

		result, err := rag.exploreFromNode(node, queryEmbedding)
		if err != nil {
			continue
		}

		for _, rnode := range result.Nodes {
			if !seen[rnode.Node.ID] {
				seen[rnode.Node.ID] = true
				retrieved = append(retrieved, rnode)
			}
		}

		if len(retrieved) >= rag.config.MaxNodes {
			break
		}
	}

	sort.Slice(retrieved, func(i, j int) bool {
		return retrieved[i].Score > retrieved[j].Score
	})

	if len(retrieved) > rag.config.MaxNodes {
		retrieved = retrieved[:rag.config.MaxNodes]
	}

	edges, err := rag.getRelevantEdges(retrieved)
	if err != nil {
		return nil, err
	}

	context := rag.buildContext(retrieved, edges)

	return &GraphRAGResult{
		Nodes:       retrieved,
		Edges:       edges,
		Query:       query,
		Context:     context,
		TotalTokens: rag.estimateTokens(context),
	}, nil
}

func (rag *GraphRAG) findStartNodes(queryEmbedding []float64) ([]*GraphNode, error) {
	allNodes, err := rag.graph.Query(&GraphQuery{})
	if err != nil {
		return nil, err
	}

	type scoredNode struct {
		node  *GraphNode
		score float64
	}

	scores := make([]scoredNode, 0, len(allNodes))

	for _, node := range allNodes {
		if len(node.Embedding) == 0 || len(queryEmbedding) == 0 {
			continue
		}

		similarity := rag.cosineSimilarity(node.Embedding, queryEmbedding)
		if similarity < rag.config.MinSimilarity {
			continue
		}

		boost := rag.getNodeTypeBoost(node.Type)
		scores = append(scores, scoredNode{
			node:  node,
			score: similarity * boost,
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	result := make([]*GraphNode, 0, len(scores))
	for i := 0; i < len(scores) && i < 10; i++ {
		result = append(result, scores[i].node)
	}

	return result, nil
}

func (rag *GraphRAG) exploreFromNode(
	startNode *GraphNode,
	queryEmbedding []float64,
) (*GraphRAGResult, error) {
	result := &GraphRAGResult{
		Nodes: make([]*RetrievedNode, 0),
	}

	visited := make(map[string]bool)
	queue := []*explorationItem{{node: startNode, depth: 0, path: []string{}}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if visited[item.node.ID] {
			continue
		}
		visited[item.node.ID] = true

		if item.depth > rag.config.MaxDepth {
			continue
		}

		similarity := rag.cosineSimilarity(item.node.Embedding, queryEmbedding)
		boost := rag.getNodeTypeBoost(item.node.Type)
		depthPenalty := 1.0 / (1.0 + float64(item.depth)*0.3)

		score := similarity * boost * depthPenalty

		result.Nodes = append(result.Nodes, &RetrievedNode{
			Node:          item.node,
			Score:         score,
			Relevance:     rag.getRelevanceLabel(score),
			PathFromQuery: item.path,
		})

		neighbors, err := rag.graph.GetNeighbors(item.node.ID, "", 1)
		if err != nil {
			continue
		}

		for _, neighbor := range neighbors {
			if !visited[neighbor.ID] {
				newPath := make([]string, len(item.path)+1)
				copy(newPath, item.path)
				newPath[len(item.path)] = neighbor.ID

				queue = append(queue, &explorationItem{
					node:  neighbor,
					depth: item.depth + 1,
					path:  newPath,
				})
			}
		}
	}

	return result, nil
}

func (rag *GraphRAG) getRelevantEdges(retrieved []*RetrievedNode) ([]*RetrievedEdge, error) {
	nodeIDs := make(map[string]bool)
	for _, rnode := range retrieved {
		nodeIDs[rnode.Node.ID] = true
	}

	allEdges, err := rag.graph.QueryEdges(&EdgeQuery{})
	if err != nil {
		return nil, err
	}

	var relevant []*RetrievedEdge

	for _, edge := range allEdges {
		if nodeIDs[edge.From] && nodeIDs[edge.To] {
			relevant = append(relevant, &RetrievedEdge{
				Edge:  edge,
				Score: edge.Weight,
			})
		}
	}

	return relevant, nil
}

func (rag *GraphRAG) buildContext(nodes []*RetrievedNode, edges []*RetrievedEdge) string {
	context := "=== Graph Context ===\n\n"

	context += "Relevant Entities:\n"
	for _, rnode := range nodes {
		node := rnode.Node
		context += fmt.Sprintf("- [%s] %s (Score: %.2f)\n", node.Type, node.Name, rnode.Score)

		if node.Type == "CodeEntity" {
			if file, ok := node.Properties["file"].(string); ok {
				context += fmt.Sprintf("  File: %s\n", file)
			}
			if content, ok := node.Properties["content"].(string); ok {
				if len(content) > 200 {
					context += fmt.Sprintf("  Content: %s...\n", content[:200])
				} else {
					context += fmt.Sprintf("  Content: %s\n", content)
				}
			}
		} else if node.Type == "Decision" {
			if rationale, ok := node.Properties["rationale"].(string); ok {
				if len(rationale) > 200 {
					context += fmt.Sprintf("  Rationale: %s...\n", rationale[:200])
				} else {
					context += fmt.Sprintf("  Rationale: %s\n", rationale)
				}
			}
		}
	}

	if len(edges) > 0 {
		context += "\nRelationships:\n"
		for _, redge := range edges {
			edge := redge.Edge
			context += fmt.Sprintf("- %s --%s--> %s (Weight: %.2f)\n",
				edge.From, edge.Type, edge.To, edge.Weight)
		}
	}

	return context
}

func (rag *GraphRAG) cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (rag *GraphRAG) getNodeTypeBoost(nodeType string) float64 {
	switch nodeType {
	case "Decision":
		return rag.config.BoostDecisions
	case "CodeEntity":
		return rag.config.BoostCode
	case "Concept":
		return rag.config.BoostConcepts
	default:
		return 1.0
	}
}

func (rag *GraphRAG) getRelevanceLabel(score float64) string {
	switch {
	case score >= 0.8:
		return "high"
	case score >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

func (rag *GraphRAG) estimateTokens(text string) int {
	return len(text) / 4
}

type explorationItem struct {
	node  *GraphNode
	depth int
	path  []string
}

func (rag *GraphRAG) RetrieveWithFilter(
	query string,
	queryEmbedding []float64,
	filter *GraphQuery,
) (*GraphRAGResult, error) {
	if filter != nil {
		filteredNodes, err := rag.graph.Query(filter)
		if err != nil {
			return nil, err
		}

		if len(filteredNodes) == 0 {
			return &GraphRAGResult{
				Nodes:   make([]*RetrievedNode, 0),
				Edges:   make([]*RetrievedEdge, 0),
				Query:   query,
				Context: "No matching entities found in the knowledge graph.",
			}, nil
		}
	}

	return rag.Retrieve(query, queryEmbedding)
}

func (rag *GraphRAG) GetCodeContext(file string, symbol string) (*GraphRAGResult, error) {
	nodeID := fmt.Sprintf("%s#%s", file, symbol)
	node, err := rag.graph.GetNode(nodeID)
	if err != nil {
		return nil, err
	}

	queryEmbedding := node.Embedding
	query := fmt.Sprintf("Code context for %s in %s", symbol, file)

	return rag.Retrieve(query, queryEmbedding)
}

func (rag *GraphRAG) GetDecisionHistory(entityID string) ([]*RetrievedNode, error) {
	edges, err := rag.graph.QueryEdges(&EdgeQuery{
		ToNodeID:  entityID,
		EdgeTypes: []string{"AFFECTS"},
	})
	if err != nil {
		return nil, err
	}

	var decisions []*RetrievedNode
	for _, edge := range edges {
		node, err := rag.graph.GetNode(edge.From)
		if err != nil {
			continue
		}
		if node.Type == "Decision" {
			decisions = append(decisions, &RetrievedNode{
				Node:      node,
				Score:     1.0,
				Relevance: "direct",
			})
		}
	}

	sort.Slice(decisions, func(i, j int) bool {
		timestampI := decisions[i].Node.Properties["timestamp"]
		timestampJ := decisions[j].Node.Properties["timestamp"]

		if timestampI == nil && timestampJ == nil {
			return false
		}
		if timestampI == nil {
			return false
		}
		if timestampJ == nil {
			return true
		}

		return timestampI.(time.Time).After(timestampJ.(time.Time))
	})

	return decisions, nil
}
