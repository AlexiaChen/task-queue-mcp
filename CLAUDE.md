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
# Build server only
make build

# Build TUI
make build-tui

# Build CLI
make build-cli

# Build all (server + TUI + CLI)
make build-all

# Run server (default port 9292)
make run

# Run all tests
make test

# Run e2e tests
make e2e
```

## Architecture

```
cmd/server/main.go          # MCP + HTTP server entry point
cmd/tui/main.go             # TUI entry point (--server flag)
cmd/cli/main.go             # CLI entry point (--server flag)
internal/
├── api/handlers.go         # REST API handlers
├── apiclient/client.go     # Shared REST API client (used by TUI & CLI)
├── mcp/                    # MCP server implementation
│   ├── server.go           # MCP server setup
│   ├── tools.go            # 8 MCP tools
│   └── resources.go        # 4 MCP resources
├── queue/                  # Business logic layer
│   ├── manager.go          # Queue manager
│   ├── models.go           # Data models
│   └── mock_storage.go     # Mock storage for testing
├── storage/sqlite.go       # SQLite persistence
├── tui/                    # TUI implementation (bubbletea)
│   ├── app.go              # Main bubbletea model
│   └── styles.go           # Lipgloss styles
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

## TUI (Terminal UI)

A bubbletea-based terminal UI that mirrors the Web UI functionality.

```bash
# Start TUI (server must be running)
./bin/task-queue-tui --server http://localhost:9292

# Build TUI
make build-tui
```

**Key bindings:**
| Key | Action |
|-----|--------|
| `j`/`k` or `↑`/`↓` | Navigate list |
| `Enter` | Open queue |
| `n` | New queue / task |
| `d` | Delete selected |
| `s` | Start task (pending → doing) |
| `f` | Finish task (doing → finished) |
| `r` | Reset task (finished → pending) |
| `p` | Prioritize task (move to front) |
| `R` | Manual refresh |
| `Esc` / `q` | Back / quit |

## CLI

A cobra-based command-line interface for scripting and automation.

```bash
# Build CLI
make build-cli

# Usage
./bin/task-queue-cli --server http://localhost:9292 queues list
./bin/task-queue-cli queues create --name "my-queue" --desc "description"
./bin/task-queue-cli queues delete <id>
./bin/task-queue-cli queues stats <id>

./bin/task-queue-cli tasks list <queue-id> [--status pending|doing|finished]
./bin/task-queue-cli tasks get <id>
./bin/task-queue-cli tasks create <queue-id> --title "task" [--desc "..."] [--priority 0]
./bin/task-queue-cli tasks start <id>
./bin/task-queue-cli tasks finish <id>
./bin/task-queue-cli tasks reset <id>
./bin/task-queue-cli tasks delete <id> [--yes]
./bin/task-queue-cli tasks prioritize <id>
```

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

---

## Task Queue Processing Instructions

When asked to process a task queue using this MCP server, follow the workflow below.

### Step 1: Identify the Queue

Use `queue_list` to find the target queue by name:

```
queue_list -> find queue with matching name -> get queue_id
```

If the queue doesn't exist, report an error and stop.

### Step 2: Process Tasks in Loop

Repeat until all tasks are finished:

1. **Get pending tasks** using `task_list`:
   ```
   task_list(queue_id=<queue_id>) -> returns tasks sorted by priority DESC, position ASC
   ```

2. **Check for pending tasks**:
   - If no pending tasks remain, exit the loop (done!)

3. **Pick the next task** (first in the sorted list - highest priority, earliest position)

4. **Start the task** using `task_update`:
   ```
   task_update(task_id=<task_id>, status="doing")
   ```

5. **Execute the task**:
   - Read the task `title` and `description`
   - Perform the work described in the task
   - This is the actual work you need to do (coding, analysis, writing, etc.)

6. **Finish the task** using `task_update`:
   ```
   task_update(task_id=<task_id>, status="finished")
   ```

7. **Loop back** to step 2

### Step 3: Interactive Continuation

When the loop exits (no more pending tasks in the current queue), **do NOT immediately print the completion report**. Instead, use the `ask_user` tool to present an interactive selection:

```
ask_user(
  question = "队列 '{queue_name}' 的所有任务已处理完毕。是否需要继续处理其他队列？",
  choices  = [
    "继续处理其他队列",
    "不，已完成，输出最终报告"
  ]
)
```

- If user selects **"继续处理其他队列"**: call `queue_list`, let the user pick a new queue, then restart from Step 1 with the new queue name.
- If user selects **"不，已完成"**: proceed to Step 4.

### Step 4: Completion

When the user confirms they are done, report:
- Total tasks processed (across all queues in this session)
- Summary of work completed per queue

### Important Rules

1. **Process one task at a time** - Do not batch process multiple tasks
2. **Respect priority order** - Higher priority tasks must be done first
3. **Respect position order** - For same priority, earlier positions go first
4. **Always update status** - Mark tasks as "doing" before work, "finished" after
5. **Handle errors gracefully** - If a task fails, report the error but continue with the next task
6. **Never skip tasks** - Process all pending tasks until the queue is empty
7. **Interactive continuation** - After a queue is fully drained, always use `ask_user` to prompt before exiting; never terminate silently

### MCP Tools Reference

| Tool | Purpose |
|------|---------|
| `queue_list` | List all queues to find target by name |
| `task_list` | Get tasks in a queue (with optional status filter) |
| `task_update` | Change task status (pending → doing → finished) |

### Task Status Flow

```
pending → doing → finished
```

- `pending`: Task is waiting to be processed
- `doing`: Task is currently being worked on
- `finished`: Task is complete
