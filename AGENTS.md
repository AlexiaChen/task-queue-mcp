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
│   ├── tools.go         # 16 MCP tools (6 readonly + 10 admin)
│   └── resources.go     # 4 MCP resources
├── memory/
│   ├── models.go              # Memory data model, categories, DTOs, errors
│   ├── storage.go             # memory.Storage interface (6 methods)
│   ├── manager.go             # MemoryManager: validation, dedup, normalization
│   ├── mock_storage.go        # In-memory mock for unit tests
│   ├── triple_models.go       # Triple (KG) data model, temporal semantics
│   ├── triple_storage.go      # TripleStorage interface (6 methods)
│   ├── triple_manager.go      # TripleManager: validation, temporal logic
│   └── mock_triple_storage.go # In-memory mock for triple unit tests
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

### Memory

```go
// Category: "decision" | "fact" | "event" | "preference" | "advice" | "general"

type Memory struct {
    ID          int64
    ProjectID   int
    Content     string
    Summary     string     // optional one-line summary
    Category    string
    Tags        string     // comma-separated
    Importance  int        // 1-5 (default 3)
    ContentHash string     // SHA-256 for dedup
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Temporal Knowledge Graph (Triples)

```go
// Temporal triple with [valid_from, valid_to) closed-open interval semantics

type Triple struct {
    ID             int64
    ProjectID      int
    Subject        string     // entity (e.g., "auth-module")
    Predicate      string     // relationship (e.g., "depends_on")
    Object         string     // target (e.g., "jwt-library")
    ValidFrom      time.Time  // when this fact became true
    ValidTo        *time.Time // nil = currently active; set = invalidated/replaced
    Confidence     float64    // 0.0-1.0 (default 1.0)
    SourceMemoryID *int64     // optional link to originating memory
    CreatedAt      time.Time
}

// StoreTripleInput.ReplaceExisting controls auto-invalidation:
//   true  → single-valued predicates (status, assigned_to): old triple gets valid_to = new.valid_from
//   false → multi-valued predicates (has_label, depends_on): both triples coexist
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

### Memories
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/projects/{id}/memories` | Store memory (dedup by content hash) |
| GET | `/api/projects/{id}/memories` | List memories (`category?`, `limit?`, `offset?`) |
| GET | `/api/projects/{id}/memories/search` | Search memories (`q`, `category?`, `limit?`) |
| DELETE | `/api/projects/{id}/memories/{mid}` | Delete memory |

### Triples (Temporal Knowledge Graph)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/projects/{id}/triples` | Store triple (auto-invalidate with `replace_existing`) |
| GET | `/api/projects/{id}/triples/{tid}` | Get triple by ID |
| GET | `/api/projects/{id}/triples` | Query triples (`subject?`, `predicate?`, `active_only?`, `at_time?`, `limit?`, `offset?`) |
| PATCH | `/api/projects/{id}/triples/{tid}` | Invalidate triple (set `valid_to`) |
| DELETE | `/api/projects/{id}/triples/{tid}` | Delete triple |

---

## MCP Tools

### Readonly (default, safe for AI agents)
| Tool | Parameters |
|------|-----------|
| `project_list` | — |
| `issue_list` | `project_id`, `status?` |
| `issue_update` | `task_id`, `status` |
| `memory_search` | `project_id`, `query`, `category?`, `limit?` |
| `memory_list` | `project_id`, `category?`, `limit?`, `offset?` |
| `triple_query` | `project_id`, `subject?`, `predicate?`, `active_only?`, `at_time?`, `limit?`, `offset?` |

### Admin (require `-readonly=false`)
`project_create`, `project_delete`, `issue_create`, `issue_delete`, `issue_prioritize`, `memory_store`, `memory_delete`, `triple_store`, `triple_invalidate`, `triple_delete`

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

- **Business logic**: always go through `queue.Manager` / `memory.MemoryManager` / `memory.TripleManager`, never call `storage` directly
- **Testing**: use `queue.NewMockStorage()` / `memory.NewMockMemoryStorage()` / `memory.NewMockTripleStorage()` — tests must not require a real DB or server
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
