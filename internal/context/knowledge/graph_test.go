package knowledge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryGraphStore_CreateNode(t *testing.T) {
	store := NewInMemoryGraphStore()

	node := &GraphNode{
		ID:   "test-1",
		Type: "Test",
		Name: "Test Node",
		Properties: map[string]any{
			"key": "value",
		},
	}

	err := store.CreateNode(node)
	require.NoError(t, err)

	retrieved, err := store.GetNode("test-1")
	require.NoError(t, err)

	assert.Equal(t, "test-1", retrieved.ID)
	assert.Equal(t, "Test", retrieved.Type)
	assert.Equal(t, "Test Node", retrieved.Name)
	assert.Equal(t, "value", retrieved.Properties["key"])
}

func TestInMemoryGraphStore_CreateEdge(t *testing.T) {
	store := NewInMemoryGraphStore()

	node1 := &GraphNode{ID: "node-1", Type: "Test", Name: "Node 1"}
	node2 := &GraphNode{ID: "node-2", Type: "Test", Name: "Node 2"}

	err := store.CreateNode(node1)
	require.NoError(t, err)

	err = store.CreateNode(node2)
	require.NoError(t, err)

	edge := &GraphEdge{
		ID:     "edge-1",
		From:   "node-1",
		To:     "node-2",
		Type:   "CONNECTS",
		Weight: 1.0,
	}

	err = store.CreateEdge(edge)
	require.NoError(t, err)

	neighbors, err := store.GetNeighbors("node-1", "", 1)
	require.NoError(t, err)
	assert.Len(t, neighbors, 1)
	assert.Equal(t, "node-2", neighbors[0].ID)
}

func TestInMemoryGraphStore_GetNeighbors(t *testing.T) {
	store := NewInMemoryGraphStore()

	for i := 1; i <= 5; i++ {
		node := &GraphNode{
			ID:   string(rune('0' + i)),
			Type: "Test",
			Name: string(rune('0' + i)),
		}
		store.CreateNode(node)
	}

	store.CreateEdge(&GraphEdge{ID: "e1", From: "1", To: "2", Type: "CONNECTS"})
	store.CreateEdge(&GraphEdge{ID: "e2", From: "1", To: "3", Type: "CONNECTS"})
	store.CreateEdge(&GraphEdge{ID: "e3", From: "2", To: "4", Type: "CONNECTS"})
	store.CreateEdge(&GraphEdge{ID: "e4", From: "3", To: "5", Type: "CONNECTS"})

	neighbors, err := store.GetNeighbors("1", "", 1)
	require.NoError(t, err)
	assert.Len(t, neighbors, 2)

	neighbors, err = store.GetNeighbors("1", "", 2)
	require.NoError(t, err)
	assert.Len(t, neighbors, 4)

	neighbors, err = store.GetNeighbors("1", "CONNECTS", 1)
	require.NoError(t, err)
	assert.Len(t, neighbors, 2)
}

func TestInMemoryGraphStore_Query(t *testing.T) {
	store := NewInMemoryGraphStore()

	store.CreateNode(&GraphNode{ID: "1", Type: "CodeEntity", Name: "func1", Labels: []string{"function", "file1.go"}})
	store.CreateNode(&GraphNode{ID: "2", Type: "CodeEntity", Name: "func2", Labels: []string{"function", "file2.go"}})
	store.CreateNode(&GraphNode{ID: "3", Type: "Decision", Name: "Decision 1"})

	results, err := store.Query(&GraphQuery{NodeTypes: []string{"CodeEntity"}})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	results, err = store.Query(&GraphQuery{NodeTypes: []string{"Decision"}})
	require.NoError(t, err)
	assert.Len(t, results, 1)

	results, err = store.Query(&GraphQuery{Labels: []string{"function"}})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestInMemoryGraphStore_DeleteNode(t *testing.T) {
	store := NewInMemoryGraphStore()

	node1 := &GraphNode{ID: "1", Type: "Test", Name: "Node 1"}
	node2 := &GraphNode{ID: "2", Type: "Test", Name: "Node 2"}

	store.CreateNode(node1)
	store.CreateNode(node2)

	store.CreateEdge(&GraphEdge{ID: "e1", From: "1", To: "2", Type: "CONNECTS"})

	err := store.DeleteNode("1")
	require.NoError(t, err)

	_, err = store.GetNode("1")
	assert.Error(t, err)

	neighbors, err := store.GetNeighbors("2", "", 1)
	require.NoError(t, err)
	assert.Len(t, neighbors, 0)
}

func TestKnowledgeGraph_AddCodeEntity(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	err := graph.AddCodeEntity("file.go", "MyFunction", "function", "func MyFunction() {}", []float64{0.1, 0.2, 0.3})
	require.NoError(t, err)

	node, err := graph.GetNode("file.go#MyFunction")
	require.NoError(t, err)

	assert.Equal(t, "CodeEntity", node.Type)
	assert.Equal(t, "MyFunction", node.Name)
	assert.Contains(t, node.Labels, "function")
	assert.Contains(t, node.Labels, "file.go")
}

func TestKnowledgeGraph_RecordDecision(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	err := graph.AddCodeEntity("file.go", "MyFunction", "function", "func MyFunction() {}", nil)
	require.NoError(t, err)

	err = graph.RecordDecision(
		"Refactored MyFunction for performance",
		"Improved algorithm efficiency by 50%",
		"agent-1",
		[]string{"file.go#MyFunction"},
	)
	require.NoError(t, err)

	nodes, err := graph.Query(&GraphQuery{NodeTypes: []string{"Decision"}})
	require.NoError(t, err)
	assert.Len(t, nodes, 1)

	edges, err := graph.QueryEdges(&EdgeQuery{FromNodeID: nodes[0].ID})
	require.NoError(t, err)
	assert.Len(t, edges, 1)
	assert.Equal(t, "AFFECTS", edges[0].Type)
}

func TestKnowledgeGraph_AddEdge(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	store.CreateNode(&GraphNode{ID: "1", Type: "Test"})
	store.CreateNode(&GraphNode{ID: "2", Type: "Test"})

	err := graph.AddEdge("1", "2", "DEPENDS_ON", 1.0)
	require.NoError(t, err)

	neighbors, err := graph.GetNeighbors("1", "", 1)
	require.NoError(t, err)
	assert.Len(t, neighbors, 1)
	assert.Equal(t, "2", neighbors[0].ID)
}

func TestKnowledgeGraph_AddConcept(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	err := graph.AddConcept("Microservices", "Architecture pattern", []string{"architecture"}, []float64{0.5, 0.5, 0.5})
	require.NoError(t, err)

	nodes, err := graph.Query(&GraphQuery{NodeTypes: []string{"Concept"}})
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, "Microservices", nodes[0].Name)
}

func TestKnowledgeGraph_AddAgent(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	err := graph.AddAgent("agent-1", "coder", []string{"write", "read"})
	require.NoError(t, err)

	node, err := graph.GetNode("agent-1")
	require.NoError(t, err)

	assert.Equal(t, "Agent", node.Type)
	assert.Equal(t, "coder", node.Properties["role"])
}

func TestKnowledgeGraph_AddSession(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	now := time.Now()
	err := graph.AddSession("session-1", "project-1", now)
	require.NoError(t, err)

	node, err := graph.GetNode("session-1")
	require.NoError(t, err)

	assert.Equal(t, "Session", node.Type)
	assert.Equal(t, "project-1", node.Properties["project_id"])
}

func TestKnowledgeGraph_BFS(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	for i := 0; i < 6; i++ {
		id := string(rune('0' + i))
		store.CreateNode(&GraphNode{ID: id, Type: "Test", Name: id})
	}

	graph.AddEdge("0", "1", "CONNECTS", 1.0)
	graph.AddEdge("0", "2", "CONNECTS", 1.0)
	graph.AddEdge("1", "3", "CONNECTS", 1.0)
	graph.AddEdge("1", "4", "CONNECTS", 1.0)
	graph.AddEdge("2", "5", "CONNECTS", 1.0)

	result, err := graph.BFS("0", &TraversalOptions{MaxDepth: 2})
	require.NoError(t, err)

	assert.Len(t, result.Visited, 6)
	assert.Equal(t, 0.0, result.Distance["0"])
	assert.Equal(t, 1.0, result.Distance["1"])
	assert.Equal(t, 1.0, result.Distance["2"])
	assert.Equal(t, 2.0, result.Distance["3"])
}

func TestKnowledgeGraph_DFS(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	for i := 0; i < 4; i++ {
		id := string(rune('0' + i))
		store.CreateNode(&GraphNode{ID: id, Type: "Test", Name: id})
	}

	graph.AddEdge("0", "1", "CONNECTS", 1.0)
	graph.AddEdge("0", "2", "CONNECTS", 1.0)
	graph.AddEdge("1", "3", "CONNECTS", 1.0)

	result, err := graph.DFS("0", &TraversalOptions{MaxDepth: 3})
	require.NoError(t, err)

	assert.Len(t, result.Visited, 4)
}

func TestKnowledgeGraph_ShortestPath(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	for i := 0; i < 5; i++ {
		id := string(rune('0' + i))
		store.CreateNode(&GraphNode{ID: id, Type: "Test", Name: id})
	}

	graph.AddEdge("0", "1", "CONNECTS", 1.0)
	graph.AddEdge("1", "2", "CONNECTS", 1.0)
	graph.AddEdge("0", "3", "CONNECTS", 1.0)
	graph.AddEdge("3", "2", "CONNECTS", 1.0)

	result, err := graph.ShortestPath("0", "2")
	require.NoError(t, err)

	assert.NotNil(t, result)
	assert.Equal(t, 2.0, result.TotalCost)
	assert.Len(t, result.PathIDs, 3)
}

func TestKnowledgeGraph_Centrality(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	for i := 0; i < 5; i++ {
		id := string(rune('0' + i))
		store.CreateNode(&GraphNode{ID: id, Type: "Test", Name: id})
	}

	graph.AddEdge("0", "1", "CONNECTS", 1.0)
	graph.AddEdge("0", "2", "CONNECTS", 1.0)
	graph.AddEdge("0", "3", "CONNECTS", 1.0)
	graph.AddEdge("0", "4", "CONNECTS", 1.0)

	centrality, err := graph.Centrality("0")
	require.NoError(t, err)
	assert.Greater(t, centrality, 0.0)
}

func TestGraphRAG_Retrieve(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	rag := NewGraphRAG(graph, &RetrievalConfig{
		MaxDepth:      2,
		MaxNodes:      10,
		MinSimilarity: 0.1,
	})

	store.CreateNode(&GraphNode{
		ID:         "node-1",
		Type:       "Concept",
		Name:       "Database",
		Embedding:  []float64{1.0, 0.0, 0.0},
		Properties: map[string]any{"description": "A database"},
	})

	store.CreateNode(&GraphNode{
		ID:         "node-2",
		Type:       "CodeEntity",
		Name:       "Query",
		Embedding:  []float64{0.9, 0.1, 0.0},
		Properties: map[string]any{"content": "func Query() {}"},
	})

	graph.AddEdge("node-1", "node-2", "REFERENCES", 0.8)

	queryEmbedding := []float64{1.0, 0.1, 0.0}
	result, err := rag.Retrieve("database query", queryEmbedding)
	require.NoError(t, err)

	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Context)
	assert.Greater(t, len(result.Nodes), 0)
}

func TestGraphRAG_CosineSimilarity(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	rag := NewGraphRAG(graph, nil)

	a := []float64{1.0, 0.0, 0.0}
	b := []float64{1.0, 0.0, 0.0}

	sim := rag.cosineSimilarity(a, b)
	assert.InDelta(t, 1.0, sim, 0.001)

	b = []float64{0.0, 1.0, 0.0}
	sim = rag.cosineSimilarity(a, b)
	assert.InDelta(t, 0.0, sim, 0.001)

	b = []float64{0.707, 0.707, 0.0}
	sim = rag.cosineSimilarity(a, b)
	assert.InDelta(t, 0.707, sim, 0.01)
}

func TestGraphRAG_GetCodeContext(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	rag := NewGraphRAG(graph, nil)

	graph.AddCodeEntity("file.go", "MyFunction", "function", "func MyFunction() {}",
		[]float64{0.5, 0.5, 0.5})
	graph.AddCodeEntity("file.go", "OtherFunction", "function", "func OtherFunction() {}",
		[]float64{0.4, 0.6, 0.5})

	graph.AddEdge("file.go#MyFunction", "file.go#OtherFunction", "CALLS", 0.7)

	result, err := rag.GetCodeContext("file.go", "MyFunction")
	require.NoError(t, err)

	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Context)
}

func TestGraphRAG_GetDecisionHistory(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	rag := NewGraphRAG(graph, nil)

	graph.AddCodeEntity("file.go", "MyFunction", "function", "func MyFunction() {}", nil)

	now := time.Now()
	node1 := &GraphNode{
		ID:   "decision-1",
		Type: "Decision",
		Name: "First decision",
		Properties: map[string]any{
			"rationale": "Because",
			"timestamp": now.Add(-2 * time.Hour),
		},
	}
	store.CreateNode(node1)
	graph.AddEdge("decision-1", "file.go#MyFunction", "AFFECTS", 1.0)

	node2 := &GraphNode{
		ID:   "decision-2",
		Type: "Decision",
		Name: "Second decision",
		Properties: map[string]any{
			"rationale": "Another reason",
			"timestamp": now,
		},
	}
	store.CreateNode(node2)
	graph.AddEdge("decision-2", "file.go#MyFunction", "AFFECTS", 1.0)

	history, err := rag.GetDecisionHistory("file.go#MyFunction")
	require.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, "Second decision", history[0].Node.Name)
}

func TestGraphRAG_RetrieveWithFilter(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	rag := NewGraphRAG(graph, nil)

	store.CreateNode(&GraphNode{ID: "1", Type: "CodeEntity", Name: "Func1"})
	store.CreateNode(&GraphNode{ID: "2", Type: "Decision", Name: "Decision1"})

	filter := &GraphQuery{NodeTypes: []string{"CodeEntity"}}
	result, err := rag.RetrieveWithFilter("test", []float64{0.5}, filter)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGraphStore_Clear(t *testing.T) {
	store := NewInMemoryGraphStore()

	store.CreateNode(&GraphNode{ID: "1", Type: "Test"})
	store.CreateEdge(&GraphEdge{ID: "e1", From: "1", To: "1", Type: "SELF"})

	assert.Equal(t, 1, store.NodeCount())
	assert.Greater(t, store.EdgeCount(), 0)

	store.Clear()

	assert.Equal(t, 0, store.NodeCount())
	assert.Equal(t, 0, store.EdgeCount())
}

func TestGraphStore_NodeCount(t *testing.T) {
	store := NewInMemoryGraphStore()

	assert.Equal(t, 0, store.NodeCount())

	store.CreateNode(&GraphNode{ID: "1", Type: "Test"})
	store.CreateNode(&GraphNode{ID: "2", Type: "Test"})

	assert.Equal(t, 2, store.NodeCount())
}

func TestGraphStore_EdgeCount(t *testing.T) {
	store := NewInMemoryGraphStore()

	store.CreateNode(&GraphNode{ID: "1", Type: "Test"})
	store.CreateNode(&GraphNode{ID: "2", Type: "Test"})

	assert.Equal(t, 0, store.EdgeCount())

	store.CreateEdge(&GraphEdge{ID: "e1", From: "1", To: "2", Type: "CONNECTS"})

	assert.Equal(t, 1, store.EdgeCount())
}

func TestKnowledgeGraph_FindConnectedComponents(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	store.CreateNode(&GraphNode{ID: "1", Type: "Test"})
	store.CreateNode(&GraphNode{ID: "2", Type: "Test"})
	store.CreateNode(&GraphNode{ID: "3", Type: "Test"})

	graph.AddEdge("1", "2", "CONNECTS", 1.0)

	components := graph.FindConnectedComponents()
	assert.Len(t, components, 2)
}

func TestKnowledgeGraph_FindCycles(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	store.CreateNode(&GraphNode{ID: "1", Type: "Test"})
	store.CreateNode(&GraphNode{ID: "2", Type: "Test"})
	store.CreateNode(&GraphNode{ID: "3", Type: "Test"})

	graph.AddEdge("1", "2", "CONNECTS", 1.0)
	graph.AddEdge("2", "3", "CONNECTS", 1.0)
	graph.AddEdge("3", "1", "CONNECTS", 1.0)

	cycles := graph.FindCycles()
	assert.Greater(t, len(cycles), 0)
}

func TestKnowledgeGraph_FindHubNodes(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)

	for i := 1; i <= 6; i++ {
		id := string(rune('0' + i))
		store.CreateNode(&GraphNode{ID: id, Type: "CodeEntity", Name: id})
	}

	graph.AddEdge("1", "2", "DEPENDS", 1.0)
	graph.AddEdge("1", "3", "DEPENDS", 1.0)
	graph.AddEdge("1", "4", "DEPENDS", 1.0)
	graph.AddEdge("2", "5", "DEPENDS", 1.0)

	hubs, err := graph.FindHubNodes("CodeEntity", 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(hubs), 3)
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "node_")
}

func TestGraphRAG_BuildContext(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	rag := NewGraphRAG(graph, nil)

	nodes := []*RetrievedNode{
		{
			Node: &GraphNode{
				ID:   "1",
				Type: "CodeEntity",
				Name: "Func1",
				Properties: map[string]any{
					"file":    "file.go",
					"content": "func Func1() {}",
				},
			},
			Score: 0.9,
		},
		{
			Node: &GraphNode{
				ID:   "2",
				Type: "Decision",
				Name: "Decision 1",
				Properties: map[string]any{
					"rationale": "Because I said so",
				},
			},
			Score: 0.8,
		},
	}

	edges := []*RetrievedEdge{
		{
			Edge:  &GraphEdge{ID: "e1", From: "1", To: "2", Type: "AFFECTS", Weight: 0.5},
			Score: 0.5,
		},
	}

	context := rag.buildContext(nodes, edges)
	assert.Contains(t, context, "Func1")
	assert.Contains(t, context, "Decision 1")
	assert.Contains(t, context, "AFFECTS")
}

func TestGraphRAG_EstimateTokens(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	rag := NewGraphRAG(graph, nil)

	text := "This is a test string with some words."
	tokens := rag.estimateTokens(text)

	assert.Greater(t, tokens, 0)
	assert.Less(t, tokens, len(text))
}

func TestGraphRAG_GetNodeTypeBoost(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	config := &RetrievalConfig{
		BoostDecisions: 2.0,
		BoostCode:      1.5,
		BoostConcepts:  1.3,
	}
	rag := NewGraphRAG(graph, config)

	boost := rag.getNodeTypeBoost("Decision")
	assert.Equal(t, 2.0, boost)

	boost = rag.getNodeTypeBoost("CodeEntity")
	assert.Equal(t, 1.5, boost)

	boost = rag.getNodeTypeBoost("Concept")
	assert.Equal(t, 1.3, boost)

	boost = rag.getNodeTypeBoost("Other")
	assert.Equal(t, 1.0, boost)
}

func TestGraphRAG_GetRelevanceLabel(t *testing.T) {
	store := NewInMemoryGraphStore()
	graph := NewKnowledgeGraph(store)
	rag := NewGraphRAG(graph, nil)

	assert.Equal(t, "high", rag.getRelevanceLabel(0.9))
	assert.Equal(t, "medium", rag.getRelevanceLabel(0.6))
	assert.Equal(t, "low", rag.getRelevanceLabel(0.3))
}

func TestTraversalResult(t *testing.T) {
	result := NewTraversalResult()

	assert.NotNil(t, result.Nodes)
	assert.NotNil(t, result.Edges)
	assert.NotNil(t, result.Paths)
	assert.NotNil(t, result.Distance)
	assert.NotNil(t, result.Visited)
}
