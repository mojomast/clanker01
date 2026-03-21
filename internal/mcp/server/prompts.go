package server

import (
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/internal/mcp"
)

type Prompt struct {
	Name        string
	Description string
	Arguments   []mcp.PromptArgument
	Handler     PromptHandler
}

type PromptHandler func(args map[string]string) (*mcp.GetPromptResult, error)

type PromptRegistry struct {
	prompts map[string]*Prompt
	mu      sync.RWMutex
}

func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		prompts: make(map[string]*Prompt),
	}
}

func (r *PromptRegistry) Register(prompt *Prompt) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.prompts[prompt.Name]; exists {
		return fmt.Errorf("prompt %s already registered", prompt.Name)
	}

	r.prompts[prompt.Name] = prompt
	return nil
}

func (r *PromptRegistry) Get(name string) (*Prompt, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	prompt, ok := r.prompts[name]
	return prompt, ok
}

func (r *PromptRegistry) List() []*Prompt {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Prompt, 0, len(r.prompts))
	for _, prompt := range r.prompts {
		list = append(list, prompt)
	}
	return list
}

func (r *PromptRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.prompts, name)
}

func (s *Server) HandleListPrompts(req *mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	prompts := s.promptRegistry.List()

	result := &mcp.ListPromptsResult{
		Prompts: make([]mcp.Prompt, 0, len(prompts)),
	}

	for _, prompt := range prompts {
		result.Prompts = append(result.Prompts, mcp.Prompt{
			Name:        prompt.Name,
			Description: prompt.Description,
			Arguments:   prompt.Arguments,
		})
	}

	return result, nil
}

func (s *Server) HandleGetPrompt(req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	prompt, ok := s.promptRegistry.Get(req.Name)
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", req.Name)
	}

	result, err := prompt.Handler(req.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	return result, nil
}

func NewPrompt(name, description string, args []mcp.PromptArgument, handler PromptHandler) *Prompt {
	return &Prompt{
		Name:        name,
		Description: description,
		Arguments:   args,
		Handler:     handler,
	}
}

func NewStaticPrompt(name, description string, messages []mcp.Message) *Prompt {
	return &Prompt{
		Name:        name,
		Description: description,
		Arguments:   []mcp.PromptArgument{},
		Handler: func(args map[string]string) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{
				Messages: messages,
			}, nil
		},
	}
}
