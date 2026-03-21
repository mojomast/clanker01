package agent

import (
	"context"
	"fmt"
	"sync"
)

type SkillManager struct {
	mu     sync.RWMutex
	skills map[string]*Skill
}

type Skill struct {
	Name        string
	Description string
	Config      map[string]any
	Enabled     bool
}

func NewSkillManager() *SkillManager {
	return &SkillManager{
		skills: make(map[string]*Skill),
	}
}

func (sm *SkillManager) Load(ctx context.Context, skillName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.skills[skillName]; exists {
		return nil
	}

	skill := &Skill{
		Name:        skillName,
		Description: fmt.Sprintf("Skill: %s", skillName),
		Config:      make(map[string]any),
		Enabled:     true,
	}

	sm.skills[skillName] = skill
	return nil
}

func (sm *SkillManager) Unload(ctx context.Context, skillName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.skills, skillName)
	return nil
}

func (sm *SkillManager) GetSkill(skillName string) (*Skill, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skill, exists := sm.skills[skillName]
	return skill, exists
}

func (sm *SkillManager) ListSkills() []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skills := make([]*Skill, 0, len(sm.skills))
	for _, skill := range sm.skills {
		skills = append(skills, skill)
	}
	return skills
}

func (sm *SkillManager) Enable(ctx context.Context, skillName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[skillName]
	if !exists {
		return fmt.Errorf("skill not found: %s", skillName)
	}

	skill.Enabled = true
	return nil
}

func (sm *SkillManager) Disable(ctx context.Context, skillName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[skillName]
	if !exists {
		return fmt.Errorf("skill not found: %s", skillName)
	}

	skill.Enabled = false
	return nil
}
