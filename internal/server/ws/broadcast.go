package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type BroadcastManager struct {
	hub     *Hub
	ctx     context.Context
	cancel  context.CancelFunc
	queue   chan *BroadcastJob
	workers int
	mu      sync.RWMutex
	stats   *BroadcastStats
}

type BroadcastJob struct {
	Message  *Message
	Target   *BroadcastTarget
	Priority int
	Callback func(success int, failed int)
}

type BroadcastStats struct {
	TotalSent     int64 `json:"total_sent"`
	TotalFailed   int64 `json:"total_failed"`
	QueuedJobs    int   `json:"queued_jobs"`
	ActiveWorkers int   `json:"active_workers"`
}

func NewBroadcastManager(hub *Hub, workers int) *BroadcastManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &BroadcastManager{
		hub:     hub,
		ctx:     ctx,
		cancel:  cancel,
		queue:   make(chan *BroadcastJob, 1024),
		workers: workers,
		stats:   &BroadcastStats{},
	}
}

func (bm *BroadcastManager) Start() {
	for i := 0; i < bm.workers; i++ {
		go bm.worker(i)
	}

	go bm.statsCollector()
}

func (bm *BroadcastManager) Stop() {
	bm.cancel()
}

func (bm *BroadcastManager) Broadcast(msg *Message, target *BroadcastTarget) error {
	select {
	case <-bm.ctx.Done():
		return fmt.Errorf("broadcast manager stopped")
	default:
	}

	if target == nil {
		target = &BroadcastTarget{All: true}
	}

	job := &BroadcastJob{
		Message:  msg,
		Target:   target,
		Priority: 0,
	}

	select {
	case <-bm.ctx.Done():
		return fmt.Errorf("broadcast manager stopped")
	case bm.queue <- job:
		return nil
	default:
		return fmt.Errorf("broadcast queue full")
	}
}

func (bm *BroadcastManager) BroadcastAsync(msg *Message, target *BroadcastTarget, callback func(int, int)) error {
	job := &BroadcastJob{
		Message:  msg,
		Target:   target,
		Priority: 0,
		Callback: callback,
	}

	select {
	case bm.queue <- job:
		return nil
	case <-bm.ctx.Done():
		return fmt.Errorf("broadcast manager stopped")
	default:
		return fmt.Errorf("broadcast queue full")
	}
}

func (bm *BroadcastManager) BroadcastAgentUpdate(agent api.Agent, metrics *api.AgentMetrics, health *api.AgentHealth) error {
	if agent == nil {
		return nil
	}
	msg := NewAgentUpdate(agent, metrics, health)
	return bm.Broadcast(msg, &BroadcastTarget{All: true})
}

func (bm *BroadcastManager) BroadcastTaskEvent(task *api.Task, eventType string, result *api.TaskResult, err error) error {
	if task == nil {
		return nil
	}
	msg := NewTaskEvent(task, eventType, result, err)
	if msg == nil {
		return nil
	}
	return bm.Broadcast(msg, &BroadcastTarget{All: true})
}

func (bm *BroadcastManager) BroadcastLog(level, message, agentID, taskID string) error {
	msg := NewLogEntry(level, message, agentID, taskID)
	return bm.Broadcast(msg, &BroadcastTarget{All: true})
}

func (bm *BroadcastManager) BroadcastToUser(msg *Message, userID string) error {
	return bm.Broadcast(msg, &BroadcastTarget{UserIDs: []string{userID}})
}

func (bm *BroadcastManager) BroadcastToAgent(msg *Message, agentID string) error {
	return bm.Broadcast(msg, &BroadcastTarget{AgentIDs: []string{agentID}})
}

func (bm *BroadcastManager) BroadcastToTask(msg *Message, taskID string) error {
	return bm.Broadcast(msg, &BroadcastTarget{TaskIDs: []string{taskID}})
}

func (bm *BroadcastManager) BroadcastAgentUpdateToUser(agent api.Agent, metrics *api.AgentMetrics, health *api.AgentHealth, userID string) error {
	msg := NewAgentUpdate(agent, metrics, health)
	return bm.BroadcastToUser(msg, userID)
}

func (bm *BroadcastManager) BroadcastTaskEventToTask(task *api.Task, eventType string, result *api.TaskResult, err error, taskID string) error {
	msg := NewTaskEvent(task, eventType, result, err)
	return bm.BroadcastToTask(msg, taskID)
}

func (bm *BroadcastManager) worker(id int) {
	for {
		select {
		case job, ok := <-bm.queue:
			if !ok {
				return
			}
			bm.processJob(job)

		case <-bm.ctx.Done():
			return
		}
	}
}

func (bm *BroadcastManager) processJob(job *BroadcastJob) {
	bm.mu.Lock()
	bm.stats.ActiveWorkers++
	bm.mu.Unlock()

	success, failed := bm.sendToRecipients(job)

	bm.mu.Lock()
	bm.stats.ActiveWorkers--
	bm.stats.TotalSent += int64(success)
	bm.stats.TotalFailed += int64(failed)
	bm.mu.Unlock()

	if job.Callback != nil {
		job.Callback(success, failed)
	}
}

func (bm *BroadcastManager) sendToRecipients(job *BroadcastJob) (success, failed int) {
	msg := job.Message
	target := job.Target

	clients := bm.getRecipients(msg, target)

	for _, client := range clients {
		if client.IsClosed() {
			failed++
			continue
		}

		if err := client.Send(msg); err != nil {
			bm.hub.Error(fmt.Errorf("failed to send to client %s: %w", client.ID(), err))
			failed++
		} else {
			success++
		}
	}

	return success, failed
}

func (bm *BroadcastManager) getRecipients(msg *Message, target *BroadcastTarget) []*Client {
	clients := bm.hub.GetAllClients()

	var recipients []*Client

	if target.All {
		for _, client := range clients {
			if !client.IsClosed() && client.Subscription().ShouldReceive(msg) {
				recipients = append(recipients, client)
			}
		}
		return recipients
	}

	userSet := make(map[string]bool)
	for _, userID := range target.UserIDs {
		userSet[userID] = true
	}

	agentSet := make(map[string]bool)
	for _, agentID := range target.AgentIDs {
		agentSet[agentID] = true
	}

	taskSet := make(map[string]bool)
	for _, taskID := range target.TaskIDs {
		taskSet[taskID] = true
	}

	for _, client := range clients {
		if client.IsClosed() {
			continue
		}
		if !client.Subscription().ShouldReceive(msg) {
			continue
		}

		if len(userSet) > 0 && !userSet[client.UserID()] {
			continue
		}

		if len(agentSet) > 0 && msg.AgentID != "" && !agentSet[msg.AgentID] {
			continue
		}

		if len(taskSet) > 0 && msg.TaskID != "" && !taskSet[msg.TaskID] {
			continue
		}

		recipients = append(recipients, client)
	}

	return recipients
}

func (bm *BroadcastManager) statsCollector() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bm.updateStats()

		case <-bm.ctx.Done():
			return
		}
	}
}

func (bm *BroadcastManager) updateStats() {
	bm.mu.Lock()
	bm.stats.QueuedJobs = len(bm.queue)
	bm.mu.Unlock()
}

func (bm *BroadcastManager) Stats() *BroadcastStats {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	stats := *bm.stats
	stats.QueuedJobs = len(bm.queue)
	return &stats
}

func (bm *BroadcastManager) QueueSize() int {
	return len(bm.queue)
}

type BufferedBroadcaster struct {
	buffer    []*Message
	maxSize   int
	interval  time.Duration
	timer     *time.Timer
	mu        sync.Mutex
	broadcast func(*Message)
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewBufferedBroadcaster(maxSize int, interval time.Duration, broadcast func(*Message)) *BufferedBroadcaster {
	ctx, cancel := context.WithCancel(context.Background())

	bb := &BufferedBroadcaster{
		buffer:    make([]*Message, 0, maxSize),
		maxSize:   maxSize,
		interval:  interval,
		timer:     time.NewTimer(interval),
		broadcast: broadcast,
		ctx:       ctx,
		cancel:    cancel,
	}

	go bb.run()

	return bb
}

func (bb *BufferedBroadcaster) Add(msg *Message) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	bb.buffer = append(bb.buffer, msg)

	if len(bb.buffer) >= bb.maxSize {
		bb.flush()
	}
}

func (bb *BufferedBroadcaster) Flush() {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	bb.flush()
}

func (bb *BufferedBroadcaster) flush() {
	if len(bb.buffer) == 0 {
		return
	}

	for _, msg := range bb.buffer {
		bb.broadcast(msg)
	}

	bb.buffer = bb.buffer[:0]
	bb.timer.Reset(bb.interval)
}

func (bb *BufferedBroadcaster) run() {
	for {
		select {
		case <-bb.timer.C:
			bb.Flush()

		case <-bb.ctx.Done():
			bb.Flush()
			bb.timer.Stop()
			return
		}
	}
}

func (bb *BufferedBroadcaster) Stop() {
	bb.cancel()
}

type MessageFilter struct {
	filters map[string]func(*Message) bool
	mu      sync.RWMutex
}

func NewMessageFilter() *MessageFilter {
	return &MessageFilter{
		filters: make(map[string]func(*Message) bool),
	}
}

func (mf *MessageFilter) AddFilter(id string, filter func(*Message) bool) {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	mf.filters[id] = filter
}

func (mf *MessageFilter) RemoveFilter(id string) {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	delete(mf.filters, id)
}

func (mf *MessageFilter) ShouldSend(msg *Message) bool {
	mf.mu.RLock()
	defer mf.mu.RUnlock()

	for _, filter := range mf.filters {
		if !filter(msg) {
			return false
		}
	}

	return true
}

func (mf *MessageFilter) Clear() {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	mf.filters = make(map[string]func(*Message) bool)
}

type PriorityBroadcaster struct {
	queues map[int]chan *Message
	mu     sync.RWMutex
}

func NewPriorityBroadcaster(queueSize int) *PriorityBroadcaster {
	return &PriorityBroadcaster{
		queues: map[int]chan *Message{
			0: make(chan *Message, queueSize),
			1: make(chan *Message, queueSize),
			2: make(chan *Message, queueSize),
		},
	}
}

func (pb *PriorityBroadcaster) Broadcast(msg *Message, priority int) error {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	queue, ok := pb.queues[priority]
	if !ok {
		return fmt.Errorf("invalid priority: %d", priority)
	}

	select {
	case queue <- msg:
		return nil
	default:
		return fmt.Errorf("priority queue %d full", priority)
	}
}

func (pb *PriorityBroadcaster) Next() (*Message, int) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	for priority := 2; priority >= 0; priority-- {
		queue, ok := pb.queues[priority]
		if !ok {
			continue
		}

		select {
		case msg := <-queue:
			return msg, priority
		default:
			continue
		}
	}

	return nil, -1
}

func MarshalBroadcastStats(stats *BroadcastStats) ([]byte, error) {
	return json.Marshal(stats)
}

func UnmarshalBroadcastStats(data []byte) (*BroadcastStats, error) {
	var stats BroadcastStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}
