package gemini

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// GenerateContent delegates to the official GenAI SDK and returns the structured response.
func (c *Client) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	if len(contents) == 0 {
		return nil, fmt.Errorf("contents cannot be empty")
	}
	resolvedModel := choose(model, c.config.DefaultModel, DefaultTextModel)
	return c.sdk.Models.GenerateContent(ctx, resolvedModel, contents, config)
}

// GenerateText is a convenience helper that constructs a simple text-only request.
func (c *Client) GenerateText(ctx context.Context, model, prompt string, config *genai.GenerateContentConfig) (string, *genai.GenerateContentResponse, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", nil, fmt.Errorf("prompt cannot be empty")
	}
	resp, err := c.GenerateContent(ctx, model, genai.Text(prompt), config)
	if err != nil {
		return "", nil, err
	}
	if text := strings.TrimSpace(resp.Text()); text != "" {
		return text, resp, nil
	}
	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part != nil && strings.TrimSpace(part.Text) != "" {
				return strings.TrimSpace(part.Text), resp, nil
			}
		}
	}
	return "", resp, nil
}

// ThinkingLevel controls the amount of reasoning the model uses.
// See: https://ai.google.dev/gemini-api/docs/text-generation
type ThinkingLevel string

const (
	ThinkingLevelLow     ThinkingLevel = "low"
	ThinkingLevelHigh    ThinkingLevel = "high"
	ThinkingLevelMinimal ThinkingLevel = "minimal"
)

func mapThinkingLevel(l ThinkingLevel) genai.ThinkingLevel {
	switch strings.ToLower(string(l)) {
	case "low":
		return genai.ThinkingLevelLow
	case "high":
		return genai.ThinkingLevelHigh
	case "minimal":
		return genai.ThinkingLevelMinimal
	default:
		return genai.ThinkingLevel(string(l))
	}
}

// ThinkingOptions configures Gemini thinking behavior.
type ThinkingOptions struct {
	// Level: "low", "high", or "minimal". Empty disables thinking.
	Level ThinkingLevel
	// IncludeThoughts: if true, thoughts are returned in the response.
	IncludeThoughts bool
	// Budget: token budget for thinking (alternative to Level). If > 0, overrides Level.
	Budget int32
}

// GenerateThinkingText enables Gemini "thinking" traces while still returning the final answer text.
// Use opts to configure thinking level (low/high) and whether to include thoughts in the response.
// If an optional config is provided it will be shallow-copied before the thinking settings are applied.
func (c *Client) GenerateThinkingText(
	ctx context.Context,
	model string,
	prompt string,
	opts *ThinkingOptions,
	config *genai.GenerateContentConfig,
) (answer string, thoughts []string, resp *genai.GenerateContentResponse, err error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		err = fmt.Errorf("prompt cannot be empty")
		return
	}

	var cfg *genai.GenerateContentConfig
	if config != nil {
		clone := *config
		cfg = &clone
	} else {
		cfg = &genai.GenerateContentConfig{}
	}

	if opts != nil {
		if cfg.ThinkingConfig == nil {
			cfg.ThinkingConfig = &genai.ThinkingConfig{}
		}
		if opts.Budget > 0 {
			b := opts.Budget
			cfg.ThinkingConfig.ThinkingBudget = &b
		} else if opts.Level != "" {
			cfg.ThinkingConfig.ThinkingLevel = mapThinkingLevel(opts.Level)
		}
		cfg.ThinkingConfig.IncludeThoughts = opts.IncludeThoughts
	}

	resp, err = c.GenerateContent(ctx, model, genai.Text(prompt), cfg)
	if err != nil {
		return
	}

	var textBuilder strings.Builder
	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part == nil {
				continue
			}
			content := strings.TrimSpace(part.Text)
			if content == "" {
				continue
			}
			if part.Thought {
				thoughts = append(thoughts, content)
				continue
			}
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n")
			}
			textBuilder.WriteString(content)
		}
		if textBuilder.Len() > 0 || len(thoughts) > 0 {
			break
		}
	}

	if textBuilder.Len() == 0 {
		answer = strings.TrimSpace(resp.Text())
	} else {
		answer = textBuilder.String()
	}
	return
}

// GenerateSystemText sends a prompt while applying a system instruction that defines the assistant persona.
// The provided config is shallow-copied so callers can reuse their template without mutation.
func (c *Client) GenerateSystemText(ctx context.Context, model, prompt string, config *genai.GenerateContentConfig) (string, *genai.GenerateContentResponse, error) {
	if c == nil {
		return "", nil, fmt.Errorf("client cannot be nil")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", nil, fmt.Errorf("prompt cannot be empty")
	}

	var cfg *genai.GenerateContentConfig
	if config != nil {
		clone := *config
		cfg = &clone
	} else {
		cfg = &genai.GenerateContentConfig{}
	}

	if cfg.SystemInstruction == nil {
		return "", nil, fmt.Errorf("system instruction is required")
	}
	if cfg.SystemInstruction.Role == "" {
		cfg.SystemInstruction.Role = genai.RoleUser
	}
	if len(cfg.SystemInstruction.Parts) == 0 {
		return "", nil, fmt.Errorf("system instruction must include at least one part")
	}

	return c.GenerateText(ctx, model, prompt, cfg)
}

// BuildTextContent converts a raw prompt into a Gemini content payload.
func BuildTextContent(role, text string) *genai.Content {
	return &genai.Content{
		Role:  role,
		Parts: []*genai.Part{genai.NewPartFromText(text)},
	}
}
