package google

import (
	"context"
	"fmt"

	genai "github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiClient represents a client for interacting with Google Cloud Gemini API
type GeminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(ctx context.Context, apiKey string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %v", err)
	}

	// Initialize with default model
	model := client.GenerativeModel("gemini-pro")

	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

// GenerateText generates text response based on prompt
func (c *GeminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	return string(resp.Candidates[0].Content.Parts[0].(genai.Text)), nil
}

// GenerateImage generates an image based on text prompt using Gemini Pro Vision
func (c *GeminiClient) GenerateImage(ctx context.Context, prompt string) ([]byte, error) {
	// Switch to vision model
	visionModel := c.client.GenerativeModel("gemini-pro-vision")

	resp, err := visionModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %v", err)
	}

	// Extract image data from response
	// Note: Implementation depends on the actual response format
	return resp.Candidates[0].Content.Parts[0].(genai.Blob).Data, nil
}

// Chat initiates a chat session with Gemini
func (c *GeminiClient) Chat(ctx context.Context) (*genai.ChatSession, error) {
	chat := c.model.StartChat()
	return chat, nil
}

// SendMessage sends a message in a chat session and returns the response
func (c *GeminiClient) SendMessage(ctx context.Context, chat *genai.ChatSession, message string) (string, error) {
	resp, err := chat.SendMessage(ctx, genai.Text(message))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}

	return string(resp.Candidates[0].Content.Parts[0].(genai.Text)), nil
}

// Close closes the Gemini client
func (c *GeminiClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
