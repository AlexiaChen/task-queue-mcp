package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"task-queue-mcp/internal/queue"
)

// registerTools registers all MCP tools
func (s *Server) registerTools() error {
	// Always register read tools
	s.mcp.AddTool(mcplib.NewTool("project_list",
		mcplib.WithDescription("List all projects with their statistics"),
	), s.handleQueueList)

	// Admin-only tools (not exposed in readonly mode)
	if !s.readonly {
		s.mcp.AddTool(mcplib.NewTool("project_create",
			mcplib.WithDescription("Create a new project"),
			mcplib.WithString("name",
				mcplib.Required(),
				mcplib.Description("Unique name for the project"),
			),
			mcplib.WithString("description",
				mcplib.Description("Optional description for the project"),
			),
		), s.handleQueueCreate)

		s.mcp.AddTool(mcplib.NewTool("project_delete",
			mcplib.WithDescription("Delete a project and all its issues"),
			mcplib.WithNumber("queue_id",
				mcplib.Required(),
				mcplib.Description("ID of the project to delete"),
			),
		), s.handleQueueDelete)
	}

	// Always register issue list (read)
	s.mcp.AddTool(mcplib.NewTool("issue_list",
		mcplib.WithDescription("List issues in a project"),
		mcplib.WithNumber("queue_id",
			mcplib.Required(),
			mcplib.Description("ID of the project"),
		),
		mcplib.WithString("status",
			mcplib.Description("Filter by status: pending, doing, or finished"),
			mcplib.Enum("pending", "doing", "finished"),
		),
	), s.handleTaskList)

	// Admin-only tools (not exposed in readonly mode)
	if !s.readonly {
		s.mcp.AddTool(mcplib.NewTool("issue_create",
			mcplib.WithDescription("Create a new issue in a project"),
			mcplib.WithNumber("queue_id",
				mcplib.Required(),
				mcplib.Description("ID of the project to add issue to"),
			),
			mcplib.WithString("title",
				mcplib.Required(),
				mcplib.Description("Title of the issue"),
			),
			mcplib.WithString("description",
				mcplib.Description("Optional description of the issue"),
			),
			mcplib.WithNumber("priority",
				mcplib.Description("Priority level (higher = more urgent)"),
				mcplib.DefaultNumber(0),
			),
		), s.handleTaskCreate)

		s.mcp.AddTool(mcplib.NewTool("issue_delete",
			mcplib.WithDescription("Delete an issue"),
			mcplib.WithNumber("task_id",
				mcplib.Required(),
				mcplib.Description("ID of the issue to delete"),
			),
		), s.handleTaskDelete)

		s.mcp.AddTool(mcplib.NewTool("issue_prioritize",
			mcplib.WithDescription("Move an issue to a higher priority position in the project (插队)"),
			mcplib.WithNumber("task_id",
				mcplib.Required(),
				mcplib.Description("ID of the issue to prioritize"),
			),
			mcplib.WithNumber("position",
				mcplib.Description("Target position (1 = front of project). If not specified, moves to front."),
				mcplib.DefaultNumber(1),
			),
		), s.handleTaskPrioritize)
	}

	// Always allow status update (AI can process issues)
	s.mcp.AddTool(mcplib.NewTool("issue_update",
		mcplib.WithDescription("Update an issue's status"),
		mcplib.WithNumber("task_id",
			mcplib.Required(),
			mcplib.Description("ID of the issue to update"),
		),
		mcplib.WithString("status",
			mcplib.Required(),
			mcplib.Description("New status for the task"),
			mcplib.Enum("pending", "doing", "finished"),
		),
	), s.handleTaskUpdate)

	return nil
}

// Queue handlers

func (s *Server) handleQueueList(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	queues, err := s.manager.ListQueues(ctx)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to list queues: %v", err)), nil
	}

	// Add stats to each queue
	type QueueWithStats struct {
		*queue.Queue
		Stats *queue.QueueStats `json:"stats"`
	}

	var result []QueueWithStats
	for _, q := range queues {
		stats, err := s.manager.GetQueueStats(ctx, q.ID)
		if err != nil {
			stats = &queue.QueueStats{}
		}
		result = append(result, QueueWithStats{
			Queue: q,
			Stats: stats,
		})
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleQueueCreate(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	description := req.GetString("description", "")

	q, err := s.manager.CreateQueue(ctx, queue.CreateQueueInput{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to create queue: %v", err)), nil
	}

	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Project created successfully:\n%s", string(data))), nil
}

func (s *Server) handleQueueDelete(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	queueID, err := req.RequireInt("queue_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	if err := s.manager.DeleteQueue(ctx, int64(queueID)); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to delete queue: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Project %d deleted successfully", queueID)), nil
}

// Task handlers

func (s *Server) handleTaskList(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	queueID, err := req.RequireInt("queue_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	var status *queue.TaskStatus
	if statusStr := req.GetString("status", ""); statusStr != "" {
		s := queue.TaskStatus(statusStr)
		status = &s
	}

	tasks, err := s.manager.ListTasks(ctx, int64(queueID), status)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to list tasks: %v", err)), nil
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleTaskCreate(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	queueID, err := req.RequireInt("queue_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	title, err := req.RequireString("title")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	description := req.GetString("description", "")
	priority := req.GetInt("priority", 0)

	task, err := s.manager.CreateTask(ctx, queue.CreateTaskInput{
		QueueID:     int64(queueID),
		Title:       title,
		Description: description,
		Priority:    priority,
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to create task: %v", err)), nil
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Issue created successfully:\n%s", string(data))), nil
}

func (s *Server) handleTaskUpdate(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	statusStr, err := req.RequireString("status")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	status := queue.TaskStatus(statusStr)
	task, err := s.manager.UpdateTask(ctx, int64(taskID), queue.UpdateTaskInput{
		Status: &status,
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to update task: %v", err)), nil
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Issue updated successfully:\n%s", string(data))), nil
}

func (s *Server) handleTaskDelete(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	if err := s.manager.DeleteTask(ctx, int64(taskID)); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to delete task: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Issue %d deleted successfully", taskID)), nil
}

func (s *Server) handleTaskPrioritize(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	position := req.GetInt("position", 1)

	task, err := s.manager.PrioritizeTask(ctx, int64(taskID), position)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to prioritize task: %v", err)), nil
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Issue prioritized successfully (moved to position %d):\n%s", position, string(data))), nil
}
