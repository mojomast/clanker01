# Handoff
## Completed: Harden config manager and task orchestration wiring
## Next Task: N/A
## Context:
Applied 10 fixes across 6 files to harden config management and task orchestration.
All fixes verified with `go build ./...` and `go vet ./...` — both pass cleanly.

## Files Modified:
- `internal/config/manager.go`
- `internal/core/orchestrator/coordinator.go`
- `internal/core/orchestrator/scheduler.go`
- `internal/core/task/verifier.go`
- `internal/core/task/bridge.go`
- `internal/config/validation.go`

## Changes Made:

### internal/config/manager.go (4 fixes)
1. **CRITICAL: Race in watchLoop** — `m.watcher.stopChan` was accessed without lock in select; Close() nils `m.watcher` under lock. Fixed by capturing `watcherStop` channel under RLock before entering loop.
2. **CRITICAL: Race in checkConfigChange** — `m.watcher` could be nil after Close(). Fixed by reading `m.watcher` under RLock, nil-checking before use.
3. **HIGH: Update() mutates in-place** — If updater modifies config and Validate fails, config was left partially mutated. Fixed by deep-copying via JSON marshal/unmarshal before applying updater. Added `deepCopyConfig()` helper.
4. **MEDIUM: Config file permissions** — Changed `0644` to `0600` on both saveYAML and saveJSON for sensitive config files.

### internal/core/orchestrator/coordinator.go (2 fixes)
5. **CRITICAL: Infinite recursion in SubmitTask** — `shouldPlan` could trigger for subtasks, causing recursive planning forever. Fixed by adding `t.ParentID == ""` guard — only top-level tasks are planned.
6. **MEDIUM: Parent marked completed before subtasks run** — Changed `api.TaskStatusCompleted` to `api.TaskStatusBlocked` so parent waits for subtask completion.

### internal/core/orchestrator/scheduler.go (1 fix)
7. **HIGH: Silent re-enqueue failure** — `_ = s.taskQueue.Enqueue(...)` discarded errors, losing tasks. Fixed to call `s.taskQueue.Fail()` if re-enqueue fails.

### internal/core/task/verifier.go (2 fixes)
8. **MEDIUM: Nil task in Verify** — Added nil guard returning invalid VerificationResult.
9. **HIGH: Unguarded type assertion** — `r.(string)` in validateObject panics on non-string. Fixed with safe `r, ok := r.(string)` + continue.

### internal/core/task/bridge.go (1 fix)
10. **MEDIUM: Nil input in ObjectiveFromAPITask** — Added nil guard returning empty Objective.

### internal/config/validation.go (1 fix)
11. **MEDIUM: Empty errors slice** — `formatValidationErrors` now guards against empty slice, returning "validation failed: unknown error".
