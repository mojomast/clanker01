package agent

import (
	"testing"

	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewStateMachine(t *testing.T) {
	sm := NewStateMachine()

	if sm == nil {
		t.Fatal("Expected state machine to be created")
	}

	if sm.CurrentState() != api.AgentStatusCreated {
		t.Errorf("Expected initial state to be created, got %s", sm.CurrentState())
	}
}

func TestStateMachineValidTransitions(t *testing.T) {
	testCases := []struct {
		from     api.AgentStatus
		to       api.AgentStatus
		expected bool
	}{
		{api.AgentStatusCreated, api.AgentStatusReady, true},
		{api.AgentStatusCreated, api.AgentStatusTerminated, true},
		{api.AgentStatusCreated, api.AgentStatusError, true},
		{api.AgentStatusReady, api.AgentStatusRunning, true},
		{api.AgentStatusReady, api.AgentStatusPaused, true},
		{api.AgentStatusReady, api.AgentStatusTerminated, true},
		{api.AgentStatusReady, api.AgentStatusError, true},
		{api.AgentStatusRunning, api.AgentStatusReady, true},
		{api.AgentStatusRunning, api.AgentStatusPaused, true},
		{api.AgentStatusRunning, api.AgentStatusTerminated, true},
		{api.AgentStatusRunning, api.AgentStatusError, true},
		{api.AgentStatusPaused, api.AgentStatusReady, true},
		{api.AgentStatusPaused, api.AgentStatusTerminated, true},
		{api.AgentStatusPaused, api.AgentStatusError, true},
		{api.AgentStatusError, api.AgentStatusReady, true},
		{api.AgentStatusError, api.AgentStatusTerminated, true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			sm := NewStateMachine()
			sm.state = tc.from

			err := sm.Transition(tc.from, tc.to)
			if tc.expected && err != nil {
				t.Errorf("Expected transition to succeed: %v", err)
			}
			if !tc.expected && err == nil {
				t.Error("Expected transition to fail")
			}
		})
	}
}

func TestStateMachineInvalidTransitions(t *testing.T) {
	testCases := []struct {
		from api.AgentStatus
		to   api.AgentStatus
	}{
		{api.AgentStatusReady, api.AgentStatusCreated},
		{api.AgentStatusRunning, api.AgentStatusCreated},
		{api.AgentStatusTerminated, api.AgentStatusReady},
		{api.AgentStatusTerminated, api.AgentStatusRunning},
		{api.AgentStatusError, api.AgentStatusRunning},
	}

	for _, tc := range testCases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			sm := NewStateMachine()
			sm.state = tc.from

			err := sm.Transition(tc.from, tc.to)
			if err == nil {
				t.Error("Expected transition to fail")
			}
		})
	}
}

func TestStateMachineWrongCurrentState(t *testing.T) {
	sm := NewStateMachine()

	err := sm.Transition(api.AgentStatusReady, api.AgentStatusRunning)
	if err == nil {
		t.Error("Expected transition to fail when current state doesn't match")
	}
}

func TestStateMachineStateAfterTransition(t *testing.T) {
	sm := NewStateMachine()

	err := sm.Transition(api.AgentStatusCreated, api.AgentStatusReady)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sm.CurrentState() != api.AgentStatusReady {
		t.Errorf("Expected state to be ready, got %s", sm.CurrentState())
	}

	err = sm.Transition(api.AgentStatusReady, api.AgentStatusRunning)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sm.CurrentState() != api.AgentStatusRunning {
		t.Errorf("Expected state to be running, got %s", sm.CurrentState())
	}

	err = sm.Transition(api.AgentStatusRunning, api.AgentStatusReady)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sm.CurrentState() != api.AgentStatusReady {
		t.Errorf("Expected state to be ready, got %s", sm.CurrentState())
	}
}

func TestStateMachineTerminatedIsFinal(t *testing.T) {
	sm := NewStateMachine()
	sm.state = api.AgentStatusRunning

	err := sm.Transition(api.AgentStatusRunning, api.AgentStatusTerminated)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = sm.Transition(api.AgentStatusTerminated, api.AgentStatusReady)
	if err == nil {
		t.Error("Expected transition from terminated to fail")
	}

	if sm.CurrentState() != api.AgentStatusTerminated {
		t.Errorf("Expected state to remain terminated, got %s", sm.CurrentState())
	}
}

func TestStateMachineErrorRecovery(t *testing.T) {
	sm := NewStateMachine()

	err := sm.Transition(api.AgentStatusCreated, api.AgentStatusReady)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = sm.Transition(api.AgentStatusReady, api.AgentStatusError)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sm.CurrentState() != api.AgentStatusError {
		t.Errorf("Expected state to be error, got %s", sm.CurrentState())
	}

	err = sm.Transition(api.AgentStatusError, api.AgentStatusReady)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sm.CurrentState() != api.AgentStatusReady {
		t.Errorf("Expected state to be ready after recovery, got %s", sm.CurrentState())
	}
}
