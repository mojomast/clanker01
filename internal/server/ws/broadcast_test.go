package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBroadcastManager(t *testing.T) {
	hub := NewHub()
	bm := NewBroadcastManager(hub, 4)

	assert.NotNil(t, bm)
	assert.Equal(t, hub, bm.hub)
	assert.Equal(t, 4, bm.workers)
	assert.NotNil(t, bm.queue)
	assert.NotNil(t, bm.stats)
}

func TestBroadcastManagerStartStop(t *testing.T) {
	hub := NewHub()
	bm := NewBroadcastManager(hub, 4)

	bm.Start()
	time.Sleep(50 * time.Millisecond)

	bm.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestBroadcastManagerBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	bm := NewBroadcastManager(hub, 2)
	bm.Start()
	defer bm.Stop()

	client := NewClient(hub, nil, "user-1")
	hub.Register(client)

	time.Sleep(50 * time.Millisecond)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err := bm.Broadcast(msg, &BroadcastTarget{All: true})
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	select {
	case receivedMsg := <-client.send:
		assert.Equal(t, msg.Type, receivedMsg.Type)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("client did not receive message")
	}

	hub.Stop()
}

func TestBroadcastManagerBroadcastAsync(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	bm := NewBroadcastManager(hub, 2)
	bm.Start()
	defer bm.Stop()

	client := NewClient(hub, nil, "user-1")
	hub.Register(client)

	time.Sleep(50 * time.Millisecond)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")

	callbackCalled := false
	var success, failed int

	err := bm.BroadcastAsync(msg, &BroadcastTarget{All: true}, func(s, f int) {
		callbackCalled = true
		success = s
		failed = f
	})
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.True(t, callbackCalled)
	assert.Equal(t, 1, success)
	assert.Equal(t, 0, failed)

	hub.Stop()
}

func TestBroadcastManagerBroadcastToUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	bm := NewBroadcastManager(hub, 2)
	bm.Start()
	defer bm.Stop()

	client1 := NewClient(hub, nil, "user-1")
	client2 := NewClient(hub, nil, "user-2")

	hub.Register(client1)
	hub.Register(client2)

	time.Sleep(50 * time.Millisecond)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err := bm.BroadcastToUser(msg, "user-1")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	select {
	case <-client1.send:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("client1 did not receive message")
	}

	select {
	case <-client2.send:
		t.Fatal("client2 should not receive message")
	case <-time.After(50 * time.Millisecond):
	}

	hub.Stop()
}

func TestBroadcastManagerBroadcastToAgent(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	bm := NewBroadcastManager(hub, 2)
	bm.Start()
	defer bm.Stop()

	client := NewClient(hub, nil, "user-1")
	client.sub.AddAgentID("agent-1")

	hub.Register(client)

	time.Sleep(50 * time.Millisecond)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err := bm.BroadcastToAgent(msg, "agent-1")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	select {
	case receivedMsg := <-client.send:
		assert.Equal(t, msg.Type, receivedMsg.Type)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("client did not receive message")
	}

	hub.Stop()
}

func TestBroadcastManagerBroadcastToTask(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	bm := NewBroadcastManager(hub, 2)
	bm.Start()
	defer bm.Stop()

	client := NewClient(hub, nil, "user-1")
	client.sub.AddTaskID("task-1")

	hub.Register(client)

	time.Sleep(50 * time.Millisecond)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err := bm.BroadcastToTask(msg, "task-1")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	select {
	case receivedMsg := <-client.send:
		assert.Equal(t, msg.Type, receivedMsg.Type)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("client did not receive message")
	}

	hub.Stop()
}

func TestBroadcastManagerStats(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	bm := NewBroadcastManager(hub, 4)
	bm.Start()
	defer bm.Stop()

	client := NewClient(hub, nil, "user-1")
	hub.Register(client)

	time.Sleep(50 * time.Millisecond)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err := bm.Broadcast(msg, &BroadcastTarget{All: true})
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	stats := bm.Stats()
	assert.NotNil(t, stats)
	assert.Equal(t, int64(1), stats.TotalSent)
	assert.Equal(t, int64(0), stats.TotalFailed)
	assert.Equal(t, 0, stats.QueuedJobs)

	hub.Stop()
}

func TestBroadcastManagerQueueSize(t *testing.T) {
	hub := NewHub()
	bm := NewBroadcastManager(hub, 2)

	size := bm.QueueSize()
	assert.Equal(t, 0, size)
}

func TestBroadcastManagerStopped(t *testing.T) {
	hub := NewHub()
	bm := NewBroadcastManager(hub, 2)

	bm.Start()
	bm.Stop()

	time.Sleep(100 * time.Millisecond)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err := bm.Broadcast(msg, &BroadcastTarget{All: true})
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "broadcast manager stopped")
	}
}

func TestNewBufferedBroadcaster(t *testing.T) {
	broadcastCalled := false
	var receivedMsg *Message

	bb := NewBufferedBroadcaster(10, 100*time.Millisecond, func(msg *Message) {
		broadcastCalled = true
		receivedMsg = msg
	})

	assert.NotNil(t, bb)
	assert.Equal(t, 10, bb.maxSize)
	assert.Equal(t, 100*time.Millisecond, bb.interval)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	bb.Add(msg)

	time.Sleep(150 * time.Millisecond)

	assert.True(t, broadcastCalled)
	assert.NotNil(t, receivedMsg)
	assert.Equal(t, msg.Type, receivedMsg.Type)
}

func TestBufferedBroadcasterFlushOnMaxSize(t *testing.T) {
	broadcastCalled := false
	var messages []*Message

	bb := NewBufferedBroadcaster(3, 1*time.Second, func(msg *Message) {
		broadcastCalled = true
		messages = append(messages, msg)
	})

	for i := 0; i < 3; i++ {
		msg := NewLogEntry("info", "test message", "agent-1", "task-1")
		bb.Add(msg)
	}

	time.Sleep(50 * time.Millisecond)

	assert.True(t, broadcastCalled)
	assert.Len(t, messages, 3)
}

func TestBufferedBroadcasterManualFlush(t *testing.T) {
	broadcastCalled := false
	var messages []*Message

	bb := NewBufferedBroadcaster(10, 1*time.Second, func(msg *Message) {
		broadcastCalled = true
		messages = append(messages, msg)
	})

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	bb.Add(msg)

	time.Sleep(50 * time.Millisecond)
	assert.False(t, broadcastCalled)

	bb.Flush()
	time.Sleep(50 * time.Millisecond)

	assert.True(t, broadcastCalled)
	assert.Len(t, messages, 1)
}

func TestBufferedBroadcasterStop(t *testing.T) {
	broadcastCalled := false

	bb := NewBufferedBroadcaster(10, 100*time.Millisecond, func(msg *Message) {
		broadcastCalled = true
	})

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	bb.Add(msg)

	bb.Stop()
	time.Sleep(150 * time.Millisecond)

	assert.True(t, broadcastCalled)
}

func TestNewMessageFilter(t *testing.T) {
	mf := NewMessageFilter()

	assert.NotNil(t, mf)
	assert.Empty(t, mf.filters)
}

func TestMessageFilterAdd(t *testing.T) {
	mf := NewMessageFilter()

	mf.AddFilter("log-only", func(msg *Message) bool {
		return msg.Type == MessageTypeLog
	})

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	assert.True(t, mf.ShouldSend(msg))

	msg2 := NewPing()
	assert.False(t, mf.ShouldSend(msg2))
}

func TestMessageFilterMultiple(t *testing.T) {
	mf := NewMessageFilter()

	mf.AddFilter("type-filter", func(msg *Message) bool {
		return msg.Type == MessageTypeLog || msg.Type == MessageTypeError
	})

	mf.AddFilter("agent-filter", func(msg *Message) bool {
		return msg.AgentID == "agent-1"
	})

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	assert.True(t, mf.ShouldSend(msg))

	msg2 := NewLogEntry("info", "test message", "agent-2", "task-1")
	assert.False(t, mf.ShouldSend(msg2))
}

func TestMessageFilterClear(t *testing.T) {
	mf := NewMessageFilter()

	mf.AddFilter("log-only-2", func(msg *Message) bool {
		return msg.Type == MessageTypeLog
	})

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	assert.True(t, mf.ShouldSend(msg))

	mf.Clear()

	assert.True(t, mf.ShouldSend(msg))
}

func TestNewPriorityBroadcaster(t *testing.T) {
	pb := NewPriorityBroadcaster(10)

	assert.NotNil(t, pb)
	assert.NotNil(t, pb.queues)
}

func TestPriorityBroadcaster(t *testing.T) {
	pb := NewPriorityBroadcaster(10)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")

	err := pb.Broadcast(msg, 1)
	assert.NoError(t, err)

	msg2 := NewPing()
	err = pb.Broadcast(msg2, 0)
	assert.NoError(t, err)

	msg3 := NewError("ERR001", "test error", "")
	err = pb.Broadcast(msg3, 2)
	assert.NoError(t, err)
}

func TestPriorityBroadcasterInvalidPriority(t *testing.T) {
	pb := NewPriorityBroadcaster(10)

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err := pb.Broadcast(msg, 10)
	assert.Error(t, err)
}

func TestPriorityBroadcasterQueueFull(t *testing.T) {
	pb := NewPriorityBroadcaster(2)

	for i := 0; i < 5; i++ {
		msg := NewLogEntry("info", "test message", "agent-1", "task-1")
		_ = pb.Broadcast(msg, 1)
	}

	msg := NewLogEntry("info", "test message", "agent-1", "task-1")
	err := pb.Broadcast(msg, 1)
	assert.Error(t, err)
}

func TestMarshalBroadcastStats(t *testing.T) {
	stats := &BroadcastStats{
		TotalSent:     100,
		TotalFailed:   5,
		QueuedJobs:    10,
		ActiveWorkers: 4,
	}

	data, err := MarshalBroadcastStats(stats)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestUnmarshalBroadcastStats(t *testing.T) {
	stats := &BroadcastStats{
		TotalSent:     100,
		TotalFailed:   5,
		QueuedJobs:    10,
		ActiveWorkers: 4,
	}

	data, err := MarshalBroadcastStats(stats)
	require.NoError(t, err)

	unmarshaledStats, err := UnmarshalBroadcastStats(data)
	assert.NoError(t, err)
	assert.Equal(t, stats.TotalSent, unmarshaledStats.TotalSent)
	assert.Equal(t, stats.TotalFailed, unmarshaledStats.TotalFailed)
	assert.Equal(t, stats.QueuedJobs, unmarshaledStats.QueuedJobs)
	assert.Equal(t, stats.ActiveWorkers, unmarshaledStats.ActiveWorkers)
}

func TestUnmarshalBroadcastStatsInvalid(t *testing.T) {
	data := []byte("invalid json")

	stats, err := UnmarshalBroadcastStats(data)
	assert.Error(t, err)
	assert.Nil(t, stats)
}
