package mcps

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	log "github.com/sirupsen/logrus"
)

// Server hosts the official MCP protocol over Streamable HTTP and legacy SSE.
type Server struct {
	service    *McpService
	mcpServer  *mcp.Server
	httpServer *http.Server
}

// NewServer creates a new MCP HTTP server backed by the given service.
func NewServer(service *McpService) *Server {
	return &Server{
		service:   service,
		mcpServer: service.BuildMCPServer(),
	}
}

// Start starts the MCP HTTP server.
//
// Endpoints:
//   - /mcp      — Streamable HTTP (recommended, MCP 2025-03-26+)
//   - /mcp/sse  — SSE transport (MCP 2024-11-05)
//   - /mcp/legacy — legacy action-based JSON POST (compat)
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	streamable := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)
	sse := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)

	mux.Handle("/mcp", streamable)
	mux.Handle("/mcp/sse", sse)
	mux.HandleFunc("/mcp/legacy", s.handleLegacy)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		// Streamable/SSE sessions are long-lived; do not set WriteTimeout.
	}

	log.Infof("Starting MCP server on %s (streamable=/mcp sse=/mcp/sse legacy=/mcp/legacy)", addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the MCP HTTP server.
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// handleLegacy preserves the previous action-based POST contract used by
// early clients before the official MCP SDK migration.
func (s *Server) handleLegacy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	response, err := s.service.HandleRequest(r.Context(), req)
	if err != nil {
		log.Errorf("legacy MCP handle request failed: %v", err)
		response = formatResponse("error", "handler error", nil, err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		log.Errorf("legacy MCP write failed: %v", err)
	}
}
