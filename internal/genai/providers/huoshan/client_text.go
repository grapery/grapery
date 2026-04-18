package huoshan

import (
	"context"
	"fmt"
	"strings"

	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// TextGenerationRequest represents a request for text generation (Responses/Chat API).
type TextGenerationRequest struct {
	Model        string   `json:"model"`
	Prompt       string   `json:"prompt"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
	Temperature  float32  `json:"temperature,omitempty"`
	TopP         float32  `json:"top_p,omitempty"`
	Stop         []string `json:"stop,omitempty"`
	// ImageURLs 参考图公网 URL（多模态）；非空时使用 vision 对话，无需先下载图片。
	ImageURLs []string `json:"image_urls,omitempty"`
	// JSONResponse 为 true 时请求模型输出 JSON 对象（方舟 response_format）。
	JSONResponse bool `json:"json_response,omitempty"`
}

// TextGenerationResponse captures the text generation result.
type TextGenerationResponse struct {
	Text         string
	Model        string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	FinishReason string
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

	useModern := len(req.ImageURLs) > 0 || req.JSONResponse
	if useModern {
		return c.generateTextModern(ctx, modelName, req, prompt)
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

	return chatResponseToResult(resp)
}

func (c *Client) generateTextModern(ctx context.Context, modelName string, req *TextGenerationRequest, prompt string) (*TextGenerationResponse, error) {
	messages := make([]*arkmodel.ChatCompletionMessage, 0, 3)
	if sysPrompt := strings.TrimSpace(req.SystemPrompt); sysPrompt != "" {
		messages = append(messages, &arkmodel.ChatCompletionMessage{
			Role:    arkmodel.ChatMessageRoleSystem,
			Content: &arkmodel.ChatCompletionMessageContent{StringValue: &sysPrompt},
		})
	}

	var userContent *arkmodel.ChatCompletionMessageContent
	if len(req.ImageURLs) > 0 {
		parts := make([]*arkmodel.ChatCompletionMessageContentPart, 0, len(req.ImageURLs)+2)
		parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
			Type: arkmodel.ChatCompletionMessageContentPartTypeText,
			Text: "User reference image(s) for this task:",
		})
		for _, raw := range req.ImageURLs {
			u := strings.TrimSpace(raw)
			if u == "" {
				continue
			}
			urlCopy := u
			parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
				Type:     arkmodel.ChatCompletionMessageContentPartTypeImageURL,
				ImageURL: &arkmodel.ChatMessageImageURL{URL: urlCopy},
			})
		}
		parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
			Type: arkmodel.ChatCompletionMessageContentPartTypeText,
			Text: prompt,
		})
		userContent = &arkmodel.ChatCompletionMessageContent{ListValue: parts}
	} else {
		userContent = &arkmodel.ChatCompletionMessageContent{StringValue: &prompt}
	}

	messages = append(messages, &arkmodel.ChatCompletionMessage{
		Role:    arkmodel.ChatMessageRoleUser,
		Content: userContent,
	})

	chatReq := arkmodel.CreateChatCompletionRequest{
		Model:    modelName,
		Messages: messages,
	}
	if req.MaxTokens > 0 {
		mt := req.MaxTokens
		chatReq.MaxTokens = &mt
	}
	if req.Temperature > 0 {
		t := req.Temperature
		chatReq.Temperature = &t
	}
	if req.TopP > 0 {
		tp := req.TopP
		chatReq.TopP = &tp
	}
	if len(req.Stop) > 0 {
		chatReq.Stop = req.Stop
	}
	if req.JSONResponse {
		rf := arkmodel.ResponseFormat{Type: arkmodel.ResponseFormatJsonObject}
		chatReq.ResponseFormat = &rf
	}

	resp, err := c.arkClient.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, err
	}
	return chatResponseToResult(resp)
}

func chatResponseToResult(resp arkmodel.ChatCompletionResponse) (*TextGenerationResponse, error) {
	result := &TextGenerationResponse{
		Model: resp.Model,
	}
	u := resp.Usage
	result.InputTokens = u.PromptTokens
	result.OutputTokens = u.CompletionTokens
	result.TotalTokens = u.TotalTokens
	if result.TotalTokens == 0 {
		result.TotalTokens = result.InputTokens + result.OutputTokens
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		result.Text = extractAssistantText(choice.Message)
		result.FinishReason = string(choice.FinishReason)
	}

	return result, nil
}

func extractAssistantText(msg arkmodel.ChatCompletionMessage) string {
	if msg.Content == nil {
		return ""
	}
	if msg.Content.StringValue != nil {
		return strings.TrimSpace(*msg.Content.StringValue)
	}
	if len(msg.Content.ListValue) == 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range msg.Content.ListValue {
		if p == nil {
			continue
		}
		if p.Type == arkmodel.ChatCompletionMessageContentPartTypeText && strings.TrimSpace(p.Text) != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(strings.TrimSpace(p.Text))
		}
	}
	return strings.TrimSpace(b.String())
}
