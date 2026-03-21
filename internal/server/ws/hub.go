package ws

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

var (
	ErrHubClosed        = errors.New("hub is closed")
	ErrClientNotFound   = errors.New("client not found")
	ErrInvalidBroadcast = errors.New("invalid broadcast target")
)

type Hub struct {
	clients     map[string]*Client
	userClients map[string]map[string]bool
	mu          sync.RWMutex
	broadcast   chan *BroadcastMessage
	register    chan *Client
	unregister  chan *Client
	errorChan   chan error
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
}

type BroadcastMessage struct {
	Message *Message
	Target  *BroadcastTarget
}

func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		clients:     make(map[string]*Client),
		userClients: make(map[string]map[string]bool),
		broadcast:   make(chan *BroadcastMessage, 256),
		register:    make(chan *Client, 64),
		unregister:  make(chan *Client, 64),
		errorChan:   make(chan error, 64),
		ctx:         ctx,
		cancel:      cancel,
		closed:      false,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case broadcast := <-h.broadcast:
			h.handleBroadcast(broadcast)

		case err := <-h.errorChan:
			h.handleError(err)

		case <-h.ctx.Done():
			h.cleanup()
			return
		}
	}
}

func (h *Hub) Register(client *Client) {
	select {
	case h.register <- client:
	case <-h.ctx.Done():
	}
}

func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	case <-h.ctx.Done():
	}
}

func (h *Hub) Broadcast(msg *Message, target *BroadcastTarget) error {
	if target == nil {
		target = &BroadcastTarget{All: true}
	}

	select {
	case h.broadcast <- &BroadcastMessage{Message: msg, Target: target}:
		return nil
	case <-h.ctx.Done():
		return ErrHubClosed
	default:
		return fmt.Errorf("broadcast channel full")
	}
}

func (h *Hub) BroadcastAll(msg *Message) error {
	return h.Broadcast(msg, &BroadcastTarget{All: true})
}

func (h *Hub) BroadcastToUser(msg *Message, userID string) error {
	return h.Broadcast(msg, &BroadcastTarget{UserIDs: []string{userID}})
}

func (h *Hub) BroadcastToAgent(msg *Message, agentID string) error {
	return h.Broadcast(msg, &BroadcastTarget{AgentIDs: []string{agentID}})
}

func (h *Hub) BroadcastToTask(msg *Message, taskID string) error {
	return h.Broadcast(msg, &BroadcastTarget{TaskIDs: []string{taskID}})
}

func (h *Hub) Error(err error) {
	select {
	case h.errorChan <- err:
	default:
	}
}

func (h *Hub) GetClient(clientID string) (*Client, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		return nil, ErrHubClosed
	}

	client, ok := h.clients[clientID]
	if !ok {
		return nil, ErrClientNotFound
	}

	return client, nil
}

func (h *Hub) GetClientsByUser(userID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var clients []*Client
	if userClients, ok := h.userClients[userID]; ok {
		for clientID := range userClients {
			if client, ok := h.clients[clientID]; ok {
				clients = append(clients, client)
			}
		}
	}

	return clients
}

func (h *Hub) GetAllClients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}

	return clients
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) UserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userClients)
}

func (h *Hub) IsClosed() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.closed
}

func (h *Hub) Stop() error {
	h.cancel()
	return nil
}

func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	h.clients[client.id] = client

	if h.userClients[client.userID] == nil {
		h.userClients[client.userID] = make(map[string]bool)
	}
	h.userClients[client.userID][client.id] = true
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.id]; ok {
		delete(h.clients, client.id)

		if userClients, ok := h.userClients[client.userID]; ok {
			delete(userClients, client.id)
			if len(userClients) == 0 {
				delete(h.userClients, client.userID)
			}
		}
	}
}

func (h *Hub) handleBroadcast(broadcast *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		return
	}

	msg := broadcast.Message
	target := broadcast.Target

	var recipients []*Client

	if target.All {
		for _, client := range h.clients {
			if !client.IsClosed() && client.Subscription().ShouldReceive(msg) {
				recipients = append(recipients, client)
			}
		}
	} else {
		userSet := make(map[string]bool)
		for _, userID := range target.UserIDs {
			userSet[userID] = true
		}

		agentSet := make(map[string]bool)
		for _, agentID := range target.AgentIDs {
			agentSet[agentID] = true
		}

		taskSet := make(map[string]bool)
		for _, taskID := range target.TaskIDs {
			taskSet[taskID] = true
		}

		for _, client := range h.clients {
			if client.IsClosed() {
				continue
			}
			if !client.Subscription().ShouldReceive(msg) {
				continue
			}

			if len(userSet) > 0 && !userSet[client.userID] {
				continue
			}

			if len(agentSet) > 0 && msg.AgentID != "" && !agentSet[msg.AgentID] {
				continue
			}

			if len(taskSet) > 0 && msg.TaskID != "" && !taskSet[msg.TaskID] {
				continue
			}

			recipients = append(recipients, client)
		}
	}

	for _, client := range recipients {
		select {
		case client.send <- msg:
		default:
		}
	}
}

func (h *Hub) handleError(err error) {
	fmt.Printf("[ws-hub] error: %v\n", err)
}

func (h *Hub) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	h.closed = true

	for _, client := range h.clients {
		client.Disconnect()
	}

	h.clients = make(map[string]*Client)
	h.userClients = make(map[string]map[string]bool)

	close(h.broadcast)
	close(h.register)
	close(h.unregister)
	close(h.errorChan)
}

func (h *Hub) SendAgentUpdate(agent api.Agent, metrics *api.AgentMetrics, health *api.AgentHealth) error {
	if agent == nil {
		return nil
	}
	msg := NewAgentUpdate(agent, metrics, health)
	return h.BroadcastAll(msg)
}

func (h *Hub) SendTaskEvent(task *api.Task, eventType string, result *api.TaskResult, err error) error {
	if task == nil {
		return nil
	}
	msg := NewTaskEvent(task, eventType, result, err)
	if msg == nil {
		return nil
	}
	return h.BroadcastAll(msg)
}

func (h *Hub) SendLog(level, message, agentID, taskID string) error {
	msg := NewLogEntry(level, message, agentID, taskID)
	return h.BroadcastAll(msg)
}

func (h *Hub) BroadcastAgentUpdateToUser(agent api.Agent, metrics *api.AgentMetrics, health *api.AgentHealth, userID string) error {
	msg := NewAgentUpdate(agent, metrics, health)
	return h.BroadcastToUser(msg, userID)
}

func (h *Hub) BroadcastTaskEventToAgent(task *api.Task, eventType string, result *api.TaskResult, err error, agentID string) error {
	msg := NewTaskEvent(task, eventType, result, err)
	return h.BroadcastToAgent(msg, agentID)
}

type HubStats struct {
	ClientCount int       `json:"client_count"`
	UserCount   int       `json:"user_count"`
	ConnectedAt time.Time `json:"connected_at"`
}

func (h *Hub) Stats() *HubStats {
	return &HubStats{
		ClientCount: h.ClientCount(),
		UserCount:   h.UserCount(),
		ConnectedAt: time.Now(),
	}
}
