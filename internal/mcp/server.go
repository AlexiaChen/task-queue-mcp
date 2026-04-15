package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/memory"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"
)

// Server wraps the MCP server with queue management
type Server struct {
	mcp            *server.MCPServer
	manager        *queue.Manager
	memoryManager  *memory.MemoryManager
	tripleManager  *memory.TripleManager
	readonly       bool
}

// ServerOption is a function that configures the server
type ServerOption func(*Server)

// WithReadonlyMode configures the server to only expose read and update tools
func WithReadonlyMode(readonly bool) ServerOption {
	return func(s *Server) {
		s.readonly = readonly
	}
}

// WithMemoryManager configures the server with a memory manager
func WithMemoryManager(mm *memory.MemoryManager) ServerOption {
	return func(s *Server) {
		s.memoryManager = mm
	}
}

// WithTripleManager configures the server with a triple manager
func WithTripleManager(tm *memory.TripleManager) ServerOption {
	return func(s *Server) {
		s.tripleManager = tm
	}
}

// NewServer creates a new MCP server for issue kanban management
func NewServer(manager *queue.Manager, opts ...ServerOption) (*Server, error) {
	s := &Server{
		mcp: server.NewMCPServer(
			"Issue Kanban MCP Server",
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
