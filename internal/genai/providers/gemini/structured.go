package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// GenerateStructuredOutput enforces JSON shaped responses using the given schema.
func (c *Client) GenerateStructuredOutput(ctx context.Context, model, prompt string, schema map[string]interface{}) (map[string]interface{}, *genai.GenerateContentResponse, error) {
	if len(schema) == 0 {
		return nil, nil, fmt.Errorf("schema cannot be empty")
	}
	config := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: schema,
	}
	text, resp, err := c.GenerateText(ctx, model, prompt, config)
	if err != nil {
		return nil, resp, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, resp, fmt.Errorf("structured output response was empty")
	}
	var structured map[string]interface{}
	if err := json.Unmarshal([]byte(text), &structured); err != nil {
		return nil, resp, fmt.Errorf("unmarshal structured output: %w", err)
	}
	return structured, resp, nil
}
