package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

const (
	StoryboardWorkflowStageBiblePlan      = "bible_plan"
	StoryboardWorkflowStageScenePlan      = "scene_plan"
	StoryboardWorkflowStagePersistContent = "persist_content"
)

// StoryboardWorkflowStageResult is a small, checkpoint-friendly result. The
// generated plans stay server-side in StoryboardGenerationRun.
type StoryboardWorkflowStageResult struct {
	StoryboardID    string `json:"storyboardId"`
	GenerationRunID string `json:"generationRunId"`
	Stage           string `json:"stage"`
	Progress        int    `json:"progress"`
	AlreadyComplete bool   `json:"alreadyComplete"`
}

type StoryboardWorkflowStageOptions struct {
	GenerationRunID     string
	ClientRequestID     string
	RegenerateStructure bool
	UserDirective       string
	SceneCount          int
	ComicStyle          string
	Language            string
}

// ExecuteStoryboardWorkflowStage executes exactly one durable text stage. A
// completed stage is returned from its persisted run, so agent retries do not
// call the model or insert scenes twice.
func (s *Service) ExecuteStoryboardWorkflowStage(ctx context.Context, userID, storyboardID, stage string, opts StoryboardWorkflowStageOptions) (*StoryboardWorkflowStageResult, error) {
	stage = strings.TrimSpace(stage)
	switch stage {
	case StoryboardWorkflowStageBiblePlan, StoryboardWorkflowStageScenePlan, StoryboardWorkflowStagePersistContent:
	default:
		return nil, fmt.Errorf("unsupported storyboard workflow stage %q", stage)
	}
	mu := s.structureResumeMutex(storyboardID)
	if !mu.TryLock() {
		return nil, fmt.Errorf("storyboard workflow stage already in progress")
	}
	defer mu.Unlock()

	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}
	if storyboard.UserID != userID {
		return nil, fmt.Errorf("permission denied: not the creator")
	}
	if opts.RegenerateStructure {
		if stage != StoryboardWorkflowStageBiblePlan {
			return nil, fmt.Errorf("storyboard revision options are only accepted by bible_plan")
		}
		opts.ClientRequestID = strings.TrimSpace(opts.ClientRequestID)
		opts.UserDirective = strings.TrimSpace(opts.UserDirective)
		if opts.UserDirective == "" {
			return nil, fmt.Errorf("storyboard revision requires a user directive")
		}
		if storyboard.WorkflowStatus == domain.WorkflowStatusPublished {
			return nil, fmt.Errorf("published storyboard cannot be revised")
		}
		if err := s.applyStoryboardTurnOptions(ctx, storyboard, StoryboardStructureGenerationOptions{SceneCount: opts.SceneCount, ComicStyle: opts.ComicStyle}); err != nil {
			return nil, err
		}
		storyboard.TurnDirective = opts.UserDirective
	}
	if !s.canGenerateStoryboardText() && stage != StoryboardWorkflowStagePersistContent {
		return nil, fmt.Errorf("storyboard AI generation is not configured")
	}
	story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
	if err != nil {
		return nil, fmt.Errorf("load storyboard story: %w", err)
	}

	run, snapshot, alignmentPrompt, sceneCount, err := s.ensureStoryboardWorkflowRun(ctx, storyboard, story, opts)
	if err != nil {
		return nil, err
	}
	switch stage {
	case StoryboardWorkflowStageBiblePlan:
		return s.ensureStoryboardBibleStage(ctx, storyboard, story, run, snapshot, alignmentPrompt, sceneCount)
	case StoryboardWorkflowStageScenePlan:
		return s.ensureStoryboardScenePlanStage(ctx, storyboard, story, run, snapshot, alignmentPrompt, sceneCount)
	default:
		return s.ensureStoryboardPersistContentStage(ctx, storyboard, run, sceneCount)
	}
}

func (s *Service) ensureStoryboardWorkflowRun(ctx context.Context, storyboard *domain.Storyboard, story *domain.Story, opts StoryboardWorkflowStageOptions) (*domain.StoryboardGenerationRun, storyboardGenerationContextSnapshot, string, int, error) {
	sceneCount := storyboard.SceneCount
	if sceneCount < MinStoryboardSceneCount || sceneCount > MaxStoryboardSceneCount {
		sceneCount = DefaultStoryboardSceneCount
	}
	if runID := strings.TrimSpace(opts.GenerationRunID); runID != "" {
		existing, err := s.repo.GetStoryboardGenerationRun(ctx, runID)
		if err != nil {
			return nil, storyboardGenerationContextSnapshot{}, "", 0, fmt.Errorf("load pinned storyboard generation run: %w", err)
		}
		if existing.StoryboardID != storyboard.ID || existing.UserID != storyboard.UserID {
			return nil, storyboardGenerationContextSnapshot{}, "", 0, fmt.Errorf("generation run does not belong to storyboard")
		}
		return restoreStoryboardWorkflowRun(existing, sceneCount)
	}
	if existing, err := s.repo.LatestStoryboardGenerationRun(ctx, storyboard.ID); err == nil && existing != nil && storyboardWorkflowRunMatchesRequest(existing, opts) {
		return restoreStoryboardWorkflowRun(existing, sceneCount)
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, storyboardGenerationContextSnapshot{}, "", 0, fmt.Errorf("load latest storyboard generation run: %w", err)
	}

	continuation := s.isStoryboardContinuation(storyboard)
	alignmentJSON, alignmentPrompt := s.storyboardAlignmentSnapshot(ctx, story, storyboard)
	snapshot := s.buildStoryboardGenerationContextSnapshot(ctx, storyboard, story, continuation)
	now := time.Now().Unix()
	run := &domain.StoryboardGenerationRun{
		ID: uuid.NewString(), StoryboardID: storyboard.ID, StoryID: storyboard.StoryID, UserID: storyboard.UserID,
		Status: domain.GenerationStatusProcessing, Progress: 5, CurrentStep: domain.StoryboardGenerationStepContext,
		RequestJSON: mustJSON(map[string]any{
			"rawInput": storyboard.RawInput, "sceneCount": sceneCount, "continuation": continuation,
			"comicStyle": storyboard.ContinuationComicStyle, "clientRequestId": strings.TrimSpace(opts.ClientRequestID),
			"regenerateStructure": opts.RegenerateStructure, "userDirective": strings.TrimSpace(opts.UserDirective),
			"language": strings.TrimSpace(opts.Language),
		}, "{}"),
		ContextSnapshotJSON: mustJSON(snapshot, "{}"), AlignmentSnapshotJSON: alignmentJSON, MetricsJSON: "{}",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateStoryboardGenerationRun(ctx, run); err != nil {
		return nil, snapshot, "", 0, fmt.Errorf("create storyboard generation run: %w", err)
	}
	s.recordPromptAudit(ctx, promptAuditInput{
		RunID: run.ID, RelatedEntityType: "storyboard", RelatedEntityID: storyboard.ID,
		Step: domain.StoryboardGenerationStepContext, PromptKind: domain.PromptKindContext,
		PromptTemplateVersion: storyboardPromptTemplateVersion, AlignmentPrompt: alignmentPrompt,
		UserPrompt: snapshot.ContextText, Provider: "internal", Model: "context-assembly", Output: run.ContextSnapshotJSON,
	})
	if err := s.repo.CreateStoryboardGenerationAssets(ctx, s.storyboardGenerationAssetsFromContext(run.ID, snapshot)); err != nil {
		return nil, snapshot, "", 0, fmt.Errorf("persist storyboard generation assets: %w", err)
	}
	return run, snapshot, alignmentPrompt, sceneCount, nil
}

func restoreStoryboardWorkflowRun(existing *domain.StoryboardGenerationRun, sceneCount int) (*domain.StoryboardGenerationRun, storyboardGenerationContextSnapshot, string, int, error) {
	var snapshot storyboardGenerationContextSnapshot
	if err := json.Unmarshal([]byte(existing.ContextSnapshotJSON), &snapshot); err != nil {
		return nil, snapshot, "", 0, fmt.Errorf("decode storyboard generation context: %w", err)
	}
	alignmentPrompt := "Human-value and user-alignment snapshot:\n" + existing.AlignmentSnapshotJSON + "\nRespect the user's selected values and safety preferences without adding moralizing exposition."
	return existing, snapshot, alignmentPrompt, sceneCount, nil
}

func (s *Service) ensureStoryboardBibleStage(ctx context.Context, storyboard *domain.Storyboard, story *domain.Story, run *domain.StoryboardGenerationRun, snapshot storyboardGenerationContextSnapshot, alignmentPrompt string, sceneCount int) (*StoryboardWorkflowStageResult, error) {
	if strings.TrimSpace(run.StoryboardBibleJSON) != "" && strings.TrimSpace(run.BeatsJSON) != "" {
		return storyboardStageResult(storyboard.ID, run, StoryboardWorkflowStageBiblePlan, true), nil
	}
	run.Status, run.CurrentStep, run.Progress = domain.GenerationStatusProcessing, domain.StoryboardGenerationStepBiblePlan, 10
	run.ErrorCode, run.ErrorMessage, run.UpdatedAt = "", "", time.Now().Unix()
	_ = s.repo.UpdateStoryboardGenerationRun(ctx, run)
	s.markLatestContentGenerationProcessing(ctx, storyboard.ID)
	plan, text, tokens, err := s.generateStoryboardBiblePlan(ctx, run, story, storyboard, snapshot, alignmentPrompt, sceneCount)
	if err != nil {
		s.failStoryboardGenerationRun(ctx, run, "bible_plan_failed", err)
		s.failLatestContentGeneration(ctx, storyboard.ID, err.Error())
		return nil, err
	}
	run.StoryboardBibleJSON = mustJSON(plan.StoryboardBible, "{}")
	run.BeatsJSON = mustJSON(plan.Beats, "[]")
	run.Progress, run.CurrentStep, run.UpdatedAt = 35, domain.StoryboardGenerationStepScenePlan, time.Now().Unix()
	run.MetricsJSON = mergeStoryboardRunMetrics(run.MetricsJSON, map[string]any{"biblePlanTokens": tokens, "biblePlanChars": len(text)})
	if err := s.repo.UpdateStoryboardGenerationRun(ctx, run); err != nil {
		return nil, fmt.Errorf("persist storyboard bible plan: %w", err)
	}
	return storyboardStageResult(storyboard.ID, run, StoryboardWorkflowStageBiblePlan, false), nil
}

func (s *Service) ensureStoryboardScenePlanStage(ctx context.Context, storyboard *domain.Storyboard, story *domain.Story, run *domain.StoryboardGenerationRun, snapshot storyboardGenerationContextSnapshot, alignmentPrompt string, sceneCount int) (*StoryboardWorkflowStageResult, error) {
	if strings.TrimSpace(run.ScenePlanJSON) != "" {
		return storyboardStageResult(storyboard.ID, run, StoryboardWorkflowStageScenePlan, true), nil
	}
	plan, err := storyboardBiblePlanFromRun(run)
	if err != nil {
		return nil, fmt.Errorf("bible_plan stage is incomplete: %w", err)
	}
	run.Status, run.CurrentStep, run.Progress = domain.GenerationStatusProcessing, domain.StoryboardGenerationStepScenePlan, 40
	run.ErrorCode, run.ErrorMessage, run.UpdatedAt = "", "", time.Now().Unix()
	_ = s.repo.UpdateStoryboardGenerationRun(ctx, run)
	scenePlan, text, tokens, err := s.generateStoryboardScenePlan(ctx, run, story, storyboard, snapshot, plan, alignmentPrompt, sceneCount)
	if err != nil {
		s.failStoryboardGenerationRun(ctx, run, "scene_plan_failed", err)
		s.failLatestContentGeneration(ctx, storyboard.ID, err.Error())
		return nil, err
	}
	run.ScenePlanJSON = mustJSON(scenePlan, "{}")
	run.Progress, run.CurrentStep, run.UpdatedAt = 65, domain.StoryboardGenerationStepImage, time.Now().Unix()
	run.MetricsJSON = mergeStoryboardRunMetrics(run.MetricsJSON, map[string]any{"scenePlanTokens": tokens, "scenePlanChars": len(text)})
	if err := s.repo.UpdateStoryboardGenerationRun(ctx, run); err != nil {
		return nil, fmt.Errorf("persist storyboard scene plan: %w", err)
	}
	return storyboardStageResult(storyboard.ID, run, StoryboardWorkflowStageScenePlan, false), nil
}

func (s *Service) ensureStoryboardPersistContentStage(ctx context.Context, storyboard *domain.Storyboard, run *domain.StoryboardGenerationRun, sceneCount int) (*StoryboardWorkflowStageResult, error) {
	plan, err := storyboardBiblePlanFromRun(run)
	if err != nil {
		return nil, fmt.Errorf("bible_plan stage is incomplete: %w", err)
	}
	var scenePlan domain.StoryboardScenePlan
	if err := json.Unmarshal([]byte(run.ScenePlanJSON), &scenePlan); err != nil || len(scenePlan.Scenes) == 0 {
		return nil, fmt.Errorf("scene_plan stage is incomplete")
	}
	assets, _ := s.repo.ListStoryboardGenerationAssets(ctx, run.ID)
	issues := s.auditStoryboardGenerationConsistency(plan, &scenePlan, assets, sceneCount, s.isStoryboardContinuation(storyboard))
	run.ConsistencyIssuesJSON = mustJSON(issues, "[]")
	if detail := firstHighStoryboardConsistencyIssue(issues); detail != "" {
		err := fmt.Errorf("story framework consistency check failed: %s", detail)
		s.failStoryboardGenerationRun(ctx, run, "story_framework_conflict", err)
		return nil, err
	}

	existing, err := s.repo.StoryboardScenes(ctx, storyboard.ID)
	if err != nil {
		return nil, fmt.Errorf("load storyboard scenes: %w", err)
	}
	already := len(existing) > 0
	revision := storyboardWorkflowRunIsRevision(run)
	for _, scene := range existing {
		if scene == nil || scene.GenerationRunID != run.ID {
			if !revision {
				return nil, fmt.Errorf("storyboard already contains scenes from another generation run")
			}
			if err := s.repo.DeleteStoryboardScenes(ctx, storyboard.ID); err != nil {
				return nil, fmt.Errorf("replace storyboard scenes for revision: %w", err)
			}
			existing = nil
			already = false
			break
		}
	}
	if !already {
		storyboard.Content = strings.TrimSpace(scenePlan.Content)
		if utf8.RuneCountInString(storyboard.Content) > storyboardContentMaxRunes {
			storyboard.Content = truncateStoryboardContentToMaxRunes(storyboard.Content, storyboardContentMaxRunes)
		}
		storyboard.IsAIGenerated = true
		storyboard.StoryboardScenes = s.storyboardScenesFromScenePlan(run.ID, &scenePlan)
		if err := s.repo.UpdateStoryboard(ctx, storyboard); err != nil {
			return nil, fmt.Errorf("persist generated storyboard content: %w", err)
		}
		if err := s.persistStoryboardScenes(ctx, storyboard); err != nil {
			return nil, err
		}
	}
	run.Progress, run.CurrentStep, run.Status = 100, domain.StoryboardGenerationStepConsistency, domain.GenerationStatusCompleted
	done := time.Now().Unix()
	run.CompletedAt, run.UpdatedAt, run.ErrorCode, run.ErrorMessage = &done, done, "", ""
	if err := s.repo.UpdateStoryboardGenerationRun(ctx, run); err != nil {
		return nil, fmt.Errorf("complete storyboard generation run: %w", err)
	}
	s.syncLatestContentGenerationCompleted(ctx, storyboard.ID, storyboard.Content)
	_ = s.repo.UpdateStoryboardWorkflow(ctx, storyboard.ID, domain.WorkflowStatusContentReady, 2)
	s.generateOrRefreshStoryboardSummary(ctx, storyboard.ID)
	return storyboardStageResult(storyboard.ID, run, StoryboardWorkflowStagePersistContent, already), nil
}

type storyboardWorkflowRunRequest struct {
	ClientRequestID     string `json:"clientRequestId"`
	RegenerateStructure bool   `json:"regenerateStructure"`
	UserDirective       string `json:"userDirective"`
	Language            string `json:"language"`
}

func storyboardWorkflowOutputLanguage(run *domain.StoryboardGenerationRun) string {
	language := strings.TrimSpace(storyboardWorkflowRunRequestFromRun(run).Language)
	if language == "" {
		return "zh-Hans"
	}
	return language
}

func storyboardWorkflowRunRequestFromRun(run *domain.StoryboardGenerationRun) storyboardWorkflowRunRequest {
	var request storyboardWorkflowRunRequest
	if run != nil {
		_ = json.Unmarshal([]byte(run.RequestJSON), &request)
	}
	return request
}

func storyboardWorkflowRunMatchesRequest(run *domain.StoryboardGenerationRun, opts StoryboardWorkflowStageOptions) bool {
	if !opts.RegenerateStructure {
		return true
	}
	request := storyboardWorkflowRunRequestFromRun(run)
	if !request.RegenerateStructure {
		return false
	}
	if id := strings.TrimSpace(opts.ClientRequestID); id != "" {
		return strings.TrimSpace(request.ClientRequestID) == id
	}
	return strings.TrimSpace(request.UserDirective) == strings.TrimSpace(opts.UserDirective)
}

func storyboardWorkflowRunIsRevision(run *domain.StoryboardGenerationRun) bool {
	return storyboardWorkflowRunRequestFromRun(run).RegenerateStructure
}

func storyboardBiblePlanFromRun(run *domain.StoryboardGenerationRun) (*domain.StoryboardBiblePlan, error) {
	var plan domain.StoryboardBiblePlan
	if err := json.Unmarshal([]byte(run.StoryboardBibleJSON), &plan.StoryboardBible); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(run.BeatsJSON), &plan.Beats); err != nil || len(plan.Beats) == 0 {
		return nil, fmt.Errorf("missing beats")
	}
	return &plan, nil
}

func storyboardStageResult(storyboardID string, run *domain.StoryboardGenerationRun, stage string, already bool) *StoryboardWorkflowStageResult {
	return &StoryboardWorkflowStageResult{StoryboardID: storyboardID, GenerationRunID: run.ID, Stage: stage, Progress: run.Progress, AlreadyComplete: already}
}

func mergeStoryboardRunMetrics(raw string, updates map[string]any) string {
	metrics := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &metrics)
	for key, value := range updates {
		metrics[key] = value
	}
	return mustJSON(metrics, "{}")
}

func (s *Service) markLatestContentGenerationProcessing(ctx context.Context, storyboardID string) {
	gen, err := s.repo.GetContentGenerationByStoryboard(ctx, storyboardID)
	if err != nil || gen == nil {
		return
	}
	gen.Status, gen.ErrorMessage = domain.GenerationStatusProcessing, ""
	_ = s.repo.UpdateContentGeneration(ctx, gen)
}
