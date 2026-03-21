package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/swarm-ai/swarm/internal/security/auth"
)

// testCredValidator is a simple credential validator for grpc tests
type testCredValidator struct{}

func (v *testCredValidator) ValidateCredentials(ctx context.Context, username, password string) (string, []string, []string, error) {
	if password != "valid_password" {
		return "", nil, nil, fmt.Errorf("invalid credentials")
	}
	return fmt.Sprintf("user_%s", username), []string{"user"}, []string{"read", "write"}, nil
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "0.0.0.0", config.Host)
	assert.Equal(t, 50051, config.Port)
	assert.False(t, config.EnableTLS)
	assert.True(t, config.EnableAuth)
	assert.True(t, config.EnableReflection)
	assert.Equal(t, 16*1024*1024, config.MaxRecvMsgSize)
	assert.Equal(t, 16*1024*1024, config.MaxSendMsgSize)
	assert.NotNil(t, config.KeepAlive)
}

func TestNewServer(t *testing.T) {
	jwtConfig := auth.DefaultJWTConfig()
	jwtAuth, err := auth.NewJWTAuthenticator(jwtConfig)
	require.NoError(t, err)

	config := DefaultConfig()
	server, err := NewServer(config, jwtAuth)

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.NotNil(t, server.StreamManager())
	assert.NotNil(t, server.Context())
}

func TestServerAddr(t *testing.T) {
	jwtConfig := auth.DefaultJWTConfig()
	jwtAuth, err := auth.NewJWTAuthenticator(jwtConfig)
	require.NoError(t, err)

	config := DefaultConfig()
	server, err := NewServer(config, jwtAuth)
	require.NoError(t, err)

	addr := server.Addr()
	assert.Empty(t, addr, "Address should be empty before server starts")

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	addr = server.Addr()
	assert.NotEmpty(t, addr, "Address should be set after server starts")
	assert.Contains(t, addr, "50051", "Address should contain the port")
}

func TestServerStartStop(t *testing.T) {
	jwtConfig := auth.DefaultJWTConfig()
	jwtAuth, err := auth.NewJWTAuthenticator(jwtConfig)
	require.NoError(t, err)

	config := DefaultConfig()
	server, err := NewServer(config, jwtAuth)
	require.NoError(t, err)

	err = server.Start()
	assert.NoError(t, err)
	assert.NotEmpty(t, server.Addr())

	err = server.Stop()
	assert.NoError(t, err)
}

func TestServerStopWithoutStart(t *testing.T) {
	jwtConfig := auth.DefaultJWTConfig()
	jwtAuth, err := auth.NewJWTAuthenticator(jwtConfig)
	require.NoError(t, err)

	config := DefaultConfig()
	server, err := NewServer(config, jwtAuth)
	require.NoError(t, err)

	err = server.Stop()
	assert.NoError(t, err)
}

func TestAuthFunc(t *testing.T) {
	jwtConfig := auth.DefaultJWTConfig()
	jwtAuth, err := auth.NewJWTAuthenticator(jwtConfig)
	require.NoError(t, err)

	config := DefaultConfig()
	config.EnableAuth = false
	server, err := NewServer(config, jwtAuth)
	require.NoError(t, err)

	ctx := context.Background()
	newCtx, err := server.AuthFunc()(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, newCtx)
}

func TestAuthFuncWithToken(t *testing.T) {
	jwtConfig := auth.DefaultJWTConfig()
	jwtAuth, err := auth.NewJWTAuthenticator(jwtConfig)
	require.NoError(t, err)
	jwtAuth.SetCredentialValidator(&testCredValidator{})

	config := DefaultConfig()
	server, err := NewServer(config, jwtAuth)
	require.NoError(t, err)

	creds := auth.Credentials{
		Username: "testuser",
		Password: "valid_password",
	}

	result, err := jwtAuth.Authenticate(context.Background(), creds)
	require.NoError(t, err)
	require.True(t, result.Success)
	assert.NotEmpty(t, result.Token)

	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer "+result.Token))

	authFunc := server.AuthFunc()
	newCtx, err := authFunc(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, newCtx)

	user, ok := GetUserFromContext(newCtx)
	assert.True(t, ok)
	assert.Equal(t, "testuser", user.Username)
}

func TestAuthFuncWithInvalidToken(t *testing.T) {
	jwtConfig := auth.DefaultJWTConfig()
	jwtAuth, err := auth.NewJWTAuthenticator(jwtConfig)
	require.NoError(t, err)

	config := DefaultConfig()
	server, err := NewServer(config, jwtAuth)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey{}, "Bearer invalid_token")

	_, err = server.AuthFunc()(ctx)
	assert.Error(t, err)
}

type contextKey struct{}

func TestStreamManagerLifecycle(t *testing.T) {
	sm := NewStreamManager()
	assert.Equal(t, 0, sm.Count())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	options := map[string]string{
		"agent_id":    "agent-1",
		"interval_ms": "1000",
	}

	stream, err := sm.RegisterStream("stream-1", "agent_metrics", ctx, cancel, options)
	require.NoError(t, err)
	require.NotNil(t, stream)
	assert.Equal(t, "stream-1", stream.GetID())
	assert.Equal(t, StreamTypeAgentMetrics, stream.GetType())
	assert.Equal(t, "agent-1", stream.GetAgentID())
	assert.Equal(t, 1*time.Second, stream.GetInterval())

	assert.Equal(t, 1, sm.Count())

	retrievedStream, err := sm.GetStream("stream-1")
	require.NoError(t, err)
	assert.Equal(t, stream.GetID(), retrievedStream.GetID())

	err = sm.UpdateStream("stream-1")
	assert.NoError(t, err)

	agentStreams := sm.GetStreamsByAgent("agent-1")
	assert.Len(t, agentStreams, 1)

	err = sm.UnregisterStream("stream-1")
	assert.NoError(t, err)
	assert.Equal(t, 0, sm.Count())
}

func TestStreamManagerNotFound(t *testing.T) {
	sm := NewStreamManager()

	_, err := sm.GetStream("nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrStreamNotFound, err)

	err = sm.UnregisterStream("nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrStreamNotFound, err)

	agentStreams := sm.GetStreamsByAgent("nonexistent")
	assert.Len(t, agentStreams, 0)
}

func TestStreamManagerCloseAll(t *testing.T) {
	sm := NewStreamManager()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	_, _ = sm.RegisterStream("stream-1", "agent_metrics", ctx1, cancel1, map[string]string{"agent_id": "agent-1"})
	_, _ = sm.RegisterStream("stream-2", "task_updates", ctx2, cancel2, map[string]string{"task_id": "task-1"})

	assert.Equal(t, 2, sm.Count())

	sm.CloseAll()

	assert.Equal(t, 0, sm.Count())
}

func TestManagedStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	stream := &ManagedStream{
		ID:        "test-stream",
		Type:      StreamTypeAgentMetrics,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
		metadata: map[string]string{
			"key1": "value1",
		},
	}

	assert.Equal(t, "test-stream", stream.GetID())
	assert.Equal(t, StreamTypeAgentMetrics, stream.GetType())
	assert.Equal(t, ctx, stream.Context())
	assert.False(t, stream.IsClosed())

	val, ok := stream.GetMetadata("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	stream.SetMetadata("key2", "value2")
	val, ok = stream.GetMetadata("key2")
	assert.True(t, ok)
	assert.Equal(t, "value2", val)

	stream.Cancel()
	assert.Eventually(t, func() bool {
		return stream.IsClosed()
	}, 1*time.Second, 100*time.Millisecond)
}

func TestGenerateStreamID(t *testing.T) {
	id1 := GenerateStreamID("test")
	id2 := GenerateStreamID("test")

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "Stream IDs should be unique")
	assert.Contains(t, id1, "test-")
}

func TestGetUserFromContext(t *testing.T) {
	user := &auth.User{
		ID:          "user-1",
		Username:    "testuser",
		Roles:       []string{"user"},
		Permissions: []string{"read", "write"},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, userContextKey{}, user)

	retrievedUser, ok := GetUserFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Username, retrievedUser.Username)

	ctx = context.Background()
	_, ok = GetUserFromContext(ctx)
	assert.False(t, ok)
}
