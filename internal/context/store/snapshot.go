package store

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type SnapshotManager struct {
	snapshots map[string]*api.ContextSnapshot
	mu        sync.RWMutex
	snapDir   string
}

type Checkpoint struct {
	ID        string
	Timestamp time.Time
	Layers    map[api.ContextLayer]string
	Metadata  map[string]any
}

func NewSnapshotManager() *SnapshotManager {
	sm := &SnapshotManager{
		snapshots: make(map[string]*api.ContextSnapshot),
		snapDir:   filepath.Join(os.TempDir(), "swarm-snapshots"),
	}
	os.MkdirAll(sm.snapDir, 0755)
	return sm
}

func (s *SnapshotManager) Save(snapshot *api.ContextSnapshot) *api.ContextSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot.TakenAt = time.Now()
	snapshot.Checksum = s.calculateChecksum(snapshot)
	snapshotsID := s.generateSnapshotID(snapshot)

	s.snapshots[snapshotsID] = snapshot

	return snapshot
}

func (s *SnapshotManager) Load(id string) (*api.ContextSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, ok := s.snapshots[id]
	if !ok {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}

	return snapshot, nil
}

func (s *SnapshotManager) List() []*api.ContextSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshots := make([]*api.ContextSnapshot, 0, len(s.snapshots))
	for _, snap := range s.snapshots {
		snapshots = append(snapshots, snap)
	}
	return snapshots
}

func (s *SnapshotManager) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.snapshots[id]; !ok {
		return fmt.Errorf("snapshot not found: %s", id)
	}

	delete(s.snapshots, id)
	return nil
}

func (s *SnapshotManager) CreateCheckpoint(store *TieredContextStore) (*Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	checkpoint := &Checkpoint{
		ID:        s.generateCheckpointID(),
		Timestamp: time.Now(),
		Layers:    make(map[api.ContextLayer]string),
		Metadata:  make(map[string]any),
	}

	layers := []api.ContextLayer{
		api.ContextLayerSession,
		api.ContextLayerAgent,
		api.ContextLayerProject,
		api.ContextLayerKnowledge,
	}

	for _, layer := range layers {
		currentLayer := layer
		if currentLayer == "" {
			continue
		}
		snapshot, err := store.Export(context.Background(), currentLayer)
		if err != nil {
			continue
		}
		snapshotID := s.generateSnapshotID(snapshot)
		s.snapshots[snapshotID] = snapshot
		if checkpoint.Layers == nil {
			checkpoint.Layers = make(map[api.ContextLayer]string)
		}
		checkpoint.Layers[currentLayer] = snapshotID
	}

	checkpoint.Metadata["entries_count"] = len(store.hot.GetAll())
	checkpoint.Metadata["hot_size"] = store.hot.Size()

	return checkpoint, nil
}

func (s *SnapshotManager) RestoreCheckpoint(checkpoint *Checkpoint, store *TieredContextStore) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, snapshotID := range checkpoint.Layers {
		snapshot, ok := s.snapshots[snapshotID]
		if !ok {
			continue
		}

		for _, entry := range snapshot.Entries {
			if err := store.Set(context.Background(), entry); err != nil {
				return fmt.Errorf("failed to restore entry %s: %w", entry.Key, err)
			}
		}
	}

	return nil
}

func (s *SnapshotManager) PersistToFile(snapshot *api.ContextSnapshot, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(snapshot); err != nil {
		return err
	}

	return nil
}

func (s *SnapshotManager) LoadFromFile(path string) (*api.ContextSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var snapshot api.ContextSnapshot
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, err
	}

	snapshotID := s.generateSnapshotID(&snapshot)
	s.snapshots[snapshotID] = &snapshot

	return &snapshot, nil
}

func (s *SnapshotManager) calculateChecksum(snapshot *api.ContextSnapshot) string {
	data := fmt.Sprintf("%s|%d|%d", snapshot.Layer, len(snapshot.Entries), snapshot.TakenAt.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *SnapshotManager) generateSnapshotID(snapshot *api.ContextSnapshot) string {
	return fmt.Sprintf("%s-%s", snapshot.Layer, snapshot.Checksum[:8])
}

func (s *SnapshotManager) generateCheckpointID() string {
	return fmt.Sprintf("ckpt-%d", time.Now().UnixNano())
}

func (s *SnapshotManager) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots = make(map[string]*api.ContextSnapshot)
}
