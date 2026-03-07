package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
)

// BuildAIGenerationRecordFromGenAPI builds an AIGenerationRecord from GenAPI request/response/error.
// Used by TokenUsageRecorder to persist image/video generation usage to the database.
// Returns nil if there is no user or entity context to attribute the record to.
func BuildAIGenerationRecordFromGenAPI(ctx context.Context, req *genapi.GenerateRequest, rsp *genapi.GenerateResponse, err error) *domain.AIGenerationRecord {
	userID := ""
	relatedEntityID := ""
	relatedEntityType := ""

	if req != nil {
		if req.UserID != 0 {
			userID = strconv.FormatInt(req.UserID, 10)
		}
		if req.Metadata != nil {
			if v, ok := req.Metadata["related_entity_id"].(string); ok && v != "" {
				relatedEntityID = v
			}
			if v, ok := req.Metadata["related_entity_type"].(string); ok && v != "" {
				relatedEntityType = v
			}
			if userID == "" {
				if v, ok := req.Metadata["user_id"].(string); ok && v != "" {
					userID = v
				}
			}
		}
	}

	// Fallback to context when request is not available (e.g. GetVideoStatus)
	if userID == "" || relatedEntityID == "" {
		if rec := genapi.UsageRecordFromContext(ctx); rec != nil {
			if userID == "" && rec.UserID != "" {
				userID = rec.UserID
			}
			if relatedEntityID == "" && rec.RelatedEntityID != "" {
				relatedEntityID = rec.RelatedEntityID
			}
			if relatedEntityType == "" && rec.RelatedEntityType != "" {
				relatedEntityType = rec.RelatedEntityType
			}
		}
	}

	// Skip recording when we have no attribution context
	if userID == "" && relatedEntityID == "" {
		return nil
	}

	record := &domain.AIGenerationRecord{
		UserID:            userID,
		RelatedEntityID:   relatedEntityID,
		RelatedEntityType: relatedEntityType,
		Provider:          "",
		Model:             "",
		Status:            domain.AITaskStatusCompleted,
		CreatedAt:         time.Now().Unix(),
		OutputResult:      "{}",
	}

	// Media type
	if rsp != nil {
		record.Provider = rsp.Provider
		switch rsp.MediaType {
		case genapi.MediaTypeImage:
			record.Type = "image"
		case genapi.MediaTypeVideo:
			record.Type = "video"
		default:
			record.Type = "image" // fallback
		}

		if rsp.Metadata != nil {
			if m, ok := rsp.Metadata["model"].(string); ok {
				record.Model = m
			}
		}

		if req != nil {
			record.OriginalPrompt = req.Prompt
		}

		if rsp.Usage != nil {
			record.InputTokens = rsp.Usage.InputTokens
			record.OutputTokens = rsp.Usage.OutputTokens
			record.TotalTokens = rsp.Usage.TotalTokens
			record.ImageCount = rsp.Usage.ImageCount
			record.VideoCount = rsp.Usage.VideoCount
		}

		record.DurationMs = rsp.Duration().Milliseconds()
		completedAt := rsp.CompletedAt.Unix()
		record.CompletedAt = &completedAt

		if rsp.Status != "" {
			switch rsp.Status {
			case "completed", "success":
				record.Status = domain.AITaskStatusCompleted
			case "failed", "error":
				record.Status = domain.AITaskStatusFailed
				record.ErrorMessage = rsp.Error
			default:
				record.Status = domain.AITaskStatus(rsp.Status)
			}
		}
	}

	if req != nil {
		record.Model = req.Model
		if record.Model == "" && rsp != nil && rsp.Metadata != nil {
			if m, ok := rsp.Metadata["model"].(string); ok {
				record.Model = m
			}
		}
	}

	if err != nil {
		record.Status = domain.AITaskStatusFailed
		record.ErrorMessage = err.Error()
		if record.TotalTokens == 0 && record.InputTokens == 0 && record.OutputTokens == 0 {
			record.TotalTokens = 0
		}
	}

	// Build input/output JSON
	inputParams := map[string]interface{}{
		"prompt": record.OriginalPrompt,
	}
	if req != nil {
		inputParams["operation"] = string(req.Operation)
		inputParams["aspectRatio"] = req.AspectRatio
	}
	if inputJSON, e := json.Marshal(inputParams); e == nil {
		record.InputParams = string(inputJSON)
	}

	outputResult := map[string]interface{}{
		"status": string(record.Status),
	}
	if rsp != nil {
		if len(rsp.ImageURLs) > 0 {
			outputResult["imageUrls"] = rsp.ImageURLs
		}
		if rsp.VideoURL != "" {
			outputResult["videoUrl"] = rsp.VideoURL
		}
		outputResult["taskId"] = rsp.TaskID
	}
	if outputJSON, e := json.Marshal(outputResult); e == nil {
		record.OutputResult = string(outputJSON)
	}

	return record
}

// NewGenAPIUsageRecorder creates a TokenUsageRecorder that persists to the database.
func NewGenAPIUsageRecorder(repo domain.Repository, onError func(string, error)) genapi.TokenUsageRecorder {
	return genapi.TokenUsageRecorderFunc(func(ctx context.Context, req *genapi.GenerateRequest, rsp *genapi.GenerateResponse, err error) {
		record := BuildAIGenerationRecordFromGenAPI(ctx, req, rsp, err)
		if record == nil {
			return
		}
		if createErr := repo.CreateAIGenerationRecord(ctx, record); createErr != nil {
			msg := fmt.Sprintf("failed to create AI generation record: %v", createErr)
			if onError != nil {
				onError(msg, createErr)
			}
		}
	})
}
