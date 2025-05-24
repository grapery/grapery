package mcps

import (
	"context"

	mcp "github.com/metoro-io/mcp-golang"
)

// Base prompt structure
type BasePrompt struct {
	Service *McpService
}

// GenerateStoryPrompt handles story generation
type GenerateStoryPrompt struct {
	BasePrompt
}

func (p *GenerateStoryPrompt) Execute(ctx context.Context, args map[string]interface{}) (*mcp.PromptResponse, error) {
	// Convert args to story creation request
	req := map[string]interface{}{
		"action":      "create_story",
		"title":       args["title"],
		"description": args["description"],
		"content":     args["content"],
	}

	response, err := p.Service.handleCreateStory(ctx, req)
	if err != nil {
		return nil, err
	}

	return mcp.NewPromptResponse("story", mcp.NewPromptMessage(
		mcp.NewTextContent(string(response)),
		mcp.RoleAssistant,
	)), nil
}

// GenerateCharacterPrompt handles character generation
type GenerateCharacterPrompt struct {
	BasePrompt
}

func (p *GenerateCharacterPrompt) Execute(ctx context.Context, args map[string]interface{}) (*mcp.PromptResponse, error) {
	// Convert args to character creation request
	req := map[string]interface{}{
		"action":      "create_character",
		"name":        args["name"],
		"description": args["description"],
		"personality": args["personality"],
		"story_id":    args["story_id"],
	}

	response, err := p.Service.handleCreateCharacter(ctx, req)
	if err != nil {
		return nil, err
	}

	return mcp.NewPromptResponse("character", mcp.NewPromptMessage(
		mcp.NewTextContent(string(response)),
		mcp.RoleAssistant,
	)), nil
}
