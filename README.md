# Issue Kanban MCP Server

[![CI](https://github.com/AlexiaChen/issue-kanban-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/AlexiaChen/issue-kanban-mcp/actions/workflows/ci.yml)
[![CD](https://github.com/AlexiaChen/issue-kanban-mcp/actions/workflows/cd.yml/badge.svg)](https://github.com/AlexiaChen/issue-kanban-mcp/actions/workflows/cd.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AlexiaChen/issue-kanban-mcp)](https://goreportcard.com/report/github.com/AlexiaChen/issue-kanban-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An AI-native issue kanban with built-in harness engineering — every completed task
makes the next one faster, safer, and smarter.

## Why This Exists

Most AI coding agents work issue-by-issue with no memory. They make the same mistakes,
miss the same edge cases, and forget the same gotchas — every single time.

This project fixes that. It combines a multi-interface kanban board (Web UI, TUI, CLI)
with an [MCP](https://modelcontextprotocol.io/) server that AI agents connect to,
plus an operational playbook that implements **harness engineering**: structured quality
gates, human-in-the-loop checkpoints, and a compound learning system (`LEARNINGS.md`)
that accumulates project knowledge across issues.

The result: your AI agent gets measurably better at working on your project over time —
without any extra effort from you.

## How It Works

```
You create issues ──► AI agent picks them up ──► Quality improves automatically

                    ┌─────────────────────────────────────┐
                    │         The Compound Loop            │
                    │                                      │
                    │  [pending]                            │
                    │     │  Load LEARNINGS.md              │
                    │     │  Check past mistakes            │
                    │     ▼                                 │
                    │  [doing]                              │
                    │     │  Research → Implement → Review  │
                    │     │  Two-pass quality gate          │
                    │     ▼                                 │
                    │  [finished]                           │
                    │     │  Capture learnings ◄── NEW!     │
                    │     │  Knowledge persists             │
                    │     ▼                                 │
                    │  ──► Next issue inherits knowledge    │
                    └─────────────────────────────────────┘
```

The kanban's `pending → doing → finished` status flow maps directly to the
compound engineering cycle: **Plan → Work → Assess → Compound**. The learning
step is what makes each cycle compound on the last.

## Quick Start

```bash
# Docker (recommended)
docker pull ghcr.io/alexiachen/issue-kanban-mcp:latest
docker run -d -p 9292:9292 -v kanban-data:/app/data ghcr.io/alexiachen/issue-kanban-mcp:latest

# Or build from source
git clone https://github.com/AlexiaChen/issue-kanban-mcp.git
cd issue-kanban-mcp
make build-all    # builds server + tui + cli → ./bin/
make run           # starts server on :9292
```

Open http://localhost:9292 — you're running.

## Four Interfaces, One Backend

| Interface | Binary | Best For |
|-----------|--------|----------|
| **Web UI** | (embedded in server) | Visual kanban management in the browser |
| **TUI** | `issue-kanban-tui` | Terminal-native kanban board with keyboard navigation |
| **CLI** | `issue-kanban-cli` | Scripting, CI/CD integration, quick operations |
| **MCP** | (built into server) | AI agents (Claude, Copilot, Gemini, etc.) |

All four interfaces hit the same REST API and SQLite database. Changes in one
are instantly visible in the others.

### Web UI

Open http://localhost:9292. Full kanban board with drag-and-drop, project management,
issue creation/editing, priority management, and 5-second auto-refresh.

### TUI

```bash
./bin/issue-kanban-tui --server http://localhost:9292
```

Three-column kanban board in your terminal. Create, edit, delete, and prioritize
issues without leaving the command line.

### CLI

```bash
./bin/issue-kanban-cli projects list
./bin/issue-kanban-cli issues list 1 --status=pending
./bin/issue-kanban-cli issues create 1 --title="Fix auth bug" --priority=high
./bin/issue-kanban-cli issues prioritize 3   # move to front
```

### MCP (for AI Agents)

The server exposes 8 MCP tools and 4 MCP resources via STDIO or HTTP/SSE transport.
AI agents connect and process issues autonomously.

---

## The Harness: How AI Quality Compounds

> This section explains what happens under the hood when an AI agent processes issues.
> You don't need to configure any of this — it's built into the agent instructions.

The project ships with an **operational playbook** (`instructions/copilot-instructions.md`)
that implements three layers of harness engineering:

### 1. Quality Gates (Every Issue)

Before marking any issue "finished," the agent runs a **two-pass self-review**:

- **Pass 1 (Critical)**: SQL injection, race conditions, XSS, unvalidated input,
  LLM output trust boundary violations
- **Pass 2 (Informational)**: Dead code, test gaps, magic numbers, completeness gaps

Mechanical issues are auto-fixed. Judgment calls go to the human.

### 2. Human-in-the-Loop Checkpoints

The agent **never** marks an issue finished without human approval:

```
Agent: "Issue #42 — review complete."
  → ✅ Mark as finished
  → 🔧 Improvements needed (describe what to change)
```

This isn't a suggestion — it's a hard gate in the workflow.

### 3. Knowledge Compounding (LEARNINGS.md)

After each issue is finished, the agent evaluates what was learned:

- Bug patterns that could recur
- Library gotchas that wasted time
- Architecture decisions with non-obvious reasoning
- What the human corrected (the agent's blind spots)

Learnings are captured in `LEARNINGS.md` with trigger keywords. Before starting
the **next** issue, the agent loads this file and matches keywords against the
new issue's title and description. Relevant learnings are applied automatically.

Over 50 issues, this prevents hours of repeated mistakes — with zero effort from you.

### Three Tiers of Knowledge

| Tier | File | Scope | How it grows |
|------|------|-------|-------------|
| Working memory | `LEARNINGS.md` | Per-project | Agent captures after each issue |
| Project conventions | `AGENTS.md` | Per-project | Promoted from LEARNINGS.md (≥3 matches) |
| Global playbook | `~/.copilot/copilot-instructions.md` | All projects | Cross-project patterns |

Each promotion is human-gated. The agent proposes; you decide.

---

## Installation

### Docker (Recommended)

```bash
docker pull ghcr.io/alexiachen/issue-kanban-mcp:latest

docker run -d \
  --name issue-kanban-mcp \
  -p 9292:9292 \
  -v kanban-data:/app/data \
  ghcr.io/alexiachen/issue-kanban-mcp:latest
```

### From Source

```bash
git clone https://github.com/AlexiaChen/issue-kanban-mcp.git
cd issue-kanban-mcp
make build-all   # server + tui + cli → ./bin/
```

The server binary is statically compiled (`CGO_ENABLED=0`) with no dynamic dependencies.

---

## Running Modes

| Mode | Command | What You Get |
|------|---------|-------------|
| HTTP | `./bin/issue-kanban-mcp -mcp=http` | Web UI + REST API + MCP SSE |
| STDIO | `./bin/issue-kanban-mcp -mcp=stdio` | MCP over stdin/stdout (for local AI clients) |
| Both | `./bin/issue-kanban-mcp -mcp=both` | All of the above |

### Readonly Mode (Default)

By default, the MCP server exposes only **3 safe tools**: `project_list`, `issue_list`,
`issue_update`. This means AI agents can read issues and update status, but cannot
create or delete anything.

To expose all 8 tools (including create/delete):

```bash
./bin/issue-kanban-mcp -readonly=false
```

---

## MCP Integration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

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

### GitHub Copilot CLI

Add to `~/.copilot/mcp-config.json`:

**STDIO (local binary):**
```json
{
  "mcpServers": {
    "issue-kanban": {
      "type": "stdio",
      "command": "/path/to/issue-kanban-mcp",
      "args": ["-mcp=stdio", "-db=/path/to/tasks.db"],
      "tools": ["*"]
    }
  }
}
```

**SSE (Docker or remote):**
```json
{
  "mcpServers": {
    "issue-kanban": {
      "type": "sse",
      "url": "http://localhost:9292/sse",
      "tools": ["*"]
    }
  }
}
```

Or use the interactive setup: `copilot mcp add`

### Deploy the Agent Playbook

For the compound engineering loop to work, deploy the operational playbook:

```bash
mkdir -p ~/.copilot
cp instructions/copilot-instructions.md ~/.copilot/copilot-instructions.md
```

Then tell your AI agent: *"Process all issues in project X"* — it handles the rest.

---

## MCP Tools

| Tool | Parameters | Mode |
|------|-----------|------|
| `project_list` | — | readonly |
| `issue_list` | `project_id`, `status?` | readonly |
| `issue_update` | `task_id`, `status` | readonly |
| `project_create` | `name`, `description?` | admin |
| `project_delete` | `project_id` | admin |
| `issue_create` | `project_id`, `title`, `description?`, `priority?` | admin |
| `issue_delete` | `task_id` | admin |
| `issue_prioritize` | `task_id` | admin |

## MCP Resources

| URI | Description |
|-----|-------------|
| `project://list` | All projects with stats |
| `project://{id}` | Project details |
| `project://{id}/issues` | All issues in project |
| `issue://{id}` | Issue details |

---

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
| PATCH | `/api/issues/{id}` | Update status |
| PUT | `/api/issues/{id}` | Edit issue (pending only) |
| DELETE | `/api/issues/{id}` | Delete issue |
| POST | `/api/issues/{id}/prioritize` | Move to front |

---

## Configuration

| Flag / Env Var | Default | Description |
|----------------|---------|-------------|
| `-port` / `PORT` | `9292` | HTTP server port |
| `-db` / `DB_PATH` | `./data/tasks.db` | SQLite database path |
| `-mcp` / `MCP_MODE` | `http` | `stdio` / `http` / `both` |
| `-readonly` / `MCP_READONLY` | `true` | `false` to expose admin MCP tools |

---

## Project Structure

```
cmd/
├── server/main.go           # MCP + HTTP server
├── tui/main.go              # Terminal UI
└── cli/main.go              # Command-line interface
internal/
├── api/handlers.go          # REST API (14 endpoints)
├── apiclient/client.go      # Shared REST client (TUI & CLI)
├── mcp/
│   ├── server.go            # MCP server setup
│   ├── tools.go             # 8 MCP tools
│   └── resources.go         # 4 MCP resources
├── queue/
│   ├── manager.go           # Business logic layer
│   ├── models.go            # Data models
│   └── mock_storage.go      # Mock storage for tests
├── storage/sqlite.go        # SQLite (pure Go, no CGO)
├── tui/                     # Bubbletea TUI
└── web/static/              # Embedded SPA
instructions/
└── copilot-instructions.md  # Agent operational playbook
AGENTS.md                    # Project knowledge base
LEARNINGS.md                 # Compound learning memory
```

---

## Development

```bash
make test             # Unit tests (mock storage, no DB required)
make test-coverage    # Coverage report (HTML)
make e2e              # End-to-end tests (starts server automatically)
make lint             # golangci-lint
make fmt              # gofmt
make dev              # Auto-reload with air
```

## Examples

```bash
make example-stdio    # STDIO client demo
make example-client   # SSE client demo (start server first with make run)
```

---

## Intellectual Roots

The harness engineering approach in this project draws from:

- **[Compound Engineering](https://every.to/p/compound-engineering)** — The idea that
  each unit of work should make the next one easier through explicit knowledge capture.
- **[gstack](https://github.com/garrytans-at-yc/gstack)** — Boil the Lake (AI makes
  completeness cheap), Search Before Building, User Sovereignty, Evidence-First review.

These principles are embedded directly in the agent workflow, not as philosophy —
as operational steps the agent executes automatically.

## License

MIT
