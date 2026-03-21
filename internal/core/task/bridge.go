package task

import (
	"fmt"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

// APITaskToTask converts an api.Task to an internal task.Task for use with the
// Planner and Verifier. Fields that don't have a direct mapping are placed in
// Metadata so they survive a round-trip conversion.
func APITaskToTask(at *api.Task) *Task {
	if at == nil {
		return nil
	}

	input := make(map[string]any)
	if at.Prompt != "" {
		input["prompt"] = at.Prompt
	}
	if at.ExpectedOutput != "" {
		input["expected_output"] = at.ExpectedOutput
	}
	if len(at.Requirements) > 0 {
		input["requirements"] = at.Requirements
	}

	metadata := make(map[string]any)
	for k, v := range at.Metadata {
		metadata[k] = v
	}
	// Preserve api-specific fields in metadata for round-trip.
	if at.ParentID != "" {
		metadata["parent_id"] = at.ParentID
	}
	if at.Type != "" {
		metadata["type"] = at.Type
	}
	if string(at.AgentType) != "" {
		metadata["agent_type"] = string(at.AgentType)
	}

	var verificationSpec *VerificationSpec
	if at.Verification != nil {
		verificationSpec = verificationSpecFromMap(at.Verification)
	}

	t := &Task{
		ID:           TaskID(at.ID),
		Name:         at.Description,
		Description:  at.Description,
		Status:       mapAPIStatusToInternal(at.Status),
		Priority:     Priority(at.Priority),
		Kind:         KindCompute, // default; can be overridden via metadata
		Input:        input,
		Dependencies: make([]TaskID, 0, len(at.Dependencies)),
		CreatedAt:    at.CreatedAt,
		UpdatedAt:    time.Now(),
		Timeout:      at.Timeout,
		RetryCount:   at.RetryCount,
		MaxRetries:   at.MaxRetries,
		Verification: verificationSpec,
		Metadata:     metadata,
	}

	if at.Prompt != "" && at.Description == "" {
		t.Name = at.Prompt
		t.Description = at.Prompt
	}

	for _, dep := range at.Dependencies {
		t.Dependencies = append(t.Dependencies, TaskID(dep))
	}

	if !at.StartedAt.IsZero() {
		st := at.StartedAt
		t.StartedAt = &st
	}
	if !at.CompletedAt.IsZero() {
		ct := at.CompletedAt
		t.CompletedAt = &ct
	}

	// Map output from api.TaskResult if available.
	if at.Result != nil && at.Result.Output != nil {
		if outputMap, ok := at.Result.Output.(map[string]any); ok {
			t.Output = outputMap
		} else {
			t.Output = map[string]any{"result": at.Result.Output}
		}
	}

	return t
}

// TaskToAPITask converts an internal task.Task back to an api.Task.
func TaskToAPITask(t *Task, parentID string, agentType api.AgentType) *api.Task {
	if t == nil {
		return nil
	}

	prompt := ""
	if p, ok := t.Input["prompt"].(string); ok {
		prompt = p
	}

	expectedOutput := ""
	if eo, ok := t.Input["expected_output"].(string); ok {
		expectedOutput = eo
	}

	var requirements []string
	if reqs, ok := t.Input["requirements"].([]string); ok {
		requirements = reqs
	} else if reqs, ok := t.Input["requirements"].([]any); ok {
		for _, r := range reqs {
			if s, ok := r.(string); ok {
				requirements = append(requirements, s)
			}
		}
	}

	deps := make([]string, 0, len(t.Dependencies))
	for _, dep := range t.Dependencies {
		deps = append(deps, string(dep))
	}

	at := &api.Task{
		ID:             string(t.ID),
		ParentID:       parentID,
		Priority:       int(t.Priority),
		Status:         mapInternalStatusToAPI(t.Status),
		AgentType:      agentType,
		Prompt:         prompt,
		Description:    t.Description,
		Requirements:   requirements,
		ExpectedOutput: expectedOutput,
		Dependencies:   deps,
		MaxRetries:     t.MaxRetries,
		RetryCount:     t.RetryCount,
		Timeout:        t.Timeout,
		CreatedAt:      t.CreatedAt,
		Metadata:       t.Metadata,
	}

	if t.StartedAt != nil {
		at.StartedAt = *t.StartedAt
	}
	if t.CompletedAt != nil {
		at.CompletedAt = *t.CompletedAt
	}

	return at
}

// TasksToAPITasks converts a slice of internal tasks to api.Task slice.
func TasksToAPITasks(tasks []*Task, parentID string, agentType api.AgentType) []*api.Task {
	result := make([]*api.Task, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, TaskToAPITask(t, parentID, agentType))
	}
	return result
}

// ObjectiveFromAPITask builds a Planner Objective from an api.Task.
func ObjectiveFromAPITask(at *api.Task) Objective {
	goals := make([]Goal, 0)

	// Build goals from requirements.
	for i, req := range at.Requirements {
		goals = append(goals, Goal{
			ID:              fmt.Sprintf("goal-%s-%d", at.ID, i),
			Description:     req,
			SuccessCriteria: req,
		})
	}

	// If no requirements, create a single goal from the prompt.
	if len(goals) == 0 {
		goals = append(goals, Goal{
			ID:              fmt.Sprintf("goal-%s-0", at.ID),
			Description:     at.Prompt,
			SuccessCriteria: at.ExpectedOutput,
		})
	}

	ctx := make(map[string]any)
	if at.Metadata != nil {
		for k, v := range at.Metadata {
			ctx[k] = v
		}
	}
	ctx["prompt"] = at.Prompt
	ctx["description"] = at.Description

	return Objective{
		ID:          at.ID,
		Description: at.Description,
		Goals:       goals,
		Context:     ctx,
	}
}

// mapAPIStatusToInternal maps an api.TaskStatus to an internal task.TaskStatus.
func mapAPIStatusToInternal(s api.TaskStatus) TaskStatus {
	switch s {
	case api.TaskStatusPending:
		return StatusPending
	case api.TaskStatusQueued:
		return StatusReady
	case api.TaskStatusRunning:
		return StatusRunning
	case api.TaskStatusCompleted:
		return StatusCompleted
	case api.TaskStatusFailed:
		return StatusFailed
	case api.TaskStatusCancelled:
		return StatusCancelled
	case api.TaskStatusBlocked:
		return StatusBlocked
	case api.TaskStatusVerifying:
		return StatusRunning // verifying maps to running internally
	default:
		return StatusPending
	}
}

// mapInternalStatusToAPI maps an internal task.TaskStatus to an api.TaskStatus.
func mapInternalStatusToAPI(s TaskStatus) api.TaskStatus {
	switch s {
	case StatusPending:
		return api.TaskStatusPending
	case StatusReady:
		return api.TaskStatusQueued
	case StatusRunning:
		return api.TaskStatusRunning
	case StatusCompleted:
		return api.TaskStatusCompleted
	case StatusFailed:
		return api.TaskStatusFailed
	case StatusCancelled:
		return api.TaskStatusCancelled
	case StatusBlocked:
		return api.TaskStatusBlocked
	default:
		return api.TaskStatusPending
	}
}

// verificationSpecFromMap builds a VerificationSpec from a generic map (typically
// deserialized from JSON in api.Task.Verification).
func verificationSpecFromMap(m map[string]any) *VerificationSpec {
	spec := &VerificationSpec{}

	if t, ok := m["type"].(string); ok {
		spec.Type = VerificationType(t)
	}

	if assertions, ok := m["assertions"].([]any); ok {
		for _, a := range assertions {
			if am, ok := a.(map[string]any); ok {
				assertion := Assertion{}
				if name, ok := am["name"].(string); ok {
					assertion.Name = name
				}
				if typ, ok := am["type"].(string); ok {
					assertion.Type = typ
				}
				if expected, ok := am["expected"]; ok {
					assertion.Expected = expected
				}
				if path, ok := am["path"].(string); ok {
					assertion.Path = path
				}
				if msg, ok := am["message"].(string); ok {
					assertion.Message = msg
				}
				spec.Assertions = append(spec.Assertions, assertion)
			}
		}
	}

	return spec
}
