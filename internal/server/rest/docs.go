package rest

import (
	"net/http"
)

const swaggerSpec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "SWARM AI REST API",
    "description": "REST API for the SWARM AI agent orchestration system",
    "version": "1.0.0",
    "contact": {
      "name": "SWARM Team",
      "email": "support@swarm.ai"
    }
  },
  "servers": [
    {
      "url": "http://localhost:8080/api/v1",
      "description": "Development server"
    }
  ],
  "tags": [
    {
      "name": "health",
      "description": "Health check endpoints"
    },
    {
      "name": "agents",
      "description": "Agent management endpoints"
    },
    {
      "name": "tasks",
      "description": "Task management endpoints"
    },
    {
      "name": "skills",
      "description": "Skill management endpoints"
    },
    {
      "name": "config",
      "description": "Configuration endpoints"
    }
  ],
  "paths": {
    "/health": {
      "get": {
        "tags": ["health"],
        "summary": "Check API health",
        "description": "Returns the health status of the API",
        "operationId": "getHealth",
        "responses": {
          "200": {
            "description": "Healthy",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/HealthResponse"
                }
              }
            }
          }
        }
      }
    },
    "/agents": {
      "get": {
        "tags": ["agents"],
        "summary": "List all agents",
        "description": "Returns a list of all registered agents",
        "operationId": "listAgents",
        "security": [{"bearerAuth": []}],
        "responses": {
          "200": {
            "description": "List of agents",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/AgentListResponse"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      },
      "post": {
        "tags": ["agents"],
        "summary": "Create a new agent",
        "description": "Creates and registers a new agent",
        "operationId": "createAgent",
        "security": [{"bearerAuth": []}],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/AgentRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Agent created successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/AgentResponse"
                }
              }
            }
          },
          "400": {
            "description": "Invalid request"
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      }
    },
    "/agents/{id}": {
      "get": {
        "tags": ["agents"],
        "summary": "Get agent details",
        "description": "Returns detailed information about a specific agent",
        "operationId": "getAgent",
        "security": [{"bearerAuth": []}],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string"
            },
            "description": "Agent ID"
          }
        ],
        "responses": {
          "200": {
            "description": "Agent details",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/AgentResponse"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized"
          },
          "404": {
            "description": "Agent not found"
          }
        }
      },
      "put": {
        "tags": ["agents"],
        "summary": "Update agent",
        "description": "Updates an existing agent's configuration",
        "operationId": "updateAgent",
        "security": [{"bearerAuth": []}],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string"
            }
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/AgentRequest"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Agent updated successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/AgentResponse"
                }
              }
            }
          },
          "400": {
            "description": "Invalid request"
          },
          "401": {
            "description": "Unauthorized"
          },
          "404": {
            "description": "Agent not found"
          }
        }
      },
      "delete": {
        "tags": ["agents"],
        "summary": "Delete agent",
        "description": "Deletes an agent",
        "operationId": "deleteAgent",
        "security": [{"bearerAuth": []}],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string"
            }
          }
        ],
        "responses": {
          "204": {
            "description": "Agent deleted successfully"
          },
          "401": {
            "description": "Unauthorized"
          },
          "404": {
            "description": "Agent not found"
          }
        }
      }
    },
    "/agents/{id}/start": {
      "post": {
        "tags": ["agents"],
        "summary": "Start agent",
        "description": "Starts a stopped or created agent",
        "operationId": "startAgent",
        "security": [{"bearerAuth": []}],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Agent started successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/AgentResponse"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized"
          },
          "404": {
            "description": "Agent not found"
          }
        }
      }
    },
    "/agents/{id}/stop": {
      "post": {
        "tags": ["agents"],
        "summary": "Stop agent",
        "description": "Stops a running agent",
        "operationId": "stopAgent",
        "security": [{"bearerAuth": []}],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Agent stopped successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/AgentResponse"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized"
          },
          "404": {
            "description": "Agent not found"
          }
        }
      }
    },
    "/tasks": {
      "get": {
        "tags": ["tasks"],
        "summary": "List all tasks",
        "description": "Returns a list of all tasks",
        "operationId": "listTasks",
        "security": [{"bearerAuth": []}],
        "responses": {
          "200": {
            "description": "List of tasks",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/TaskListResponse"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      },
      "post": {
        "tags": ["tasks"],
        "summary": "Create a new task",
        "description": "Creates and queues a new task",
        "operationId": "createTask",
        "security": [{"bearerAuth": []}],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/TaskRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Task created successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/TaskResponse"
                }
              }
            }
          },
          "400": {
            "description": "Invalid request"
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      }
    },
    "/tasks/{id}": {
      "get": {
        "tags": ["tasks"],
        "summary": "Get task details",
        "description": "Returns detailed information about a specific task",
        "operationId": "getTask",
        "security": [{"bearerAuth": []}],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Task details",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/TaskResponse"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized"
          },
          "404": {
            "description": "Task not found"
          }
        }
      }
    },
    "/skills": {
      "get": {
        "tags": ["skills"],
        "summary": "List all skills",
        "description": "Returns a list of all loaded skills",
        "operationId": "listSkills",
        "security": [{"bearerAuth": []}],
        "responses": {
          "200": {
            "description": "List of skills",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/SkillListResponse"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      },
      "post": {
        "tags": ["skills"],
        "summary": "Load a skill",
        "description": "Loads and registers a new skill",
        "operationId": "loadSkill",
        "security": [{"bearerAuth": []}],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/SkillRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Skill loaded successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/SkillResponse"
                }
              }
            }
          },
          "400": {
            "description": "Invalid request"
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      }
    },
    "/config": {
      "get": {
        "tags": ["config"],
        "summary": "Get configuration",
        "description": "Returns the current configuration",
        "operationId": "getConfig",
        "security": [{"bearerAuth": []}],
        "responses": {
          "200": {
            "description": "Current configuration",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ConfigResponse"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      }
    },
    "schemas": {
      "HealthResponse": {
        "type": "object",
        "properties": {
          "status": {
            "type": "string",
            "example": "healthy"
          },
          "timestamp": {
            "type": "string",
            "format": "date-time"
          },
          "version": {
            "type": "string"
          },
          "uptime": {
            "type": "string"
          }
        }
      },
      "AgentRequest": {
        "type": "object",
        "required": ["name"],
        "properties": {
          "type": {
            "type": "string",
            "enum": ["architect", "coder", "tester", "reviewer", "researcher", "coordinator"],
            "default": "coder"
          },
          "name": {
            "type": "string"
          },
          "model": {
            "type": "string"
          },
          "skills": {
            "type": "array",
            "items": {
              "type": "string"
            }
          },
          "config": {
            "type": "object"
          }
        }
      },
      "AgentResponse": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string"
          },
          "type": {
            "type": "string"
          },
          "name": {
            "type": "string"
          },
          "status": {
            "type": "string",
            "enum": ["created", "ready", "running", "paused", "error", "terminated"]
          },
          "created_at": {
            "type": "string",
            "format": "date-time"
          },
          "updated_at": {
            "type": "string",
            "format": "date-time"
          },
          "config": {
            "type": "object"
          },
          "metrics": {
            "type": "object"
          }
        }
      },
      "AgentListResponse": {
        "type": "object",
        "properties": {
          "count": {
            "type": "integer"
          },
          "agents": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/AgentResponse"
            }
          }
        }
      },
      "TaskRequest": {
        "type": "object",
        "required": ["prompt"],
        "properties": {
          "type": {
            "type": "string"
          },
          "prompt": {
            "type": "string"
          },
          "agent_type": {
            "type": "string",
            "enum": ["architect", "coder", "tester", "reviewer", "researcher", "coordinator"]
          },
          "agent_id": {
            "type": "string"
          },
          "priority": {
            "type": "integer",
            "default": 0
          },
          "max_retries": {
            "type": "integer",
            "default": 3
          },
          "timeout": {
            "type": "string"
          },
          "metadata": {
            "type": "object"
          }
        }
      },
      "TaskResponse": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string"
          },
          "type": {
            "type": "string"
          },
          "prompt": {
            "type": "string"
          },
          "status": {
            "type": "string",
            "enum": ["pending", "queued", "running", "blocked", "completed", "failed", "cancelled"]
          },
          "agent_type": {
            "type": "string"
          },
          "assigned_agent": {
            "type": "string"
          },
          "created_at": {
            "type": "string",
            "format": "date-time"
          },
          "started_at": {
            "type": "string",
            "format": "date-time"
          },
          "completed_at": {
            "type": "string",
            "format": "date-time"
          },
          "result": {
            "type": "object"
          },
          "error": {
            "type": "string"
          },
          "retry_count": {
            "type": "integer"
          },
          "metadata": {
            "type": "object"
          }
        }
      },
      "TaskListResponse": {
        "type": "object",
        "properties": {
          "count": {
            "type": "integer"
          },
          "tasks": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/TaskResponse"
            }
          }
        }
      },
      "SkillRequest": {
        "type": "object",
        "required": ["name"],
        "properties": {
          "name": {
            "type": "string"
          },
          "version": {
            "type": "string"
          },
          "config": {
            "type": "object"
          },
          "enable": {
            "type": "boolean",
            "default": true
          }
        }
      },
      "SkillResponse": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string"
          },
          "name": {
            "type": "string"
          },
          "version": {
            "type": "string"
          },
          "description": {
            "type": "string"
          },
          "status": {
            "type": "string"
          },
          "loaded_at": {
            "type": "string",
            "format": "date-time"
          },
          "config": {
            "type": "object"
          },
          "tools": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/ToolInfo"
            }
          }
        }
      },
      "SkillListResponse": {
        "type": "object",
        "properties": {
          "count": {
            "type": "integer"
          },
          "skills": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/SkillResponse"
            }
          }
        }
      },
      "ToolInfo": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string"
          },
          "description": {
            "type": "string"
          },
          "parameters": {
            "type": "object"
          },
          "returns": {
            "type": "object"
          }
        }
      },
      "ConfigResponse": {
        "type": "object",
        "properties": {
          "version": {
            "type": "string"
          },
          "project": {
            "type": "object"
          },
          "llm": {
            "type": "object"
          },
          "agents": {
            "type": "object"
          },
          "skills": {
            "type": "object"
          },
          "server": {
            "type": "object"
          },
          "security": {
            "type": "object"
          }
        }
      }
    }
  }
}`

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(swaggerUIHTML))
}

func (s *Server) handleSwaggerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(swaggerSpec))
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SWARM AI API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin: 0;
            padding: 0;
        }
        .topbar-wrapper .download-url-wrapper .download-url-button {
            display: none;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "/api/v1/swagger.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                docExpansion: "list",
                filter: true,
                showRequestHeaders: true
            });
        };
    </script>
</body>
</html>`
