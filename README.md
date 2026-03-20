# Task Queue MCP Server

A Go-based MCP (Model Context Protocol) Server that manages multiple Task Queues with Web UI and REST API.

## Features

- **MCP Tools**: 8 tools for queue and task management
- **MCP Resources**: 4 resources for reading queue/task data
- **REST API**: Full CRUD API for queues and tasks
- **Web UI**: Embedded single-page application for visual management
- **SQLite Storage**: Persistent data storage
- **Multiple Transports**: STDIO and HTTP/SSE support

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

## Installation

### From Source

```bash
git clone <repo-url>
cd task-queue-mcp
make build
```

### Binary

The binary is statically compiled with no dynamic dependencies:

```bash
./bin/task-queue-mcp -port=9292 -mcp=http
```

## Running Modes

| Mode | Command | Description |
|------|---------|-------------|
| HTTP | `./bin/task-queue-mcp -mcp=http` | Web UI + REST API + MCP SSE |
| STDIO | `./bin/task-queue-mcp -mcp=stdio` | For MCP clients like Claude Desktop |
| Both | `./bin/task-queue-mcp -mcp=both` | Both transports enabled |

## MCP Integration

### Claude Desktop Configuration

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "task-queue": {
      "command": "/path/to/task-queue-mcp",
      "args": ["-mcp=stdio", "-db=/path/to/tasks.db"]
    }
  }
}
```

### MCP Tools

| Tool | Description |
|------|-------------|
| `queue_list` | List all queues with stats |
| `queue_create` | Create a new queue |
| `queue_delete` | Delete a queue |
| `task_list` | List tasks in a queue |
| `task_create` | Create a new task |
| `task_update` | Update task status |
| `task_delete` | Delete a task |
| `task_prioritize` | Move task to front (插队) |

### MCP Resources

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

## REST API

### Queues

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/queues` | List all queues |
| POST | `/api/queues` | Create queue |
| GET | `/api/queues/{id}` | Get queue |
| DELETE | `/api/queues/{id}` | Delete queue |
| GET | `/api/queues/{id}/tasks` | Get queue tasks |
| GET | `/api/queues/{id}/stats` | Get queue stats |

### Tasks

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/tasks` | Create task |
| GET | `/api/tasks/{id}` | Get task |
| PATCH | `/api/tasks/{id}` | Update task |
| DELETE | `/api/tasks/{id}` | Delete task |
| POST | `/api/tasks/{id}/start` | Start task (pending → doing) |
| POST | `/api/tasks/{id}/finish` | Finish task (doing → finished) |
| POST | `/api/tasks/{id}/prioritize` | Prioritize task (插队) |

## Web UI

Open http://localhost:9292 in your browser to access the Web UI for managing queues and tasks visually.

## Project Structure

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
examples/
├── mcp-client/main.go      # SSE client example
└── stdio-client/main.go    # STDIO client example
```

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `9292` | HTTP server port |
| `-db` | `./data/tasks.db` | SQLite database path |
| `-mcp` | `http` | MCP mode: stdio, http, or both |

Environment variables are also supported:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9292` | HTTP server port |
| `DB_PATH` | `./data/tasks.db` | SQLite database path |
| `MCP_MODE` | `http` | MCP mode: stdio, http, or both |

## Development

```bash
# Run in development mode with auto-reload (requires air)
make dev

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt
```

## Examples

### STDIO Client (like Claude Desktop)

```bash
make example-stdio
```

### HTTP Client

```bash
# Start server first
make run

# In another terminal
make example-client
```

## License

MIT
