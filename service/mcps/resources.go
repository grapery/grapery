package mcps

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	storyResourceURI     = "grapery://stories"
	characterResourceURI = "grapery://characters"
)

// registerResources wires in-memory story/character catalog resources.
func (s *McpService) registerResources(server *mcp.Server) {
	server.AddResource(&mcp.Resource{
		URI:         storyResourceURI,
		Name:        "stories",
		Description: "当前 MCP 会话中缓存的故事列表（JSON）",
		MIMEType:    "application/json",
	}, s.readStoriesResource)

	server.AddResource(&mcp.Resource{
		URI:         characterResourceURI,
		Name:        "characters",
		Description: "当前 MCP 会话中缓存的角色列表（JSON）",
		MIMEType:    "application/json",
	}, s.readCharactersResource)
}

func (s *McpService) readStoriesResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stories := make([]*Story, 0, len(s.stories))
	for _, story := range s.stories {
		stories = append(stories, story)
	}

	data, err := json.Marshal(stories)
	if err != nil {
		return nil, err
	}

	uri := storyResourceURI
	if req != nil && req.Params != nil && req.Params.URI != "" {
		uri = req.Params.URI
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func (s *McpService) readCharactersResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	characters := make([]*Character, 0, len(s.characters))
	for _, character := range s.characters {
		characters = append(characters, character)
	}

	data, err := json.Marshal(characters)
	if err != nil {
		return nil, err
	}

	uri := characterResourceURI
	if req != nil && req.Params != nil && req.Params.URI != "" {
		uri = req.Params.URI
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}
