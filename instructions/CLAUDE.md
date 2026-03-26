# Issue Kanban Processor Instruction

This instruction guides Claude Code to process issues from a specified project in the Issue Kanban MCP Server until all issues are completed.

## Prerequisites

Ensure the Issue Kanban MCP Server is configured like below in your Claude Desktop or Claude Code settings:

```json
{
  "mcpServers": {
    "issue-kanban": {
      "command": "/path/to/issue-kanban-mcp",
      "args": ["-mcp=[stdio | http]", "-db=/path/to/tasks.db"]
    }
  }
}
```

## Instruction

When asked to process an issue kanban, follow this workflow:

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
  question = "Project '{queue_name}' (id={queue_id}) is fully processed. Continue with the current queue?",
  choices  = [
    "Continue current project (re-check for newly added pending issues)",
    "Switch to another project",
    "No, done — print final report"
  ]
)
```

- If user selects **"Continue current queue"**: loop back to Step 2 with the same `queue_id` (new tasks may have been added).
- If user selects **"Switch to another project"**: call `project_list`, let the user pick a new queue, then restart from Step 1 with the new queue name.
- If user selects **"No, done"**: proceed to Step 4.

### Step 4: Completion

When the user confirms they are done, report:
- Total issues processed (across all projects in this session)
- Summary of work completed per project

## Important Rules

1. **Process one issue at a time** - Do not batch process multiple issues
2. **Respect priority order** - Higher priority issues must be done first
3. **Respect position order** - For same priority, earlier positions go first
4. **Always update status** - Mark issues as "doing" before work, "finished" after
5. **Handle errors gracefully** - If an issue fails, report the error but continue with the next issue
6. **Never skip issues** - Process all pending issues until the project is empty
7. **Interactive continuation** - After a project is fully drained, always use `ask_user` to prompt before exiting; never terminate silently

## Example Usage

User prompt:
> Process all issues in the "code-review" project

Expected behavior:
1. Find project named "code-review"
2. Get all issues, pick the first pending one
3. Mark as "doing", do the code review work
4. Mark as "finished"
5. Repeat until no pending issues remain
    6. **Ask user** (via `ask_user` tool): "Project 'code-review' (id=N) is fully processed. Continue with the current project?"
    7. If user says "Continue current queue" → re-check same queue; if "Switch to another project" → pick new queue; if no → output final report

## MCP Tools Reference

| Tool | Purpose |
|------|---------|
| `project_list` | List all projects to find target by name |
| `issue_list` | Get issues in a project (with optional status filter) |
| `issue_update` | Change issue status (pending → doing → finished) |

## Issue Status Flow

```
pending → doing → finished
```

- `pending`: Issue is waiting to be processed
- `doing`: Issue is currently being worked on
- `finished`: Issue is complete
