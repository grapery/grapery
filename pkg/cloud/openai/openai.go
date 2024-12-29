package openai

import (
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Constants
const (
	APIKey = ""
)

// OpenAIClient wraps the OpenAI API client
type OpenAIClient struct {
	Client *openai.Client
}

// NewOpenAIClient creates a new OpenAI client instance
func NewOpenAIClient() *OpenAIClient {
	opts := []option.RequestOption{
		option.WithAPIKey(APIKey),
	}
	return &OpenAIClient{
		Client: openai.NewClient(opts...),
	}
}
