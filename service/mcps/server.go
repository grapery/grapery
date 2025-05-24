package mcps

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Implement proper origin checking
	},
}

// Server represents the MCP server
type Server struct {
	service *McpService
	server  *http.Server
}

// NewServer creates a new MCP server instance
func NewServer(service *McpService) *Server {
	return &Server{
		service: service,
	}
}

// Start starts the MCP server
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleWebSocket)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Infof("Starting MCP server on %s", addr)
	return s.server.ListenAndServe()
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// handleWebSocket handles WebSocket connections for MCP
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("WebSocket error: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			response, err := s.service.HandleRequest(r.Context(), message)
			if err != nil {
				errorResponse := map[string]interface{}{
					"status":  "error",
					"message": err.Error(),
				}
				response, _ = json.Marshal(errorResponse)
			}

			if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
				log.Errorf("Failed to write message: %v", err)
				break
			}
		}
	}
}
