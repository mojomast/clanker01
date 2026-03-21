# SWARM Architecture

This document provides a detailed overview of the SWARM architecture, design decisions, and implementation details.

## Table of Contents

- [System Overview](#system-overview)
- [Component Architecture](#component-architecture)
- [Module Descriptions](#module-descriptions)
- [Data Flow](#data-flow)
- [Design Decisions](#design-decisions)
- [Performance Considerations](#performance-considerations)

## System Overview

SWARM is a multi-agent AI coding platform that enables parallel execution of specialized AI agents. The system is designed around three core principles:

1. **Modularity**: Each component is independent and can be developed/tested/deployed separately
2. **Parallel Execution**: Multiple agents work simultaneously on different tasks
3. **Extensibility**: Skills can be hot-loaded without restarting the system

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SWARM SYSTEM                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐   │
│  │                      CLIENT LAYER                            │   │
│  │  ┌─────────────┐  ┌─────────────┐                           │   │
│  │  │   TUI CLI   │  │  Remote      │                           │   │
│  │  │             │  │  Client      │                           │   │
│  │  └─────────────┘  └─────────────┘                           │   │
│  └───────────────────────────────────────────────────────────────────────┘   │
│                           │                                        │   │
│                           ▼                                        │   │
│  ┌───────────────────────────────────────────────────────────────────────┐   │
│  │                      API LAYER                            │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │   │
│  │  │   gRPC     │  │  WebSocket   │  │   REST API  │    │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │   │
│  └───────────────────────────────────────────────────────────────────────┘   │
│                           │                                        │   │
│                           ▼                                        │   │
│  ┌───────────────────────────────────────────────────────────────────────┐   │
│  │                      CORE LAYER                            │   │
│  │  ┌─────────────────────────────────────────────────────────┐       │   │
│  │  │            Orchestrator & Task Manager          │       │   │
│  │  └─────────────────────────────────────────────────────────┘       │   │
│  │                              │                              │       │   │
│  │                              ▼                              │       │   │
│  │  ┌─────────────────────────────────────────────────────┐       │   │
│  │  │              Agent Runtime                │       │   │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐    │       │   │
│  │  │  │Architect│ │ Coder  │ │Tester  │    │       │   │
│  │  │  └────────┘ └────────┘ └────────┘    │       │   │
│  │  └─────────────────────────────────────────────┘       │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                           │   │
│  ┌─────────────────────────────────────────────────────────────┐       │   │
│  │                 SUPPORTING LAYERS                  │       │   │
│  │  ┌───────────────┐  ┌───────────────┐           │       │   │
│  │  │ MCP Connector │  │ Skill System   │           │       │   │
│  │  └───────────────┘  └───────────────┘           │       │   │
│  └─────────────────────────────────────────────────────────────┘       │   │
│                                                           │   │
│  ┌─────────────────────────────────────────────────────────────┐       │   │
│  │                 INFRASTRUCTURE LAYERS            │       │   │
│  │  ┌───────────────┐  ┌───────────────┐           │       │   │
│  │  │ Context Store │  │ LLM Providers │           │       │   │
│  │  └───────────────┘  └───────────────┘           │       │   │
│  └─────────────────────────────────────────────────────────────┘       │   │
│                                                           │   │
│  ┌─────────────────────────────────────────────────────────────┐       │   │
│  │              SECURITY & MONITORING              │       │   │
│  │  ┌─────────────────────────────────────┐               │       │   │
│  │  │ Auth + RBAC + Monitoring/Logging │               │       │   │
│  │  └─────────────────────────────────────┘               │       │   │
│  └─────────────────────────────────────────────────────────────┘       │
│                                                                   │
└───────────────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### Client Layer

#### TUI CLI (Terminal User Interface)

**Location**: `internal/tui/`, `internal/tui/components/`

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), provides an Elm-architecture based terminal UI:

- **Model-View-Update (MVU)**: State management through message passing
- **Real-time Updates**: WebSocket connection for live data
- **Multiple Views**: Dashboard, Agents, Tasks, Logs, Config
- **Theming**: Dark theme with extensible color schemes
- **Accessibility**: Keyboard navigation and screen reader support

#### Remote Client

**Location**: `cmd/swarm/`

Cobra-based CLI that can connect to remote SWARM instances:

- Connects via gRPC or WebSocket
- Supports all CLI commands locally or remotely
- Manages connection state and reconnection

### API Layer

#### gRPC Server

**Location**: `internal/server/grpc/`

Protocol Buffer-based high-performance API:

- **Bidirectional Streaming**: Real-time agent updates
- **Load Balancing**: Supports multiple agent pools
- **Connection Management**: Keepalive and graceful shutdown
- **Authentication**: JWT middleware integration

#### WebSocket Server

**Location**: `internal/server/ws/`

Real-time event streaming:

- **Message Broadcasting**: Fan-out to all connected clients
- **Topic Subscriptions**: Clients can subscribe to specific events
- **Connection Pooling**: Efficient connection management
- **Binary/Text Support**: Automatic format detection

#### REST API

**Location**: `internal/server/rest/`

HTTP API with OpenAPI documentation:

- **OpenAPI 3.0**: Auto-generated documentation
- **Rate Limiting**: Token bucket per-client limits
- **Structured Logging**: Request/response logging
- **Middleware Stack**: Auth, RBAC, logging, rate limiting

### Core Layer

#### Orchestrator

**Location**: `internal/core/orchestrator/`

Central coordinator for task execution:

- **Task Queue**: Priority-based queue with dependencies
- **Dependency Graph**: Topological sort and cycle detection
- **Agent Allocation**: Load balancing and capability matching
- **Conflict Resolution**: Handle resource conflicts
- **Error Recovery**: Retry, reassign, decompose, escalate strategies

#### Agent Runtime

**Location**: `internal/core/agent/`

Agent lifecycle and execution:

- **State Machine**: Created → Ready → Running → (Paused/Error) → Terminated
- **Role-Based Pools**: Architect, Coder, Tester, Reviewer, Researcher, Coordinator
- **Tool Execution**: MCP connector for skill/tool calls
- **Metrics Collection**: Tasks completed, tokens used, costs incurred

### Supporting Layers

#### Skill System

**Location**: `internal/skills/`, `skills/builtin/`

Modular plugin system:

- **Multi-Runtime**: Go, Python, Node.js, WASM, Native binaries
- **Hot Loading**: Load/unload without restart
- **Sandboxing**: Security profiles (restricted, standard, elevated)
- **Registry**: Skill discovery and version management
- **Built-in Skills**: Filesystem, Git, Database, Web operations

#### MCP Connector

**Location**: `internal/mcp/`

Model Context Protocol implementation:

- **JSON-RPC 2.0**: Over stdio, HTTP, WebSocket
- **Server Registry**: Dynamic server registration
- **Resource/Tool/Prompt Endpoints**: Full MCP spec support
- **Built-in Servers**: Filesystem, Git, HTTP, Memory

#### Context Store

**Location**: `internal/context/`

Tiered storage architecture:

- **Hot Store**: In-memory with LRU eviction (fast access)
- **Warm Store**: Redis with TTL (persistent cache)
- **Cold Store**: PostgreSQL (long-term storage)
- **Snapshots**: Checkpoint/restore functionality
- **Semantic Search**: Graph-based RAG for retrieval

#### LLM Providers

**Location**: `internal/providers/`

Universal provider interface:

- **75+ Providers**: Anthropic, OpenAI, Google, Azure, AWS, Ollama, etc.
- **Message Normalization**: Unified request/response format
- **Retry Circuit Breaker**: Automatic retry with exponential backoff
- **Semantic Caching**: Cache similar prompts/responses
- **Cost Tracking**: Token usage and cost per provider/model

### Security & Monitoring

#### Authentication

**Location**: `internal/security/auth/`

- **JWT**: Token generation and validation with multiple algorithms (HS256/384/512)
- **mTLS**: Certificate-based authentication
- **Session Management**: In-memory sessions with expiration and cleanup
- **HTTP Middleware**: Auth, session, mTLS support

#### RBAC Authorization

**Location**: `internal/security/rbac/`

- **Role-Based Access**: Admin, User, Readonly roles
- **Permission Checking**: Resource-based and wildcard permissions
- **Middleware**: HTTP, gRPC, WebSocket interceptors
- **Resource Ownership**: Agent/task/skill permissions

#### Monitoring & Logging

**Location**: `internal/monitoring/`

- **Metrics Collection**: Counters, gauges, histograms
- **Distributed Tracing**: Span creation and propagation
- **Structured Logging**: Debug, Info, Warn, Error levels with context
- **Alerting**: Threshold-based notifications with multiple operators

## Data Flow

### Task Execution Flow

```
1. User submits task via CLI/TUI/API
   ↓
2. Orchestrator validates task and checks dependencies
   ↓
3. Task added to priority queue
   ↓
4. Orchestrator selects available agent based on:
   - Agent type matches task requirements
   - Agent capacity/load
   - Priority and dependencies
   ↓
5. Task assigned to agent pool
   ↓
6. Agent fetches task and decomposes if needed (via LLM)
   ↓
7. Agent retrieves context from Context Store
   ↓
8. Agent uses LLM Provider to generate response
   ↓
9. Agent calls Skills/MCP tools as needed
   ↓
10. Agent updates task progress and results
   ↓
11. Orchestrator updates task status
   ↓
12. Clients notified via WebSocket/gRPC streams
```

### Agent Communication Flow

```
1. Agent A creates sub-task for Agent B
   ↓
2. Task submitted to Orchestrator
   ↓
3. Orchestrator assigns to Agent B
   ↓
4. Agent B executes and returns result
   ↓
5. Result stored in Knowledge Graph
   ↓
6. Agent A receives result via shared context
```

### Skill Execution Flow

```
1. Agent receives LLM response with tool call
   ↓
2. Skill Loader loads skill if not cached
   ↓
3. Skill sandbox initialized with security profile
   ↓
4. Skill executed via JSON-RPC (stdio/HTTP/WS)
   ↓
5. Result returned to agent
   ↓
6. Agent includes result in next LLM prompt
```

## Design Decisions

### Why Bubbletea for TUI?

- **Choice**: Selected over alternatives like termui, tcell
- **Rationale**:
  - Elm architecture is well-tested and predictable
  - Excellent documentation and examples
  - Active community and maintenance
  - Built-in support for complex layouts and animations

### Why MCP (Model Context Protocol)?

- **Choice**: Standardized on MCP vs custom protocol
- **Rationale**:
  - Universal connector for tools, databases, APIs
  - Growing ecosystem of MCP servers
  - Future-proof as MCP adoption increases
  - Enables code-sharing across AI platforms

### Why gRPC + WebSocket + REST?

- **Choice**: Three protocols instead of one
- **Rationale**:
  - **gRPC**: Best for high-performance streaming and bidirectional communication
  - **WebSocket**: Best for web clients and browser-based UIs
  - **REST**: Best for simple integrations and third-party tools
  - Each serves different use cases optimally

### Why Tiered Context Store?

- **Choice**: Hot/Warm/Cold vs single store
- **Rationale**:
  - **Hot**: Fast access for active contexts (LRU ensures memory efficiency)
  - **Warm**: Persistent cache for recently used contexts
  - **Cold**: Long-term storage for historical data and compliance
  - Balances performance, cost, and durability

### Why Multi-Runtime Skill System?

- **Choice**: Support Go, Python, Node.js, WASM, Native
- **Rationale**:
  - **Go**: Native performance, compile-time safety
  - **Python**: Extensive ML/AI library ecosystem
  - **Node.js**: Rich web scraping and API libraries
  - **WASM**: Language-agnostic, sandboxed execution
  - **Native**: Legacy tool integration without rewriting

### Why RBAC over ABAC?

- **Choice**: Role-Based Access Control over Attribute-Based
- **Rationale**:
  - Simpler to understand and configure
  - Fits most organizational structures
  - Sufficient for multi-agent use cases
  - Can evolve to ABAC if needed without major redesign

## Performance Considerations

### Concurrency Model

- **Agents**: Parallel execution by design, managed via pools
- **Tasks**: Priority queue ensures high-priority tasks execute first
- **API**: Connection pooling and reuse for efficiency
- **Context**: LRU eviction prevents memory bloat

### Scalability

- **Horizontal Scaling**: Add more agents to increase capacity
- **Vertical Scaling**: Increase agent pool sizes for more concurrent tasks
- **Distributed Deployment**: Run SWARM across multiple servers
- **Database**: PostgreSQL supports large-scale deployments

### Resource Management

- **Token Limits**: Track usage per provider/model
- **Cost Awareness**: Real-time cost tracking and alerts
- **Memory**: LRU and tiered storage prevent exhaustion
- **CPU**: Agent pools and timeouts prevent runaway processes

### Reliability

- **Retry Strategies**: Circuit breakers and exponential backoff
- **Graceful Degradation**: Non-critical failures don't stop system
- **Error Recovery**: Multiple strategies (retry, reassign, decompose, escalate)
- **Checkpointing**: Session snapshots prevent data loss

## Future Enhancements

Planned architecture improvements:

1. **Distributed Execution**: Run agents across multiple machines
2. **Federated Learning**: Share learned patterns between SWARM instances
3. **Advanced Skill Marketplace**: Browse, rate, purchase skills
4. **Multi-Modal Agents**: Vision, audio, and code generation
5. **Web UI**: Browser-based interface alongside TUI
6. **Mobile Support**: iOS and Android apps for remote monitoring
