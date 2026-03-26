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
	), s.handleProjectList)

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
		), s.handleProjectCreate)

		s.mcp.AddTool(mcplib.NewTool("project_delete",
			mcplib.WithDescription("Delete a project and all its issues"),
			mcplib.WithNumber("project_id",
				mcplib.Required(),
				mcplib.Description("ID of the project to delete"),
			),
		), s.handleProjectDelete)
	}

	// Always register issue list (read)
	s.mcp.AddTool(mcplib.NewTool("issue_list",
		mcplib.WithDescription("List issues in a project"),
		mcplib.WithNumber("project_id",
			mcplib.Required(),
			mcplib.Description("ID of the project"),
		),
		mcplib.WithString("status",
			mcplib.Description("Filter by status: pending, doing, or finished"),
			mcplib.Enum("pending", "doing", "finished"),
		),
	), s.handleIssueList)

	// Admin-only tools (not exposed in readonly mode)
	if !s.readonly {
		s.mcp.AddTool(mcplib.NewTool("issue_create",
			mcplib.WithDescription("Create a new issue in a project"),
			mcplib.WithNumber("project_id",
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
			mcplib.WithString("priority",
				mcplib.Description("Priority level: low, medium, or high (default: low)"),
				mcplib.Enum("low", "medium", "high"),
			),
		), s.handleIssueCreate)

		s.mcp.AddTool(mcplib.NewTool("issue_delete",
			mcplib.WithDescription("Delete an issue"),
			mcplib.WithNumber("task_id",
				mcplib.Required(),
				mcplib.Description("ID of the issue to delete"),
			),
		), s.handleIssueDelete)

		s.mcp.AddTool(mcplib.NewTool("issue_prioritize",
			mcplib.WithDescription("Move a pending issue ahead of lower-priority pending issues in the project (插队)"),
			mcplib.WithNumber("task_id",
				mcplib.Required(),
				mcplib.Description("ID of the issue to prioritize"),
			),
		), s.handleIssuePrioritize)
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
	), s.handleIssueUpdate)

	return nil
}

// Queue handlers

func (s *Server) handleProjectList(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	queues, err := s.manager.ListProjects(ctx)
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
		stats, err := s.manager.GetProjectStats(ctx, q.ID)
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

func (s *Server) handleProjectCreate(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	description := req.GetString("description", "")

	q, err := s.manager.CreateProject(ctx, queue.CreateQueueInput{
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

func (s *Server) handleProjectDelete(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	if err := s.manager.DeleteProject(ctx, int64(projectID)); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to delete queue: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Project %d deleted successfully", projectID)), nil
}

// Task handlers

func (s *Server) handleIssueList(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	var status *queue.TaskStatus
	if statusStr := req.GetString("status", ""); statusStr != "" {
		s := queue.TaskStatus(statusStr)
		status = &s
	}

	tasks, err := s.manager.ListIssues(ctx, int64(projectID), status)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to list tasks: %v", err)), nil
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleIssueCreate(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	title, err := req.RequireString("title")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	description := req.GetString("description", "")
	priorityStr := req.GetString("priority", "low")
	priority, err := queue.ParsePriority(priorityStr)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Invalid priority: %v", err)), nil
	}

	task, err := s.manager.CreateIssue(ctx, queue.CreateTaskInput{
		ProjectID:   int64(projectID),
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

func (s *Server) handleIssueUpdate(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	statusStr, err := req.RequireString("status")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	status := queue.TaskStatus(statusStr)
	task, err := s.manager.UpdateIssue(ctx, int64(taskID), queue.UpdateTaskInput{
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

func (s *Server) handleIssueDelete(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	if err := s.manager.DeleteIssue(ctx, int64(taskID)); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to delete task: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Issue %d deleted successfully", taskID)), nil
}

func (s *Server) handleIssuePrioritize(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	taskID, err := req.RequireInt("task_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	task, err := s.manager.PrioritizeIssue(ctx, int64(taskID))
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to prioritize task: %v", err)), nil
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Issue prioritized successfully:\n%s", string(data))), nil
}
