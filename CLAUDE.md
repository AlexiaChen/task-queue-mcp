# Issue Kanban MCP Server

A Go-based MCP (Model Context Protocol) Server that manages multiple Issue Kanbans with Web UI and REST API.

## Project Overview

This is an MCP server implementation in Go that provides:
- **MCP Tools**: 3 readonly tools + 5 admin tools for project and issue management
- **MCP Resources**: 4 resources for reading project/issue data
- **REST API**: Full CRUD API for projects and issues
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
│   ├── manager.go          # Project manager
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
| `project_list` | List all projects with stats |
| `issue_list` | List issues in a project |
| `issue_update` | Update issue status |

### Admin Tools (require `-readonly=false`)

| Tool | Description |
|------|-------------|
| `project_create` | Create a new project |
| `project_delete` | Delete a project |
| `issue_create` | Create a new issue |
| `issue_delete` | Delete an issue |
| `issue_prioritize` | Move issue to front (插队) |

## MCP Resources

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

## TUI (Terminal UI)

A bubbletea-based terminal UI that mirrors the Web UI functionality.

```bash
# Start TUI (server must be running)
./bin/issue-kanban-tui --server http://localhost:9292

# Build TUI
make build-tui
```

**Key bindings:**
| Key | Action |
|-----|--------|
| `j`/`k` or `↑`/`↓` | Navigate list |
| `Enter` | Open project |
| `n` | New project / issue |
| `e` | Edit selected issue (pending only) |
| `d` | Delete selected |
| `p` | Prioritize issue (move to front) |
| `R` | Manual refresh |
| `Esc` / `q` | Back / quit |

## CLI

A cobra-based command-line interface for scripting and automation.

```bash
# Build CLI
make build-cli

# Usage
./bin/issue-kanban-cli --server http://localhost:9292 projects list
./bin/issue-kanban-cli projects create --name "my-project" --desc "description"
./bin/issue-kanban-cli projects delete <id>
./bin/issue-kanban-cli projects stats <id>

./bin/issue-kanban-cli issues list <project-id> [--status pending|doing|finished]
./bin/issue-kanban-cli issues get <id>
./bin/issue-kanban-cli issues create <project-id> --title "issue" [--desc "..."] [--priority 0]
./bin/issue-kanban-cli issues edit <id> [--title "new title"] [--desc "new desc"] [--priority 1]
./bin/issue-kanban-cli issues delete <id> [--yes]
./bin/issue-kanban-cli issues prioritize <id>
```

## Running Modes

```bash
# HTTP mode (Web UI + REST API + MCP SSE) - readonly by default
./bin/issue-kanban-mcp -port=9292 -mcp=http

# STDIO mode (for MCP clients like Claude) - readonly by default
./bin/issue-kanban-mcp -mcp=stdio

# Both modes
./bin/issue-kanban-mcp -port=9292 -mcp=both

# Admin mode (full access to all MCP tools)
./bin/issue-kanban-mcp -readonly=false
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
# Unit tests (includes CLI tests)
make test

# Test with coverage
make test-coverage

# E2E tests (starts server automatically)
make e2e

# E2E tests against running server
make run &
make e2e-quick
```

### CLI Testing Strategy

CLI tests live in `cmd/cli/main_test.go` and use `net/http/httptest.NewServer` to spin up a mock REST API server for each test case. This means:
- Tests run without a real server or database
- Every subcommand (`projects list/create/delete/stats`, `issues list/get/create/edit/delete/prioritize`) has a dedicated test
- Tests verify both output text and HTTP method/path correctness
- The `runCmd(t, ts, ...)` helper wires the CLI to the mock server via `--server` flag

## Static Build

The binary is compiled statically with no dynamic dependencies:

```bash
make build  # Uses CGO_ENABLED=0
```

## API Endpoints

### Projects
- `GET /api/projects` - List all projects
- `POST /api/projects` - Create project
- `GET /api/projects/{id}` - Get project
- `DELETE /api/projects/{id}` - Delete project
- `GET /api/projects/{id}/issues` - Get project issues
- `GET /api/projects/{id}/stats` - Get project stats

### Issues
- `POST /api/issues` - Create issue
- `GET /api/issues/{id}` - Get issue
- `PATCH /api/issues/{id}` - Update issue **status** (pending/doing/finished) — used by MCP
- `PUT /api/issues/{id}` - Edit issue content (title/description/priority, pending only)
- `DELETE /api/issues/{id}` - Delete issue
- `POST /api/issues/{id}/prioritize` - Prioritize issue (插队)

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

## Issue Kanban Processing Instructions

When asked to process an issue kanban using this MCP server, follow the workflow below.

### Step 1: Identify the Project

Use `project_list` to find the target project by name:

```
project_list -> find project with matching name -> get queue_id
```

If the project doesn't exist, report an error and stop.

### Step 2: Process Issues in Loop

Repeat until all issues are finished:

1. **Get pending issues** using `issue_list`:
   ```
   issue_list(queue_id=<queue_id>) -> returns issues sorted by priority DESC, position ASC
   ```

2. **Check for pending issues**:
   - If no pending issues remain, exit the loop (done!)

3. **Pick the next issue** (first in the sorted list - highest priority, earliest position)

4. **Start the issue** using `issue_update`:
   ```
   issue_update(task_id=<task_id>, status="doing")
   ```

5. **Execute the issue**:
   - Read the issue `title` and `description`
   - Perform the work described in the issue
   - This is the actual work you need to do (coding, analysis, writing, etc.)

6. **Finish the issue** using `issue_update`:
   ```
   issue_update(task_id=<task_id>, status="finished")
   ```

7. **Loop back** to step 2

### Step 3: Interactive Continuation

When the loop exits (no more pending issues in the current project), **do NOT immediately print the completion report**. Instead, use the `ask_user` tool to present an interactive selection:

```
ask_user(
  question = "Project '{project_name}' (id={project_id}) is fully processed. Continue with the current project?",
  choices  = [
    "Continue current project (re-check for newly added pending issues)",
    "Switch to another project",
    "No, done — print final report"
  ]
)
```

- If user selects **"Continue current project"**: loop back to Step 2 with the same `queue_id` (new issues may have been added).
- If user selects **"Switch to another project"**: call `project_list`, let the user pick a new project, then restart from Step 1 with the new project name.
- If user selects **"No, done"**: proceed to Step 4.

### Step 4: Completion

When the user confirms they are done, report:
- Total issues processed (across all projects in this session)
- Summary of work completed per project

### Important Rules

1. **Process one issue at a time** - Do not batch process multiple issues
2. **Respect priority order** - Higher priority issues must be done first
3. **Respect position order** - For same priority, earlier positions go first
4. **Always update status** - Mark issues as "doing" before work, "finished" after
5. **Handle errors gracefully** - If an issue fails, report the error but continue with the next issue
6. **Never skip issues** - Process all pending issues until the project is empty
7. **Interactive continuation** - After a project is fully drained, always use `ask_user` to prompt before exiting; never terminate silently

### MCP Tools Reference

| Tool | Purpose |
|------|---------|
| `project_list` | List all projects to find target by name |
| `issue_list` | Get issues in a project (with optional status filter) |
| `issue_update` | Change issue status (pending → doing → finished) |

### Issue Status Flow

```
pending → doing → finished
```

- `pending`: Issue is waiting to be processed
- `doing`: Issue is currently being worked on
- `finished`: Issue is complete
