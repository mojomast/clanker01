# SWARM: Multi-Agent AI Coding Platform

[![Built by GLM-4.7](https://img.shields.io/badge/built%20by-GLM--4.7-blue.svg)](https://github.com/THUDM)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

SWARM is an open-source, terminal-first multi-agent AI coding platform designed for **parallel agent execution** with specialized roles. Built on MCP (Model Context Protocol) as its universal connector, SWARM provides a modular, extensible architecture supporting **any LLM provider** with client/server capabilities for remote operation.

> **Note**: SWARM is under active development. The architecture is fully wired and all core features — including LLM provider API calls (Anthropic and OpenAI with SSE streaming), remote client connection, REST config endpoints, task planning/verification, and the orchestrator — are implemented and functional. See [ARCHITECTURE.md](ARCHITECTURE.md) for details.

## 🚀 Key Features

- **Parallel Agent Execution**: Multiple specialized agents work simultaneously on different tasks
- **MCP-Native**: Universal connector for tools, databases, APIs via Model Context Protocol
- **Skill System**: Modular, hot-loadable capabilities like plugins with multi-runtime support
- **Provider Agnostic**: Works with 75+ LLM providers (Anthropic, OpenAI, Claude, etc.)
- **Terminal-First**: Beautiful TUI built with Bubbletea for real-time streaming
- **gRPC/WebSocket/REST APIs**: Full client/server capabilities for remote operation
- **Comprehensive Security**: JWT authentication, mTLS support, RBAC authorization
- **Monitoring & Logging**: Built-in metrics collection, distributed tracing, structured logging

## 📋 Table of Contents

- [Architecture](#-architecture)
- [Quick Start](#-quick-start)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Usage](#-usage)
  - [CLI Usage](#cli-usage)
  - [TUI Usage](#tui-usage)
  - [Server Usage](#server-usage)
- [Developing Skills](#-developing-skills)
- [Architecture Modules](#-architecture-modules)
- [API Reference](#-api-reference)

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              SWARM PLATFORM                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐    ┌─────────────────────────────────────────────────┐    │
│  │   TUI CLI   │    │                  SWARM CORE                      │    │
│  │  (Bubbletea)│◄──►│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │    │
│  │             │    │  │ Orchestrator│  │ Agent Router│  │Task Queue│ │    │
│  │  └─────────────┘    │  └──────┬──────┘  └──────┬──────┘  └────┬─────┘ │    │
│  │                     │         │                │              │       │    │
│  │                     │         ▼                ▼              ▼       │    │
│  │  ┌─────────────┐    │  ┌─────────────────────────────────────────┐   │    │
│  │  │   Remote    │    │  │              Agent Runtime              │   │    │
│  │  │   Client    │◄──►│  │  ┌───────┐ ┌───────┐ ┌───────┐ ┌──────┐│   │    │
│  │  │  (gRPC/WS)  │    │  │  │Architct│ │ Coder │ │Tester │ │Review││   │    │
│  │  └─────────────┘    │  │  └───┬───┘ └───┬───┘ └───┬───┘ └──┬───┘│   │    │
│  │                     │  │      │         │         │        │     │   │    │
│  │                     │  └──────┼─────────┼─────────┼────────┼─────┘   │    │
│  │                     └─────────┼─────────┼─────────┼────────┼─────────┘    │
│  │                               │         │         │        │              │
│  │                               ▼         ▼         ▼        ▼              │
│  │  ┌───────────────────────────────────────────────────────────────────┐   │
│  │  │                      SKILL SYSTEM (Plugin Layer)                   │   │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────────┐  │   │
│  │  │  │ FileOps │ │ GitSkill│ │  Shell  │ │ TestRun │ │ Custom Skill│  │   │
│  │  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────────┘  │   │
│  │  └───────────────────────────────────────────────────────────────────┘   │
│  │                                           │                               │
│  │                                           ▼                               │
│  │  ┌───────────────────────────────────────────────────────────────────┐   │
│  │  │                    MCP CONNECTOR LAYER                             │   │
│  │  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐              │   │
│  │  │  │ MCP Server   │ │ MCP Server   │ │ MCP Server   │   ...        │   │
│  │  │  │ (filesystem) │ │  (git)       │ │ (postgres)   │              │   │
│  │  │  └──────────────┘ └──────────────┘ └──────────────┘              │   │
│  │  └───────────────────────────────────────────────────────────────────┘   │
│  │                                           │                               │
│  │                                           ▼                               │
│  │  ┌───────────────────────────────────────────────────────────────────┐   │
│  │  │                      LLM PROVIDER LAYER                            │   │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐          │   │
│  │  │  │ Claude │ │ OpenAI │ │ Google │ │ Ollama │ │ LiteLLM│          │   │
│  │  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘          │   │
│  └───────────────────────────────────────────────────────────────────────────┘
```

## ⚡ Quick Start

```bash
# Install SWARM
go install github.com/mojomast/clanker01/cmd/swarm@latest

# Initialize configuration
swarm init

# Start the TUI
swarm

# Or start as a server
swarm serve
```

## 📦 Installation

### Prerequisites

- Go 1.24 or higher
- Git
- A supported LLM provider API key

### From Source

```bash
# Clone the repository
git clone https://github.com/mojomast/clanker01.git
cd clanker01

# Build the binary
go build -o swarm ./cmd/swarm

# Or install to $GOPATH/bin
go install ./cmd/swarm
```

### From Binary

Download the latest binary for your platform from [releases](https://github.com/mojomast/clanker01/releases).

## ⚙️ Configuration

SWARM uses a YAML configuration file located at `~/.config/swarm/config.yaml` (or specified via `--config` flag).

### Example Configuration

```yaml
version: "1.0"
project_name: "my-swarm-project"

providers:
  anthropic:
    api_key: "your-anthropic-api-key"
    models:
      - id: "claude-3-sonnet-20240229"
        alias: "claude-3-sonnet"
        max_tokens: 200000

  openai:
    api_key: "your-openai-api-key"
    models:
      - id: "gpt-4"
        alias: "gpt-4"
        max_tokens: 128000

agents:
  architect:
    model: "claude-3-sonnet"
    max_concurrent: 3

  coder:
    model: "gpt-4"
    max_concurrent: 5

  tester:
    model: "claude-3-haiku"
    max_concurrent: 2

skills:
  directory: "~/.config/swarm/skills"
  auto_load: true

server:
  host: "0.0.0.0"
  port: 8080
  enable_tls: false

logging:
  level: "info"
  format: "json"
  file: "~/.config/swarm/logs/swarm.log"
```

### Environment Variables

- `SWARM_CONFIG`: Path to config file
- `SWARM_LOG_LEVEL`: Log level (debug, info, warn, error)
- `ANTHROPIC_API_KEY`: Anthropic API key
- `OPENAI_API_KEY`: OpenAI API key
- `GITHUB_TOKEN`: GitHub token for MCP operations

## 🎯 Usage

### CLI Usage

```bash
# Show help
swarm --help

# Connect to a remote SWARM server
swarm connect --url https://swarm.example.com --token your-token

# Agent management
swarm agent list
swarm agent create --type coder --model gpt-4
swarm agent delete --id agent-123
swarm agent info --id agent-123
swarm agent stats --agent agent-123

# Skill management
swarm skill list
swarm skill install --name filesystem --version 1.0.0
swarm skill search --query "git operations"
swarm skill info --name filesystem
```

### TUI Usage

```bash
# Launch the TUI interface
swarm

# TUI Keybindings:
# q         - Quit
# Tab       - Cycle between views
# 1-5       - Jump to specific view (Dashboard/Agents/Tasks/Logs/Config)
# ?         - Show help
# Esc       - Go back
# Ctrl+C    - Force quit
```

The TUI provides real-time views of:
- **Dashboard**: Agent overview, task queue, recent activity
- **Agents**: Individual agent status, metrics, current tasks
- **Tasks**: Task list with filtering, sorting, and details
- **Logs**: Real-time log streaming with filtering
- **Config**: Interactive configuration editor

### Server Usage

Start SWARM as a server for remote access:

```bash
# Start with default settings
swarm serve

# Start with custom configuration
swarm serve --config ./config.yaml

# Start with TLS
swarm serve --tls-cert ./cert.pem --tls-key ./key.pem

# Server will listen on:
# - gRPC: :8080
# - WebSocket: ws://localhost:8080/ws
# - REST API: http://localhost:8080/api
```

### Remote Client Usage

Connect to a running SWARM server:

```bash
# Connect to server
swarm connect --url https://swarm.example.com:8080

# Use a token for authentication
swarm connect --url https://swarm.example.com:8080 --token your-jwt-token

# Interact with remote SWARM via CLI or TUI
swarm agent list  # Lists agents on remote server
```

## 🔌 Developing Skills

SWARM skills are modular capabilities that can be hot-loaded. Skills can be written in:

- **Go** (native plugins)
- **Python** (via virtual environments)
- **Node.js** (via npm)
- **WebAssembly** (via WASM runtime)
- **Native binaries** (via JSON-RPC)

### Skill Manifest

Every skill requires a `skill.yaml` manifest:

```yaml
apiVersion: "swarm.ai/v1"
kind: "Skill"
metadata:
  name: "my-skill"
  version: "1.0.0"
  displayName: "My Custom Skill"
  description: "Performs custom operations"
  author: "Your Name"
  license: "Apache-2.0"
  tags: ["custom", "tool"]
spec:
  runtime: "go"
  entrypoint: "github.com/user/skills/myskill"
  tools:
    - name: "do_work"
      description: "Performs custom work"
      parameters:
        type: "object"
        required: ["input"]
        properties:
          input:
            type: "string"
            description: "Input to process"
  permissions:
    filesystem:
      read: ["**"]
      write: ["**"]
    network:
      allow: true
      allowedHosts: ["*"]
```

### Example Skill (Go)

```go
package main

import (
    "context"
    "github.com/swarm-ai/swarm/internal/skills/loader"
)

type MySkill struct {
    manifest *loader.SkillManifest
}

func (s *MySkill) Meta() *loader.SkillManifest {
    return s.manifest
}

func (s *MySkill) Initialize(ctx context.Context, config *loader.Config) error {
    // Initialization logic
    return nil
}

func (s *MySkill) Execute(ctx context.Context, tool string, args map[string]interface{}) (*loader.Result, error) {
    return &loader.Result{
        Success: true,
        Data: map[string]interface{}{
            "output": "processed " + args["input"].(string),
        },
    }, nil
}

func main() {
    skill := &MySkill{
        manifest: &loader.SkillManifest{
            // ... manifest data
        },
    }
    // Skill runtime entry point
}
```

## 🏛️ Architecture Modules

SWARM is organized into independent modules that can be developed in parallel:

### Phase 1: Foundation
- **A1: LLM Provider Layer** - Universal interface for 75+ LLM providers with retry, caching, cost tracking
- **A2: MCP Connector** - Model Context Protocol implementation for universal tool/database connectivity
- **A3: Configuration System** - YAML/JSON config loading with validation and environment overrides
- **G1: Authentication Framework** - JWT tokens, mTLS support, session management

### Phase 2: Core
- **B1: Agent Runtime** - Agent lifecycle, state machine, role-based pools
- **B2: Agent Orchestration** - Task scheduling, dependency graphs, conflict resolution, error recovery
- **C1: Context Store** - Tiered storage (hot/warm/cold) with LRU eviction and snapshots
- **D1: Skill Loader** - Multi-runtime skill loading with sandboxing and security profiles

### Phase 3: Features
- **B3: Task Decomposition** - LLM-based planning with 5 decomposition strategies (parallel, sequential, pipeline, map-reduce, divide-conquer)
- **C2: Knowledge Graph** - Code entity indexing, decision recording, graph-based RAG
- **C3: Session Management** - Hierarchical summarization, persistence, recovery with autosave
- **D2: Built-in Skills** - Filesystem, git, database, web operations
- **D3: Skill Registry** - Skill indexing, discovery, REST API for management
- **E1: TUI Core** - Bubbletea-based terminal UI with layouts, theming, key bindings

### Phase 4: Integration
- **E2: TUI Components** - Dashboard, agent view, task queue, logs, config, modals
- **E3: CLI Commands** - Cobra-based CLI with agent, skill, connect commands
- **F1: gRPC Server** - Protocol buffers, streaming, authentication middleware
- **F2: WebSocket Server** - Real-time updates, broadcasting, connection management
- **F3: REST API** - OpenAPI docs, rate limiting, structured logging
- **G2: RBAC Authorization** - Role-based access control, permission checking, middleware
- **G3: Monitoring & Logging** - Metrics collection, distributed tracing, alerting

## 📚 API Reference

### REST API

Base URL: `http://localhost:8080/api`

#### Authentication

All API endpoints require authentication via:
- JWT token in `Authorization: Bearer <token>` header
- Session cookie
- mTLS certificate

#### Endpoints

- `GET /v1/agents` - List all agents
- `POST /v1/agents` - Create new agent
- `GET /v1/agents/{id}` - Get agent details
- `PUT /v1/agents/{id}` - Update agent
- `DELETE /v1/agents/{id}` - Delete agent
- `POST /v1/agents/{id}/start` - Start agent
- `POST /v1/agents/{id}/stop` - Stop agent
- `POST /v1/agents/{id}/pause` - Pause agent
- `POST /v1/agents/{id}/resume` - Resume agent

- `GET /v1/tasks` - List all tasks
- `POST /v1/tasks` - Create new task
- `GET /v1/tasks/{id}` - Get task details
- `POST /v1/tasks/{id}/execute` - Execute task
- `POST /v1/tasks/{id}/cancel` - Cancel task

- `GET /v1/skills` - List all skills
- `POST /v1/skills` - Register new skill
- `GET /v1/skills/{name}` - Get skill details
- `DELETE /v1/skills/{name}` - Uninstall skill

OpenAPI/Swagger documentation available at: `http://localhost:8080/api/docs`

### gRPC API

Port: `:8080`

Services:
- `AgentService` - Agent CRUD and lifecycle operations
- `TaskService` - Task management and execution
- `SkillService` - Skill registration and discovery

### WebSocket API

URL: `ws://localhost:8080/ws`

Message Types:
- `agent_update` - Real-time agent status updates
- `task_event` - Task creation, completion, failure events
- `log_stream` - Streaming log messages
- `ping` / `pong` - Connection keep-alive

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests for specific module
go test ./internal/core/agent/...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.html coverage.out

# Run specific test
go test ./internal/core/agent -run TestAgentExecute -v
```

## Test Coverage

To generate and view test coverage:

```bash
# Run all tests with coverage
go test -coverprofile=coverage.out ./...

# View HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# View per-package coverage summary
go tool cover -func=coverage.out
```

Run `go test -cover ./...` to see current per-package coverage numbers.

## 🔒 Security

### Authentication Methods
- JWT token-based authentication
- Session-based authentication with expiration
- mTLS (mutual TLS) for enhanced security
- Pluggable credential validation (no hardcoded credentials)
- Token revocation via in-memory blacklist
- Cryptographically secure token/session/entity ID generation

### Authorization
- RBAC (Role-Based Access Control)
- Three default roles: admin, user, readonly
- Resource-based permissions (e.g., `tasks:read:task123`)
- Wildcard permissions (e.g., `agents:*`)

### Skill Sandboxing
- Three security profiles: restricted, standard, elevated
- Filesystem permission enforcement with glob patterns
- Network access control
- Environment variable filtering
- Resource limits (memory, CPU, timeout)

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- Built by [GLM-4.7](https://github.com/THUDM)
- Powered by [MCP (Model Context Protocol)](https://modelcontextprotocol.io/)
- TUI built with [Bubbletea](https://github.com/charmbracelet/bubbletea)
- Inspired by modern multi-agent AI research

## 📞 Support

- GitHub Issues: https://github.com/mojomast/clanker01/issues
- Documentation: https://github.com/mojomast/clanker01/wiki
- Discord: [Join our community](https://discord.gg/swarm)

---

**this jawn is 100% built by glm** 🚀
