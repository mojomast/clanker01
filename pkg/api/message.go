package api

import (
	"time"
)

type MessageType string

// MessageType constants for agent-to-agent communication.
// Currently only the following types are actively used. Additional message types
// (e.g., task_status, clarification, artifact_update, etc.) can be added here
// as the messaging system is implemented.
const (
	MessageTypeTaskAssignment    MessageType = "task_assignment"
	MessageTypeAssistanceRequest MessageType = "assistance_request"
	MessageTypeContextShare      MessageType = "context_share"
	MessageTypeConsensusRequest  MessageType = "consensus_request"
	MessageTypeHeartbeat         MessageType = "heartbeat"
)

type MessagePriority int

const (
	PriorityLow      MessagePriority = 0
	PriorityNormal   MessagePriority = 1
	PriorityHigh     MessagePriority = 2
	PriorityCritical MessagePriority = 3
)

type AgentMessage struct {
	ID            string
	CorrelationID string
	Timestamp     time.Time

	Sender   AgentRef
	Receiver AgentRef

	Type     MessageType
	Priority MessagePriority
	TTL      time.Duration

	Payload  any
	Metadata map[string]any

	RequiresAck bool
	AckDeadline time.Time
}

type AgentRef struct {
	ID   string
	Type AgentType
	Role string
}
