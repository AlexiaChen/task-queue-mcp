# Task Queue MCP Server Design Document

## Overview

A Go-based MCP (Model Context Protocol) Server that manages multiple Task Queues, providing both MCP interfaces for LLM integration and a Web UI for human management.

## Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21+ |
| MCP Library | github.com/mark3labs/mcp-go |
| Database | SQLite (modernc.org/sqlite or mattn/go-sqlite3) |
| Web Framework | net/http (standard library) |
| Frontend | Embedded static files (HTML/CSS/JS) |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Task Queue MCP Server                   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ STDIO       │  │ SSE/HTTP    │  │ Web UI (HTTP)       │  │
│  │ Transport   │  │ Transport   │  │ (Embedded Static)   │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                     │              │
│         └────────────────┼─────────────────────┘              │
│                          ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                   MCP Server Core                        │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │ │
│  │  │   Tools     │  │  Resources  │  │    Prompts      │  │ │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘  │ │
│  └──────────────────────────┬──────────────────────────────┘ │
│                             ▼                                 │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                  Queue Manager                           │ │
│  │  - Queue CRUD operations                                │ │
│  │  - Task CRUD operations                                 │ │
│  │  - Task state transitions                               │ │
│  │  - Priority management                                  │ │
│  └──────────────────────────┬──────────────────────────────┘ │
│                             ▼                                 │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                  SQLite Database                         │ │
│  │  - queues table                                         │ │
│  │  - tasks table                                          │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Data Model

### Queue

| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER | Primary key, auto-increment |
| name | TEXT | Unique queue name |
| description | TEXT | Optional description |
| created_at | DATETIME | Creation timestamp |
| updated_at | DATETIME | Last update timestamp |

### Task

| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER | Primary key, auto-increment |
| queue_id | INTEGER | Foreign key to queue |
| title | TEXT | Task title |
| description | TEXT | Optional task description |
| status | TEXT | pending, doing, finished |
| priority | INTEGER | Priority level (higher = more urgent) |
| position | INTEGER | Position within queue for ordering |
| created_at | DATETIME | Creation timestamp |
| updated_at | DATETIME | Last update timestamp |
| started_at | DATETIME | When task moved to doing |
| finished_at | DATETIME | When task moved to finished |

## MCP Interface Design

### Tools (Actions)

| Tool Name | Description | Parameters |
|-----------|-------------|------------|
| `queue_list` | List all queues | None |
| `queue_create` | Create a new queue | name, description? |
| `queue_delete` | Delete a queue | queue_id |
| `task_list` | List tasks in a queue | queue_id, status? |
| `task_create` | Create a new task | queue_id, title, description?, priority? |
| `task_update` | Update task status | task_id, status |
| `task_delete` | Delete a task | task_id |
| `task_prioritize` | Move task to front of queue | task_id, position? |

### Resources (Data Access)

| URI Pattern | Description | MIME Type |
|-------------|-------------|-----------|
| `queue://list` | List all queues | application/json |
| `queue://{queue_id}` | Get queue details | application/json |
| `queue://{queue_id}/tasks` | Get all tasks in queue | application/json |
| `task://{task_id}` | Get task details | application/json |

### Example Tool Implementations

#### queue_create Tool

```go
s.AddTool(
    mcp.NewTool("queue_create",
        mcp.WithDescription("Create a new task queue"),
        mcp.WithString("name",
            mcp.Required(),
            mcp.Description("Unique name for the queue"),
        ),
        mcp.WithString("description",
            mcp.Description("Optional description for the queue"),
        ),
    ),
    handleQueueCreate,
)
```

#### task_create Tool

```go
s.AddTool(
    mcp.NewTool("task_create",
        mcp.WithDescription("Create a new task in a queue"),
        mcp.WithNumber("queue_id",
            mcp.Required(),
            mcp.Description("ID of the queue to add task to"),
        ),
        mcp.WithString("title",
            mcp.Required(),
            mcp.Description("Title of the task"),
        ),
        mcp.WithString("description",
            mcp.Description("Optional description of the task"),
        ),
        mcp.WithNumber("priority",
            mcp.Description("Priority level (higher = more urgent)"),
            mcp.DefaultNumber(0),
        ),
    ),
    handleTaskCreate,
)
```

#### task_prioritize Tool (插队功能)

```go
s.AddTool(
    mcp.NewTool("task_prioritize",
        mcp.WithDescription("Move a task to a higher priority position in the queue"),
        mcp.WithNumber("task_id",
            mcp.Required(),
            mcp.Description("ID of the task to prioritize"),
        ),
        mcp.WithNumber("position",
            mcp.Description("Target position (1 = front of queue). If not specified, moves to front."),
        ),
    ),
    handleTaskPrioritize,
)
```

### Resource Template Example

```go
// Dynamic resource for queue tasks
s.AddResource(
    mcp.NewResource(
        "queue://{queue_id}/tasks",
        "Queue Tasks",
        mcp.WithResourceDescription("Get all tasks in a specific queue"),
        mcp.WithMIMEType("application/json"),
    ),
    handleQueueTasksResource,
)
```

## Web UI Design

### Pages

1. **Dashboard** (`/`) - Overview of all queues with task counts
2. **Queue Detail** (`/queue/{id}`) - View and manage tasks in a queue

### Features

- View all queues with pending/doing/finished counts
- Create/delete queues
- View tasks in a queue with status indicators
- Create/delete tasks
- Change task status (pending → doing → finished)
- Drag-and-drop or button to prioritize tasks (move to front)
- Real-time updates via SSE (optional enhancement)

### API Endpoints (REST)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/queues | List all queues |
| POST | /api/queues | Create queue |
| GET | /api/queues/{id} | Get queue details |
| DELETE | /api/queues/{id} | Delete queue |
| GET | /api/queues/{id}/tasks | Get queue tasks |
| POST | /api/tasks | Create task |
| PATCH | /api/tasks/{id} | Update task |
| DELETE | /api/tasks/{id} | Delete task |
| POST | /api/tasks/{id}/prioritize | Prioritize task |

## Project Structure

```
task-queue-mcp/
├── cmd/
│   └── server/
│       └── main.go           # Entry point
├── internal/
│   ├── api/
│   │   └── handlers.go       # REST API handlers
│   ├── mcp/
│   │   ├── server.go         # MCP server setup
│   │   ├── tools.go          # MCP tool handlers
│   │   └── resources.go      # MCP resource handlers
│   ├── queue/
│   │   ├── manager.go        # Queue business logic
│   │   └── models.go         # Data models
│   ├── storage/
│   │   ├── sqlite.go         # SQLite implementation
│   │   └── migrations.go     # Database migrations
│   └── web/
│       ├── embed.go          # Static file embedding
│       └── static/           # Frontend files
│           ├── index.html
│           ├── style.css
│           └── app.js
├── doc/
│   └── task-queue-mcp-design.md
├── go.mod
├── go.sum
└── README.md
```

## Implementation Phases

### Phase 1: Core Infrastructure
1. Initialize Go module
2. Set up SQLite database with migrations
3. Implement data models
4. Create storage layer

### Phase 2: Queue Manager
1. Implement Queue CRUD operations
2. Implement Task CRUD operations
3. Implement priority/position management
4. Add status transition logic

### Phase 3: MCP Server
1. Set up MCP server with mcp-go
2. Implement all MCP tools
3. Implement all MCP resources
4. Configure STDIO + SSE transports

### Phase 4: Web UI
1. Create HTML/CSS/JS frontend
2. Implement REST API endpoints
3. Embed static files
4. Integrate with queue manager

### Phase 5: Integration & Testing
1. End-to-end testing
2. MCP client testing
3. Web UI testing
4. Documentation

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_PATH` | `./data/tasks.db` | SQLite database path |
| `MCP_TRANSPORT` | `stdio` | MCP transport mode (stdio/sse/both) |

## Dependencies

```go
require (
    github.com/mark3labs/mcp-go v0.x.x
    modernc.org/sqlite v1.x.x  // Pure Go SQLite
)
```

## Verification Plan

1. **MCP Tools Testing**: Use MCP client to call each tool and verify responses
2. **MCP Resources Testing**: Read each resource and verify data format
3. **Web UI Testing**: Manual testing of all UI operations
4. **REST API Testing**: Use curl or HTTP client to test all endpoints
5. **Integration Testing**: Verify MCP operations reflect in Web UI and vice versa

## Success Criteria

- [ ] MCP server starts and responds to initialization
- [ ] All 8 MCP tools work correctly
- [ ] All 4 MCP resources return correct data
- [ ] Web UI displays queues and tasks
- [ ] Tasks can be created, updated, deleted via both MCP and Web UI
- [ ] Task prioritization (插队) works correctly
- [ ] Data persists across server restarts
