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

type SkillHandler struct {
	proto.UnimplementedSkillServiceServer
	server *Server
}

func NewSkillHandler(server *Server) *SkillHandler {
	return &SkillHandler{
		server: server,
	}
}

func (h *SkillHandler) RegisterSkill(ctx context.Context, req *proto.RegisterSkillRequest) (*proto.RegisterSkillResponse, error) {
	manifest := req.Manifest
	if manifest == nil {
		return nil, status.Error(codes.InvalidArgument, "manifest is required")
	}
	if manifest.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "manifest metadata is required")
	}

	skill := &proto.Skill{
		Id:          fmt.Sprintf("%s@%s", manifest.Metadata.Name, manifest.Metadata.Version),
		Name:        manifest.Metadata.Name,
		Version:     manifest.Metadata.Version,
		DisplayName: manifest.Metadata.DisplayName,
		Description: manifest.Metadata.Description,
		Author:      manifest.Metadata.Author,
		License:     manifest.Metadata.License,
		Tags:        manifest.Metadata.Tags,
		Icon:        manifest.Metadata.Icon,
		Homepage:    manifest.Metadata.Homepage,
		Repository:  manifest.Metadata.Repository,
		Deprecated:  manifest.Metadata.Deprecated,
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	return &proto.RegisterSkillResponse{
		Skill: skill,
	}, nil
}

func (h *SkillHandler) GetSkill(ctx context.Context, req *proto.GetSkillRequest) (*proto.GetSkillResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "skill name is required")
	}

	skill := &proto.Skill{
		Id:          fmt.Sprintf("%s@%s", req.Name, req.Version),
		Name:        req.Name,
		Version:     req.Version,
		DisplayName: "Code Generation",
		Description: "Generates code snippets and functions",
		Author:      "Swarm AI",
		License:     "MIT",
		Tags:        []string{"code", "generation", "ai"},
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	return &proto.GetSkillResponse{
		Skill: skill,
	}, nil
}

func (h *SkillHandler) ListSkills(ctx context.Context, req *proto.ListSkillsRequest) (*proto.ListSkillsResponse, error) {
	skills := []*proto.Skill{
		{
			Id:          "code_generation@1.0.0",
			Name:        "code_generation",
			Version:     "1.0.0",
			DisplayName: "Code Generation",
			Description: "Generates code snippets and functions",
			Author:      "Swarm AI",
			License:     "MIT",
			Tags:        []string{"code", "generation"},
			CreatedAt:   timestamppb.Now(),
			UpdatedAt:   timestamppb.Now(),
		},
		{
			Id:          "testing@1.0.0",
			Name:        "testing",
			Version:     "1.0.0",
			DisplayName: "Testing",
			Description: "Automated testing capabilities",
			Author:      "Swarm AI",
			License:     "MIT",
			Tags:        []string{"testing", "qa"},
			CreatedAt:   timestamppb.Now(),
			UpdatedAt:   timestamppb.Now(),
		},
	}

	return &proto.ListSkillsResponse{
		Skills:     skills,
		TotalCount: int32(len(skills)),
	}, nil
}

func (h *SkillHandler) UpdateSkill(ctx context.Context, req *proto.UpdateSkillRequest) (*proto.UpdateSkillResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "skill name is required")
	}
	if req.Manifest == nil {
		return nil, status.Error(codes.InvalidArgument, "manifest is required")
	}
	if req.Manifest.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "manifest metadata is required")
	}

	skill := &proto.Skill{
		Id:          fmt.Sprintf("%s@%s", req.Name, req.Version),
		Name:        req.Name,
		Version:     req.Version,
		DisplayName: req.Manifest.Metadata.DisplayName,
		Description: req.Manifest.Metadata.Description,
		Author:      req.Manifest.Metadata.Author,
		License:     req.Manifest.Metadata.License,
		Tags:        req.Manifest.Metadata.Tags,
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	return &proto.UpdateSkillResponse{
		Skill: skill,
	}, nil
}

func (h *SkillHandler) DeleteSkill(ctx context.Context, req *proto.DeleteSkillRequest) (*emptypb.Empty, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "skill name is required")
	}

	return &emptypb.Empty{}, nil
}

func (h *SkillHandler) DiscoverSkills(ctx context.Context, req *proto.DiscoverSkillsRequest) (*proto.DiscoverSkillsResponse, error) {
	matches := []*proto.SkillMatch{
		{
			SkillId: "code_generation@1.0.0",
			Score:   0.95,
			Context: map[string]string{
				"match_reason": "Query matches skill description",
			},
		},
		{
			SkillId: "testing@1.0.0",
			Score:   0.75,
			Context: map[string]string{
				"match_reason": "Partial match",
			},
		},
	}

	return &proto.DiscoverSkillsResponse{
		Matches: matches,
	}, nil
}

func (h *SkillHandler) GetSkillManifest(ctx context.Context, req *proto.GetSkillManifestRequest) (*proto.GetSkillManifestResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "skill name is required")
	}

	manifest := &proto.SkillManifest{
		ApiVersion: "v1",
		Kind:       "Skill",
		Metadata: &proto.SkillMetadata{
			Name:        req.Name,
			Version:     req.Version,
			DisplayName: "Code Generation",
			Description: "Generates code snippets and functions",
			Author:      "Swarm AI",
			License:     "MIT",
			Tags:        []string{"code", "generation"},
		},
		Spec: &proto.SkillSpec{
			Runtime:    "go",
			Entrypoint: "main.go",
			Tools: []*proto.ToolDef{
				{
					Name:        "generate_code",
					Description: "Generates code from a prompt",
					Parameters: map[string]string{
						"prompt":   "The code generation prompt",
						"language": "The programming language",
					},
				},
			},
		},
	}

	return &proto.GetSkillManifestResponse{
		Manifest: manifest,
	}, nil
}

func (h *SkillHandler) StreamSkillUpdates(req *proto.StreamSkillUpdatesRequest, stream proto.SkillService_StreamSkillUpdatesServer) error {
	streamID := GenerateStreamID("skill-updates")
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	options := map[string]string{
		"skill_names": fmt.Sprintf("%v", req.SkillNames),
	}

	_, err := h.server.StreamManager().RegisterStream(streamID, string(StreamTypeSkillUpdates), ctx, cancel, options)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to register stream: %v", err)
	}
	defer h.server.StreamManager().UnregisterStream(streamID)

	skills := []*proto.Skill{
		{
			Id:          "code_generation@1.0.0",
			Name:        "code_generation",
			Version:     "1.0.0",
			DisplayName: "Code Generation",
			Description: "Generates code snippets and functions",
			CreatedAt:   timestamppb.Now(),
			UpdatedAt:   timestamppb.Now(),
		},
	}

	for _, skill := range skills {
		select {
		case <-ctx.Done():
			return nil
		default:
			update := &proto.SkillUpdate{
				SkillId:   skill.Id,
				Skill:     skill,
				Action:    "registered",
				Timestamp: timestamppb.Now(),
			}

			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "failed to send skill update: %v", err)
			}

			h.server.StreamManager().UpdateStream(streamID)
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}
