package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type OrchestratorState struct {
	Version   string
	StartedAt time.Time

	Agents map[string]*AgentState
	Pools  map[api.AgentType]*PoolState

	Tasks map[string]*TaskState
	Queue *QueueState

	Context *ContextState
	Metrics *OrchestratorMetrics
}

type AgentState struct {
	ID           string
	Type         api.AgentType
	Status       api.AgentStatus
	CurrentTask  string
	LastActivity time.Time
	Metrics      api.AgentMetrics
}

type PoolState struct {
	Type        api.AgentType
	MinSize     int
	MaxSize     int
	CurrentSize int
	Available   []string
}

type TaskState struct {
	ID            string
	Status        api.TaskStatus
	AssignedAgent string
	Progress      float64
	StartedAt     time.Time
	CompletedAt   time.Time
}

type QueueState struct {
	PendingCount   int
	RunningCount   int
	CompletedCount int
	FailedCount    int
}

type ContextState struct {
	SessionID string
	Entries   int
}

type OrchestratorMetrics struct {
	TasksSubmitted    int64
	TasksCompleted    int64
	TasksFailed       int64
	TotalDuration     time.Duration
	AvgTaskDuration   time.Duration
	AgentErrors       int64
	ConflictsDetected int64
}

type Checkpoint struct {
	ID        string
	Name      string
	CreatedAt time.Time
	State     *OrchestratorState
	Metadata  map[string]any
}

type StateManager struct {
	mu             sync.RWMutex
	dataDir        string
	checkpointsDir string
	currentState   *OrchestratorState
}

func NewStateManager(dataDir string) (*StateManager, error) {
	checkpointsDir := filepath.Join(dataDir, "checkpoints")
	if err := os.MkdirAll(checkpointsDir, 0755); err != nil {
		return nil, fmt.Errorf("create checkpoints dir: %w", err)
	}

	return &StateManager{
		dataDir:        dataDir,
		checkpointsDir: checkpointsDir,
		currentState: &OrchestratorState{
			Version:   "1.0.0",
			StartedAt: time.Now(),
			Agents:    make(map[string]*AgentState),
			Pools:     make(map[api.AgentType]*PoolState),
			Tasks:     make(map[string]*TaskState),
			Metrics:   &OrchestratorMetrics{},
		},
	}, nil
}

func (sm *StateManager) GetState() *OrchestratorState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.currentState
}

func (sm *StateManager) UpdateAgentState(agentID string, state *AgentState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentState.Agents[agentID] = state
}

func (sm *StateManager) UpdateTaskState(taskID string, state *TaskState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentState.Tasks[taskID] = state
}

func (sm *StateManager) UpdateQueueState(state *QueueState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentState.Queue = state
}

func (sm *StateManager) UpdateMetrics(metrics *OrchestratorMetrics) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentState.Metrics = metrics
}

func (sm *StateManager) Save() error {
	sm.mu.RLock()
	state := sm.currentState
	sm.mu.RUnlock()

	path := filepath.Join(sm.dataDir, "state.json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}

	return nil
}

func (sm *StateManager) Load() (*OrchestratorState, error) {
	path := filepath.Join(sm.dataDir, "state.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return sm.currentState, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}

	var state OrchestratorState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	sm.mu.Lock()
	sm.currentState = &state
	sm.mu.Unlock()

	return &state, nil
}

func (sm *StateManager) CreateCheckpoint(name string) (*Checkpoint, error) {
	sm.mu.RLock()
	state := sm.currentState
	sm.mu.RUnlock()

	checkpoint := &Checkpoint{
		ID:        generateID(),
		Name:      name,
		CreatedAt: time.Now(),
		State:     state,
		Metadata:  make(map[string]any),
	}

	path := filepath.Join(sm.checkpointsDir, checkpoint.ID+".json")

	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal checkpoint: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write checkpoint: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return nil, fmt.Errorf("rename checkpoint: %w", err)
	}

	return checkpoint, nil
}

func (sm *StateManager) RestoreCheckpoint(id string) (*OrchestratorState, error) {
	path := filepath.Join(sm.checkpointsDir, id+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint: %w", err)
	}

	sm.mu.Lock()
	sm.currentState = checkpoint.State
	sm.mu.Unlock()

	return checkpoint.State, nil
}

func (sm *StateManager) ListCheckpoints() ([]*Checkpoint, error) {
	entries, err := os.ReadDir(sm.checkpointsDir)
	if err != nil {
		return nil, fmt.Errorf("read checkpoints dir: %w", err)
	}

	var checkpoints []*Checkpoint
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(sm.checkpointsDir, entry.Name()))
		if err != nil {
			continue
		}

		var checkpoint Checkpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			continue
		}

		checkpoints = append(checkpoints, &checkpoint)
	}

	return checkpoints, nil
}

func (sm *StateManager) DeleteCheckpoint(id string) error {
	path := filepath.Join(sm.checkpointsDir, id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete checkpoint: %w", err)
	}
	return nil
}

func (sm *StateManager) Clear() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentState = &OrchestratorState{
		Version:   "1.0.0",
		StartedAt: time.Now(),
		Agents:    make(map[string]*AgentState),
		Pools:     make(map[api.AgentType]*PoolState),
		Tasks:     make(map[string]*TaskState),
		Metrics:   &OrchestratorMetrics{},
	}

	return os.RemoveAll(sm.checkpointsDir)
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
