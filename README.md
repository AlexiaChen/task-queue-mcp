# Issue Kanban MCP Server

[![CI](https://github.com/AlexiaChen/issue-kanban-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/AlexiaChen/issue-kanban-mcp/actions/workflows/ci.yml)
[![CD](https://github.com/AlexiaChen/issue-kanban-mcp/actions/workflows/cd.yml/badge.svg)](https://github.com/AlexiaChen/issue-kanban-mcp/actions/workflows/cd.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AlexiaChen/issue-kanban-mcp)](https://goreportcard.com/report/github.com/AlexiaChen/issue-kanban-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go-based MCP (Model Context Protocol) Server that manages multiple Issue Kanbans with Web UI and REST API.

## Features

- **MCP Tools**: 8 tools for project and issue management
- **MCP Resources**: 4 resources for reading project/issue data
- **REST API**: Full CRUD API for projects and issues
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

### Docker (Recommended)

Pull the image from GitHub Container Registry:

```bash
docker pull ghcr.io/alexiachen/issue-kanban-mcp:latest
```

Run with HTTP mode (Web UI + REST API + MCP SSE):

```bash
docker run -d \
  --name issue-kanban-mcp \
  -p 9292:9292 \
  -v task-queue-data:/app/data \
  ghcr.io/alexiachen/issue-kanban-mcp:latest
```

Access:
- **Web UI**: http://localhost:9292
- **REST API**: http://localhost:9292/api/...
- **MCP SSE**: http://localhost:9292/sse

### From Source

```bash
git clone https://github.com/alexiachen/issue-kanban-mcp.git
cd issue-kanban-mcp
make build
```

### Binary

The binary is statically compiled with no dynamic dependencies:

```bash
./bin/issue-kanban-mcp -port=9292 -mcp=http
```

## Running Modes

| Mode | Command | Description |
|------|---------|-------------|
| HTTP | `./bin/issue-kanban-mcp -mcp=http` | Web UI + REST API + MCP SSE |
| STDIO | `./bin/issue-kanban-mcp -mcp=stdio` | For MCP clients like Claude Desktop |
| Both | `./bin/issue-kanban-mcp -mcp=both` | Both transports enabled |

## MCP Integration

### Local STDIO Mode

For local MCP clients (Claude Desktop, Copilot CLI, etc.), use STDIO mode:

```bash
# Build the binary
make build

# Run in STDIO mode
./bin/issue-kanban-mcp -mcp=stdio -db=/path/to/tasks.db
```

### Claude Desktop Configuration

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "issue-kanban": {
      "command": "/path/to/issue-kanban-mcp",
      "args": ["-mcp=stdio", "-db=/path/to/tasks.db"]
    }
  }
}
```

### GitHub Copilot CLI Configuration

Copilot CLI supports MCP servers via `~/.copilot/mcp-config.json`.

#### Option 1: STDIO Mode (Local Binary)

```json
{
  "mcpServers": {
    "issue-kanban": {
      "type": "stdio",
      "command": "/path/to/issue-kanban-mcp",
      "args": ["-mcp=stdio", "-db=/path/to/tasks.db"],
      "env": {},
      "tools": ["*"]
    }
  }
}
```

#### Option 2: HTTP/SSE Mode (Docker or Remote Server)

```json
{
  "mcpServers": {
    "issue-kanban": {
      "type": "sse",
      "url": "http://localhost:9292/sse",
      "headers": {},
      "tools": ["*"]
    }
  }
}
```

#### Interactive Setup (Recommended)

Copilot CLI also provides an interactive command to add MCP servers:

```bash
copilot mcp add
```

This will guide you through the configuration process interactively.

For more details, see [Copilot CLI MCP Documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers).

### MCP Tools

| Tool | Description |
|------|-------------|
| `project_list` | List all projects with stats |
| `project_create` | Create a new project |
| `project_delete` | Delete a project |
| `issue_list` | List issues in a project |
| `issue_create` | Create a new issue |
| `issue_update` | Update issue status |
| `issue_delete` | Delete an issue |
| `issue_prioritize` | Move issue to front (插队) |

### MCP Resources

| URI | Description |
|-----|-------------|
| `project://list` | List all projects |
| `project://{id}` | Get project details |
| `project://{id}/issues` | Get project issues |
| `issue://{id}` | Get issue details |

## Issue Status

Issues have three states:

- `pending` - Waiting to be processed
- `doing` - Currently being processed
- `finished` - Completed

## REST API

### Projects

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/projects` | List all projects |
| POST | `/api/projects` | Create project |
| GET | `/api/projects/{id}` | Get project |
| DELETE | `/api/projects/{id}` | Delete project |
| GET | `/api/projects/{id}/issues` | Get project issues |
| GET | `/api/projects/{id}/stats` | Get project stats |

### Issues

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/issues` | Create issue |
| GET | `/api/issues/{id}` | Get issue |
| PATCH | `/api/issues/{id}` | Update issue |
| DELETE | `/api/issues/{id}` | Delete issue |
| POST | `/api/issues/{id}/start` | Start issue (pending → doing) |
| POST | `/api/issues/{id}/finish` | Finish issue (doing → finished) |
| POST | `/api/issues/{id}/prioritize` | Prioritize issue (插队) |

## Web UI

Open http://localhost:9292 in your browser to access the Web UI for managing projects and issues visually.

## AI Agent Instructions

The `instructions/` directory contains ready-to-use instructions for AI agents to process issues from projects automatically.

| File | Description |
|------|-------------|
| `CLAUDE.md` | Instruction for Claude Code / Claude Desktop |
| `AGENTS.md` | Instruction for other AI agents (GitHub Copilot, Gemini, etc.) |

### Usage

Copy the content of `CLAUDE.md` or `AGENTS.md` into your project's instruction file (e.g., `.claude/CLAUDE.md` or `AGENTS.md`). When you ask the AI to "process all issues in project X", it will:

1. Find the project by name
2. Loop through pending issues (sorted by priority, then position)
3. For each issue: mark as "doing" → execute the work → mark as "finished"
4. Stop when all issues are completed

This enables autonomous issue processing where the AI acts as a worker for your projects.

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
│   ├── manager.go          # Project manager
│   ├── models.go           # Data models
│   └── mock_storage.go     # Mock storage for testing
├── storage/sqlite.go       # SQLite persistence
└── web/                    # Web UI (embedded)
    └── static/
examples/
├── mcp-client/main.go      # SSE client example
└── stdio-client/main.go    # STDIO client example
instructions/
├── CLAUDE.md               # Claude Code issue processing instruction
└── AGENTS.md               # Generic AI agent issue processing instruction
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
