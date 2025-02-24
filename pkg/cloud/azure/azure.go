package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
)

// AzureOpenAIClient represents a client for interacting with Azure OpenAI API
type AzureOpenAIClient struct {
	client         *azopenai.Client
	deploymentName string
}

// NewAzureOpenAIClient creates a new Azure OpenAI API client
func NewAzureOpenAIClient(endpoint, apiKey, deploymentName string) (*AzureOpenAIClient, error) {
	cred := azcore.NewKeyCredential(apiKey)
	client, err := azopenai.NewClientWithKeyCredential(endpoint, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure OpenAI client: %v", err)
	}

	return &AzureOpenAIClient{
		client:         client,
		deploymentName: deploymentName,
	}, nil
}

// GenerateText generates text using the specified deployment
func (c *AzureOpenAIClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.GetCompletions(ctx, azopenai.CompletionsOptions{
		Prompt:         []string{prompt},
		MaxTokens:      to.Ptr[int32](1000),
		Temperature:    to.Ptr[float32](0.7),
		DeploymentName: &c.deploymentName,
	}, nil)

	if err != nil {
		return "", fmt.Errorf("failed to generate text: %v", err)
	}

	return *resp.Choices[0].Text, nil
}

// GenerateImage generates an image based on the prompt
func (c *AzureOpenAIClient) GenerateImage(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.GetImageGenerations(ctx, azopenai.ImageGenerationOptions{
		Prompt:         to.Ptr(prompt),
		N:              to.Ptr[int32](1),
		Size:           to.Ptr(azopenai.ImageSizeSize1024X1024),
		ResponseFormat: to.Ptr(azopenai.ImageGenerationResponseFormatURL),
		DeploymentName: &c.deploymentName,
	}, nil)

	if err != nil {
		return "", fmt.Errorf("failed to generate image: %v", err)
	}

	return *resp.Data[0].URL, nil
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string
	Content string
}

// Chat conducts a chat conversation
func (c *AzureOpenAIClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	chatMessages := make([]azopenai.ChatRequestMessageClassification, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case "user":
			chatMessages[i] = &azopenai.ChatRequestUserMessage{}
		case "assistant":
			chatMessages[i] = &azopenai.ChatRequestAssistantMessage{}
		case "system":
			chatMessages[i] = &azopenai.ChatRequestSystemMessage{}
		default:
			return "", fmt.Errorf("unsupported role: %s", msg.Role)
		}
	}

	resp, err := c.client.GetChatCompletions(ctx, azopenai.ChatCompletionsOptions{
		Messages:       chatMessages,
		DeploymentName: &c.deploymentName,
		Temperature:    to.Ptr[float32](0.7),
	}, nil)

	if err != nil {
		return "", fmt.Errorf("failed to get chat completion: %v", err)
	}

	return *resp.Choices[0].Message.Content, nil
}

// FineTuneModel represents fine-tuning model configuration
