package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const storyboardPromptTemplateVersion = "storyboard_generation_redesign_v1"

type promptAuditInput struct {
	RunID                 string
	RelatedEntityType     string
	RelatedEntityID       string
	Step                  string
	PromptKind            string
	PromptTemplateVersion string
	AlignmentPrompt       string
	Provider              string
	Model                 string
	Temperature           float64
	MaxTokens             int
	SystemPrompt          string
	UserPrompt            string
	ReferencePreamble     string
	FinalPrompt           string
	ReferenceImageURLs    []string
	Output                string
	TokenUsage            map[string]any
	Metadata              map[string]any
}

func (s *Service) recordPromptAudit(ctx context.Context, in promptAuditInput) string {
	if strings.TrimSpace(in.PromptTemplateVersion) == "" {
		in.PromptTemplateVersion = storyboardPromptTemplateVersion
	}
	tokenUsageJSON := mustJSON(in.TokenUsage, "{}")
	metadataJSON := mustJSON(in.Metadata, "{}")
	fullPrompt := strings.Join([]string{
		in.SystemPrompt,
		in.AlignmentPrompt,
		in.ReferencePreamble,
		in.UserPrompt,
		in.FinalPrompt,
	}, "\n\n")
	record := &domain.AIPromptAuditRecord{
		ID:                    uuid.NewString(),
		RunID:                 in.RunID,
		RelatedEntityType:     in.RelatedEntityType,
		RelatedEntityID:       in.RelatedEntityID,
		Step:                  in.Step,
		PromptKind:            in.PromptKind,
		PromptTemplateVersion: in.PromptTemplateVersion,
		AlignmentSnapshotHash: stableSHA256(in.AlignmentPrompt),
		FullPromptHash:        stableSHA256(fullPrompt),
		Provider:              in.Provider,
		Model:                 in.Model,
		Temperature:           in.Temperature,
		MaxTokens:             in.MaxTokens,
		SystemPrompt:          in.SystemPrompt,
		UserPrompt:            in.UserPrompt,
		AlignmentPrompt:       in.AlignmentPrompt,
		ReferencePreamble:     in.ReferencePreamble,
		FinalPrompt:           in.FinalPrompt,
		ReferenceImageURLs:    in.ReferenceImageURLs,
		Output:                in.Output,
		TokenUsageJSON:        tokenUsageJSON,
		MetadataJSON:          metadataJSON,
		CreatedAt:             time.Now().Unix(),
	}
	if err := s.repo.CreateAIPromptAuditRecord(ctx, record); err != nil {
		s.logger.Warn("failed to create prompt audit record",
			zap.String("runId", in.RunID),
			zap.String("step", in.Step),
			zap.String("promptKind", in.PromptKind),
			zap.Error(err))
		return ""
	}
	return record.ID
}

func stableSHA256(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func mustJSON(v any, fallback string) string {
	if v == nil {
		return fallback
	}
	b, err := json.Marshal(v)
	if err != nil || !json.Valid(b) {
		return fallback
	}
	return string(b)
}
