package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	mcplib "task-queue-mcp/internal/mcp"
	"task-queue-mcp/internal/api"
	"task-queue-mcp/internal/queue"
	"task-queue-mcp/internal/storage"
	"task-queue-mcp/internal/web"
)

func main() {
	// Parse flags
	httpPort := flag.Int("port", 9292, "HTTP server port")
	dbPath := flag.String("db", "./data/tasks.db", "SQLite database path")
	mcpMode := flag.String("mcp", "http", "MCP mode: stdio, http, or both")
	flag.Parse()

	// Initialize storage
	store, err := storage.NewSQLiteStorage(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize queue manager
	manager := queue.NewManager(store)

	// Create MCP server
	mcpServer, err := mcplib.NewServer(manager)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Handle different modes
	if *mcpMode == "stdio" {
		// STDIO-only mode for MCP clients
		log.Println("Starting MCP server in STDIO mode...")
		if err := server.ServeStdio(mcpServer.GetMCPServer()); err != nil {
			log.Fatalf("Server error: %v", err)
		}
		return
	}

	// HTTP mode (with optional STDIO)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start STDIO server in goroutine if needed
	if *mcpMode == "both" {
		go func() {
			log.Println("Starting MCP STDIO server...")
			if err := server.ServeStdio(mcpServer.GetMCPServer()); err != nil {
				log.Printf("STDIO server error: %v", err)
			}
		}()
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Register REST API
	apiHandler := api.NewHandler(manager)
	apiHandler.RegisterRoutes(mux)

	// Serve static files
	staticFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create static FS: %v", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve index.html for root
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, staticFS, "index.html")
	})

	// Start SSE server for MCP over HTTP
	sseServer := server.NewSSEServer(mcpServer.GetMCPServer())
	mux.Handle("GET /sse", sseServer)
	mux.Handle("POST /message", sseServer)

	// Start HTTP server
	httpAddr := fmt.Sprintf(":%d", *httpPort)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting HTTP server on %s", httpAddr)
		log.Printf("Web UI: http://localhost%s", httpAddr)
		log.Printf("MCP SSE: http://localhost%s/sse", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
