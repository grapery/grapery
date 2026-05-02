package service

import (
	"context"
	"fmt"
	"strings"

	huoshanclient "github.com/grapestree/fgrapery/grapery/internal/genai/providers/huoshan"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

// storyboardLLMTextHuoshanThenGemini 故事板文本步骤：优先火山方舟对话，失败或空结果再 Gemini（与碎片链路一致）。
func (s *Service) storyboardLLMTextHuoshanThenGemini(
	ctx context.Context,
	prompt string,
	step string,
	hMaxTokens int,
	hTemp float32,
	hJSON bool,
	gTemp float32,
	gMaxOut int32,
) (text string, inTok, outTok, totTok int, provider string, err error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", 0, 0, 0, "", fmt.Errorf("empty prompt")
	}

	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	if huoshanOK {
		hc := s.genAPI.HuoshanInternalClient()
		resp, hsErr := hc.GenerateText(ctx, &huoshanclient.TextGenerationRequest{
			Prompt:       prompt,
			MaxTokens:    hMaxTokens,
			Temperature:  hTemp,
			JSONResponse: hJSON,
		})
		if hsErr == nil && resp != nil && strings.TrimSpace(resp.Text) != "" {
			inTok = resp.InputTokens
			outTok = resp.OutputTokens
			totTok = resp.TotalTokens
			if totTok == 0 {
				totTok = inTok + outTok
			}
			return strings.TrimSpace(resp.Text), inTok, outTok, totTok, "huoshan", nil
		}
		if hsErr != nil {
			s.logger.Warn("storyboard text LLM: huoshan failed, falling back to gemini",
				zap.String("step", step),
				zap.Error(hsErr))
		} else {
			s.logger.Warn("storyboard text LLM: huoshan returned empty, falling back to gemini",
				zap.String("step", step))
		}
	}

	if s.geminiClient == nil {
		if !huoshanOK {
			return "", 0, 0, 0, "", fmt.Errorf("no text LLM (configure HUOSHAN_API_KEY or GEMINI_API_KEY)")
		}
		return "", 0, 0, 0, "", fmt.Errorf("huoshan text failed and gemini is not configured")
	}

	cfg := &genai.GenerateContentConfig{
		Temperature:     &gTemp,
		MaxOutputTokens: gMaxOut,
	}
	gText, gemResp, gErr := s.geminiClient.GenerateText(ctx, "", prompt, cfg)
	if gErr != nil {
		return "", 0, 0, 0, "", gErr
	}
	gText = strings.TrimSpace(gText)
	if gemResp != nil && gemResp.UsageMetadata != nil {
		inTok = int(gemResp.UsageMetadata.PromptTokenCount)
		outTok = int(gemResp.UsageMetadata.CandidatesTokenCount)
		totTok = int(gemResp.UsageMetadata.TotalTokenCount)
	}
	return gText, inTok, outTok, totTok, "gemini", nil
}
