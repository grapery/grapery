package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const characterTaskLeaseTTL = 2 * time.Minute

type characterTaskContextKey struct{}

// StartCharacterGenerationRecovery scans durable work at startup and after expired
// leases. Multiple servers may scan safely: only one database claimant can write.
func (s *Service) StartCharacterGenerationRecovery(ctx context.Context) {
	repo, ok := s.repo.(domain.CharacterTaskExecutionRepository)
	if !ok {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		ids, err := repo.RecoverableCharacterTasks(ctx, 20)
		if err != nil && ctx.Err() == nil {
			s.logger.Warn("character recovery scan failed", zap.Error(err))
		}
		for _, id := range ids {
			go s.runCharacterGenerationTask(ctx, id)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) runCharacterGenerationTask(parent context.Context, taskID string) {
	repo, ok := s.repo.(domain.CharacterTaskExecutionRepository)
	if !ok {
		return
	}
	task, err := repo.ClaimCharacterTask(parent, taskID, uuid.NewString(), characterTaskLeaseTTL)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	// The lease descriptor is immutable; the worker owns its mutable progress copy.
	lease := *task
	ctx = context.WithValue(ctx, characterTaskContextKey{}, &lease)
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := repo.RenewCharacterTask(ctx, &lease, characterTaskLeaseTTL); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	if err := s.executeCharacterGenerationTask(ctx, task); err != nil {
		if errors.Is(err, domain.ErrCharacterTaskLeaseLost) || ctx.Err() != nil {
			return
		}
		task.Status = domain.CharacterGenerationStatusFailed
		if task.CharacterID != "" {
			task.Status = domain.CharacterGenerationStatusPartial
		}
		task.ErrorMessage = err.Error()
		now := time.Now().Unix()
		task.CompletedAt = &now
		if err := s.repo.UpdateCharacterGenerationTask(ctx, task); err != nil {
			return
		}
		_ = s.NotifyCharacterGenerationFailed(ctx, task.UserID, task.StoryID, task.ID, task.ErrorMessage)
	}
}

func (s *Service) createCharacterWithExecutionLease(ctx context.Context, character *domain.Character) error {
	if task, ok := ctx.Value(characterTaskContextKey{}).(*domain.CharacterGenerationTask); ok {
		return s.repo.(domain.CharacterTaskExecutionRepository).CreateCharacterForTask(ctx, task, character)
	}
	return s.repo.CreateCharacter(ctx, character)
}

func (s *Service) updateCharacterWithExecutionLease(ctx context.Context, character *domain.Character) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if task, ok := ctx.Value(characterTaskContextKey{}).(*domain.CharacterGenerationTask); ok {
		return s.repo.(domain.CharacterTaskExecutionRepository).UpdateCharacterForTask(ctx, task, character)
	}
	return s.repo.UpdateCharacter(ctx, character)
}

func (s *Service) characterTaskStage(ctx context.Context, task *domain.CharacterGenerationTask, key, status string, cause error, progress int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if task.Stages == nil {
		task.Stages = map[string]domain.CharacterGenerationStage{}
	}
	stage := task.Stages[key]
	stage.Status, stage.Error = status, ""
	if status == "processing" {
		stage.Attempts++
	}
	if cause != nil {
		stage.Error = cause.Error()
	}
	task.Stages[key] = stage
	step := map[string]string{"attributes": domain.CharacterGenerationStepExtract, "character": domain.CharacterGenerationStepCreate, "portrait": domain.CharacterGenerationStepPortrait, "threeViews": domain.CharacterGenerationStepThreeViews}[key]
	task.CurrentStep, task.Progress = step, progress
	return s.repo.UpdateCharacterGenerationTask(ctx, task)
}

func characterRequestAttributes(req CharacterGenerationTaskRequest) GeneratedCharacterAttributes {
	attrs := GeneratedCharacterAttributes{Description: strings.TrimSpace(req.Description), Personality: strings.TrimSpace(req.Personality), Background: strings.TrimSpace(req.Background), Appearance: strings.TrimSpace(req.Appearance), ShortTermGoal: strings.TrimSpace(req.ShortTermGoal), LongTermGoal: strings.TrimSpace(req.LongTermGoal), HandlingStyle: strings.TrimSpace(req.HandlingStyle), CognitionRange: strings.TrimSpace(req.CognitionRange), AbilityFeatures: strings.TrimSpace(req.AbilityFeatures), DressPreference: strings.TrimSpace(req.DressPreference)}
	if req.Suggestion != nil {
		mergeGeneratedCharacterAttributes(&attrs, &GeneratedCharacterAttributes{Description: strings.TrimSpace(req.Suggestion.Description), Appearance: strings.TrimSpace(req.Suggestion.Appearance), Background: strings.TrimSpace(req.Suggestion.Background)})
	}
	return attrs
}

func (s *Service) executeCharacterGenerationTask(ctx context.Context, task *domain.CharacterGenerationTask) error {
	var req CharacterGenerationTaskRequest
	if err := json.Unmarshal([]byte(task.RequestJSON), &req); err != nil {
		return fmt.Errorf("invalid persisted character request: %w", err)
	}
	attrs := characterRequestAttributes(req)
	name := strings.TrimSpace(req.Name)
	if req.Suggestion != nil && (name == "" || legacyFragmentPlaceholderCharacterName(name)) {
		name = strings.TrimSpace(req.Suggestion.Name)
	}
	if name == "" || legacyFragmentPlaceholderCharacterName(name) {
		name = s.resolveFragmentCharacterSuggestionName(ctx, task.SourceFragmentID, task.SourceFragmentCharacterKey)
	}
	if name == "" {
		return errors.New("character name is required")
	}
	ref := strings.TrimSpace(req.ReferenceImage)
	if ref == "" && req.Suggestion != nil {
		ref = strings.TrimSpace(firstNonEmpty(req.Suggestion.ReferenceImage, req.Suggestion.ReferenceImageURL))
	}
	if task.Stages == nil {
		task.Stages = map[string]domain.CharacterGenerationStage{}
	}
	if task.Stages["attributes"].Status == "completed" {
		var saved GeneratedCharacterAttributes
		if json.Unmarshal([]byte(task.AttributesJSON), &saved) == nil {
			mergeGeneratedCharacterAttributes(&attrs, &saved)
		} else {
			return errors.New("saved character attributes are invalid")
		}
	} else {
		if err := s.characterTaskStage(ctx, task, "attributes", "processing", nil, 12); err != nil {
			return err
		}
		generated, genErr := s.GenerateCharacterWithAI(ctx, task.UserID, GenerateCharacterRequest{CreativeControls: req.CreativeControls, WorkflowReleaseID: req.WorkflowReleaseID, StoryID: task.StoryID, Prompt: firstNonEmpty(req.Prompt, "补全故事角色的未设定字段"), Name: name, KnownAttributes: &attrs})
		if genErr == nil {
			mergeGeneratedCharacterAttributes(&attrs, generated)
		}
		status := "completed"
		if genErr != nil {
			status = "failed"
		}
		raw, _ := json.Marshal(attrs)
		task.AttributesJSON = string(raw)
		if err := s.characterTaskStage(ctx, task, "attributes", status, genErr, 30); err != nil {
			return err
		}
		if genErr != nil && strings.TrimSpace(attrs.Description+attrs.Appearance+attrs.Background) == "" {
			return genErr
		}
	}
	var ch *domain.Character
	var err error
	if task.CharacterID != "" {
		ch, err = s.repo.CharacterByID(ctx, task.CharacterID)
	} else {
		if err := s.characterTaskStage(ctx, task, "character", "processing", nil, 35); err != nil {
			return err
		}
		ch, err = s.CreateCharacter(ctx, task.UserID, CreateCharacterRequest{Name: name, StoryID: task.StoryID, Description: attrs.Description, Personality: attrs.Personality, Background: attrs.Background, Appearance: attrs.Appearance, ShortTermGoal: attrs.ShortTermGoal, LongTermGoal: attrs.LongTermGoal, HandlingStyle: attrs.HandlingStyle, CognitionRange: attrs.CognitionRange, AbilityFeatures: attrs.AbilityFeatures, DressPreference: attrs.DressPreference, SourceType: "ai", SourcePrompt: req.Prompt, ReferenceImage: ref, NeedsPortrait: true, SourceFragmentID: task.SourceFragmentID, SourceFragmentCharacterKey: task.SourceFragmentCharacterKey})
	}
	if err != nil {
		_ = s.characterTaskStage(ctx, task, "character", "failed", err, 35)
		return err
	}
	if ch == nil {
		return errors.New("character record unavailable")
	}
	// A retry may successfully fill attributes that failed before record creation.
	// Backfill only empty persisted fields: edits made while the task was partial remain authoritative.
	if task.Stages["attributes"].Status == "completed" {
		existing := GeneratedCharacterAttributes{Description: ch.Description, Personality: ch.Personality, Background: ch.Background, Appearance: ch.Appearance, ShortTermGoal: ch.ShortTermGoal, LongTermGoal: ch.LongTermGoal, HandlingStyle: ch.HandlingStyle, CognitionRange: ch.CognitionRange, AbilityFeatures: ch.AbilityFeatures, DressPreference: ch.DressPreference}
		before := existing
		mergeGeneratedCharacterAttributes(&existing, &attrs)
		if before != existing {
			ch.Description, ch.Personality, ch.Background, ch.Appearance = existing.Description, existing.Personality, existing.Background, existing.Appearance
			ch.ShortTermGoal, ch.LongTermGoal, ch.HandlingStyle, ch.CognitionRange, ch.AbilityFeatures, ch.DressPreference = existing.ShortTermGoal, existing.LongTermGoal, existing.HandlingStyle, existing.CognitionRange, existing.AbilityFeatures, existing.DressPreference
			if err := s.updateCharacterWithExecutionLease(ctx, ch); err != nil {
				return err
			}
		}
	}

	task.CharacterID = ch.ID
	if err := s.characterTaskStage(ctx, task, "character", "completed", nil, 45); err != nil {
		return err
	}
	if strings.TrimSpace(ch.Portrait) == "" {
		if err := s.characterTaskStage(ctx, task, "portrait", "processing", nil, 55); err != nil {
			return err
		}
		if ref != "" {
			ch.Portrait, ch.PortraitGenerationStatus = ref, "generated"
			err = s.saveCharacterVisualFields(ctx, ch, map[string]any{"portrait": ch.Portrait, "portrait_generation_status": "generated"})
		} else {
			_, err = s.GenerateCharacterPortrait(ctx, task.UserID, ch.ID, GenerateCharacterPortraitRequest{})
		}
		if err != nil {
			if saveErr := s.characterTaskStage(ctx, task, "portrait", "failed", err, 65); saveErr != nil {
				return saveErr
			}
		} else if err := s.characterTaskStage(ctx, task, "portrait", "completed", nil, 70); err != nil {
			return err
		}
	} else if err := s.characterTaskStage(ctx, task, "portrait", "completed", nil, 70); err != nil {
		return err
	}
	ch, err = s.repo.CharacterByID(ctx, ch.ID)
	if err != nil {
		return err
	}
	if ch.Views == nil || strings.TrimSpace(ch.Views.Sheet) == "" {
		if strings.TrimSpace(ch.Portrait) == "" && ref == "" {
			if err := s.characterTaskStage(ctx, task, "threeViews", "blocked", errors.New("a confirmed portrait/reference is required before the turnaround"), 78); err != nil {
				return err
			}
		} else {
			if err := s.characterTaskStage(ctx, task, "threeViews", "processing", nil, 78); err != nil {
				return err
			}
			if req.Suggestion != nil && strings.TrimSpace(req.Suggestion.ThreeViewSheetURL) != "" {
				ch.Views = &domain.CharacterThreeViews{Sheet: strings.TrimSpace(req.Suggestion.ThreeViewSheetURL)}
				raw, _ := json.Marshal(ch.Views)
				err = s.saveCharacterVisualFields(ctx, ch, map[string]any{"views_json": string(raw)})
			} else {
				_, err = s.GenerateCharacterThreeViews(ctx, task.UserID, ch.ID, GenerateCharacterThreeViewsRequest{})
			}
			status := "completed"
			if err != nil {
				status = "failed"
			}
			if saveErr := s.characterTaskStage(ctx, task, "threeViews", status, err, 95); saveErr != nil {
				return saveErr
			}
		}
	} else if err := s.characterTaskStage(ctx, task, "threeViews", "completed", nil, 95); err != nil {
		return err
	}
	task.Status = domain.CharacterGenerationStatusCompleted
	var failures []string
	for _, key := range []string{"attributes", "character", "portrait", "threeViews"} {
		st := task.Stages[key]
		if st.Status != "completed" {
			task.Status = domain.CharacterGenerationStatusPartial
			failures = append(failures, key+": "+st.Error)
		}
	}
	task.Progress, task.ErrorMessage = 100, strings.Join(failures, "; ")
	now := time.Now().Unix()
	task.CompletedAt = &now
	raw, _ := json.Marshal(map[string]string{"characterId": ch.ID})
	task.ResultJSON = string(raw)
	if err := s.repo.UpdateCharacterGenerationTask(ctx, task); err != nil {
		return err
	}
	if task.Status == domain.CharacterGenerationStatusCompleted {
		_ = s.NotifyCharacterGenerationComplete(ctx, task.UserID, task.StoryID, ch.ID, name)
	}
	return nil
}

func (s *Service) saveCharacterVisualFields(ctx context.Context, ch *domain.Character, fields map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	lease, _ := ctx.Value(characterTaskContextKey{}).(*domain.CharacterGenerationTask)
	if repo, ok := s.repo.(domain.CharacterVisualRepository); ok {
		return repo.UpdateCharacterVisualAssets(ctx, lease, ch, fields)
	}
	return s.updateCharacterWithExecutionLease(ctx, ch)
}
