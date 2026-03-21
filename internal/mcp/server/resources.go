package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/internal/mcp"
)

type Resource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Handler     ResourceHandler
}

type ResourceHandler func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)

type ResourceRegistry struct {
	resources map[string]*Resource
	mu        sync.RWMutex
}

func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		resources: make(map[string]*Resource),
	}
}

func (r *ResourceRegistry) Register(res *Resource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[res.URI]; exists {
		return fmt.Errorf("resource %s already registered", res.URI)
	}

	r.resources[res.URI] = res
	return nil
}

func (r *ResourceRegistry) Get(uri string) (*Resource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res, ok := r.resources[uri]
	return res, ok
}

func (r *ResourceRegistry) List() []*Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Resource, 0, len(r.resources))
	for _, res := range r.resources {
		list = append(list, res)
	}
	return list
}

func (r *ResourceRegistry) Unregister(uri string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.resources, uri)
}

func (s *Server) HandleListResources(req *mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	resources := s.resourceRegistry.List()

	result := &mcp.ListResourcesResult{
		Resources: make([]mcp.Resource, 0, len(resources)),
	}

	for _, res := range resources {
		result.Resources = append(result.Resources, mcp.Resource{
			URI:         res.URI,
			Name:        res.Name,
			Description: res.Description,
			MimeType:    res.MimeType,
		})
	}

	return result, nil
}

func (s *Server) HandleReadResource(req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	res, ok := s.resourceRegistry.Get(req.URI)
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", req.URI)
	}

	result, err := res.Handler(s.ctx, req.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}

	return result, nil
}

func NewTextResource(uri, name, description, content string) *Resource {
	return &Resource{
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    "text/plain",
		Handler: func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []mcp.ResourceContents{
					{
						URI:  uri,
						Text: content,
					},
				},
			}, nil
		},
	}
}

func NewDynamicResource(uri, name, description, mimeType string, handler ResourceHandler) *Resource {
	return &Resource{
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    mimeType,
		Handler:     handler,
	}
}
