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
	"github.com/swarm-ai/swarm/pkg/api"
)

type AgentHandler struct {
	proto.UnimplementedAgentServiceServer
	server *Server
}

func NewAgentHandler(server *Server) *AgentHandler {
	return &AgentHandler{
		server: server,
	}
}

func (h *AgentHandler) CreateAgent(ctx context.Context, req *proto.CreateAgentRequest) (*proto.CreateAgentResponse, error) {
	var resourceLimits api.ResourceLimits
	if req.ResourceLimits != nil {
		resourceLimits = api.ResourceLimits{
			MaxMemoryMB:      int(req.ResourceLimits.MaxMemoryMb),
			MaxCPUPercent:    int(req.ResourceLimits.MaxCpuPercent),
			MaxTokensPerTask: int(req.ResourceLimits.MaxTokensPerTask),
			MaxTasksPerHour:  int(req.ResourceLimits.MaxTasksPerHour),
		}
	}

	_ = &api.AgentConfig{
		ID:             req.Id,
		Type:           api.AgentType(req.Type.String()),
		Name:           req.Name,
		Model:          req.Model,
		SystemPrompt:   req.SystemPrompt,
		Skills:         req.Skills,
		MaxConcurrent:  int(req.MaxConcurrent),
		Timeout:        time.Duration(req.TimeoutMs) * time.Millisecond,
		MaxRetries:     int(req.MaxRetries),
		ResourceLimits: resourceLimits,
	}

	agent := &proto.Agent{
		Id:            req.Id,
		Type:          req.Type,
		Name:          req.Name,
		Model:         req.Model,
		Status:        proto.AgentStatus_AGENT_STATUS_CREATED,
		Skills:        req.Skills,
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		LastHeartbeat: timestamppb.Now(),
	}

	return &proto.CreateAgentResponse{
		Agent: agent,
	}, nil
}

func (h *AgentHandler) GetAgent(ctx context.Context, req *proto.GetAgentRequest) (*proto.GetAgentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	agent := &proto.Agent{
		Id:            req.Id,
		Type:          proto.AgentType_AGENT_TYPE_CODER,
		Name:          "Test Agent",
		Model:         "gpt-4",
		Status:        proto.AgentStatus_AGENT_STATUS_READY,
		Skills:        []string{"code_generation", "testing"},
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		LastHeartbeat: timestamppb.Now(),
	}

	return &proto.GetAgentResponse{
		Agent: agent,
	}, nil
}

func (h *AgentHandler) ListAgents(ctx context.Context, req *proto.ListAgentsRequest) (*proto.ListAgentsResponse, error) {
	agents := []*proto.Agent{
		{
			Id:            "agent-1",
			Type:          proto.AgentType_AGENT_TYPE_CODER,
			Name:          "Code Generator",
			Model:         "gpt-4",
			Status:        proto.AgentStatus_AGENT_STATUS_READY,
			Skills:        []string{"code_generation"},
			CreatedAt:     timestamppb.Now(),
			UpdatedAt:     timestamppb.Now(),
			LastHeartbeat: timestamppb.Now(),
		},
		{
			Id:            "agent-2",
			Type:          proto.AgentType_AGENT_TYPE_TESTER,
			Name:          "Test Runner",
			Model:         "gpt-4",
			Status:        proto.AgentStatus_AGENT_STATUS_RUNNING,
			Skills:        []string{"testing", "verification"},
			CreatedAt:     timestamppb.Now(),
			UpdatedAt:     timestamppb.Now(),
			LastHeartbeat: timestamppb.Now(),
		},
	}

	return &proto.ListAgentsResponse{
		Agents:     agents,
		TotalCount: int32(len(agents)),
	}, nil
}

func (h *AgentHandler) UpdateAgent(ctx context.Context, req *proto.UpdateAgentRequest) (*proto.UpdateAgentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	agent := &proto.Agent{
		Id:            req.Id,
		Type:          proto.AgentType_AGENT_TYPE_CODER,
		Name:          req.Name,
		Model:         "gpt-4",
		Status:        proto.AgentStatus_AGENT_STATUS_READY,
		Skills:        req.Skills,
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		LastHeartbeat: timestamppb.Now(),
	}

	return &proto.UpdateAgentResponse{
		Agent: agent,
	}, nil
}

func (h *AgentHandler) DeleteAgent(ctx context.Context, req *proto.DeleteAgentRequest) (*emptypb.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	return &emptypb.Empty{}, nil
}

func (h *AgentHandler) StartAgent(ctx context.Context, req *proto.StartAgentRequest) (*proto.StartAgentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	agent := &proto.Agent{
		Id:            req.Id,
		Type:          proto.AgentType_AGENT_TYPE_CODER,
		Name:          "Test Agent",
		Model:         "gpt-4",
		Status:        proto.AgentStatus_AGENT_STATUS_READY,
		Skills:        []string{},
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		LastHeartbeat: timestamppb.Now(),
	}

	return &proto.StartAgentResponse{
		Agent: agent,
	}, nil
}

func (h *AgentHandler) StopAgent(ctx context.Context, req *proto.StopAgentRequest) (*proto.StopAgentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	agent := &proto.Agent{
		Id:            req.Id,
		Type:          proto.AgentType_AGENT_TYPE_CODER,
		Name:          "Test Agent",
		Model:         "gpt-4",
		Status:        proto.AgentStatus_AGENT_STATUS_TERMINATED,
		Skills:        []string{},
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		LastHeartbeat: timestamppb.Now(),
	}

	return &proto.StopAgentResponse{
		Agent: agent,
	}, nil
}

func (h *AgentHandler) PauseAgent(ctx context.Context, req *proto.PauseAgentRequest) (*proto.PauseAgentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	agent := &proto.Agent{
		Id:            req.Id,
		Type:          proto.AgentType_AGENT_TYPE_CODER,
		Name:          "Test Agent",
		Model:         "gpt-4",
		Status:        proto.AgentStatus_AGENT_STATUS_PAUSED,
		Skills:        []string{},
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		LastHeartbeat: timestamppb.Now(),
	}

	return &proto.PauseAgentResponse{
		Agent: agent,
	}, nil
}

func (h *AgentHandler) ResumeAgent(ctx context.Context, req *proto.ResumeAgentRequest) (*proto.ResumeAgentResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	agent := &proto.Agent{
		Id:            req.Id,
		Type:          proto.AgentType_AGENT_TYPE_CODER,
		Name:          "Test Agent",
		Model:         "gpt-4",
		Status:        proto.AgentStatus_AGENT_STATUS_READY,
		Skills:        []string{},
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
		LastHeartbeat: timestamppb.Now(),
	}

	return &proto.ResumeAgentResponse{
		Agent: agent,
	}, nil
}

func (h *AgentHandler) GetAgentMetrics(ctx context.Context, req *proto.GetAgentMetricsRequest) (*proto.GetAgentMetricsResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	metrics := &proto.AgentMetrics{
		TasksCompleted:  100,
		TasksFailed:     5,
		AvgDurationMs:   5000,
		TotalTokensUsed: 1000000,
		TotalCost:       10.50,
		LastActivity:    timestamppb.Now(),
	}

	return &proto.GetAgentMetricsResponse{
		Metrics: metrics,
	}, nil
}

func (h *AgentHandler) GetAgentHealth(ctx context.Context, req *proto.GetAgentHealthRequest) (*proto.GetAgentHealthResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	health := &proto.AgentHealth{
		Status:         "healthy",
		LastHeartbeat:  timestamppb.Now(),
		CpuUsage:       25.5,
		MemoryUsageMb:  512,
		ActiveRequests: 3,
		ErrorCount:     0,
	}

	return &proto.GetAgentHealthResponse{
		Health: health,
	}, nil
}

func (h *AgentHandler) StreamAgentMetrics(req *proto.StreamAgentMetricsRequest, stream proto.AgentService_StreamAgentMetricsServer) error {
	if req.AgentId == "" {
		return status.Error(codes.InvalidArgument, "agent ID is required")
	}

	streamID := GenerateStreamID(fmt.Sprintf("agent-metrics-%s", req.AgentId))
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	options := map[string]string{
		"agent_id":    req.AgentId,
		"interval_ms": fmt.Sprintf("%d", req.IntervalMs),
	}

	_, err := h.server.StreamManager().RegisterStream(streamID, string(StreamTypeAgentMetrics), ctx, cancel, options)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to register stream: %v", err)
	}
	defer h.server.StreamManager().UnregisterStream(streamID)

	ticker := time.NewTicker(time.Duration(req.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			metrics := &proto.AgentMetrics{
				TasksCompleted:  100,
				TasksFailed:     5,
				AvgDurationMs:   5000,
				TotalTokensUsed: 1000000,
				TotalCost:       10.50,
				LastActivity:    timestamppb.Now(),
			}

			update := &proto.AgentMetricUpdate{
				AgentId:   req.AgentId,
				Metrics:   metrics,
				Timestamp: timestamppb.Now(),
			}

			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "failed to send metrics update: %v", err)
			}

			h.server.StreamManager().UpdateStream(streamID)
		}
	}
}

func (h *AgentHandler) StreamAgentHealth(req *proto.StreamAgentHealthRequest, stream proto.AgentService_StreamAgentHealthServer) error {
	if req.AgentId == "" {
		return status.Error(codes.InvalidArgument, "agent ID is required")
	}

	streamID := GenerateStreamID(fmt.Sprintf("agent-health-%s", req.AgentId))
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	options := map[string]string{
		"agent_id":    req.AgentId,
		"interval_ms": fmt.Sprintf("%d", req.IntervalMs),
	}

	_, err := h.server.StreamManager().RegisterStream(streamID, string(StreamTypeAgentHealth), ctx, cancel, options)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to register stream: %v", err)
	}
	defer h.server.StreamManager().UnregisterStream(streamID)

	ticker := time.NewTicker(time.Duration(req.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			health := &proto.AgentHealth{
				Status:         "healthy",
				LastHeartbeat:  timestamppb.Now(),
				CpuUsage:       25.5,
				MemoryUsageMb:  512,
				ActiveRequests: 3,
				ErrorCount:     0,
			}

			update := &proto.AgentHealthUpdate{
				AgentId:   req.AgentId,
				Health:    health,
				Timestamp: timestamppb.Now(),
			}

			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "failed to send health update: %v", err)
			}

			h.server.StreamManager().UpdateStream(streamID)
		}
	}
}
