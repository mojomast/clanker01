package registry

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type SkillRegistry struct {
	mu      sync.RWMutex
	skills  map[string]*SkillManifest
	index   *SkillIndex
	sources []DiscoverySource
}

func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills:  make(map[string]*SkillManifest),
		index:   NewSkillIndex(),
		sources: []DiscoverySource{},
	}
}

func (r *SkillRegistry) AddSource(source DiscoverySource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.sources {
		if s.Name() == source.Name() {
			return fmt.Errorf("source already exists: %s", source.Name())
		}
	}

	r.sources = append(r.sources, source)
	return nil
}

func (r *SkillRegistry) RemoveSource(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, s := range r.sources {
		if s.Name() == name {
			r.sources = append(r.sources[:i], r.sources[i+1:]...)
			return
		}
	}
}

func (r *SkillRegistry) DiscoverAll(ctx context.Context) ([]*SkillManifest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var allManifests []*SkillManifest
	for _, source := range r.sources {
		manifests, err := source.Discover(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}
		allManifests = append(allManifests, manifests...)
	}

	return allManifests, nil
}

func (r *SkillRegistry) Register(manifest *SkillManifest) error {
	if manifest == nil {
		return fmt.Errorf("cannot register nil manifest")
	}

	id := manifest.ID()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.skills[id]; ok {
		return fmt.Errorf("skill already registered: %s", id)
	}

	r.skills[id] = manifest
	r.index.Add(manifest)

	return nil
}

func (r *SkillRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.skills[id]; !ok {
		return fmt.Errorf("skill not found: %s", id)
	}

	delete(r.skills, id)
	r.index.Remove(id)

	return nil
}

func (r *SkillRegistry) Get(name string, version string) (*SkillManifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if version != "" {
		id := name + "@" + version
		if m, ok := r.skills[id]; ok {
			return m, nil
		}
		return nil, fmt.Errorf("skill not found: %s", id)
	}

	var latest *SkillManifest
	for id, m := range r.skills {
		prefix := name + "@"
		if strings.HasPrefix(id, prefix) {
			if latest == nil || compareVersions(m.Metadata.Version, latest.Metadata.Version) > 0 {
				latest = m
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	return latest, nil
}

func (r *SkillRegistry) GetByID(id string) (*SkillManifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.skills[id]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", id)
	}

	return m, nil
}

func (r *SkillRegistry) List(limit int) []*SkillMatch {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.index.ListAll()
	if limit > 0 && limit < len(ids) {
		ids = ids[:limit]
	}

	var matches []*SkillMatch
	for _, id := range ids {
		_, ok := r.skills[id]
		if !ok {
			continue
		}
		matches = append(matches, &SkillMatch{
			SkillID: id,
			Score:   1.0,
		})
	}

	return matches
}

func (r *SkillRegistry) ListByTag(tag string, limit int) []*SkillMatch {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.index.GetByTag(tag)
	if limit > 0 && limit < len(ids) {
		ids = ids[:limit]
	}

	var matches []*SkillMatch
	for _, id := range ids {
		_, ok := r.skills[id]
		if !ok {
			continue
		}
		matches = append(matches, &SkillMatch{
			SkillID: id,
			Score:   1.0,
		})
	}

	return matches
}

func (r *SkillRegistry) ListByRuntime(runtime string, limit int) []*SkillMatch {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.index.GetByRuntime(runtime)
	if limit > 0 && limit < len(ids) {
		ids = ids[:limit]
	}

	var matches []*SkillMatch
	for _, id := range ids {
		_, ok := r.skills[id]
		if !ok {
			continue
		}
		matches = append(matches, &SkillMatch{
			SkillID: id,
			Score:   1.0,
		})
	}

	return matches
}

func (r *SkillRegistry) Discover(query string, limit int) []*SkillMatch {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matches := make(map[string]*SkillMatch)

	for id, manifest := range r.skills {
		score := r.calculateScore(query, manifest)
		if score > 0 {
			if _, ok := matches[id]; !ok {
				matches[id] = &SkillMatch{
					SkillID: id,
					Score:   score,
					Context: map[string]any{},
				}
			}
		}
	}

	sorted := make([]*SkillMatch, 0, len(matches))
	for _, m := range matches {
		sorted = append(sorted, m)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}

	return sorted
}

func (r *SkillRegistry) calculateScore(query string, manifest *SkillManifest) float64 {
	var totalScore float64

	nameScore := fuzzyMatch(query, manifest.Metadata.Name)
	if nameScore > 0 {
		totalScore += nameScore * 0.8
	}

	displayNameScore := fuzzyMatch(query, manifest.Metadata.DisplayName)
	if displayNameScore > 0 {
		totalScore += displayNameScore * 0.7
	}

	descriptionScore := fuzzyMatch(query, manifest.Metadata.Description)
	if descriptionScore > 0 {
		totalScore += descriptionScore * 0.3
	}

	for _, tag := range manifest.Metadata.Tags {
		tagScore := fuzzyMatch(query, tag)
		if tagScore > 0 {
			totalScore += tagScore * 0.5
		}
	}

	for _, trigger := range manifest.Spec.Triggers {
		for _, pattern := range trigger.Patterns {
			if regexMatch(query, pattern) {
				totalScore += 0.4
			}
		}
	}

	if totalScore > 1.0 {
		totalScore = 1.0
	}

	return totalScore
}

func (r *SkillRegistry) Versions(name string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var versions []string
	prefix := name + "@"

	for id := range r.skills {
		if strings.HasPrefix(id, prefix) {
			parts := strings.Split(id, "@")
			if len(parts) == 2 {
				versions = append(versions, parts[1])
			}
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for skill: %s", name)
	}

	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})

	return versions, nil
}

func (r *SkillRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.index.Count()
}

func (r *SkillRegistry) IsRegistered(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.index.Has(id)
}

func (r *SkillRegistry) Sources() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.sources))
	for i, source := range r.sources {
		names[i] = source.Name()
	}
	return names
}

func (r *SkillRegistry) AutoDiscover(ctx context.Context) error {
	manifests, err := r.DiscoverAll(ctx)
	if err != nil {
		return fmt.Errorf("discover skills: %w", err)
	}

	for _, manifest := range manifests {
		if err := r.Register(manifest); err != nil {
			continue
		}
	}

	return nil
}
