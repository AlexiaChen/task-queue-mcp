// Example STDIO MCP Client demonstrating how Claude Desktop would use this MCP Server
//
// This example shows how to connect to the MCP server via STDIO transport,
// which is how Claude Desktop and other MCP clients typically connect.
//
// Usage:
//   go run ./examples/stdio-client/main.go
//
// The client spawns the server as a subprocess and communicates via STDIN/STDOUT.

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

	fmt.Println("=== STDIO MCP Client Example ===")
	fmt.Println("This simulates how Claude Desktop connects to the MCP server")
	fmt.Println()

	// Create STDIO client by spawning the server
	// In production, the path would be to the compiled binary
	c, err := client.NewStdioMCPClient(
		"go",           // command
		[]string{},     // environment variables
		"run",          // args...
		"./cmd/server",
		"-mcp=stdio",
		"-db=./data/example.db",
	)
	if err != nil {
		log.Fatalf("Failed to create STDIO client: %v", err)
	}
	defer c.Close()

	fmt.Println("Spawning server process and connecting via STDIO...")

	// Initialize the connection
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "STDIO Client Example",
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

	// Demonstrate tool usage
	fmt.Println("=== Using MCP Tools ===")

	// 1. Create a queue
	fmt.Println("1. Creating a queue...")
	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "queue_create",
			Arguments: map[string]interface{}{
				"name":        "My Task Queue",
				"description": "A simple task queue",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printToolResult(result)

	// 2. Create some tasks
	fmt.Println("2. Creating tasks...")
	for i := 1; i <= 3; i++ {
		_, err = c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "task_create",
				Arguments: map[string]interface{}{
					"queue_id":    1,
					"title":       fmt.Sprintf("Task #%d", i),
					"description": fmt.Sprintf("This is task number %d", i),
					"priority":    10 - i,
				},
			},
		})
		if err != nil {
			log.Printf("Failed to create task %d: %v", i, err)
		}
	}
	fmt.Println("Created 3 tasks")
	fmt.Println()

	// 3. List all queues with stats
	fmt.Println("3. Listing all queues...")
	result, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "queue_list",
			Arguments: map[string]interface{}{},
		},
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printToolResult(result)

	// 4. Start the first task
	fmt.Println("4. Starting task 1...")
	result, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_update",
			Arguments: map[string]interface{}{
				"task_id": 1,
				"status":  "doing",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printToolResult(result)

	// 5. Prioritize the last task (插队)
	fmt.Println("5. Prioritizing task 3 (moving to front)...")
	result, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_prioritize",
			Arguments: map[string]interface{}{
				"task_id":  3,
				"position": 1,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printToolResult(result)

	// 6. List tasks to see the result
	fmt.Println("6. Listing tasks in queue...")
	result, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task_list",
			Arguments: map[string]interface{}{
				"queue_id": 1,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printToolResult(result)

	// 7. Read a resource
	fmt.Println("7. Reading queue resource...")
	resourceResult, err := c.ReadResource(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "queue://1/tasks",
		},
	})
	if err != nil {
		log.Printf("Failed to read resource: %v", err)
	} else {
		for _, content := range resourceResult.Contents {
			if textContent, ok := mcp.AsTextResourceContents(content); ok {
				fmt.Println(textContent.Text)
			}
		}
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("This is how Claude Desktop would interact with the Task Queue MCP Server!")
}

func printToolResult(result *mcp.CallToolResult) {
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}
	fmt.Println()
}
