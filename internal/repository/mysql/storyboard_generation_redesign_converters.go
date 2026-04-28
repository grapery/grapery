package mysql

import (
	"encoding/json"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func storyboardGenerationRunToModel(d *domain.StoryboardGenerationRun) *StoryboardGenerationRun {
	if d == nil {
		return nil
	}
	var completedAt *time.Time
	if d.CompletedAt != nil {
		t := unixToTime(*d.CompletedAt)
		completedAt = &t
	}
	return &StoryboardGenerationRun{
		ID:                    d.ID,
		StoryboardID:          d.StoryboardID,
		StoryID:               d.StoryID,
		UserID:                d.UserID,
		Status:                d.Status,
		Progress:              d.Progress,
		CurrentStep:           d.CurrentStep,
		RequestJSON:           validJSONOrObject(d.RequestJSON),
		ContextSnapshotJSON:   validJSONOrObject(d.ContextSnapshotJSON),
		AlignmentSnapshotJSON: validJSONOrObject(d.AlignmentSnapshotJSON),
		StoryboardBibleJSON:   validJSONOrObject(d.StoryboardBibleJSON),
		BeatsJSON:             validJSONOrArray(d.BeatsJSON),
		ScenePlanJSON:         validJSONOrObject(d.ScenePlanJSON),
		ConsistencyIssuesJSON: validJSONOrArray(d.ConsistencyIssuesJSON),
		MetricsJSON:           validJSONOrObject(d.MetricsJSON),
		ErrorCode:             d.ErrorCode,
		ErrorMessage:          d.ErrorMessage,
		CreatedAt:             unixToTime(d.CreatedAt),
		UpdatedAt:             unixToTime(d.UpdatedAt),
		CompletedAt:           completedAt,
	}
}

func modelToStoryboardGenerationRun(m *StoryboardGenerationRun) *domain.StoryboardGenerationRun {
	if m == nil {
		return nil
	}
	return &domain.StoryboardGenerationRun{
		ID:                    m.ID,
		StoryboardID:          m.StoryboardID,
		StoryID:               m.StoryID,
		UserID:                m.UserID,
		Status:                m.Status,
		Progress:              m.Progress,
		CurrentStep:           m.CurrentStep,
		RequestJSON:           m.RequestJSON,
		ContextSnapshotJSON:   m.ContextSnapshotJSON,
		AlignmentSnapshotJSON: m.AlignmentSnapshotJSON,
		StoryboardBibleJSON:   m.StoryboardBibleJSON,
		BeatsJSON:             m.BeatsJSON,
		ScenePlanJSON:         m.ScenePlanJSON,
		ConsistencyIssuesJSON: m.ConsistencyIssuesJSON,
		MetricsJSON:           m.MetricsJSON,
		ErrorCode:             m.ErrorCode,
		ErrorMessage:          m.ErrorMessage,
		CreatedAt:             timeToUnix(m.CreatedAt),
		UpdatedAt:             timeToUnix(m.UpdatedAt),
		CompletedAt:           timeToInt64PtrFromPtr(m.CompletedAt),
	}
}

func storyboardGenerationAssetToModel(d *domain.StoryboardGenerationAsset) *StoryboardGenerationAsset {
	if d == nil {
		return nil
	}
	return &StoryboardGenerationAsset{
		ID:           d.ID,
		RunID:        d.RunID,
		Kind:         d.Kind,
		AssetKey:     d.AssetKey,
		EntityID:     d.EntityID,
		ImageURL:     d.ImageURL,
		Source:       d.Source,
		MetadataJSON: validJSONOrObject(d.MetadataJSON),
		CreatedAt:    unixToTime(d.CreatedAt),
	}
}

func modelToStoryboardGenerationAsset(m *StoryboardGenerationAsset) *domain.StoryboardGenerationAsset {
	if m == nil {
		return nil
	}
	return &domain.StoryboardGenerationAsset{
		ID:           m.ID,
		RunID:        m.RunID,
		Kind:         m.Kind,
		AssetKey:     m.AssetKey,
		EntityID:     m.EntityID,
		ImageURL:     m.ImageURL,
		Source:       m.Source,
		MetadataJSON: m.MetadataJSON,
		CreatedAt:    timeToUnix(m.CreatedAt),
	}
}

func aiPromptAuditRecordToModel(d *domain.AIPromptAuditRecord) *AIPromptAuditRecord {
	if d == nil {
		return nil
	}
	refJSON, _ := json.Marshal(d.ReferenceImageURLs)
	return &AIPromptAuditRecord{
		ID:                     d.ID,
		RunID:                  d.RunID,
		RelatedEntityType:      d.RelatedEntityType,
		RelatedEntityID:        d.RelatedEntityID,
		Step:                   d.Step,
		PromptKind:             d.PromptKind,
		PromptTemplateVersion:  d.PromptTemplateVersion,
		AlignmentSnapshotHash:  d.AlignmentSnapshotHash,
		FullPromptHash:         d.FullPromptHash,
		Provider:               d.Provider,
		Model:                  d.Model,
		Temperature:            d.Temperature,
		MaxTokens:              d.MaxTokens,
		SystemPrompt:           d.SystemPrompt,
		UserPrompt:             d.UserPrompt,
		AlignmentPrompt:        d.AlignmentPrompt,
		ReferencePreamble:      d.ReferencePreamble,
		FinalPrompt:            d.FinalPrompt,
		ReferenceImageURLsJSON: string(refJSON),
		Output:                 d.Output,
		TokenUsageJSON:         validJSONOrObject(d.TokenUsageJSON),
		MetadataJSON:           validJSONOrObject(d.MetadataJSON),
		CreatedAt:              unixToTime(d.CreatedAt),
	}
}

func modelToAIPromptAuditRecord(m *AIPromptAuditRecord) *domain.AIPromptAuditRecord {
	if m == nil {
		return nil
	}
	var refs []string
	_ = json.Unmarshal([]byte(m.ReferenceImageURLsJSON), &refs)
	return &domain.AIPromptAuditRecord{
		ID:                    m.ID,
		RunID:                 m.RunID,
		RelatedEntityType:     m.RelatedEntityType,
		RelatedEntityID:       m.RelatedEntityID,
		Step:                  m.Step,
		PromptKind:            m.PromptKind,
		PromptTemplateVersion: m.PromptTemplateVersion,
		AlignmentSnapshotHash: m.AlignmentSnapshotHash,
		FullPromptHash:        m.FullPromptHash,
		Provider:              m.Provider,
		Model:                 m.Model,
		Temperature:           m.Temperature,
		MaxTokens:             m.MaxTokens,
		SystemPrompt:          m.SystemPrompt,
		UserPrompt:            m.UserPrompt,
		AlignmentPrompt:       m.AlignmentPrompt,
		ReferencePreamble:     m.ReferencePreamble,
		FinalPrompt:           m.FinalPrompt,
		ReferenceImageURLs:    refs,
		Output:                m.Output,
		TokenUsageJSON:        m.TokenUsageJSON,
		MetadataJSON:          m.MetadataJSON,
		CreatedAt:             timeToUnix(m.CreatedAt),
	}
}

func validJSONOrObject(s string) string {
	if json.Valid([]byte(s)) {
		return s
	}
	return "{}"
}

func validJSONOrArray(s string) string {
	if json.Valid([]byte(s)) {
		return s
	}
	return "[]"
}

func timeToInt64PtrFromPtr(t *time.Time) *int64 {
	if t == nil || t.IsZero() {
		return nil
	}
	v := t.Unix()
	return &v
}
