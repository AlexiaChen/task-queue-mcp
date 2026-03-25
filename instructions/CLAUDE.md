# Task Queue Processor Instruction

This instruction guides Claude Code to process tasks from a specified queue in the Task Queue MCP Server until all tasks are completed.

## Prerequisites

Ensure the Task Queue MCP Server is configured like below in your Claude Desktop or Claude Code settings:

```json
{
  "mcpServers": {
    "task-queue": {
      "command": "/path/to/task-queue-mcp",
      "args": ["-mcp=[stdio | http]", "-db=/path/to/tasks.db"]
    }
  }
}
```

## Instruction

When asked to process a task queue, follow this workflow:

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
  question = "队列 '{queue_name}' (id={queue_id}) 的所有任务已处理完毕。是否需要继续处理当前队列？",
  choices  = [
    "继续处理当前队列（重新检查是否有新的 Pending 任务加入）",
    "切换到其他队列",
    "不，已完成，输出最终报告"
  ]
)
```

- If user selects **"继续处理当前队列"**: loop back to Step 2 with the same `queue_id` (new tasks may have been added).
- If user selects **"切换到其他队列"**: call `queue_list`, let the user pick a new queue, then restart from Step 1 with the new queue name.
- If user selects **"不，已完成"**: proceed to Step 4.

### Step 4: Completion

When the user confirms they are done, report:
- Total tasks processed (across all queues in this session)
- Summary of work completed per queue

## Important Rules

1. **Process one task at a time** - Do not batch process multiple tasks
2. **Respect priority order** - Higher priority tasks must be done first
3. **Respect position order** - For same priority, earlier positions go first
4. **Always update status** - Mark tasks as "doing" before work, "finished" after
5. **Handle errors gracefully** - If a task fails, report the error but continue with the next task
6. **Never skip tasks** - Process all pending tasks until the queue is empty
7. **Interactive continuation** - After a queue is fully drained, always use `ask_user` to prompt before exiting; never terminate silently

## Example Usage

User prompt:
> Process all tasks in the "code-review" queue

Expected behavior:
1. Find queue named "code-review"
2. Get all tasks, pick the first pending one
3. Mark as "doing", do the code review work
4. Mark as "finished"
5. Repeat until no pending tasks remain
6. **Ask user** (via `ask_user` tool): "队列 'code-review' (id=N) 的所有任务已处理完毕。是否需要继续处理当前队列？"
7. If user says "继续当前队列" → re-check same queue; if "切换队列" → pick new queue; if no → output final report

## MCP Tools Reference

| Tool | Purpose |
|------|---------|
| `queue_list` | List all queues to find target by name |
| `task_list` | Get tasks in a queue (with optional status filter) |
| `task_update` | Change task status (pending → doing → finished) |

## Task Status Flow

```
pending → doing → finished
```

- `pending`: Task is waiting to be processed
- `doing`: Task is currently being worked on
- `finished`: Task is complete
