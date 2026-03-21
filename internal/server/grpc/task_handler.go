package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/swarm-ai/swarm/internal/server/grpc/proto"
)

type TaskHandler struct {
	proto.UnimplementedTaskServiceServer
	server *Server
}

func NewTaskHandler(server *Server) *TaskHandler {
	return &TaskHandler{
		server: server,
	}
}

func (h *TaskHandler) CreateTask(ctx context.Context, req *proto.CreateTaskRequest) (*proto.CreateTaskResponse, error) {
	task := &proto.Task{
		Id:            req.Id,
		Name:          req.Name,
		Description:   req.Description,
		Prompt:        req.Prompt,
		Status:        proto.TaskStatus_TASK_STATUS_PENDING,
		Kind:          req.Kind,
		Priority:      req.Priority,
		AgentType:     req.AgentType,
		AssignedAgent: req.AssignedAgent,
		Dependencies:  req.Dependencies,
		Input:         req.Input,
		TimeoutMs:     req.TimeoutMs,
		MaxRetries:    req.MaxRetries,
		Metadata:      req.Metadata,
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
	}

	return &proto.CreateTaskResponse{
		Task: task,
	}, nil
}

func (h *TaskHandler) GetTask(ctx context.Context, req *proto.GetTaskRequest) (*proto.GetTaskResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "task ID is required")
	}

	task := &proto.Task{
		Id:            req.Id,
		Name:          "Test Task",
		Description:   "A test task",
		Prompt:        "Complete this task",
		Status:        proto.TaskStatus_TASK_STATUS_COMPLETED,
		Kind:          proto.TaskKind_TASK_KIND_COMPUTE,
		Priority:      2,
		AgentType:     proto.AgentType_AGENT_TYPE_CODER,
		AssignedAgent: "agent-1",
		Input:         map[string]string{"param1": "value1"},
		Output:        map[string]string{"result": "success"},
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		StartedAt:     timestamppb.Now(),
		CompletedAt:   timestamppb.Now(),
		Progress:      100.0,
	}

	return &proto.GetTaskResponse{
		Task: task,
	}, nil
}

func (h *TaskHandler) ListTasks(ctx context.Context, req *proto.ListTasksRequest) (*proto.ListTasksResponse, error) {
	tasks := []*proto.Task{
		{
			Id:            "task-1",
			Name:          "Task 1",
			Description:   "First task",
			Status:        proto.TaskStatus_TASK_STATUS_COMPLETED,
			Kind:          proto.TaskKind_TASK_KIND_COMPUTE,
			Priority:      int32(proto.Priority_PRIORITY_NORMAL),
			AgentType:     proto.AgentType_AGENT_TYPE_CODER,
			AssignedAgent: "agent-1",
			CreatedAt:     timestamppb.Now(),
			UpdatedAt:     timestamppb.Now(),
			Progress:      100.0,
		},
		{
			Id:            "task-2",
			Name:          "Task 2",
			Description:   "Second task",
			Status:        proto.TaskStatus_TASK_STATUS_RUNNING,
			Kind:          proto.TaskKind_TASK_KIND_IO,
			Priority:      3,
			AgentType:     proto.AgentType_AGENT_TYPE_TESTER,
			AssignedAgent: "agent-2",
			CreatedAt:     timestamppb.Now(),
			UpdatedAt:     timestamppb.Now(),
			Progress:      50.0,
		},
	}

	return &proto.ListTasksResponse{
		Tasks:      tasks,
		TotalCount: int32(len(tasks)),
	}, nil
}

func (h *TaskHandler) UpdateTask(ctx context.Context, req *proto.UpdateTaskRequest) (*proto.UpdateTaskResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "task ID is required")
	}

	task := &proto.Task{
		Id:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		Prompt:      req.Prompt,
		Priority:    req.Priority,
		Input:       req.Input,
		Metadata:    req.Metadata,
		Status:      proto.TaskStatus_TASK_STATUS_PENDING,
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	return &proto.UpdateTaskResponse{
		Task: task,
	}, nil
}

func (h *TaskHandler) DeleteTask(ctx context.Context, req *proto.DeleteTaskRequest) (*emptypb.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "task ID is required")
	}

	return &emptypb.Empty{}, nil
}

func (h *TaskHandler) ExecuteTask(ctx context.Context, req *proto.ExecuteTaskRequest) (*proto.ExecuteTaskResponse, error) {
	if req.TaskId == "" {
		return nil, status.Error(codes.InvalidArgument, "task ID is required")
	}

	result := &proto.TaskResult{
		TaskId:    req.TaskId,
		Success:   true,
		Output:    "Task completed successfully",
		Artifacts: []*proto.Artifact{},
		Metrics: &proto.TaskMetrics{
			StartTime:  timestamppb.Now(),
			EndTime:    timestamppb.Now(),
			DurationMs: 5000,
			TokensUsed: 1000,
			ToolCalls:  5,
			Messages:   10,
		},
		CompletedAt: timestamppb.Now(),
	}

	return &proto.ExecuteTaskResponse{
		Result: result,
	}, nil
}

func (h *TaskHandler) CancelTask(ctx context.Context, req *proto.CancelTaskRequest) (*proto.CancelTaskResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "task ID is required")
	}

	task := &proto.Task{
		Id:          req.Id,
		Name:        "Cancelled Task",
		Description: "A cancelled task",
		Status:      proto.TaskStatus_TASK_STATUS_CANCELLED,
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	return &proto.CancelTaskResponse{
		Task: task,
	}, nil
}

func (h *TaskHandler) RetryTask(ctx context.Context, req *proto.RetryTaskRequest) (*proto.RetryTaskResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "task ID is required")
	}

	task := &proto.Task{
		Id:          req.Id,
		Name:        "Retried Task",
		Description: "A retried task",
		Status:      proto.TaskStatus_TASK_STATUS_PENDING,
		RetryCount:  1,
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	return &proto.RetryTaskResponse{
		Task: task,
	}, nil
}

func (h *TaskHandler) ListTaskDependencies(ctx context.Context, req *proto.ListTaskDependenciesRequest) (*proto.ListTaskDependenciesResponse, error) {
	if req.TaskId == "" {
		return nil, status.Error(codes.InvalidArgument, "task ID is required")
	}

	dependencies := []*proto.TaskDependency{
		{
			TaskId:    req.TaskId,
			DependsOn: "task-1",
			Status:    "completed",
		},
		{
			TaskId:    req.TaskId,
			DependsOn: "task-2",
			Status:    "completed",
		},
	}

	return &proto.ListTaskDependenciesResponse{
		Dependencies: dependencies,
	}, nil
}

func (h *TaskHandler) StreamTaskUpdates(req *proto.StreamTaskUpdatesRequest, stream proto.TaskService_StreamTaskUpdatesServer) error {
	if req.TaskId == "" {
		return status.Error(codes.InvalidArgument, "task ID is required")
	}

	streamID := GenerateStreamID(fmt.Sprintf("task-updates-%s", req.TaskId))
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	options := map[string]string{
		"task_id": req.TaskId,
	}

	_, err := h.server.StreamManager().RegisterStream(streamID, string(StreamTypeTaskUpdates), ctx, cancel, options)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to register stream: %v", err)
	}
	defer h.server.StreamManager().UnregisterStream(streamID)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	statuses := []proto.TaskStatus{
		proto.TaskStatus_TASK_STATUS_PENDING,
		proto.TaskStatus_TASK_STATUS_QUEUED,
		proto.TaskStatus_TASK_STATUS_RUNNING,
		proto.TaskStatus_TASK_STATUS_COMPLETED,
	}

	for i, taskStatus := range statuses {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			update := &proto.TaskUpdate{
				TaskId:    req.TaskId,
				Status:    taskStatus,
				Timestamp: timestamppb.Now(),
				Message:   fmt.Sprintf("Task status: %s", taskStatus.String()),
			}

			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "failed to send task update: %v", err)
			}

			h.server.StreamManager().UpdateStream(streamID)

			if i >= len(statuses)-1 {
				return nil
			}
		}
	}

	return nil
}

func (h *TaskHandler) StreamTaskProgress(req *proto.StreamTaskProgressRequest, stream proto.TaskService_StreamTaskProgressServer) error {
	if req.TaskId == "" {
		return status.Error(codes.InvalidArgument, "task ID is required")
	}

	streamID := GenerateStreamID(fmt.Sprintf("task-progress-%s", req.TaskId))
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	options := map[string]string{
		"task_id":     req.TaskId,
		"interval_ms": fmt.Sprintf("%d", req.IntervalMs),
	}

	_, err := h.server.StreamManager().RegisterStream(streamID, string(StreamTypeTaskProgress), ctx, cancel, options)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to register stream: %v", err)
	}
	defer h.server.StreamManager().UnregisterStream(streamID)

	interval := time.Duration(req.IntervalMs) * time.Millisecond
	if interval == 0 {
		interval = 500 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	progress := 0.0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			progress += 10.0
			if progress > 100.0 {
				progress = 100.0
			}

			update := &proto.TaskProgressUpdate{
				TaskId:        req.TaskId,
				Progress:      progress,
				Timestamp:     timestamppb.Now(),
				StatusMessage: fmt.Sprintf("Progress: %.1f%%", progress),
			}

			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "failed to send progress update: %v", err)
			}

			h.server.StreamManager().UpdateStream(streamID)

			if progress >= 100.0 {
				return nil
			}
		}
	}
}
