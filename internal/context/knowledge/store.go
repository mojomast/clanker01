package knowledge

import (
	"fmt"
	"sync"
)

type InMemoryGraphStore struct {
	mu    sync.RWMutex
	nodes map[string]*GraphNode
	edges map[string][]*GraphEdge
	adj   map[string][]*GraphEdge
}

func NewInMemoryGraphStore() *InMemoryGraphStore {
	return &InMemoryGraphStore{
		nodes: make(map[string]*GraphNode),
		edges: make(map[string][]*GraphEdge),
		adj:   make(map[string][]*GraphEdge),
	}
}

func (s *InMemoryGraphStore) CreateNode(node *GraphNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node.ID] = node
	return nil
}

func (s *InMemoryGraphStore) CreateEdge(edge *GraphEdge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[edge.From]; !exists {
		return fmt.Errorf("source node not found: %s", edge.From)
	}

	if _, exists := s.nodes[edge.To]; !exists {
		return fmt.Errorf("target node not found: %s", edge.To)
	}

	s.edges[edge.ID] = append(s.edges[edge.ID], edge)
	s.adj[edge.From] = append(s.adj[edge.From], edge)
	return nil
}

func (s *InMemoryGraphStore) GetNode(id string) (*GraphNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", id)
	}
	return node, nil
}

func (s *InMemoryGraphStore) GetNeighbors(
	id string,
	edgeType string,
	depth int,
) ([]*GraphNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*GraphNode
	visited := map[string]bool{id: true}

	var bfs func(current string, currentDepth int)
	bfs = func(current string, currentDepth int) {
		if currentDepth >= depth {
			return
		}
		for _, edge := range s.adj[current] {
			if edgeType != "" && edge.Type != edgeType {
				continue
			}
			if visited[edge.To] {
				continue
			}
			visited[edge.To] = true
			if node, ok := s.nodes[edge.To]; ok {
				result = append(result, node)
			}
			bfs(edge.To, currentDepth+1)
		}
	}

	bfs(id, 0)
	return result, nil
}

func (s *InMemoryGraphStore) Query(pattern *GraphQuery) ([]*GraphNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*GraphNode

	for _, node := range s.nodes {
		if s.matchesQuery(node, pattern) {
			results = append(results, node)
		}
	}

	return results, nil
}

func (s *InMemoryGraphStore) QueryEdges(pattern *EdgeQuery) ([]*GraphEdge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*GraphEdge

	for _, edges := range s.edges {
		for _, edge := range edges {
			if s.matchesEdgeQuery(edge, pattern) {
				results = append(results, edge)
			}
		}
	}

	return results, nil
}

func (s *InMemoryGraphStore) DeleteNode(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.nodes[id]; !ok {
		return fmt.Errorf("node not found: %s", id)
	}

	delete(s.nodes, id)

	for fromID := range s.adj {
		var filtered []*GraphEdge
		for _, edge := range s.adj[fromID] {
			if edge.To != id {
				filtered = append(filtered, edge)
			}
		}
		s.adj[fromID] = filtered
	}

	for edgeID, edges := range s.edges {
		var filtered []*GraphEdge
		for _, edge := range edges {
			if edge.From != id && edge.To != id {
				filtered = append(filtered, edge)
			}
		}
		if len(filtered) == 0 {
			delete(s.edges, edgeID)
		} else {
			s.edges[edgeID] = filtered
		}
	}

	return nil
}

func (s *InMemoryGraphStore) DeleteEdge(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.edges[id]; !ok {
		return fmt.Errorf("edge not found: %s", id)
	}

	delete(s.edges, id)

	for fromID, edges := range s.adj {
		var filtered []*GraphEdge
		for _, edge := range edges {
			if edge.ID != id {
				filtered = append(filtered, edge)
			}
		}
		s.adj[fromID] = filtered
		if len(filtered) == 0 {
			delete(s.adj, fromID)
		}
	}

	return nil
}

func (s *InMemoryGraphStore) matchesQuery(node *GraphNode, pattern *GraphQuery) bool {
	if len(pattern.NodeTypes) > 0 {
		found := false
		for _, nt := range pattern.NodeTypes {
			if node.Type == nt {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(pattern.Labels) > 0 {
		labelSet := make(map[string]bool)
		for _, l := range node.Labels {
			labelSet[l] = true
		}
		for _, pl := range pattern.Labels {
			if !labelSet[pl] {
				return false
			}
		}
	}

	if len(pattern.Properties) > 0 {
		for key, value := range pattern.Properties {
			if nodeValue, ok := node.Properties[key]; !ok || nodeValue != value {
				return false
			}
		}
	}

	return true
}

func (s *InMemoryGraphStore) matchesEdgeQuery(edge *GraphEdge, pattern *EdgeQuery) bool {
	if len(pattern.EdgeTypes) > 0 {
		found := false
		for _, et := range pattern.EdgeTypes {
			if edge.Type == et {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if pattern.FromNodeID != "" && edge.From != pattern.FromNodeID {
		return false
	}

	if pattern.ToNodeID != "" && edge.To != pattern.ToNodeID {
		return false
	}

	if pattern.MinWeight > 0 && edge.Weight < pattern.MinWeight {
		return false
	}

	if pattern.MaxWeight > 0 && edge.Weight > pattern.MaxWeight {
		return false
	}

	if len(pattern.Properties) > 0 {
		for key, value := range pattern.Properties {
			if edgeValue, ok := edge.Properties[key]; !ok || edgeValue != value {
				return false
			}
		}
	}

	return true
}

func (s *InMemoryGraphStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes = make(map[string]*GraphNode)
	s.edges = make(map[string][]*GraphEdge)
	s.adj = make(map[string][]*GraphEdge)
}

func (s *InMemoryGraphStore) NodeCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.nodes)
}

func (s *InMemoryGraphStore) EdgeCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, edges := range s.edges {
		count += len(edges)
	}
	return count
}
