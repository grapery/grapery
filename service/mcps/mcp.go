package mcps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/grapery/grapery/config"
)

// Story represents a story with its metadata and content
type Story struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Characters  []string `json:"characters"`
	Content     string   `json:"content"`
	CreatedAt   int64    `json:"created_at"`
	UpdatedAt   int64    `json:"updated_at"`
	Likes       int      `json:"likes"`
	Followers   []string `json:"followers"`
	StoryPoints []string `json:"story_points"`
	AuthorID    string   `json:"author_id"`
}

// Character represents a story character
type Character struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Personality string   `json:"personality"`
	StoryID     string   `json:"story_id"`
	CreatedAt   int64    `json:"created_at"`
	UpdatedAt   int64    `json:"updated_at"`
	Followers   []string `json:"followers"`
	Storyline   string   `json:"storyline"`
}

// StoryImage represents an image associated with a story or character
type StoryImage struct {
	ID          string `json:"id"`
	StoryID     string `json:"story_id"`
	CharacterID string `json:"character_id,omitempty"`
	URL         string `json:"url"`
	Type        string `json:"type"` // "story" or "character"
	CreatedAt   int64  `json:"created_at"`
}

// StoryVersion represents a version of a story
type StoryVersion struct {
	ID        string   `json:"id"`
	StoryID   string   `json:"story_id"`
	Content   string   `json:"content"`
	CreatedAt int64    `json:"created_at"`
	Likes     int      `json:"likes"`
	Followers []string `json:"followers"`
}

// User represents a user in the system
type User struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	Followers     []string `json:"followers"`
	Following     []string `json:"following"`
	IsVIP         bool     `json:"is_vip"`
	VIPExpireTime int64    `json:"vip_expire_time"`
}

// StoryPoint represents a story point with reward
type StoryPoint struct {
	ID          string  `json:"id"`
	StoryID     string  `json:"story_id"`
	Description string  `json:"description"`
	Reward      float64 `json:"reward"`
	Status      string  `json:"status"` // "open", "claimed", "completed"
	ClaimedBy   string  `json:"claimed_by,omitempty"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

// MCPResource represents a resource in the MCP protocol
type MCPResource struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt int64                  `json:"created_at"`
	UpdatedAt int64                  `json:"updated_at"`
}

// MCPPrompt represents a prompt in the MCP protocol
type MCPPrompt struct {
	ID        string   `json:"id"`
	Content   string   `json:"content"`
	Context   []string `json:"context"`
	Tools     []string `json:"tools"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

// MCPTool represents a tool in the MCP protocol
type MCPTool struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
	CreatedAt   int64    `json:"created_at"`
	UpdatedAt   int64    `json:"updated_at"`
}

// MCPRoot represents a root in the MCP protocol
type MCPRoot struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Resources []string `json:"resources"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

// MCPTransport represents a transport in the MCP protocol
type MCPTransport struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt int64                  `json:"created_at"`
	UpdatedAt int64                  `json:"updated_at"`
}

// Response represents a standardized API response
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// McpService implements the MCP protocol for story management
type McpService struct {
	stories       map[string]*Story
	characters    map[string]*Character
	images        map[string]*StoryImage
	users         map[string]*User
	storyVersions map[string]*StoryVersion
	storyPoints   map[string]*StoryPoint
	resources     map[string]*MCPResource
	prompts       map[string]*MCPPrompt
	tools         map[string]*MCPTool
	roots         map[string]*MCPRoot
	transports    map[string]*MCPTransport
	mu            sync.RWMutex
	config        *config.Config
}

// NewMcpService creates a new MCP service instance
func NewMcpService() *McpService {
	return &McpService{
		stories:       make(map[string]*Story),
		characters:    make(map[string]*Character),
		images:        make(map[string]*StoryImage),
		users:         make(map[string]*User),
		storyVersions: make(map[string]*StoryVersion),
		storyPoints:   make(map[string]*StoryPoint),
		resources:     make(map[string]*MCPResource),
		prompts:       make(map[string]*MCPPrompt),
		tools:         make(map[string]*MCPTool),
		roots:         make(map[string]*MCPRoot),
		transports:    make(map[string]*MCPTransport),
	}
}

// formatResponse creates a standardized response
func formatResponse(status string, message string, data interface{}, err error) []byte {
	resp := Response{
		Status:  status,
		Message: message,
		Data:    data,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	response, _ := json.Marshal(resp)
	return response
}

// validateRequiredFields checks if all required fields are present in the request
func validateRequiredFields(req map[string]interface{}, fields ...string) error {
	for _, field := range fields {
		if _, ok := req[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

// validateFieldType checks if a field has the expected type
func validateFieldType(req map[string]interface{}, field string, expectedType string) error {
	value, ok := req[field]
	if !ok {
		return fmt.Errorf("missing field: %s", field)
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field %s must be a string", field)
		}
	case "int":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("field %s must be a number", field)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %s must be a boolean", field)
		}
	case "map":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("field %s must be an object", field)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("field %s must be an array", field)
		}
	}
	return nil
}

// HandleRequest processes incoming MCP requests
func (s *McpService) HandleRequest(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	// Validate action field
	if err := validateRequiredFields(req, "action"); err != nil {
		return formatResponse("error", "", nil, err), nil
	}

	action, _ := req["action"].(string)
	var response []byte
	var err error

	// Use a read lock for get operations
	if strings.HasPrefix(action, "get_") {
		s.mu.RLock()
		defer s.mu.RUnlock()
	} else {
		// Use a write lock for all other operations
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	switch action {
	case "create_resource":
		if err := validateRequiredFields(req, "type", "data"); err != nil {
			return formatResponse("error", "", nil, err), nil
		}
		if err := validateFieldType(req, "data", "map"); err != nil {
			return formatResponse("error", "", nil, err), nil
		}
		response, err = s.handleCreateResource(ctx, req)
	case "get_resource":
		if err := validateRequiredFields(req, "id"); err != nil {
			return formatResponse("error", "", nil, err), nil
		}
		response, err = s.handleGetResource(ctx, req)
	case "create_prompt":
		response, err = s.handleCreatePrompt(ctx, req)
	case "get_prompt":
		response, err = s.handleGetPrompt(ctx, req)
	case "create_tool":
		response, err = s.handleCreateTool(ctx, req)
	case "get_tool":
		response, err = s.handleGetTool(ctx, req)
	case "create_root":
		response, err = s.handleCreateRoot(ctx, req)
	case "get_root":
		response, err = s.handleGetRoot(ctx, req)
	case "create_transport":
		response, err = s.handleCreateTransport(ctx, req)
	case "get_transport":
		response, err = s.handleGetTransport(ctx, req)
	case "follow_character":
		response, err = s.handleFollowCharacter(ctx, req)
	case "unfollow_character":
		response, err = s.handleUnfollowCharacter(ctx, req)
	case "like_story":
		response, err = s.handleLikeStory(ctx, req)
	case "unlike_story":
		response, err = s.handleUnlikeStory(ctx, req)
	case "regenerate_character":
		response, err = s.handleRegenerateCharacter(ctx, req)
	case "follow_story":
		response, err = s.handleFollowStory(ctx, req)
	case "unfollow_story":
		response, err = s.handleUnfollowStory(ctx, req)
	case "create_story_version":
		response, err = s.handleCreateStoryVersion(ctx, req)
	case "continue_story_version":
		response, err = s.handleContinueStoryVersion(ctx, req)
	case "follow_user":
		response, err = s.handleFollowUser(ctx, req)
	case "unfollow_user":
		response, err = s.handleUnfollowUser(ctx, req)
	case "analyze_character_storyline":
		response, err = s.handleAnalyzeCharacterStoryline(ctx, req)
	case "subscribe_vip":
		response, err = s.handleSubscribeVIP(ctx, req)
	case "unsubscribe_vip":
		response, err = s.handleUnsubscribeVIP(ctx, req)
	case "create_story_point":
		response, err = s.handleCreateStoryPoint(ctx, req)
	case "set_story_point_reward":
		response, err = s.handleSetStoryPointReward(ctx, req)
	case "like_story_version":
		response, err = s.handleLikeStoryVersion(ctx, req)
	case "unlike_story_version":
		response, err = s.handleUnlikeStoryVersion(ctx, req)
	default:
		err = fmt.Errorf("unknown action: %s", action)
	}

	if err != nil {
		return formatResponse("error", "", nil, err), nil
	}
	return response, nil
}

// Initialize sets up the MCP service with configuration
func (s *McpService) Initialize(cfg *config.Config) error {
	s.config = cfg
	return nil
}

// Shutdown performs cleanup when the service is shutting down
func (s *McpService) Shutdown() error {
	// Perform any necessary cleanup
	return nil
}
