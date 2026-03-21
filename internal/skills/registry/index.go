package registry

import (
	"sort"
	"strings"
	"sync"
)

type SkillIndex struct {
	mu        sync.RWMutex
	byName    map[string][]string
	byTag     map[string][]string
	byTrigger map[string][]string
	bySource  map[string][]string
	byRuntime map[string][]string
	skillIDs  map[string]bool
}

func NewSkillIndex() *SkillIndex {
	return &SkillIndex{
		byName:    make(map[string][]string),
		byTag:     make(map[string][]string),
		byTrigger: make(map[string][]string),
		bySource:  make(map[string][]string),
		byRuntime: make(map[string][]string),
		skillIDs:  make(map[string]bool),
	}
}

func (idx *SkillIndex) Add(manifest *SkillManifest) {
	id := manifest.ID()

	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.skillIDs[id] {
		return
	}
	idx.skillIDs[id] = true

	idx.byName[manifest.Metadata.Name] = appendUnique(
		idx.byName[manifest.Metadata.Name], id)

	for _, tag := range manifest.Metadata.Tags {
		idx.byTag[tag] = appendUnique(idx.byTag[tag], id)
	}

	for _, trigger := range manifest.Spec.Triggers {
		for _, pattern := range trigger.Patterns {
			idx.byTrigger[pattern] = appendUnique(idx.byTrigger[pattern], id)
		}
	}

	if manifest.Source != "" {
		idx.bySource[manifest.Source] = appendUnique(idx.bySource[manifest.Source], id)
	}

	if manifest.Spec.Runtime != "" {
		idx.byRuntime[manifest.Spec.Runtime] = appendUnique(idx.byRuntime[manifest.Spec.Runtime], id)
	}
}

func (idx *SkillIndex) Remove(id string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if !idx.skillIDs[id] {
		return
	}
	delete(idx.skillIDs, id)

	for name := range idx.byName {
		idx.byName[name] = removeID(idx.byName[name], id)
	}

	for tag := range idx.byTag {
		idx.byTag[tag] = removeID(idx.byTag[tag], id)
	}

	for pattern := range idx.byTrigger {
		idx.byTrigger[pattern] = removeID(idx.byTrigger[pattern], id)
	}

	for source := range idx.bySource {
		idx.bySource[source] = removeID(idx.bySource[source], id)
	}

	for runtime := range idx.byRuntime {
		idx.byRuntime[runtime] = removeID(idx.byRuntime[runtime], id)
	}
}

func (idx *SkillIndex) GetByName(name string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]string, len(idx.byName[name]))
	copy(result, idx.byName[name])
	return result
}

func (idx *SkillIndex) GetByTag(tag string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]string, len(idx.byTag[tag]))
	copy(result, idx.byTag[tag])
	return result
}

func (idx *SkillIndex) GetByTrigger(pattern string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]string, len(idx.byTrigger[pattern]))
	copy(result, idx.byTrigger[pattern])
	return result
}

func (idx *SkillIndex) GetBySource(source string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]string, len(idx.bySource[source]))
	copy(result, idx.bySource[source])
	return result
}

func (idx *SkillIndex) GetByRuntime(runtime string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]string, len(idx.byRuntime[runtime]))
	copy(result, idx.byRuntime[runtime])
	return result
}

func (idx *SkillIndex) ListAll() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ids := make([]string, 0, len(idx.skillIDs))
	for id := range idx.skillIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (idx *SkillIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.skillIDs)
}

func (idx *SkillIndex) Has(id string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.skillIDs[id]
}

func appendUnique(slice []string, id string) []string {
	for _, existing := range slice {
		if existing == id {
			return slice
		}
	}
	return append(slice, id)
}

func removeID(slice []string, id string) []string {
	for i, existing := range slice {
		if existing == id {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func fuzzyMatch(query, target string) float64 {
	if query == "" || target == "" {
		return 0.0
	}

	query = strings.ToLower(query)
	target = strings.ToLower(target)

	if query == target {
		return 1.0
	}

	if strings.Contains(target, query) {
		return 0.9
	}

	queryWords := strings.Fields(query)
	targetWords := strings.Fields(target)

	matchCount := 0
	for _, qw := range queryWords {
		for _, tw := range targetWords {
			if strings.Contains(tw, qw) {
				matchCount++
				break
			}
		}
	}

	if matchCount > 0 {
		return float64(matchCount) / float64(len(queryWords)) * 0.8
	}

	return 0.0
}

func regexMatch(input, pattern string) bool {
	lowerInput := strings.ToLower(input)
	lowerPattern := strings.ToLower(pattern)

	if !strings.Contains(lowerPattern, ".*") {
		return strings.Contains(lowerInput, lowerPattern)
	}

	parts := strings.Split(lowerPattern, ".*")

	if !strings.HasPrefix(lowerInput, parts[0]) {
		return false
	}

	remaining := lowerInput[len(parts[0]):]

	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			continue
		}
		idx := strings.Index(remaining, part)
		if idx == -1 {
			return false
		}
		if i == len(parts)-1 {
			remaining = remaining[idx:]
			if remaining != part {
				return false
			}
		} else {
			remaining = remaining[idx+len(part):]
		}
	}

	return true
}
