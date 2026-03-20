// Example MCP Client demonstrating how to use the Task Queue MCP Server
//
// This example shows how an MCP client (like Claude Desktop) would connect to
// and interact with the Task Queue MCP Server via SSE transport.
//
// Run:
//   1. Start the server: go run ./cmd/server -port=9292 -mcp=http
//   2. Run this client: go run ./examples/mcp-client/main.go
//
// The client demonstrates:
//   - Connecting to MCP server via SSE transport
//   - Listing available tools
//   - Calling tools to manage queues and tasks
//   - Reading resources to get data

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to MCP server via SSE transport
	// The server should be running with -mcp=http flag
	serverURL := "http://localhost:9292/sse"

	fmt.Println("=== MCP Client Example (SSE Transport) ===")
	fmt.Printf("Connecting to MCP server at %s...\n", serverURL)

	c, err := client.NewSSEMCPClient(serverURL)
	if err != nil {
		log.Fatalf("Failed to create SSE client: %v", err)
	}
	defer c.Close()

	// Initialize the connection
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "Task Queue MCP Client Example",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := c.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	fmt.Printf("Connected to: %s (version %s)\n\n",
		serverInfo.ServerInfo.Name,
		serverInfo.ServerInfo.Version)

	// Step 1: List available tools
	fmt.Println("=== Step 1: List Available Tools ===")
	toolsResp, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("Found %d tools:\n", len(toolsResp.Tools))
	for i, tool := range toolsResp.Tools {
		fmt.Printf("  %d. %s - %s\n", i+1, tool.Name, tool.Description)
	}
	fmt.Println()

	// Step 2: Create a queue
	fmt.Println("=== Step 2: Create a Queue ===")
	createQueueResult, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "queue_create",
			Arguments: map[string]interface{}{
				"name":        "Example Queue",
				"description": "A queue created by MCP client example",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create queue: %v", err)
	}
	printToolResult(createQueueResult)

	// Step 3: List all queues
	fmt.Println("=== Step 3: List All Queues ===")
	listQueuesResult, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "queue_list",
			Arguments: map[string]interface{}{},
		},
	})
	if err != nil {
		log.Fatalf("Failed to list queues: %v", err)
	}
	printToolResult(listQueuesResult)

	// Step 4: Create tasks in the queue
	fmt.Println("=== Step 4: Create Tasks ===")
	for i := 1; i <= 3; i++ {
		_, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "task_create",
				Arguments: map[string]interface{}{
					"queue_id":    1,
					"title":       fmt.Sprintf("Task %d", i),
					"description": fmt.Sprintf("Description for task %d", i),
					"priority":    i,
				},
			},
		})
		if err != nil {
			log.Fatalf("Failed to create task: %v", err)
		}
		fmt.Printf("Created task %d\n", i)
	}
	fmt.Println()

	// Step 5: List tasks in queue
	fmt.Println("=== Step 5: List Tasks in Queue ===")
	listTasksResult, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_list",
			Arguments: map[string]interface{}{
				"queue_id": 1,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to list tasks: %v", err)
	}
	printToolResult(listTasksResult)

	// Step 6: Start a task (pending -> doing)
	fmt.Println("=== Step 6: Start Task 1 ===")
	startResult, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_update",
			Arguments: map[string]interface{}{
				"task_id": 1,
				"status":  "doing",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to start task: %v", err)
	}
	printToolResult(startResult)

	// Step 7: Prioritize task 3 (插队)
	fmt.Println("=== Step 7: Prioritize Task 3 (插队) ===")
	prioritizeResult, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_prioritize",
			Arguments: map[string]interface{}{
				"task_id":  3,
				"position": 1,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to prioritize task: %v", err)
	}
	printToolResult(prioritizeResult)

	// Step 8: Read resources
	fmt.Println("=== Step 8: Read MCP Resources ===")

	// List available resources
	resourcesResp, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		log.Printf("Failed to list resources: %v", err)
	} else {
		fmt.Printf("Found %d resources:\n", len(resourcesResp.Resources))
		for i, res := range resourcesResp.Resources {
			fmt.Printf("  %d. %s (%s)\n", i+1, res.URI, res.Name)
		}
		fmt.Println()

		// Read the queue list resource
		readResult, err := c.ReadResource(ctx, mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "queue://list",
			},
		})
		if err != nil {
			log.Printf("Failed to read resource: %v", err)
		} else {
			fmt.Println("Resource content (queue://list):")
			for _, content := range readResult.Contents {
				if textContent, ok := mcp.AsTextResourceContents(content); ok {
					fmt.Println(textContent.Text)
				}
			}
		}
	}

	fmt.Println("\n=== MCP Client Example Complete ===")
}

func printToolResult(result *mcp.CallToolResult) {
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}
	fmt.Println()
}
