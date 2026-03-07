package huoshan

import (
	"context"
	"fmt"
	"strings"

	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// TextGenerationRequest represents a request for text generation (Responses/Chat API).
type TextGenerationRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	SystemPrompt string  `json:"system_prompt,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature float32  `json:"temperature,omitempty"`
	TopP        float32  `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// TextGenerationResponse captures the text generation result.
type TextGenerationResponse struct {
	Text          string
	Model         string
	InputTokens   int
	OutputTokens  int
	TotalTokens   int
	FinishReason  string
}

// GenerateText performs text generation using the Chat Completions API.
func (c *Client) GenerateText(ctx context.Context, req *TextGenerationRequest) (*TextGenerationResponse, error) {
	if c.arkClient == nil {
		return nil, fmt.Errorf("ark client is not configured")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = strings.TrimSpace(choose(c.config.TextModel, defaultTextModel))
	}
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}

	messages := make([]*arkmodel.ChatCompletionMessage, 0, 3)
	if sysPrompt := strings.TrimSpace(req.SystemPrompt); sysPrompt != "" {
		messages = append(messages, &arkmodel.ChatCompletionMessage{
			Role:    arkmodel.ChatMessageRoleSystem,
			Content: &arkmodel.ChatCompletionMessageContent{StringValue: &sysPrompt},
		})
	}
	messages = append(messages, &arkmodel.ChatCompletionMessage{
		Role:    arkmodel.ChatMessageRoleUser,
		Content: &arkmodel.ChatCompletionMessageContent{StringValue: &prompt},
	})

	chatReq := arkmodel.ChatCompletionRequest{
		Model:    modelName,
		Messages: messages,
	}
	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		chatReq.Temperature = req.Temperature
	}
	if req.TopP > 0 {
		chatReq.TopP = req.TopP
	}
	if len(req.Stop) > 0 {
		chatReq.Stop = req.Stop
	}

	resp, err := c.arkClient.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	result := &TextGenerationResponse{
		Model: resp.Model,
	}
	result.InputTokens = resp.Usage.PromptTokens
	result.OutputTokens = resp.Usage.CompletionTokens
	result.TotalTokens = resp.Usage.TotalTokens
	if result.TotalTokens == 0 {
		result.TotalTokens = result.InputTokens + result.OutputTokens
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message.Content != nil && choice.Message.Content.StringValue != nil {
			result.Text = *choice.Message.Content.StringValue
		}
		result.FinishReason = string(choice.FinishReason)
	}

	return result, nil
}
