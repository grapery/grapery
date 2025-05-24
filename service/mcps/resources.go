package mcps

import (
	"context"
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
)

// Base resource structure
type BaseResource struct {
	Service *McpService
}

// StoryResource handles story resources
type StoryResource struct {
	BaseResource
}

func (r *StoryResource) Get(ctx context.Context) (*mcp.ResourceResponse, error) {
	r.Service.mu.RLock()
	defer r.Service.mu.RUnlock()

	stories := make([]*Story, 0, len(r.Service.stories))
	for _, story := range r.Service.stories {
		stories = append(stories, story)
	}

	data, err := json.Marshal(stories)
	if err != nil {
		return nil, err
	}

	return mcp.NewResourceResponse(mcp.NewTextEmbeddedResource(
		"story://",
		string(data),
		"application/json",
	)), nil
}

// CharacterResource handles character resources
type CharacterResource struct {
	BaseResource
}

func (r *CharacterResource) Get(ctx context.Context) (*mcp.ResourceResponse, error) {
	r.Service.mu.RLock()
	defer r.Service.mu.RUnlock()

	characters := make([]*Character, 0, len(r.Service.characters))
	for _, character := range r.Service.characters {
		characters = append(characters, character)
	}

	data, err := json.Marshal(characters)
	if err != nil {
		return nil, err
	}

	return mcp.NewResourceResponse(mcp.NewTextEmbeddedResource(
		"character://",
		string(data),
		"application/json",
	)), nil
}
