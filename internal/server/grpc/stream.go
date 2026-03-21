package grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrStreamNotFound = errors.New("stream not found")
	ErrStreamClosed   = errors.New("stream closed")
)

type StreamManager struct {
	mu       sync.RWMutex
	streams  map[string]*ManagedStream
	registry map[string][]string
}

type ManagedStream struct {
	ID        string
	Type      StreamType
	CreatedAt time.Time
	UpdatedAt time.Time

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	AgentID   string
	TaskID    string
	SkillIDs  []string
	Interval  time.Duration
	LastEvent time.Time

	metadata map[string]string
}

type StreamType string

const (
	StreamTypeAgentMetrics StreamType = "agent_metrics"
	StreamTypeAgentHealth  StreamType = "agent_health"
	StreamTypeTaskUpdates  StreamType = "task_updates"
	StreamTypeTaskProgress StreamType = "task_progress"
	StreamTypeSkillUpdates StreamType = "skill_updates"
)

func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams:  make(map[string]*ManagedStream),
		registry: make(map[string][]string),
	}
}

func (sm *StreamManager) RegisterStream(streamID, streamType string, ctx context.Context, cancel context.CancelFunc, options map[string]string) (*ManagedStream, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.streams[streamID]; exists {
		return nil, status.Errorf(codes.AlreadyExists, "stream %s already exists", streamID)
	}

	stream := &ManagedStream{
		ID:        streamID,
		Type:      StreamType(streamType),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
		metadata:  options,
	}

	for key, value := range options {
		switch key {
		case "agent_id":
			stream.AgentID = value
		case "task_id":
			stream.TaskID = value
		case "interval_ms":
			if duration, err := time.ParseDuration(value + "ms"); err == nil {
				stream.Interval = duration
			}
		}
	}

	sm.streams[streamID] = stream

	if stream.AgentID != "" {
		sm.registry[stream.AgentID] = append(sm.registry[stream.AgentID], streamID)
	}
	if stream.TaskID != "" {
		sm.registry[stream.TaskID] = append(sm.registry[stream.TaskID], streamID)
	}

	return stream, nil
}

func (sm *StreamManager) GetStream(streamID string) (*ManagedStream, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stream, exists := sm.streams[streamID]
	if !exists {
		return nil, ErrStreamNotFound
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.ctx.Err() != nil {
		return nil, ErrStreamClosed
	}

	return stream, nil
}

func (sm *StreamManager) UnregisterStream(streamID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stream, exists := sm.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	if stream.AgentID != "" {
		streams := sm.registry[stream.AgentID]
		for i, id := range streams {
			if id == streamID {
				sm.registry[stream.AgentID] = append(streams[:i], streams[i+1:]...)
				break
			}
		}
	}

	if stream.TaskID != "" {
		streams := sm.registry[stream.TaskID]
		for i, id := range streams {
			if id == streamID {
				sm.registry[stream.TaskID] = append(streams[:i], streams[i+1:]...)
				break
			}
		}
	}

	delete(sm.streams, streamID)

	if stream.cancel != nil {
		stream.cancel()
	}

	return nil
}

func (sm *StreamManager) GetStreamsByAgent(agentID string) []*ManagedStream {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	streamIDs := sm.registry[agentID]
	streams := make([]*ManagedStream, 0, len(streamIDs))

	for _, streamID := range streamIDs {
		if stream, exists := sm.streams[streamID]; exists {
			stream.mu.Lock()
			if stream.ctx.Err() == nil {
				streams = append(streams, stream)
			}
			stream.mu.Unlock()
		}
	}

	return streams
}

func (sm *StreamManager) GetStreamsByTask(taskID string) []*ManagedStream {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	streamIDs := sm.registry[taskID]
	streams := make([]*ManagedStream, 0, len(streamIDs))

	for _, streamID := range streamIDs {
		if stream, exists := sm.streams[streamID]; exists {
			stream.mu.Lock()
			if stream.ctx.Err() == nil {
				streams = append(streams, stream)
			}
			stream.mu.Unlock()
		}
	}

	return streams
}

func (sm *StreamManager) CloseAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for streamID, stream := range sm.streams {
		if stream.cancel != nil {
			stream.cancel()
		}
		delete(sm.streams, streamID)
	}

	for key := range sm.registry {
		delete(sm.registry, key)
	}
}

func (sm *StreamManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.streams)
}

func (sm *StreamManager) UpdateStream(streamID string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stream, exists := sm.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	stream.UpdatedAt = time.Now()
	stream.LastEvent = time.Now()

	return nil
}

func (s *ManagedStream) Context() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx
}

func (s *ManagedStream) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *ManagedStream) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx.Err() != nil
}

func (s *ManagedStream) GetMetadata(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.metadata[key]
	return val, ok
}

func (s *ManagedStream) SetMetadata(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.metadata == nil {
		s.metadata = make(map[string]string)
	}
	s.metadata[key] = value
}

func (s *ManagedStream) GetID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ID
}

func (s *ManagedStream) GetType() StreamType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Type
}

func (s *ManagedStream) GetAgentID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AgentID
}

func (s *ManagedStream) GetTaskID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TaskID
}

func (s *ManagedStream) GetInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Interval
}

func (s *ManagedStream) GetLastEvent() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastEvent
}

func (s *ManagedStream) GetCreatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CreatedAt
}

func (s *ManagedStream) GetUpdatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.UpdatedAt
}

func GenerateStreamID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
