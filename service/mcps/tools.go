package mcps

import (
	"context"

	mcp "github.com/metoro-io/mcp-golang"
)

// Base tool structure
type BaseTool struct {
	Service *McpService
}

// CreateStoryTool handles story creation
type CreateStoryTool struct {
	BaseTool
}

func (t *CreateStoryTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.ToolResponse, error) {
	response, err := t.Service.handleCreateStory(ctx, args)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(response))), nil
}

// GetStoryTool handles story retrieval
type GetStoryTool struct {
	BaseTool
}

func (t *GetStoryTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.ToolResponse, error) {
	response, err := t.Service.handleGetStory(ctx, args)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(response))), nil
}

// CreateCharacterTool handles character creation
type CreateCharacterTool struct {
	BaseTool
}

func (t *CreateCharacterTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.ToolResponse, error) {
	response, err := t.Service.handleCreateCharacter(ctx, args)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(response))), nil
}

// GetCharacterTool handles character retrieval
type GetCharacterTool struct {
	BaseTool
}

func (t *GetCharacterTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.ToolResponse, error) {
	response, err := t.Service.handleGetCharacter(ctx, args)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(response))), nil
}

// FollowCharacterTool handles following a character
type FollowCharacterTool struct {
	BaseTool
}

func (t *FollowCharacterTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.ToolResponse, error) {
	response, err := t.Service.handleFollowCharacter(ctx, args)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(response))), nil
}

// UnfollowCharacterTool handles unfollowing a character
type UnfollowCharacterTool struct {
	BaseTool
}

func (t *UnfollowCharacterTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.ToolResponse, error) {
	response, err := t.Service.handleUnfollowCharacter(ctx, args)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(response))), nil
}

// LikeStoryTool handles liking a story
type LikeStoryTool struct {
	BaseTool
}

func (t *LikeStoryTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.ToolResponse, error) {
	response, err := t.Service.handleLikeStory(ctx, args)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(response))), nil
}

// UnlikeStoryTool handles unliking a story
type UnlikeStoryTool struct {
	BaseTool
}

func (t *UnlikeStoryTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.ToolResponse, error) {
	response, err := t.Service.handleUnlikeStory(ctx, args)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(response))), nil
}
