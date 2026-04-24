package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/memory"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"
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

	// Memory tools (only if memory manager is configured)
	if s.memoryManager != nil {
		// Readonly memory tools
		s.mcp.AddTool(mcplib.NewTool("memory_search",
			mcplib.WithDescription("Search memories using full-text search within a project"),
			mcplib.WithNumber("project_id",
				mcplib.Required(),
				mcplib.Description("ID of the project to search in. Use 0 for global (cross-project) memories."),
			),
			mcplib.WithString("query",
				mcplib.Required(),
				mcplib.Description("Search query string"),
			),
			mcplib.WithString("category",
				mcplib.Description("Filter by category: decision, fact, event, preference, advice, general"),
				mcplib.Enum("decision", "fact", "event", "preference", "advice", "general"),
			),
			mcplib.WithNumber("limit",
				mcplib.Description("Maximum number of results (default: 20)"),
			),
		), s.handleMemorySearch)

		s.mcp.AddTool(mcplib.NewTool("memory_list",
			mcplib.WithDescription("List memories in a project, optionally filtered by category"),
			mcplib.WithNumber("project_id",
				mcplib.Required(),
				mcplib.Description("ID of the project. Use 0 for global (cross-project) memories."),
			),
			mcplib.WithString("category",
				mcplib.Description("Filter by category: decision, fact, event, preference, advice, general"),
				mcplib.Enum("decision", "fact", "event", "preference", "advice", "general"),
			),
			mcplib.WithNumber("limit",
				mcplib.Description("Maximum number of results (default: 50)"),
			),
			mcplib.WithNumber("offset",
				mcplib.Description("Offset for pagination (default: 0)"),
			),
		), s.handleMemoryList)

		// memory_store is readonly — the agent loop requires storing memories
		s.mcp.AddTool(mcplib.NewTool("memory_store",
			mcplib.WithDescription("Store a new memory in a project. Deduplicates by content hash."),
			mcplib.WithNumber("project_id",
				mcplib.Required(),
				mcplib.Description("ID of the project. Use 0 to store a global (cross-project) memory."),
			),
			mcplib.WithString("content",
				mcplib.Required(),
				mcplib.Description("Memory content text (max 50KB)"),
			),
			mcplib.WithString("summary",
				mcplib.Description("Brief summary of the memory"),
			),
			mcplib.WithString("category",
				mcplib.Description("Category: decision, fact, event, preference, advice, general (default: general)"),
				mcplib.Enum("decision", "fact", "event", "preference", "advice", "general"),
			),
			mcplib.WithString("tags",
				mcplib.Description("Comma-separated tags"),
			),
			mcplib.WithString("source",
				mcplib.Description("Source of the memory"),
			),
			mcplib.WithNumber("importance",
				mcplib.Description("Importance level 1-5 (default: 3)"),
			),
		), s.handleMemoryStore)

		// Admin-only memory tools
		if !s.readonly {
			s.mcp.AddTool(mcplib.NewTool("memory_delete",
				mcplib.WithDescription("Delete a memory from a project"),
				mcplib.WithNumber("project_id",
					mcplib.Required(),
					mcplib.Description("ID of the project the memory belongs to"),
				),
				mcplib.WithNumber("memory_id",
					mcplib.Required(),
					mcplib.Description("ID of the memory to delete"),
				),
			), s.handleMemoryDelete)
		}
	}

	// Triple tools (only if triple manager is configured)
	if s.tripleManager != nil {
		// Readonly triple tools
		s.mcp.AddTool(mcplib.NewTool("triple_query",
			mcplib.WithDescription("Query knowledge graph triples within a project. Supports filtering by subject/predicate/object (substring match), active-only, and point-in-time queries."),
			mcplib.WithNumber("project_id",
				mcplib.Required(),
				mcplib.Description("ID of the project. Use 0 for global (cross-project) triples."),
			),
			mcplib.WithString("subject",
				mcplib.Description("Filter by subject (substring match)"),
			),
			mcplib.WithString("predicate",
				mcplib.Description("Filter by predicate (substring match)"),
			),
			mcplib.WithString("object",
				mcplib.Description("Filter by object (substring match)"),
			),
			mcplib.WithBoolean("active_only",
				mcplib.Description("Only return currently active triples (valid_to IS NULL)"),
			),
			mcplib.WithString("point_in_time",
				mcplib.Description("Return triples valid at this time (RFC3339 format, e.g. 2024-06-01T00:00:00Z)"),
			),
			mcplib.WithNumber("limit",
				mcplib.Description("Maximum number of results (default: 50)"),
			),
			mcplib.WithNumber("offset",
				mcplib.Description("Offset for pagination (default: 0)"),
			),
		), s.handleTripleQuery)

		// triple_store is readonly — the agent loop requires storing triples
		s.mcp.AddTool(mcplib.NewTool("triple_store",
			mcplib.WithDescription("Store a knowledge graph triple (subject-predicate-object fact) with temporal validity. Use replace_existing=true for single-valued predicates like 'status' or 'assigned_to'."),
			mcplib.WithNumber("project_id",
				mcplib.Required(),
				mcplib.Description("ID of the project. Use 0 to store a global (cross-project) triple."),
			),
			mcplib.WithString("subject",
				mcplib.Required(),
				mcplib.Description("Entity the triple is about (max 1024 chars)"),
			),
			mcplib.WithString("predicate",
				mcplib.Required(),
				mcplib.Description("Relationship or property (max 1024 chars)"),
			),
			mcplib.WithString("object",
				mcplib.Required(),
				mcplib.Description("Value or target entity (max 1024 chars)"),
			),
			mcplib.WithString("valid_from",
				mcplib.Description("When this fact became true (RFC3339, default: now)"),
			),
			mcplib.WithString("valid_to",
				mcplib.Description("When this fact stopped being true (RFC3339, omit for still-active facts)"),
			),
			mcplib.WithNumber("confidence",
				mcplib.Description("Confidence level 0.0-1.0 (default: 1.0)"),
			),
			mcplib.WithNumber("source_memory_id",
				mcplib.Description("Optional ID of the memory this triple was extracted from"),
			),
			mcplib.WithBoolean("replace_existing",
				mcplib.Description("If true, auto-invalidate active triples with same subject+predicate but different object"),
			),
		), s.handleTripleStore)

		// Admin-only triple tools
		if !s.readonly {
			s.mcp.AddTool(mcplib.NewTool("triple_invalidate",
				mcplib.WithDescription("Invalidate a triple by setting its valid_to timestamp, marking it as no longer active"),
				mcplib.WithNumber("project_id",
					mcplib.Required(),
					mcplib.Description("ID of the project"),
				),
				mcplib.WithNumber("triple_id",
					mcplib.Required(),
					mcplib.Description("ID of the triple to invalidate"),
				),
				mcplib.WithString("valid_to",
					mcplib.Description("When the fact stopped being true (RFC3339, default: now)"),
				),
			), s.handleTripleInvalidate)

			s.mcp.AddTool(mcplib.NewTool("triple_delete",
				mcplib.WithDescription("Permanently delete a triple from the knowledge graph"),
				mcplib.WithNumber("project_id",
					mcplib.Required(),
					mcplib.Description("ID of the project"),
				),
				mcplib.WithNumber("triple_id",
					mcplib.Required(),
					mcplib.Description("ID of the triple to delete"),
				),
			), s.handleTripleDelete)
		}
	}

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

// Memory handlers

func (s *Server) handleMemorySearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	query, err := req.RequireString("query")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	opts := memory.SearchOptions{
		Category: req.GetString("category", ""),
		Limit:    req.GetInt("limit", 0),
	}

	results, err := s.memoryManager.Search(ctx, int64(projectID), query, opts)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to search memories: %v", err)), nil
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleMemoryList(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	opts := memory.ListOptions{
		Category: req.GetString("category", ""),
		Limit:    req.GetInt("limit", 0),
		Offset:   req.GetInt("offset", 0),
	}

	mems, err := s.memoryManager.List(ctx, int64(projectID), opts)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to list memories: %v", err)), nil
	}

	data, err := json.MarshalIndent(mems, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleMemoryStore(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	content, err := req.RequireString("content")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	input := memory.StoreMemoryInput{
		ProjectID:  int64(projectID),
		Content:    content,
		Summary:    req.GetString("summary", ""),
		Category:   req.GetString("category", ""),
		Tags:       req.GetString("tags", ""),
		Source:     req.GetString("source", ""),
		Importance: req.GetInt("importance", 0),
	}

	mem, err := s.memoryManager.Store(ctx, input)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to store memory: %v", err)), nil
	}

	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Memory stored successfully:\n%s", string(data))), nil
}

func (s *Server) handleMemoryDelete(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	memoryID, err := req.RequireInt("memory_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	if err := s.memoryManager.Delete(ctx, int64(projectID), int64(memoryID)); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to delete memory: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Memory %d deleted successfully from project %d", memoryID, projectID)), nil
}

// Triple handlers

func (s *Server) handleTripleQuery(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	opts := memory.QueryTripleOptions{
		Subject:     req.GetString("subject", ""),
		Predicate:   req.GetString("predicate", ""),
		Object:      req.GetString("object", ""),
		ActiveOnly:  req.GetBool("active_only", false),
		PointInTime: req.GetString("point_in_time", ""),
		Limit:       req.GetInt("limit", 0),
		Offset:      req.GetInt("offset", 0),
	}

	results, err := s.tripleManager.Query(ctx, int64(projectID), opts)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to query triples: %v", err)), nil
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(string(data)), nil
}

func (s *Server) handleTripleStore(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	subject, err := req.RequireString("subject")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	predicate, err := req.RequireString("predicate")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	object, err := req.RequireString("object")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	input := memory.StoreTripleInput{
		ProjectID:       int64(projectID),
		Subject:         subject,
		Predicate:       predicate,
		Object:          object,
		ValidFrom:       req.GetString("valid_from", ""),
		ValidTo:         req.GetString("valid_to", ""),
		Confidence:      req.GetFloat("confidence", 0),
		ReplaceExisting: req.GetBool("replace_existing", false),
	}

	sourceMemoryID := req.GetInt("source_memory_id", 0)
	if sourceMemoryID > 0 {
		id := int64(sourceMemoryID)
		input.SourceMemoryID = &id
	}

	triple, err := s.tripleManager.Store(ctx, input)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to store triple: %v", err)), nil
	}

	data, err := json.MarshalIndent(triple, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Triple stored successfully:\n%s", string(data))), nil
}

func (s *Server) handleTripleInvalidate(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	tripleID, err := req.RequireInt("triple_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	validToStr := req.GetString("valid_to", "")
	validTo := time.Now().UTC()
	if validToStr != "" {
		parsed, parseErr := time.Parse(time.RFC3339, validToStr)
		if parseErr != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Invalid valid_to format: %v", parseErr)), nil
		}
		validTo = parsed.UTC()
	}

	if err := s.tripleManager.Invalidate(ctx, int64(projectID), int64(tripleID), validTo); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to invalidate triple: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Triple %d invalidated successfully (valid_to=%s)", tripleID, validTo.Format(time.RFC3339))), nil
}

func (s *Server) handleTripleDelete(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	projectID, err := req.RequireInt("project_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	tripleID, err := req.RequireInt("triple_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	if err := s.tripleManager.Delete(ctx, int64(projectID), int64(tripleID)); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to delete triple: %v", err)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Triple %d deleted successfully from project %d", tripleID, projectID)), nil
}
