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
			"queue://list",
			"All Queues",
			mcplib.WithResourceDescription("List all task queues"),
			mcplib.WithMIMEType("application/json"),
		),
		s.handleQueueListResource,
	)

	// Dynamic resource: get specific queue
	s.mcp.AddResource(
		mcplib.NewResource(
			"queue://{queue_id}",
			"Queue Details",
			mcplib.WithResourceDescription("Get details of a specific queue"),
			mcplib.WithMIMEType("application/json"),
		),
		s.handleQueueResource,
	)

	// Dynamic resource: get tasks in a queue
	s.mcp.AddResource(
		mcplib.NewResource(
			"queue://{queue_id}/tasks",
			"Queue Tasks",
			mcplib.WithResourceDescription("Get all tasks in a specific queue"),
			mcplib.WithMIMEType("application/json"),
		),
		s.handleQueueTasksResource,
	)

	// Dynamic resource: get specific task
	s.mcp.AddResource(
		mcplib.NewResource(
			"task://{task_id}",
			"Task Details",
			mcplib.WithResourceDescription("Get details of a specific task"),
			mcplib.WithMIMEType("application/json"),
		),
		s.handleTaskResource,
	)

	return nil
}

// Resource handlers

func (s *Server) handleQueueListResource(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	queues, err := s.manager.ListQueues(ctx)
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

func (s *Server) handleQueueResource(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	// Extract queue_id from URI
	queueID, err := extractQueueID(req.Params.URI)
	if err != nil {
		return nil, err
	}

	q, err := s.manager.GetQueue(ctx, queueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}

	stats, err := s.manager.GetQueueStats(ctx, queueID)
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

func (s *Server) handleQueueTasksResource(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	// Extract queue_id from URI
	queueID, err := extractQueueIDFromTasksURI(req.Params.URI)
	if err != nil {
		return nil, err
	}

	tasks, err := s.manager.ListTasks(ctx, queueID, nil)
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

func (s *Server) handleTaskResource(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	// Extract task_id from URI
	taskID, err := extractTaskID(req.Params.URI)
	if err != nil {
		return nil, err
	}

	task, err := s.manager.GetTask(ctx, taskID)
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

func extractQueueID(uri string) (int64, error) {
	re := regexp.MustCompile(`queue://(\d+)`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid queue URI format: %s", uri)
	}
	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid queue ID: %s", matches[1])
	}
	return id, nil
}

func extractQueueIDFromTasksURI(uri string) (int64, error) {
	re := regexp.MustCompile(`queue://(\d+)/tasks`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid queue tasks URI format: %s", uri)
	}
	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid queue ID: %s", matches[1])
	}
	return id, nil
}

func extractTaskID(uri string) (int64, error) {
	re := regexp.MustCompile(`task://(\d+)`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid task URI format: %s", uri)
	}
	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid task ID: %s", matches[1])
	}
	return id, nil
}
