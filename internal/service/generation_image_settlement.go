package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// A retried operation with a durable result settles that result instead of calling the provider again.
func (s *AIGenerationService) resumeImageSettlement(ctx context.Context, req *GenerateImageRequest, record *domain.AIGenerationRecord) (*GenerateImageResult, error) {
	if !matchesImageOperation(req, record) {
		return nil, &domain.GenerationBillingError{Err: fmt.Errorf("image operation intent conflict")}
	}
	var output struct {
		ImageURLs []string `json:"imageURLs"`
	}
	if err := json.Unmarshal([]byte(record.OutputResult), &output); err != nil || len(output.ImageURLs) == 0 {
		return nil, &domain.GenerationBillingError{Err: fmt.Errorf("image operation already started; no recoverable output yet")}
	}
	tokens := record.TotalTokens
	if tokens <= 0 {
		tokens = len(output.ImageURLs) * common.AIImageBillingUnitTokens
	}
	sponsored, _ := record.Metadata["sponsored"].(bool)
	if sponsored != domain.IsSponsoredGeneration(ctx) {
		return nil, &domain.GenerationBillingError{Err: fmt.Errorf("image sponsorship intent conflict")}
	}
	if !sponsored {
		if _, err := applyGenerationCharge(ctx, s.repo, req.UserID, record.ID, req.RelatedEntityID, "ai_image_generation", tokens); err != nil {
			return nil, &domain.GenerationBillingError{Err: err}
		}
	}
	return &GenerateImageResult{ImageURLs: output.ImageURLs, RecordID: record.ID, TokensUsed: record.TotalTokens}, nil
}

func imageOperationFingerprint(req *GenerateImageRequest) string {
	copy := *req
	copy.OperationID = ""
	copy.Metadata = nil
	raw, _ := json.Marshal(copy)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
func cloneImageGenerationMetadata(input map[string]interface{}) map[string]interface{} {
	output := make(map[string]interface{}, len(input)+4)
	for key, value := range input {
		output[key] = value
	}
	return output
}

func matchesImageOperation(req *GenerateImageRequest, record *domain.AIGenerationRecord) bool {
	return record.UserID == req.UserID && record.OriginalPrompt == req.Prompt && record.RelatedEntityID == req.RelatedEntityID && record.Metadata["operationFingerprint"] == imageOperationFingerprint(req)
}
func hasStoredImageOutput(record *domain.AIGenerationRecord) bool {
	var result struct {
		ImageURLs []string `json:"imageURLs"`
	}
	return json.Unmarshal([]byte(record.OutputResult), &result) == nil && len(result.ImageURLs) > 0
}
