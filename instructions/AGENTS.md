# Issue Kanban Processor Agent Instruction

This instruction configures an AI agent to autonomously process issues from a specified project in the Issue Kanban MCP Server until all issues are completed.

## MCP Server Configuration

### STDIO Mode (Local)

The server defaults to readonly mode (safe for AI agents):

```json
{
  "servers": {
    "issue-kanban": {
      "type": "stdio",
      "command": "/path/to/issue-kanban-mcp",
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
    "issue-kanban": {
      "type": "sse",
      "url": "http://localhost:9292/sse",
      "tools": ["*"]
    }
  }
}
```

> **Note**: Readonly mode is enabled by default. To disable for admin access, use `-readonly=false` flag or set `MCP_READONLY=false` environment variable.

## Agent Behavior

When instructed to process an issue kanban, the agent MUST:

### 1. Initialization Phase

- Call `project_list` to retrieve all available projects
- Match the specified queue name (case-sensitive) to get the `queue_id`
- If no match found, report error and terminate

### 2. Issue Processing Loop

The agent must execute this loop continuously:

```
WHILE true:
    tasks = issue_list(queue_id=queue_id)

    pending_issues = filter(issues, status="pending")
    pending_issues = sort(pending_issues, by=[priority DESC, position ASC])

    IF pending_issues is empty:
        BREAK  # All issues completed

    current_issue = pending_issues[0]  # First issue in order

    # Start issue
    issue_update(task_id=current_task.id, status="doing")

    # Execute the actual work
    execute(current_issue.title, current_issue.description)

    # Complete issue
    issue_update(task_id=current_task.id, status="finished")
```

### 3. Issue Execution

For each issue:
- **Understand**: Read and comprehend `title` and `description`
- **Plan**: Determine the steps needed to complete the issue
- **Execute**: Perform the actual work (code, analysis, documentation, etc.)
- **Verify**: Ensure the work meets the issue requirements
- **Complete**: Mark as finished only when truly done

### 4. Interactive Continuation

After the issue processing loop exits (no more pending issues), the agent MUST interactively ask the user whether to continue before generating the completion report:

```
PRESENT interactive selection to user:
    question = "Project '{queue_name}' (id={queue_id}) is fully processed. Continue with the current queue?"
    choices  = [
        "Continue current project (re-check for newly added pending issues)",
        "Switch to another project (call project_list to select a new project)",
        "No, done — print final report"
    ]

IF user selects "Continue current project":
    GOTO Issue Processing Loop (re-check for new pending issues in same queue_id)

IF user selects "Switch to another project":
    project_list()  # Show all projects for user to pick next project
    GOTO Initialization Phase with new project_name

IF user selects "No, done":
    CONTINUE to Completion Report
```

**Key requirements**:
- This prompt MUST appear after every project is fully drained, before the final report
- The agent must NOT silently exit — always pause and wait for user input
- "Continue current project" is preferred over "Switch project" because new issues may have been added while the agent was processing

### 5. Completion Report

After the user confirms they are done (or selects to stop), provide:
- Number of issues processed
- List of completed issues with brief summaries
- Any errors or warnings encountered

## Ordering Rules

Issues are processed in this order:
1. **Higher priority first** - `priority` field (descending)
2. **Earlier position first** - `position` field (ascending) - for tie-breaking

Example: Issue A (priority=10, position=2) runs before Issue B (priority=5, position=1)

## Error Handling

- If an issue cannot be completed:
  1. Document the error clearly
  2. Keep the issue in "doing" status (do not mark as finished)
  3. Continue to the next pending issue
  4. Report all failed issues at the end

- If the MCP server is unavailable:
  1. Report connection error
  2. Stop processing immediately
  3. Do not attempt to retry automatically

## Available MCP Tools (Readonly Mode)

The following tools are available to AI agents in readonly mode:

| Tool | Description | Parameters |
|------|-------------|------------|
| `project_list` | List all projects | (none) |
| `issue_list` | List tasks in queue | `queue_id`, `status?` |
| `issue_update` | Update task status | `task_id`, `status` |

> **Admin Tools**: `project_create`, `project_delete`, `issue_create`, `issue_delete`, `issue_prioritize` are only available when readonly mode is disabled (`-readonly=false`). Use the Web UI or REST API for project/issue management.

## Issue Status Values

| Status | Meaning |
|--------|---------|
| `pending` | Waiting to be processed |
| `doing` | Currently being worked on |
| `finished` | Successfully completed |

## Example Invocation

**User:**
> Process all issues in project "daily-tasks"

**Agent Response:**
```
Found project "daily-tasks" (id: 3)

Processing issue #15: "Review PR #42" (priority: 10)
  → Marked as doing
  → Reviewed pull request, left comments
  → Marked as finished

Processing issue #16: "Update documentation" (priority: 5)
  → Marked as doing
  → Updated README and API docs
  → Marked as finished

Processing issue #17: "Run tests" (priority: 5)
  → Marked as doing
  → All tests passed
  → Marked as finished

No more pending issues.

[Interactive prompt]
Project "daily-tasks" (id: 3) is fully processed. Continue with the current project?
> 1. Continue current project (re-check for newly added pending issues)
> 2. Switch to another project (call project_list to select a new project)
> 3. No, done — print final report

[User selects: No, done]

Summary:
- Processed: 3 issues
- Completed: 3 issues
- Failed: 0 issues
```

## Constraints

1. **Single-threaded**: Process one issue at a time, never in parallel
2. **No skipping**: Every pending issue must be attempted
3. **Honest status**: Only mark "finished" when truly complete
4. **Persistent**: Continue until all issues are done or unrecoverable error
5. **Interactive continuation**: Always pause and ask the user before exiting; never auto-terminate silently
