package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestHotStore_BasicOperations(t *testing.T) {
	store := NewHotStore(1024 * 1024)
	require.NotNil(t, store)

	entry := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerSession,
		Content:   "test content",
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"foo": "bar"},
	}

	err := store.Set(entry)
	require.NoError(t, err)

	retrieved, found := store.Get("test-key")
	require.True(t, found)
	assert.Equal(t, "test-key", retrieved.Key)
	assert.Equal(t, api.ContextLayerSession, retrieved.Layer)
	assert.Equal(t, "test content", retrieved.Content)

	store.Delete("test-key")
	_, found = store.Get("test-key")
	assert.False(t, found)
}

func TestHotStore_LRUEviction(t *testing.T) {
	store := NewHotStore(2000)
	require.NotNil(t, store)

	entries := make([]*api.ContextEntry, 10)
	for i := 0; i < 10; i++ {
		entries[i] = &api.ContextEntry{
			Key:       "key-" + string(rune('0'+i)),
			Layer:     api.ContextLayerSession,
			Content:   "content-xxxxx",
			CreatedAt: time.Now(),
		}
		require.NoError(t, store.Set(entries[i]))
	}

	initialSize := store.Len()
	initialBytes := store.Size()
	assert.Equal(t, 10, initialSize)
	t.Logf("Initial size: %d entries, %d bytes", initialSize, initialBytes)

	for i := 0; i < 5; i++ {
		retrieved, found := store.Get("key-" + string(rune('0'+i)))
		assert.True(t, found)
		assert.NotNil(t, retrieved)
		assert.Equal(t, entries[i].Key, retrieved.Key)
	}

	longContent := "very long content to trigger eviction with lots and lots of data that will definitely cause evictions with even more data and more and more and more"
	for len(longContent) < 1000 {
		longContent += " " + longContent
	}

	store.Set(&api.ContextEntry{
		Key:       "new-key",
		Layer:     api.ContextLayerSession,
		Content:   longContent,
		CreatedAt: time.Now(),
	})

	t.Logf("Final size: %d entries, %d bytes", store.Len(), store.Size())
	assert.Less(t, store.Len(), initialSize)
}

func TestHotStore_UpdateExisting(t *testing.T) {
	store := NewHotStore(1024 * 1024)

	entry1 := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerSession,
		Content:   "original content",
		CreatedAt: time.Now(),
	}
	require.NoError(t, store.Set(entry1))

	entry2 := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerSession,
		Content:   "updated content",
		CreatedAt: time.Now(),
	}
	require.NoError(t, store.Set(entry2))

	retrieved, _ := store.Get("test-key")
	assert.Equal(t, "updated content", retrieved.Content)
	assert.Equal(t, 1, store.Len())
}

func TestHotStore_Clear(t *testing.T) {
	store := NewHotStore(1024 * 1024)

	for i := 0; i < 5; i++ {
		entry := &api.ContextEntry{
			Key:       "key-" + string(rune('0'+i)),
			Layer:     api.ContextLayerSession,
			Content:   "content",
			CreatedAt: time.Now(),
		}
		require.NoError(t, store.Set(entry))
	}

	assert.Equal(t, 5, store.Len())
	assert.Greater(t, store.Size(), int64(0))

	store.Clear()

	assert.Equal(t, 0, store.Len())
	assert.Equal(t, int64(0), store.Size())
}

func TestWarmStore_BasicOperations(t *testing.T) {
	store := NewWarmStore(5 * time.Minute)
	require.NotNil(t, store)

	entry := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerAgent,
		Content:   "test content",
		CreatedAt: time.Now(),
	}

	err := store.Set(entry)
	require.NoError(t, err)

	retrieved, err := store.Get("test-key")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "test-key", retrieved.Key)
	assert.Equal(t, api.ContextLayerAgent, retrieved.Layer)
}

func TestWarmStore_NotFound(t *testing.T) {
	store := NewWarmStore(5 * time.Minute)

	retrieved, err := store.Get("non-existent")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestWarmStore_Delete(t *testing.T) {
	store := NewWarmStore(5 * time.Minute)

	entry := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerAgent,
		Content:   "test content",
		CreatedAt: time.Now(),
	}
	require.NoError(t, store.Set(entry))

	err := store.Delete("test-key")
	require.NoError(t, err)

	retrieved, _ := store.Get("test-key")
	assert.Nil(t, retrieved)
}

func TestWarmStore_Exists(t *testing.T) {
	store := NewWarmStore(5 * time.Minute)

	entry := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerAgent,
		Content:   "test content",
		CreatedAt: time.Now(),
	}
	require.NoError(t, store.Set(entry))

	exists, err := store.Exists("test-key")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.Exists("non-existent")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestWarmStore_Clear(t *testing.T) {
	store := NewWarmStore(5 * time.Minute)

	for i := 0; i < 5; i++ {
		entry := &api.ContextEntry{
			Key:       "key-" + string(rune('0'+i)),
			Layer:     api.ContextLayerAgent,
			Content:   "content",
			CreatedAt: time.Now(),
		}
		require.NoError(t, store.Set(entry))
	}

	store.Clear()

	for i := 0; i < 5; i++ {
		retrieved, _ := store.Get("key-" + string(rune('0'+i)))
		assert.Nil(t, retrieved)
	}
}

func TestColdStore_BasicOperations(t *testing.T) {
	store := NewColdStore()
	require.NotNil(t, store)

	entry := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerProject,
		Content:   "test content",
		Embedding: []float64{0.1, 0.2, 0.3},
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"type": "code"},
	}

	err := store.Set(entry)
	require.NoError(t, err)

	retrieved, err := store.Get("test-key")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "test-key", retrieved.Key)
	assert.Equal(t, api.ContextLayerProject, retrieved.Layer)
	assert.Equal(t, []float64{0.1, 0.2, 0.3}, retrieved.Embedding)
}

func TestColdStore_NotFound(t *testing.T) {
	store := NewColdStore()

	retrieved, err := store.Get("non-existent")
	require.Error(t, err)
	assert.Nil(t, retrieved)
}

func TestColdStore_Delete(t *testing.T) {
	store := NewColdStore()

	entry := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerProject,
		Content:   "test content",
		CreatedAt: time.Now(),
	}
	require.NoError(t, store.Set(entry))

	err := store.Delete("test-key")
	require.NoError(t, err)

	_, err = store.Get("test-key")
	assert.Error(t, err)
}

func TestColdStore_Query(t *testing.T) {
	store := NewColdStore()

	entry1 := &api.ContextEntry{
		Key:       "session-key",
		Layer:     api.ContextLayerSession,
		Content:   "session data",
		CreatedAt: time.Now(),
	}
	entry2 := &api.ContextEntry{
		Key:       "project-key",
		Layer:     api.ContextLayerProject,
		Content:   "project data",
		CreatedAt: time.Now(),
	}

	require.NoError(t, store.Set(entry1))
	require.NoError(t, store.Set(entry2))

	results, err := store.Query(&api.ContextQuery{Layer: api.ContextLayerSession})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "session-key", results[0].Key)
}

func TestColdStore_GetAll(t *testing.T) {
	store := NewColdStore()

	for i := 0; i < 3; i++ {
		entry := &api.ContextEntry{
			Key:       "key-" + string(rune('0'+i)),
			Layer:     api.ContextLayerProject,
			Content:   "content",
			CreatedAt: time.Now(),
		}
		require.NoError(t, store.Set(entry))
	}

	all := store.GetAll()
	assert.Len(t, all, 3)
}

func TestColdStore_Clear(t *testing.T) {
	store := NewColdStore()

	for i := 0; i < 5; i++ {
		entry := &api.ContextEntry{
			Key:       "key-" + string(rune('0'+i)),
			Layer:     api.ContextLayerProject,
			Content:   "content",
			CreatedAt: time.Now(),
		}
		require.NoError(t, store.Set(entry))
	}

	store.Clear()

	all := store.GetAll()
	assert.Len(t, all, 0)
}

func TestSnapshotManager_SaveAndLoad(t *testing.T) {
	manager := NewSnapshotManager()
	require.NotNil(t, manager)

	snapshot := &api.ContextSnapshot{
		Layer: api.ContextLayerSession,
		Entries: []*api.ContextEntry{
			{
				Key:       "key1",
				Layer:     api.ContextLayerSession,
				Content:   "content1",
				CreatedAt: time.Now(),
			},
			{
				Key:       "key2",
				Layer:     api.ContextLayerSession,
				Content:   "content2",
				CreatedAt: time.Now(),
			},
		},
	}

	saved := manager.Save(snapshot)
	require.NotNil(t, saved)
	assert.NotEmpty(t, saved.Checksum)
	assert.NotEmpty(t, saved.TakenAt)

	snapshotID := fmt.Sprintf("%s-%s", saved.Layer, saved.Checksum[:8])
	loaded, err := manager.Load(snapshotID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, snapshot.Layer, loaded.Layer)
	assert.Len(t, loaded.Entries, 2)
}

func TestSnapshotManager_List(t *testing.T) {
	manager := NewSnapshotManager()

	snapshot1 := &api.ContextSnapshot{
		Layer:   api.ContextLayerSession,
		Entries: []*api.ContextEntry{},
	}
	snapshot2 := &api.ContextSnapshot{
		Layer:   api.ContextLayerAgent,
		Entries: []*api.ContextEntry{},
	}

	manager.Save(snapshot1)
	manager.Save(snapshot2)

	snapshots := manager.List()
	assert.Len(t, snapshots, 2)
}

func TestSnapshotManager_Delete(t *testing.T) {
	manager := NewSnapshotManager()

	snapshot := &api.ContextSnapshot{
		Layer:   api.ContextLayerSession,
		Entries: []*api.ContextEntry{},
	}

	saved := manager.Save(snapshot)
	id := fmt.Sprintf("%s-%s", saved.Layer, saved.Checksum[:8])

	err := manager.Delete(id)
	require.NoError(t, err)

	_, err = manager.Load(id)
	assert.Error(t, err)
}

func TestSnapshotManager_Checkpoint(t *testing.T) {
	manager := NewSnapshotManager()
	store := NewTieredContextStore(1024*1024, 5*time.Minute)

	entry := &api.ContextEntry{
		Key:       "test-key",
		Layer:     api.ContextLayerSession,
		Content:   "test content",
		CreatedAt: time.Now(),
	}
	require.NoError(t, store.Set(context.Background(), entry))

	checkpoint, err := manager.CreateCheckpoint(store)
	require.NoError(t, err)
	require.NotNil(t, checkpoint)
	assert.NotEmpty(t, checkpoint.ID)
	assert.NotEmpty(t, checkpoint.Layers)

	store.Clear()

	err = manager.RestoreCheckpoint(checkpoint, store)
	require.NoError(t, err)

	retrieved, err := store.Get(context.Background(), "test-key", api.ContextLayerSession)
	require.NoError(t, err)
	assert.Equal(t, "test-key", retrieved.Key)
}

func TestSnapshotManager_Clear(t *testing.T) {
	manager := NewSnapshotManager()

	snapshot := &api.ContextSnapshot{
		Layer:   api.ContextLayerSession,
		Entries: []*api.ContextEntry{},
	}
	manager.Save(snapshot)

	manager.Clear()

	snapshots := manager.List()
	assert.Len(t, snapshots, 0)
}
