package groqapi

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestClient_Chat(t *testing.T) {
	GROQ_API_KEY := os.Getenv("GROQ_API_KEY")
	testClient := NewClient(GROQ_API_KEY,
		http.DefaultClient,
		WithBaseURL("https://api.groq.com"))
	if testClient == nil {
		t.Error("testClient is nil")
	}
	testReq := ChatRequest{
		Messages: []Message{
			{
				Role:    "system",
				Content: "you are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Explain the importance of fast language models",
			},
		},
		Model:     "llama3-8b-8192",
		TopP:      1.0,
		MaxTokens: 500,
		Seed:      int(time.Now().Unix()),
		Stream:    false,
	}
	resp, err := testClient.Chat(testReq)
	if err != nil {
		t.Error(err)
	}
	t.Logf("resp: %v", resp)
}
