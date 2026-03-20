package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"task-queue-mcp/internal/queue"
)

// Server wraps the MCP server with queue management
type Server struct {
	mcp     *server.MCPServer
	manager *queue.Manager
}

// NewServer creates a new MCP server for task queue management
func NewServer(manager *queue.Manager) (*Server, error) {
	s := &Server{
		mcp: server.NewMCPServer(
			"Task Queue MCP Server",
			"1.0.0",
			server.WithToolCapabilities(true),
			server.WithResourceCapabilities(true, true),
		),
		manager: manager,
	}

	if err := s.registerTools(); err != nil {
		return nil, err
	}

	if err := s.registerResources(); err != nil {
		return nil, err
	}

	return s, nil
}

// GetMCPServer returns the underlying MCP server
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcp
}
