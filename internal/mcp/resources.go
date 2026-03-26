package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"task-queue-mcp/internal/queue"
)

// registerResources registers all MCP resources
func (s *Server) registerResources() error {
	// Static resource: list all queues
	s.mcp.AddResource(
		mcplib.NewResource(
			"project://list",
			"All Projects",
			mcplib.WithResourceDescription("List all projects"),
			mcplib.WithMIMEType("application/json"),
		),
		s.handleProjectListResource,
	)

	// Dynamic resource: get specific project
	s.mcp.AddResource(
		mcplib.NewResource(
			"project://{project_id}",
			"Project Details",
			mcplib.WithResourceDescription("Get details of a specific project"),
			mcplib.WithMIMEType("application/json"),
		),
		s.handleProjectResource,
	)

	// Dynamic resource: get issues in a project
	s.mcp.AddResource(
		mcplib.NewResource(
			"project://{project_id}/issues",
			"Project Issues",
			mcplib.WithResourceDescription("Get all issues in a specific project"),
			mcplib.WithMIMEType("application/json"),
		),
		s.handleProjectIssuesResource,
	)

	// Dynamic resource: get specific issue
	s.mcp.AddResource(
		mcplib.NewResource(
			"issue://{task_id}",
			"Issue Details",
			mcplib.WithResourceDescription("Get details of a specific issue"),
			mcplib.WithMIMEType("application/json"),
		),
		s.handleIssueResource,
	)

	return nil
}

// Resource handlers

func (s *Server) handleProjectListResource(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	queues, err := s.manager.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
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
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handleProjectResource(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	// Extract project_id from URI
	queueID, err := extractProjectID(req.Params.URI)
	if err != nil {
		return nil, err
	}

	q, err := s.manager.GetProject(ctx, queueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}

	stats, err := s.manager.GetProjectStats(ctx, queueID)
	if err != nil {
		stats = &queue.QueueStats{}
	}

	result := struct {
		*queue.Queue
		Stats *queue.QueueStats `json:"stats"`
	}{
		Queue: q,
		Stats: stats,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handleProjectIssuesResource(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	// Extract project_id from URI
	queueID, err := extractProjectIDFromIssuesURI(req.Params.URI)
	if err != nil {
		return nil, err
	}

	tasks, err := s.manager.ListIssues(ctx, queueID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handleIssueResource(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	// Extract task_id from URI
	taskID, err := extractIssueID(req.Params.URI)
	if err != nil {
		return nil, err
	}

	task, err := s.manager.GetIssue(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// URI extraction helpers

func extractProjectID(uri string) (int64, error) {
	re := regexp.MustCompile(`project://(\d+)`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid project URI format: %s", uri)
	}
	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid project ID: %s", matches[1])
	}
	return id, nil
}

func extractProjectIDFromIssuesURI(uri string) (int64, error) {
	re := regexp.MustCompile(`project://(\d+)/issues`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid project issues URI format: %s", uri)
	}
	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid project ID: %s", matches[1])
	}
	return id, nil
}

func extractIssueID(uri string) (int64, error) {
	re := regexp.MustCompile(`issue://(\d+)`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid issue URI format: %s", uri)
	}
	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid issue ID: %s", matches[1])
	}
	return id, nil
}
