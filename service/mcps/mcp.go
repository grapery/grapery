package mcps

import (
	"context"
	"encoding/json"
	"fmt"
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

// McpService implements the MCP protocol for story management
type McpService struct {
	stories       map[string]*Story
	characters    map[string]*Character
	images        map[string]*StoryImage
	users         map[string]*User
	storyVersions map[string]*StoryVersion
	storyPoints   map[string]*StoryPoint
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
	}
}

// HandleRequest processes incoming MCP requests
func (s *McpService) HandleRequest(ctx context.Context, request []byte) ([]byte, error) {
	var req map[string]interface{}
	if err := json.Unmarshal(request, &req); err != nil {
		return nil, fmt.Errorf("invalid request format: %v", err)
	}

	action, ok := req["action"].(string)
	if !ok {
		return nil, fmt.Errorf("missing action in request")
	}

	switch action {
	case "create_story":
		return s.handleCreateStory(ctx, req)
	case "get_story":
		return s.handleGetStory(ctx, req)
	case "create_character":
		return s.handleCreateCharacter(ctx, req)
	case "get_character":
		return s.handleGetCharacter(ctx, req)
	case "generate_image":
		return s.handleGenerateImage(ctx, req)
	case "chat_with_character":
		return s.handleCharacterChat(ctx, req)
	case "follow_character":
		return s.handleFollowCharacter(ctx, req)
	case "unfollow_character":
		return s.handleUnfollowCharacter(ctx, req)
	case "like_story":
		return s.handleLikeStory(ctx, req)
	case "unlike_story":
		return s.handleUnlikeStory(ctx, req)
	case "regenerate_character":
		return s.handleRegenerateCharacter(ctx, req)
	case "follow_story":
		return s.handleFollowStory(ctx, req)
	case "unfollow_story":
		return s.handleUnfollowStory(ctx, req)
	case "like_story_version":
		return s.handleLikeStoryVersion(ctx, req)
	case "unlike_story_version":
		return s.handleUnlikeStoryVersion(ctx, req)
	case "create_story_version":
		return s.handleCreateStoryVersion(ctx, req)
	case "continue_story_version":
		return s.handleContinueStoryVersion(ctx, req)
	case "follow_user":
		return s.handleFollowUser(ctx, req)
	case "unfollow_user":
		return s.handleUnfollowUser(ctx, req)
	case "analyze_character_storyline":
		return s.handleAnalyzeCharacterStoryline(ctx, req)
	case "subscribe_vip":
		return s.handleSubscribeVIP(ctx, req)
	case "unsubscribe_vip":
		return s.handleUnsubscribeVIP(ctx, req)
	case "create_story_point":
		return s.handleCreateStoryPoint(ctx, req)
	case "set_story_point_reward":
		return s.handleSetStoryPointReward(ctx, req)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
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
