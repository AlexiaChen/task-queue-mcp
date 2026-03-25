# Task Queue Processor Agent Instruction

This instruction configures an AI agent to autonomously process tasks from a specified queue in the Task Queue MCP Server until all tasks are completed.

## MCP Server Configuration

### STDIO Mode (Local)

The server defaults to readonly mode (safe for AI agents):

```json
{
  "servers": {
    "task-queue": {
      "type": "stdio",
      "command": "/path/to/task-queue-mcp",
      "args": ["-mcp=stdio", "-db=/path/to/tasks.db"],
      "tools": ["*"]
    }
  }
}
```

### SSE Mode (Remote/Docker)

```json
{
  "servers": {
    "task-queue": {
      "type": "sse",
      "url": "http://localhost:9292/sse",
      "tools": ["*"]
    }
  }
}
```

> **Note**: Readonly mode is enabled by default. To disable for admin access, use `-readonly=false` flag or set `MCP_READONLY=false` environment variable.

## Available Interfaces

This project ships three binaries:

| Binary | Purpose |
|--------|---------|
| `task-queue-mcp` | MCP server + REST API + Web UI |
| `task-queue-tui` | Terminal UI (bubbletea) — `--server http://...` |
| `task-queue-cli` | CLI (cobra) — `--server http://...` |

Build all: `make build-all`

## Agent Behavior

When instructed to process a task queue, the agent MUST:

### 1. Initialization Phase

- Call `queue_list` to retrieve all available queues
- Match the specified queue name (case-sensitive) to get the `queue_id`
- If no match found, report error and terminate

### 2. Task Processing Loop

The agent must execute this loop continuously:

```
WHILE true:
    tasks = task_list(queue_id=queue_id)

    pending_tasks = filter(tasks, status="pending")
    pending_tasks = sort(pending_tasks, by=[priority DESC, position ASC])

    IF pending_tasks is empty:
        BREAK  # All tasks completed

    current_task = pending_tasks[0]  # First task in order

    # Start task
    task_update(task_id=current_task.id, status="doing")

    # Execute the actual work
    execute(current_task.title, current_task.description)

    # Complete task
    task_update(task_id=current_task.id, status="finished")
```

### 3. Task Execution

For each task:
- **Understand**: Read and comprehend `title` and `description`
- **Plan**: Determine the steps needed to complete the task
- **Execute**: Perform the actual work (code, analysis, documentation, etc.)
- **Verify**: Ensure the work meets the task requirements
- **Complete**: Mark as finished only when truly done

### 4. Interactive Continuation

After the task processing loop exits (no more pending tasks), the agent MUST interactively ask the user whether to continue before generating the completion report:

```
PRESENT interactive selection to user:
    question = "队列 '{queue_name}' 的所有任务已处理完毕。是否需要继续处理其他任务？"
    choices  = [
        "继续处理其他队列（再次调用 queue_list 选择队列）",
        "不，已完成，输出最终报告"
    ]

IF user selects "继续处理其他队列":
    queue_list()  # Show all queues for user to pick next queue
    GOTO Initialization Phase with new queue_name

IF user selects "不，已完成":
    CONTINUE to Completion Report
```

**For Claude Code / Copilot CLI agents**, use the `ask_user` tool:

```
ask_user(
  question = "队列 '{queue_name}' 的所有任务已处理完毕。是否需要继续处理其他队列？",
  choices  = [
    "继续处理其他队列",
    "不，已完成，输出最终报告"
  ]
)
```

**Key requirements**:
- This prompt MUST appear after every queue is fully drained, before the final report
- The agent must NOT silently exit — always pause and wait for user input
- If the user selects to continue, restart the full Initialization → Loop → Interactive Continuation cycle for the new queue

### 5. Completion Report

After the user confirms they are done (or selects to stop), provide:
- Number of tasks processed (across all queues in this session)
- List of completed tasks with brief summaries per queue
- Any errors or warnings encountered

## Ordering Rules

Tasks are processed in this order:
1. **Higher priority first** - `priority` field (descending)
2. **Earlier position first** - `position` field (ascending) - for tie-breaking

Example: Task A (priority=10, position=2) runs before Task B (priority=5, position=1)

## Error Handling

- If a task cannot be completed:
  1. Document the error clearly
  2. Keep the task in "doing" status (do not mark as finished)
  3. Continue to the next pending task
  4. Report all failed tasks at the end

- If the MCP server is unavailable:
  1. Report connection error
  2. Stop processing immediately
  3. Do not attempt to retry automatically

## Available MCP Tools (Readonly Mode)

The following tools are available to AI agents in readonly mode:

| Tool | Description | Parameters |
|------|-------------|------------|
| `queue_list` | List all queues | (none) |
| `task_list` | List tasks in queue | `queue_id`, `status?` |
| `task_update` | Update task status | `task_id`, `status` |

> **Admin Tools**: `queue_create`, `queue_delete`, `task_create`, `task_delete`, `task_prioritize` are only available when readonly mode is disabled (`-readonly=false`). Use the Web UI or REST API for queue/task management.

## Task Status Values

| Status | Meaning |
|--------|---------|
| `pending` | Waiting to be processed |
| `doing` | Currently being worked on |
| `finished` | Successfully completed |

## Example Invocation

**User:**
> Process all tasks in queue "daily-tasks"

**Agent Response:**
```
Found queue "daily-tasks" (id: 3)

Processing task #15: "Review PR #42" (priority: 10)
  → Marked as doing
  → Reviewed pull request, left comments
  → Marked as finished

Processing task #16: "Update documentation" (priority: 5)
  → Marked as doing
  → Updated README and API docs
  → Marked as finished

Processing task #17: "Run tests" (priority: 5)
  → Marked as doing
  → All tests passed
  → Marked as finished

No more pending tasks.

[Interactive prompt]
队列 "daily-tasks" 的所有任务已处理完毕。是否需要继续处理其他任务？
> 1. 继续处理其他队列（再次调用 queue_list 选择队列）
> 2. 不，已完成，输出最终报告

[User selects: 不，已完成]

Summary:
- Processed: 3 tasks
- Completed: 3 tasks
- Failed: 0 tasks
```

## Constraints

1. **Single-threaded**: Process one task at a time, never in parallel
2. **No skipping**: Every pending task must be attempted
3. **Honest status**: Only mark "finished" when truly complete
4. **Persistent**: Continue until all tasks are done or unrecoverable error
5. **Interactive continuation**: Always pause and ask the user before exiting; never auto-terminate silently
