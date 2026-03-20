package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"task-queue-mcp/internal/queue"
)

// Server wraps the MCP server with queue management
type Server struct {
	mcp      *server.MCPServer
	manager  *queue.Manager
	readonly bool
}

// ServerOption is a function that configures the server
type ServerOption func(*Server)

// WithReadonlyMode configures the server to only expose read and update tools
func WithReadonlyMode(readonly bool) ServerOption {
	return func(s *Server) {
		s.readonly = readonly
	}
}

// NewServer creates a new MCP server for task queue management
func NewServer(manager *queue.Manager, opts ...ServerOption) (*Server, error) {
	s := &Server{
		mcp: server.NewMCPServer(
			"Task Queue MCP Server",
			"1.0.0",
			server.WithToolCapabilities(true),
			server.WithResourceCapabilities(true, true),
		),
		manager:  manager,
		readonly: false,
	}

	for _, opt := range opts {
		opt(s)
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
