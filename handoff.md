# Handoff
## Completed: Fix critical race conditions, path traversal, and lock ordering bugs
## Next Task: N/A
## Context:
Applied 3 categories of fixes across 3 files to address critical and high-severity bugs.

## Files Modified:
- `internal/config/manager.go`
- `internal/skills/builtin/web/skill.go`
- `internal/server/rest/handlers.go`

## Changes Made:

### internal/config/manager.go (Issue 1: watchLoop race on errorChan close)
1. **Removed `close(m.errorChan)` and `m.errorChan = nil` from Close()** — The watchLoop goroutine could send on a closed channel causing a panic. The stopChan close already signals the goroutine to exit; errorChan will be GC'd.
2. **Captured `m.stopChan` as local `stopChan` in watchLoop** — Close() sets `m.stopChan = nil` which would cause the select to read a nil field without lock protection. Now both `watcherStop` and `stopChan` are captured under RLock before entering the loop.
3. **Removed stale `m.errorChan != nil` check** — Since errorChan is no longer nilled, the nil check is unnecessary.

### internal/skills/builtin/web/skill.go (Issue 2: path traversal in download)
1. **Added path sanitization** — Clean with `filepath.Clean`, resolve relative paths to absolute, reject paths containing `..` after cleaning, and verify the resolved path is within the current working directory.

### internal/server/rest/handlers.go (Issue 3: response built after unlock)
1. **Moved response construction before `s.mu.Unlock()` in 9 handlers:**
   - `handleUpdateAgent`, `handleStartAgent`, `handleStopAgent`, `handlePauseAgent`, `handleResumeAgent`
   - `handleUpdateTask`, `handleStartTask`, `handleCancelTask`
   - `handleUpdateSkill`
   
   Previously, the lock was released before building the response struct from the mutated object, allowing another goroutine to modify it between unlock and response construction.
