# Task Queue MCP Server

A Go-based MCP (Model Context Protocol) Server that manages multiple Task Queues with Web UI and REST API.

## Project Overview

This is an MCP server implementation in Go that provides:
- **MCP Tools**: 3 readonly tools + 5 admin tools for queue and task management
- **MCP Resources**: 4 resources for reading queue/task data
- **REST API**: Full CRUD API for queues and tasks
- **Web UI**: Embedded single-page application for visual management
- **SQLite Storage**: Persistent data storage

## Quick Start

```bash
# Build
make build

# Run server (default port 9292)
make run

# Run all tests
make test

# Run e2e tests
make e2e
```

## Architecture

```
cmd/server/main.go          # Entry point
internal/
├── api/handlers.go         # REST API handlers
├── mcp/                    # MCP server implementation
│   ├── server.go           # MCP server setup
│   ├── tools.go            # 8 MCP tools
│   └── resources.go        # 4 MCP resources
├── queue/                  # Business logic layer
│   ├── manager.go          # Queue manager
│   ├── models.go           # Data models
│   └── mock_storage.go     # Mock storage for testing
├── storage/sqlite.go       # SQLite persistence
└── web/                    # Web UI (embedded)
    └── static/
```

## MCP Tools

### Readonly Mode (Default - for AI Agents)

| Tool | Description |
|------|-------------|
| `queue_list` | List all queues with stats |
| `task_list` | List tasks in a queue |
| `task_update` | Update task status |

### Admin Tools (require `-readonly=false`)

| Tool | Description |
|------|-------------|
| `queue_create` | Create a new queue |
| `queue_delete` | Delete a queue |
| `task_create` | Create a new task |
| `task_delete` | Delete a task |
| `task_prioritize` | Move task to front (插队) |

## MCP Resources

| URI | Description |
|-----|-------------|
| `queue://list` | List all queues |
| `queue://{id}` | Get queue details |
| `queue://{id}/tasks` | Get queue tasks |
| `task://{id}` | Get task details |

## Task Status

Tasks have three states:
- `pending` - Waiting to be processed
- `doing` - Currently being processed
- `finished` - Completed

## Running Modes

```bash
# HTTP mode (Web UI + REST API + MCP SSE) - readonly by default
./bin/task-queue-mcp -port=9292 -mcp=http

# STDIO mode (for MCP clients like Claude) - readonly by default
./bin/task-queue-mcp -mcp=stdio

# Both modes
./bin/task-queue-mcp -port=9292 -mcp=both

# Admin mode (full access to all MCP tools)
./bin/task-queue-mcp -readonly=false
```

> **Note**: Readonly mode is enabled by default for AI safety. Use `-readonly=false` or `MCP_READONLY=false` for admin access.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9292` | HTTP server port |
| `DB_PATH` | `./data/tasks.db` | SQLite database path |
| `MCP_MODE` | `http` | MCP mode: stdio, http, or both |
| `MCP_READONLY` | `true` | Set to `false` to expose all MCP tools |

## Testing

```bash
# Unit tests
make test

# Test with coverage
make test-coverage

# E2E tests (starts server automatically)
make e2e

# E2E tests against running server
make run &
make e2e-quick
```

## Static Build

The binary is compiled statically with no dynamic dependencies:

```bash
make build  # Uses CGO_ENABLED=0
```

## API Endpoints

### Queues
- `GET /api/queues` - List all queues
- `POST /api/queues` - Create queue
- `GET /api/queues/{id}` - Get queue
- `DELETE /api/queues/{id}` - Delete queue
- `GET /api/queues/{id}/tasks` - Get queue tasks
- `GET /api/queues/{id}/stats` - Get queue stats

### Tasks
- `POST /api/tasks` - Create task
- `GET /api/tasks/{id}` - Get task
- `PATCH /api/tasks/{id}` - Update task
- `DELETE /api/tasks/{id}` - Delete task
- `POST /api/tasks/{id}/start` - Start task (pending → doing)
- `POST /api/tasks/{id}/finish` - Finish task (doing → finished)
- `POST /api/tasks/{id}/prioritize` - Prioritize task (插队)

## Key Dependencies

- `github.com/mark3labs/mcp-go` - MCP Go SDK
- `modernc.org/sqlite` - Pure Go SQLite driver

## Code Conventions

- Use `queue.Manager` for all business logic
- Storage interface allows swapping implementations
- Mock storage available for testing (`queue.NewMockStorage()`)
- All handlers are in `internal/api/handlers.go`
- MCP tools/resources are in `internal/mcp/`
