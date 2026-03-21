package agent

import (
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/pkg/api"
)

type StateMachine struct {
	mu    sync.RWMutex
	state api.AgentStatus
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		state: api.AgentStatusCreated,
	}
}

func (sm *StateMachine) CurrentState() api.AgentStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

func (sm *StateMachine) Transition(from, to api.AgentStatus) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state != from {
		return fmt.Errorf("cannot transition from %s to %s: current state is %s", from, to, sm.state)
	}

	if !isValidTransition(from, to) {
		return fmt.Errorf("invalid state transition: %s -> %s", from, to)
	}

	sm.state = to
	return nil
}

func isValidTransition(from, to api.AgentStatus) bool {
	validTransitions := map[api.AgentStatus][]api.AgentStatus{
		api.AgentStatusCreated: {
			api.AgentStatusReady,
			api.AgentStatusTerminated,
			api.AgentStatusError,
		},
		api.AgentStatusReady: {
			api.AgentStatusRunning,
			api.AgentStatusPaused,
			api.AgentStatusTerminated,
			api.AgentStatusError,
		},
		api.AgentStatusRunning: {
			api.AgentStatusReady,
			api.AgentStatusPaused,
			api.AgentStatusError,
			api.AgentStatusTerminated,
		},
		api.AgentStatusPaused: {
			api.AgentStatusReady,
			api.AgentStatusTerminated,
			api.AgentStatusError,
		},
		api.AgentStatusError: {
			api.AgentStatusReady,
			api.AgentStatusTerminated,
		},
		api.AgentStatusTerminated: {},
	}

	validStates, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, validState := range validStates {
		if validState == to {
			return true
		}
	}

	return false
}
