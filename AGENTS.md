# Issue Kanban MCP Server — Project Knowledge Base

This file provides project-specific context for AI agents working in this repository.
For agent workflow and processing instructions, see the global instructions at
`~/.copilot/copilot-instructions.md` (or `instructions/copilot-instructions.md` in this repo).

---

## Project Overview

A Go-based MCP (Model Context Protocol) server that manages multiple Issue Kanbans
with a Web UI, REST API, TUI, and CLI. Module: `github.com/AlexiaChen/issue-kanban-mcp`

---

## Architecture

```
cmd/
├── server/main.go       # MCP + HTTP server entry point
├── tui/main.go          # TUI entry point (--server flag)
└── cli/main.go          # CLI entry point (--server flag)

internal/
├── api/handlers.go      # REST API handlers (all routes)
├── apiclient/client.go  # Shared REST client (used by TUI & CLI)
├── mcp/
│   ├── server.go        # MCP server setup & routing
│   ├── tools.go         # 8 MCP tools (3 readonly + 5 admin)
│   └── resources.go     # 4 MCP resources
├── queue/
│   ├── manager.go       # Business logic layer (always use this, not storage directly)
│   ├── models.go        # Data models: Project, Issue, Priority, Status
│   └── mock_storage.go  # Mock storage for tests
├── storage/sqlite.go    # SQLite persistence (modernc.org/sqlite — pure Go, no CGO)
├── tui/
│   ├── app.go           # Bubbletea model
│   └── styles.go        # Lipgloss styles
└── web/static/          # Embedded SPA (Web UI)
```

---

## Data Model

```go
// Priority: PriorityLow=0, PriorityMedium=1, PriorityHigh=2
// Status: "pending" | "doing" | "finished"

type Project struct {
    ID          int
    Name        string
    Description string
    CreatedAt   time.Time
}

type Issue struct {
    ID          int
    ProjectID   int
    Title       string
    Description string
    Status      string    // pending | doing | finished
    Priority    Priority  // 0=low 1=medium 2=high
    Position    int       // ordering within same priority
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

---

## Build & Run

```bash
make build        # server only (CGO_ENABLED=0, static binary)
make build-tui    # TUI binary
make build-cli    # CLI binary
make build-all    # all three binaries → ./bin/

make run          # run server on :9292

make test         # unit tests (uses mock storage, no real DB)
make test-coverage
make e2e          # e2e tests (starts server automatically)
make e2e-quick    # e2e against already-running server
```

---

## REST API

### Projects
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/projects` | List all projects |
| POST | `/api/projects` | Create project |
| GET | `/api/projects/{id}` | Get project |
| DELETE | `/api/projects/{id}` | Delete project |
| GET | `/api/projects/{id}/issues` | Get project issues |
| GET | `/api/projects/{id}/stats` | Get project stats |

### Issues
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/issues` | Create issue |
| GET | `/api/issues/{id}` | Get issue |
| PATCH | `/api/issues/{id}` | Update **status only** (used by MCP) |
| PUT | `/api/issues/{id}` | Edit title/description/priority (pending only) |
| DELETE | `/api/issues/{id}` | Delete issue |
| POST | `/api/issues/{id}/prioritize` | Move to front (插队) |

---

## MCP Tools

### Readonly (default, safe for AI agents)
| Tool | Parameters |
|------|-----------|
| `project_list` | — |
| `issue_list` | `project_id`, `status?` |
| `issue_update` | `task_id`, `status` |

### Admin (require `-readonly=false`)
`project_create`, `project_delete`, `issue_create`, `issue_delete`, `issue_prioritize`

---

## Running Modes

```bash
# HTTP (Web UI + REST + MCP SSE) — readonly by default
./bin/issue-kanban-mcp -port=9292 -mcp=http

# STDIO (for MCP clients like Claude/Copilot)
./bin/issue-kanban-mcp -mcp=stdio

# Both
./bin/issue-kanban-mcp -port=9292 -mcp=both

# Admin mode
./bin/issue-kanban-mcp -readonly=false
```

---

## Code Conventions

- **Business logic**: always go through `queue.Manager`, never call `storage` directly
- **Testing**: use `queue.NewMockStorage()` — tests must not require a real DB or server
- **TDD discipline**: RED-GREEN-REFACTOR for all new code. Write failing test first, implement minimal code to pass, refactor. No production code without a failing test. See Step 4b in the playbook.
- **Systematic debugging**: For any bug fix, follow the 4-phase root cause protocol (Step 4d) before attempting fixes. No random fix-and-check cycles.
- **Verification before completion**: Run `make test` and verify output BEFORE claiming tests pass. Evidence before claims, always.
- **YAGNI**: Don't add features, options, or abstractions not required by the current issue. Simpler = better.
- **Builds**: `CGO_ENABLED=0` for static linking; never add CGO dependencies
- **CLI tests**: use `net/http/httptest.NewServer` to mock the REST API in `cmd/cli/main_test.go`
- **No sycophancy in code review**: When receiving feedback, verify against codebase reality before implementing. Push back with technical reasoning if feedback is wrong. No performative agreement.

---

## Compound Engineering Integration

This project uses an AI-driven compound engineering workflow. The key artifacts:

| File | Purpose | Who maintains |
|------|---------|--------------|
| `instructions/copilot-instructions.md` | Agent operational playbook | Human + Agent (promotion) |
| `LEARNINGS.md` | Per-issue knowledge capture (append-only) | Agent proposes, human approves |
| `AGENTS.md` | Project conventions (this file) | Promoted from LEARNINGS.md |

**How it works**: When the AI agent processes an issue via the kanban, it automatically:
1. Loads `LEARNINGS.md` during pre-flight to avoid repeating past mistakes
2. Captures new learnings after each issue is marked finished
3. Proposes promoting frequently-matched learnings to this file

Users don't need to manage this — it happens transparently during normal issue processing.

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9292` | HTTP server port |
| `DB_PATH` | `./data/tasks.db` | SQLite database path |
| `MCP_MODE` | `http` | `stdio` / `http` / `both` |
| `MCP_READONLY` | `true` | `false` to expose admin tools |

---

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/mark3labs/mcp-go` | MCP Go SDK |
| `modernc.org/sqlite` | Pure Go SQLite (no CGO) |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/spf13/cobra` | CLI framework |
