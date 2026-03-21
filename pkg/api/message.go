package api

import (
	"time"
)

type MessageType string

const (
	MessageTypeTaskAssignment     MessageType = "task_assignment"
	MessageTypeTaskStatus         MessageType = "task_status"
	MessageTypeTaskCompletion     MessageType = "task_completion"
	MessageTypeTaskFailure        MessageType = "task_failure"
	MessageTypeAssistanceRequest  MessageType = "assistance_request"
	MessageTypeAssistanceResponse MessageType = "assistance_response"
	MessageTypeClarification      MessageType = "clarification"
	MessageTypeFeedback           MessageType = "feedback"
	MessageTypeContextShare       MessageType = "context_share"
	MessageTypeContextRequest     MessageType = "context_request"
	MessageTypeArtifactUpdate     MessageType = "artifact_update"
	MessageTypeBlockingNotice     MessageType = "blocking_notice"
	MessageTypeDependencyReady    MessageType = "dependency_ready"
	MessageTypeConflictDetected   MessageType = "conflict_detected"
	MessageTypeConsensusRequest   MessageType = "consensus_request"
	MessageTypeConsensusVote      MessageType = "consensus_vote"
	MessageTypeHeartbeat          MessageType = "heartbeat"
	MessageTypeStatusQuery        MessageType = "status_query"
	MessageTypeErrorReport        MessageType = "error_report"
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
