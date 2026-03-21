package knowledge

import (
	"container/list"
	"math"
)

type TraversalOptions struct {
	MaxDepth       int
	MaxNodes       int
	EdgeTypes      []string
	NodeTypes      []string
	IncludeWeights bool
}

type TraversalResult struct {
	Nodes    []*GraphNode
	Edges    []*GraphEdge
	Paths    [][]string
	Distance map[string]float64
	Visited  map[string]bool
	MaxDepth int
}

type PathResult struct {
	Path      []*GraphNode
	PathIDs   []string
	TotalCost float64
}

func NewTraversalResult() *TraversalResult {
	return &TraversalResult{
		Nodes:    make([]*GraphNode, 0),
		Edges:    make([]*GraphEdge, 0),
		Paths:    make([][]string, 0),
		Distance: make(map[string]float64),
		Visited:  make(map[string]bool),
	}
}

func (g *KnowledgeGraph) BFS(startID string, opts *TraversalOptions) (*TraversalResult, error) {
	store := g.store
	if opts == nil {
		opts = &TraversalOptions{MaxDepth: 3}
	}

	result := NewTraversalResult()
	result.Distance[startID] = 0
	result.Visited[startID] = true

	queue := list.New()
	queue.PushBack(&queueItem{id: startID, depth: 0})

	for queue.Len() > 0 {
		item := queue.Remove(queue.Front()).(*queueItem)

		if item.depth >= opts.MaxDepth {
			continue
		}

		if opts.MaxNodes > 0 && len(result.Visited) >= opts.MaxNodes {
			break
		}

		node, err := store.GetNode(item.id)
		if err != nil {
			continue
		}

		if !contains(result.Nodes, node.ID) {
			result.Nodes = append(result.Nodes, node)
		}

		neighbors, err := store.GetNeighbors(item.id, "", 1)
		if err != nil {
			continue
		}

		for _, neighbor := range neighbors {
			if !result.Visited[neighbor.ID] {
				result.Visited[neighbor.ID] = true
				result.Distance[neighbor.ID] = float64(item.depth + 1)
				queue.PushBack(&queueItem{id: neighbor.ID, depth: item.depth + 1})
			}
		}
	}

	return result, nil
}

func (g *KnowledgeGraph) DFS(startID string, opts *TraversalOptions) (*TraversalResult, error) {
	store := g.store
	if opts == nil {
		opts = &TraversalOptions{MaxDepth: 3}
	}

	result := NewTraversalResult()
	result.Distance[startID] = 0
	result.Visited[startID] = true

	stack := []*queueItem{{id: startID, depth: 0}}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if item.depth >= opts.MaxDepth {
			continue
		}

		if opts.MaxNodes > 0 && len(result.Visited) >= opts.MaxNodes {
			break
		}

		node, err := store.GetNode(item.id)
		if err != nil {
			continue
		}

		if !contains(result.Nodes, node.ID) {
			result.Nodes = append(result.Nodes, node)
		}

		neighbors, err := store.GetNeighbors(item.id, "", 1)
		if err != nil {
			continue
		}

		for _, neighbor := range neighbors {
			if !result.Visited[neighbor.ID] {
				result.Visited[neighbor.ID] = true
				result.Distance[neighbor.ID] = float64(item.depth + 1)
				stack = append(stack, &queueItem{id: neighbor.ID, depth: item.depth + 1})
			}
		}
	}

	return result, nil
}

func (g *KnowledgeGraph) ShortestPath(fromID, toID string) (*PathResult, error) {
	store := g.store

	dist := make(map[string]float64)
	prev := make(map[string]string)
	visited := make(map[string]bool)

	dist[fromID] = 0

	for len(visited) < 1000 {
		current := ""
		minDist := math.Inf(1)

		for id, d := range dist {
			if !visited[id] && d < minDist {
				minDist = d
				current = id
			}
		}

		if current == "" || current == toID {
			break
		}

		visited[current] = true

		neighbors, err := store.GetNeighbors(current, "", 1)
		if err != nil {
			continue
		}

		edges, err := store.QueryEdges(&EdgeQuery{FromNodeID: current})
		if err != nil {
			continue
		}

		for _, neighbor := range neighbors {
			weight := 1.0
			for _, edge := range edges {
				if edge.To == neighbor.ID {
					weight = edge.Weight
					break
				}
			}

			alt := dist[current] + weight
			if existing, ok := dist[neighbor.ID]; !ok || alt < existing {
				dist[neighbor.ID] = alt
				prev[neighbor.ID] = current
			}
		}
	}

	if _, ok := dist[toID]; !ok {
		return nil, nil
	}

	path := make([]*GraphNode, 0)
	pathIDs := make([]string, 0)

	current := toID
	for current != "" {
		node, err := store.GetNode(current)
		if err != nil {
			break
		}
		path = append([]*GraphNode{node}, path...)
		pathIDs = append([]string{current}, pathIDs...)
		current = prev[current]
	}

	return &PathResult{
		Path:      path,
		PathIDs:   pathIDs,
		TotalCost: dist[toID],
	}, nil
}

func (g *KnowledgeGraph) FindConnectedComponents() [][]string {
	store := g.store
	visited := make(map[string]bool)
	var components [][]string

	_, _ = store.Query(&GraphQuery{})

	for !allNodesVisited(store, visited) {
		var start string
		allNodes, _ := store.Query(&GraphQuery{})
		for _, node := range allNodes {
			if !visited[node.ID] {
				start = node.ID
				break
			}
		}

		if start == "" {
			break
		}

		result, _ := g.BFS(start, &TraversalOptions{MaxDepth: 100})

		component := make([]string, 0, len(result.Visited))
		for id := range result.Visited {
			component = append(component, id)
			visited[id] = true
		}

		if len(component) > 0 {
			components = append(components, component)
		}
	}

	return components
}

func (g *KnowledgeGraph) FindCycles() [][]string {
	store := g.store
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cycles [][]string

	allNodes, _ := store.Query(&GraphQuery{})

	var dfs func(nodeID string, path []string)
	dfs = func(nodeID string, path []string) {
		visited[nodeID] = true
		recStack[nodeID] = true
		path = append(path, nodeID)

		edges, err := store.QueryEdges(&EdgeQuery{FromNodeID: nodeID})
		if err == nil {
			for _, edge := range edges {
				if !visited[edge.To] {
					dfs(edge.To, path)
				} else if recStack[edge.To] {
					cycleStart := -1
					for i, id := range path {
						if id == edge.To {
							cycleStart = i
							break
						}
					}
					if cycleStart >= 0 {
						cycle := make([]string, len(path)-cycleStart+1)
						copy(cycle, path[cycleStart:])
						cycle[len(cycle)-1] = edge.To
						cycles = append(cycles, cycle)
					}
				}
			}
		}

		recStack[nodeID] = false
	}

	for _, node := range allNodes {
		if !visited[node.ID] {
			dfs(node.ID, make([]string, 0))
		}
	}

	return cycles
}

func (g *KnowledgeGraph) Centrality(nodeID string) (float64, error) {
	result, err := g.BFS(nodeID, &TraversalOptions{MaxDepth: 3})
	if err != nil {
		return 0, err
	}

	if len(result.Visited) == 0 {
		return 0, nil
	}

	totalDist := 0.0
	for _, dist := range result.Distance {
		totalDist += dist
	}

	if totalDist == 0 {
		return float64(len(result.Visited)), nil
	}

	return float64(len(result.Visited)) / totalDist, nil
}

func (g *KnowledgeGraph) FindHubNodes(hubType string, topK int) ([]*GraphNode, error) {
	store := g.store

	nodes, err := store.Query(&GraphQuery{NodeTypes: []string{hubType}})
	if err != nil {
		return nil, err
	}

	type nodeScore struct {
		node  *GraphNode
		score float64
	}
	scores := make([]nodeScore, 0, len(nodes))

	for _, node := range nodes {
		neighbors, _ := store.GetNeighbors(node.ID, "", 1)

		degree := len(neighbors)
		centrality, _ := g.Centrality(node.ID)
		score := float64(degree) + centrality*10

		scores = append(scores, nodeScore{node: node, score: score})
	}

	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score < scores[j].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	result := make([]*GraphNode, 0, topK)
	for i := 0; i < topK && i < len(scores); i++ {
		result = append(result, scores[i].node)
	}

	return result, nil
}

type queueItem struct {
	id    string
	depth int
}

func contains(nodes []*GraphNode, id string) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func allNodesVisited(store GraphStore, visited map[string]bool) bool {
	allNodes, err := store.Query(&GraphQuery{})
	if err != nil {
		return true
	}

	for _, node := range allNodes {
		if !visited[node.ID] {
			return false
		}
	}
	return true
}
