package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

type storyboardGenerationContextSnapshot struct {
	StoryID      string                         `json:"storyId"`
	StoryTitle   string                         `json:"storyTitle"`
	StoryGenre   string                         `json:"storyGenre,omitempty"`
	StoryStyle   any                            `json:"storyStyle,omitempty"`
	RawInput     string                         `json:"rawInput"`
	Continuation bool                           `json:"continuation"`
	IsStandalone bool                           `json:"isStandalone"`
	ContextText  string                         `json:"contextText"`
	Characters   []storyboardContextCharacter   `json:"characters,omitempty"`
	Scenes       []storyboardContextScene       `json:"scenes,omitempty"`
	ParentTail   []storyboardContextParentScene `json:"parentTailScenes,omitempty"`
}

type storyboardContextCharacter struct {
	ID          string                      `json:"id"`
	Key         string                      `json:"key"`
	Name        string                      `json:"name"`
	Role        string                      `json:"role,omitempty"`
	Description string                      `json:"description,omitempty"`
	Personality string                      `json:"personality,omitempty"`
	Appearance  string                      `json:"appearance,omitempty"`
	Views       *domain.CharacterThreeViews `json:"turnaroundImages,omitempty"`
}

type storyboardContextScene struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	TimeOfDay   string `json:"timeOfDay,omitempty"`
}

type storyboardContextParentScene struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
}

func (s *Service) generateStoryboardWithRedesignPipeline(ctx context.Context, storyboard *domain.Storyboard, story *domain.Story) error {
	if s.aiGenService == nil {
		return fmt.Errorf("AI generation service not configured")
	}
	sceneCount := storyboard.SceneCount
	if sceneCount < MinStoryboardSceneCount || sceneCount > MaxStoryboardSceneCount {
		sceneCount = DefaultStoryboardSceneCount
	}
	continuation := s.isStoryboardContinuation(storyboard)
	alignmentJSON, alignmentPrompt := s.storyboardAlignmentSnapshot(ctx, story, storyboard)
	contextSnapshot := s.buildStoryboardGenerationContextSnapshot(ctx, storyboard, story, continuation)
	contextJSON := mustJSON(contextSnapshot, "{}")
	now := time.Now().Unix()
	run := &domain.StoryboardGenerationRun{
		ID:                    uuid.NewString(),
		StoryboardID:          storyboard.ID,
		StoryID:               storyboard.StoryID,
		UserID:                storyboard.UserID,
		Status:                domain.GenerationStatusProcessing,
		Progress:              5,
		CurrentStep:           domain.StoryboardGenerationStepContext,
		RequestJSON:           mustJSON(map[string]any{"rawInput": storyboard.RawInput, "sceneCount": sceneCount, "continuation": continuation, "comicStyle": storyboard.ContinuationComicStyle}, "{}"),
		ContextSnapshotJSON:   contextJSON,
		AlignmentSnapshotJSON: alignmentJSON,
		MetricsJSON:           "{}",
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := s.repo.CreateStoryboardGenerationRun(ctx, run); err != nil {
		return fmt.Errorf("create storyboard generation run: %w", err)
	}
	s.recordPromptAudit(ctx, promptAuditInput{
		RunID:                 run.ID,
		RelatedEntityType:     "storyboard",
		RelatedEntityID:       storyboard.ID,
		Step:                  domain.StoryboardGenerationStepContext,
		PromptKind:            domain.PromptKindContext,
		PromptTemplateVersion: storyboardPromptTemplateVersion,
		AlignmentPrompt:       alignmentPrompt,
		UserPrompt:            contextSnapshot.ContextText,
		Provider:              "internal",
		Model:                 "context-assembly",
		Output:                contextJSON,
	})

	assets := s.storyboardGenerationAssetsFromContext(run.ID, contextSnapshot)
	if err := s.repo.CreateStoryboardGenerationAssets(ctx, assets); err != nil {
		return fmt.Errorf("persist storyboard generation assets: %w", err)
	}

	plan, planText, planTokens, err := s.generateStoryboardBiblePlan(ctx, run, story, storyboard, contextSnapshot, alignmentPrompt, sceneCount)
	if err != nil {
		s.failStoryboardGenerationRun(ctx, run, "bible_plan_failed", err)
		return err
	}
	run.StoryboardBibleJSON = mustJSON(plan.StoryboardBible, "{}")
	run.BeatsJSON = mustJSON(plan.Beats, "[]")
	run.Progress = 35
	run.CurrentStep = domain.StoryboardGenerationStepScenePlan
	run.UpdatedAt = time.Now().Unix()
	_ = s.repo.UpdateStoryboardGenerationRun(ctx, run)

	scenePlan, sceneText, sceneTokens, err := s.generateStoryboardScenePlan(ctx, run, story, storyboard, contextSnapshot, plan, alignmentPrompt, sceneCount)
	if err != nil {
		s.failStoryboardGenerationRun(ctx, run, "scene_plan_failed", err)
		return err
	}
	run.ScenePlanJSON = mustJSON(scenePlan, "{}")
	run.Progress = 65
	run.CurrentStep = domain.StoryboardGenerationStepImage
	run.MetricsJSON = mustJSON(map[string]any{
		"biblePlanTokens": planTokens,
		"scenePlanTokens": sceneTokens,
		"biblePlanChars":  len(planText),
		"scenePlanChars":  len(sceneText),
	}, "{}")
	run.UpdatedAt = time.Now().Unix()
	_ = s.repo.UpdateStoryboardGenerationRun(ctx, run)

	storyboard.Content = strings.TrimSpace(scenePlan.Content)
	if utf8.RuneCountInString(storyboard.Content) > storyboardContentMaxRunes {
		storyboard.Content = truncateStoryboardContentToMaxRunes(storyboard.Content, storyboardContentMaxRunes)
	}
	storyboard.IsAIGenerated = true
	storyboard.StoryboardScenes = s.storyboardScenesFromScenePlan(run.ID, scenePlan)

	issues := s.auditStoryboardGenerationConsistency(plan, scenePlan, assets, sceneCount, continuation)
	run.ConsistencyIssuesJSON = mustJSON(issues, "[]")
	if detail := firstHighStoryboardConsistencyIssue(issues); detail != "" {
		_ = s.repo.UpdateStoryboardGenerationRun(ctx, run)
		err := fmt.Errorf("story framework consistency check failed: %s", detail)
		s.failStoryboardGenerationRun(ctx, run, "story_framework_conflict", err)
		return err
	}
	run.Progress = 100
	run.CurrentStep = domain.StoryboardGenerationStepConsistency
	run.Status = domain.GenerationStatusCompleted
	done := time.Now().Unix()
	run.CompletedAt = &done
	run.UpdatedAt = done
	_ = s.repo.UpdateStoryboardGenerationRun(ctx, run)
	s.recordPromptAudit(ctx, promptAuditInput{
		RunID:                 run.ID,
		RelatedEntityType:     "storyboard",
		RelatedEntityID:       storyboard.ID,
		Step:                  domain.StoryboardGenerationStepConsistency,
		PromptKind:            domain.PromptKindConsistencyAudit,
		PromptTemplateVersion: storyboardPromptTemplateVersion,
		AlignmentPrompt:       alignmentPrompt,
		UserPrompt:            "Best-effort deterministic consistency audit over bible, beats, scene plan, reference assets, and generated prompts.",
		Provider:              "internal",
		Model:                 "deterministic-audit",
		Output:                run.ConsistencyIssuesJSON,
	})
	return nil
}

func (s *Service) repairStoryboardJSONResponse(
	ctx context.Context,
	run *domain.StoryboardGenerationRun,
	storyboard *domain.Storyboard,
	broken string,
	detail string,
	step string,
	promptKind string,
	metadataOp string,
) (string, int, error) {
	if s.aiGenService == nil {
		return "", 0, fmt.Errorf("AI generation service not configured")
	}
	snippet := broken
	const maxRepairSnippet = 12000
	if len(snippet) > maxRepairSnippet {
		snippet = snippet[:maxRepairSnippet]
	}
	prompt := resolveStoryboardJSONRepairPrompt(storyboard, snippet, detail, step, metadataOp)
	if storyboard.WorkflowReleaseID != "" && shouldWarnStoryboardPromptFallback(prompt.FallbackReason) {
		s.logger.Warn("storyboard JSON repair workflow prompt fallback",
			zap.String("storyboardId", storyboard.ID),
			zap.String("workflowReleaseId", storyboard.WorkflowReleaseID),
			zap.String("reason", prompt.FallbackReason))
	}

	res, err := s.aiGenService.GenerateText(ctx, &GenerateTextRequest{
		UserID:                storyboard.UserID,
		OriginalPrompt:        prompt.UserPrompt,
		SystemPrompt:          prompt.SystemPrompt,
		Model:                 prompt.Model,
		Temperature:           prompt.Temperature,
		MaxTokens:             prompt.MaxTokens,
		RelatedEntityID:       storyboard.ID,
		RelatedEntityType:     "storyboard",
		RunID:                 run.ID,
		Step:                  step,
		PromptKind:            promptKind,
		PromptTemplateVersion: prompt.TemplateVersion,
		AlignmentPrompt:       "",
		Metadata: map[string]interface{}{
			"operation":                    metadataOp,
			"storyboardId":                 storyboard.ID,
			"storyId":                      storyboard.StoryID,
			"workflowReleaseId":            storyboard.WorkflowReleaseID,
			"workflowPromptApplied":        prompt.Applied,
			"workflowPromptFallbackReason": prompt.FallbackReason,
		},
	})
	if err != nil {
		return "", 0, err
	}
	return res.Text, res.TokensUsed, nil
}

func (s *Service) generateStoryboardBiblePlan(ctx context.Context, run *domain.StoryboardGenerationRun, story *domain.Story, storyboard *domain.Storyboard, snapshot storyboardGenerationContextSnapshot, alignmentPrompt string, sceneCount int) (*domain.StoryboardBiblePlan, string, int, error) {
	prompt := resolveStoryboardBiblePrompt(story, storyboard, snapshot, alignmentPrompt, sceneCount)
	prompt.SystemPrompt += "\nWrite all user-facing narrative fields in " + storyboardWorkflowOutputLanguage(run) + ". Keep machine keys and executable image prompts in English."
	if storyboard.WorkflowReleaseID != "" && shouldWarnStoryboardPromptFallback(prompt.FallbackReason) {
		s.logger.Warn("storyboard workflow prompt fallback",
			zap.String("storyboardId", storyboard.ID),
			zap.String("workflowReleaseId", storyboard.WorkflowReleaseID),
			zap.String("reason", prompt.FallbackReason))
	}
	res, err := s.aiGenService.GenerateText(ctx, &GenerateTextRequest{
		UserID:                storyboard.UserID,
		OriginalPrompt:        prompt.UserPrompt,
		SystemPrompt:          prompt.SystemPrompt,
		Model:                 prompt.Model,
		Temperature:           prompt.Temperature,
		MaxTokens:             prompt.MaxTokens,
		RelatedEntityID:       storyboard.ID,
		RelatedEntityType:     "storyboard",
		RunID:                 run.ID,
		Step:                  domain.StoryboardGenerationStepBiblePlan,
		PromptKind:            domain.PromptKindBiblePlanUser,
		PromptTemplateVersion: prompt.TemplateVersion,
		AlignmentPrompt:       alignmentPrompt,
		Metadata: map[string]interface{}{
			"operation":                    "storyboard_bible_plan",
			"storyboardId":                 storyboard.ID,
			"storyId":                      storyboard.StoryID,
			"workflowReleaseId":            storyboard.WorkflowReleaseID,
			"workflowPromptApplied":        prompt.Applied,
			"workflowPromptFallbackReason": prompt.FallbackReason,
		},
	})
	if err != nil {
		return nil, "", 0, fmt.Errorf("generate storyboard bible plan: %w", err)
	}
	text := res.Text
	totalTok := res.TokensUsed
	var parseErr error
	var validateErr error
	var plan *domain.StoryboardBiblePlan
	for attempt := 0; attempt < 2; attempt++ {
		plan, parseErr = s.parseStoryboardBiblePlan(text)
		if parseErr == nil {
			validateErr = validateStoryboardBiblePlan(plan, sceneCount)
		} else {
			validateErr = nil
		}
		if parseErr == nil && validateErr == nil {
			return plan, res.Text, totalTok, nil
		}
		detail := ""
		if parseErr != nil {
			detail = fmt.Sprintf("parse: %v", parseErr)
		}
		if validateErr != nil {
			if detail != "" {
				detail += "; "
			}
			detail += fmt.Sprintf("validate: %v", validateErr)
		}
		if attempt == 1 {
			if parseErr != nil {
				return nil, res.Text, totalTok, parseErr
			}
			return nil, res.Text, totalTok, validateErr
		}
		repaired, repairTok, rerr := s.repairStoryboardJSONResponse(ctx, run, storyboard, text, detail, domain.StoryboardGenerationStepBiblePlan, domain.PromptKindBiblePlanUser, "storyboard_bible_plan_json_repair")
		if rerr != nil {
			if parseErr != nil {
				return nil, res.Text, totalTok, parseErr
			}
			return nil, res.Text, totalTok, validateErr
		}
		text = repaired
		totalTok += repairTok
	}
	return nil, res.Text, totalTok, fmt.Errorf("unexpected bible plan repair exit")
}

func (s *Service) generateStoryboardScenePlan(ctx context.Context, run *domain.StoryboardGenerationRun, story *domain.Story, storyboard *domain.Storyboard, snapshot storyboardGenerationContextSnapshot, plan *domain.StoryboardBiblePlan, alignmentPrompt string, sceneCount int) (*domain.StoryboardScenePlan, string, int, error) {
	prompt := resolveStoryboardScenePrompt(story, storyboard, snapshot, plan, alignmentPrompt, sceneCount)
	prompt.SystemPrompt += "\nWrite content, scene titles, descriptions, dialogue and narration in " + storyboardWorkflowOutputLanguage(run) + ". Keep imagePrompt and machine-readable enum values in English."
	if storyboard.WorkflowReleaseID != "" && shouldWarnStoryboardPromptFallback(prompt.FallbackReason) {
		s.logger.Warn("storyboard scene plan workflow prompt fallback",
			zap.String("storyboardId", storyboard.ID),
			zap.String("workflowReleaseId", storyboard.WorkflowReleaseID),
			zap.String("reason", prompt.FallbackReason))
	}
	res, err := s.aiGenService.GenerateText(ctx, &GenerateTextRequest{
		UserID:                storyboard.UserID,
		OriginalPrompt:        prompt.UserPrompt,
		SystemPrompt:          prompt.SystemPrompt,
		Model:                 prompt.Model,
		Temperature:           prompt.Temperature,
		MaxTokens:             prompt.MaxTokens,
		RelatedEntityID:       storyboard.ID,
		RelatedEntityType:     "storyboard",
		RunID:                 run.ID,
		Step:                  domain.StoryboardGenerationStepScenePlan,
		PromptKind:            domain.PromptKindSceneWriterUser,
		PromptTemplateVersion: prompt.TemplateVersion,
		AlignmentPrompt:       alignmentPrompt,
		Metadata: map[string]interface{}{
			"operation":                    "storyboard_scene_plan",
			"storyboardId":                 storyboard.ID,
			"storyId":                      storyboard.StoryID,
			"workflowReleaseId":            storyboard.WorkflowReleaseID,
			"workflowPromptApplied":        prompt.Applied,
			"workflowPromptFallbackReason": prompt.FallbackReason,
		},
	})
	if err != nil {
		return nil, "", 0, fmt.Errorf("generate storyboard scene plan: %w", err)
	}
	text := res.Text
	totalTok := res.TokensUsed
	var parseErr error
	var validateErr error
	var scenePlan *domain.StoryboardScenePlan
	for attempt := 0; attempt < 2; attempt++ {
		scenePlan, parseErr = s.parseStoryboardScenePlan(text)
		if parseErr == nil {
			applyStoryboardScenePlanFallbacks(scenePlan, plan)
			validateErr = validateStoryboardScenePlan(scenePlan, plan, sceneCount)
		} else {
			validateErr = nil
		}
		if parseErr == nil && validateErr == nil {
			return scenePlan, res.Text, totalTok, nil
		}
		detail := ""
		if parseErr != nil {
			detail = fmt.Sprintf("parse: %v", parseErr)
		}
		if validateErr != nil {
			if detail != "" {
				detail += "; "
			}
			detail += fmt.Sprintf("validate: %v", validateErr)
		}
		if attempt == 1 {
			if parseErr != nil {
				return nil, res.Text, totalTok, parseErr
			}
			return nil, res.Text, totalTok, validateErr
		}
		repaired, repairTok, rerr := s.repairStoryboardJSONResponse(ctx, run, storyboard, text, detail, domain.StoryboardGenerationStepScenePlan, domain.PromptKindSceneWriterUser, "storyboard_scene_plan_json_repair")
		if rerr != nil {
			if parseErr != nil {
				return nil, res.Text, totalTok, parseErr
			}
			return nil, res.Text, totalTok, validateErr
		}
		text = repaired
		totalTok += repairTok
	}
	return nil, res.Text, totalTok, fmt.Errorf("unexpected scene plan repair exit")
}

func (s *Service) buildStoryboardGenerationContextSnapshot(ctx context.Context, storyboard *domain.Storyboard, story *domain.Story, continuation bool) storyboardGenerationContextSnapshot {
	characters := make([]storyboardContextCharacter, 0, len(storyboard.CharacterRefs))
	for idx, ref := range storyboard.CharacterRefs {
		ch, err := s.repo.CharacterByID(ctx, ref.CharacterID)
		if err != nil || ch == nil {
			continue
		}
		key := fmt.Sprintf("char_%d", idx+1)
		characters = append(characters, storyboardContextCharacter{
			ID:          ch.ID,
			Key:         key,
			Name:        ch.Name,
			Role:        firstNonEmpty(ref.Role, ch.Role),
			Description: ch.Description,
			Personality: ch.Personality,
			Appearance:  firstNonEmpty(ch.Appearance, ch.Description),
			Views:       ch.Views,
		})
	}
	scenes := make([]storyboardContextScene, 0, len(storyboard.SceneRefs))
	for idx, ref := range storyboard.SceneRefs {
		sc, err := s.repo.StorySceneByID(ctx, story.ID, ref.StorySceneID)
		if err != nil || sc == nil {
			continue
		}
		scenes = append(scenes, storyboardContextScene{
			ID:          sc.ID,
			Key:         fmt.Sprintf("loc_%d", idx+1),
			Title:       sc.Title,
			Description: sc.Description,
			Location:    sc.Location,
			TimeOfDay:   sc.TimeOfDay,
		})
	}
	parentTail := s.storyboardParentTailScenes(ctx, storyboard)
	return storyboardGenerationContextSnapshot{
		StoryID:      story.ID,
		StoryTitle:   story.Title,
		StoryGenre:   story.Genre,
		StoryStyle:   story.Style,
		RawInput:     storyboard.RawInput,
		Continuation: continuation,
		IsStandalone: storyboard.IsStandalone,
		ContextText:  s.buildStoryboardContext(ctx, storyboard, story),
		Characters:   characters,
		Scenes:       scenes,
		ParentTail:   parentTail,
	}
}

func (s *Service) storyboardGenerationAssetsFromContext(runID string, snapshot storyboardGenerationContextSnapshot) []*domain.StoryboardGenerationAsset {
	var assets []*domain.StoryboardGenerationAsset
	add := func(kind, key, entityID, url, slot string) {
		if strings.TrimSpace(url) == "" {
			return
		}
		assets = append(assets, &domain.StoryboardGenerationAsset{
			ID:           uuid.NewString(),
			RunID:        runID,
			Kind:         kind,
			AssetKey:     key,
			EntityID:     entityID,
			ImageURL:     strings.TrimSpace(url),
			Source:       slot,
			MetadataJSON: mustJSON(map[string]any{"slot": slot}, "{}"),
			CreatedAt:    time.Now().Unix(),
		})
	}
	for _, ch := range snapshot.Characters {
		if ch.Views == nil {
			continue
		}
		turnURL := strings.TrimSpace(ch.Views.Sheet)
		if turnURL == "" {
			// Legacy: fallback to第一个可用分图，避免旧数据无端缺失
			turnURL = strings.TrimSpace(firstNonEmpty(ch.Views.Front, ch.Views.Side, ch.Views.Back))
		}
		add(domain.StoryboardGenerationAssetCharacterTurnaround, ch.Key, ch.ID, turnURL, "sheet")
	}
	for _, scene := range snapshot.ParentTail {
		add(domain.StoryboardGenerationAssetParentTailScene, "parent_tail_"+scene.ID, scene.ID, scene.Image, "parent_tail_scene")
	}
	return assets
}

func (s *Service) storyboardParentTailScenes(ctx context.Context, storyboard *domain.Storyboard) []storyboardContextParentScene {
	if storyboard.IsStandalone || storyboard.ParentID == "" || storyboard.ParentID == domain.StoryboardRootMarker {
		return nil
	}
	parentScenes, err := s.repo.StoryboardScenes(ctx, storyboard.ParentID)
	if err != nil || len(parentScenes) == 0 {
		return nil
	}
	sort.Slice(parentScenes, func(i, j int) bool { return parentScenes[i].Sequence < parentScenes[j].Sequence })
	start := len(parentScenes) - parentTailSceneCount
	if start < 0 {
		start = 0
	}
	out := make([]storyboardContextParentScene, 0, len(parentScenes)-start)
	for _, sc := range parentScenes[start:] {
		if sc == nil {
			continue
		}
		out = append(out, storyboardContextParentScene{
			ID:          sc.ID,
			Title:       sc.Title,
			Description: sc.Description,
			Image:       sc.Image,
		})
	}
	return out
}

func (s *Service) storyboardScenesFromScenePlan(runID string, plan *domain.StoryboardScenePlan) []domain.StoryboardScene {
	if plan == nil {
		return nil
	}
	out := make([]domain.StoryboardScene, 0, len(plan.Scenes))
	for i, item := range plan.Scenes {
		visualState := mustJSON(item.VisualState, "{}")
		seq := item.Sequence
		if seq <= 0 {
			seq = i
		}
		out = append(out, domain.StoryboardScene{
			Sequence:        seq,
			Title:           strings.TrimSpace(item.Title),
			Description:     strings.TrimSpace(item.Description),
			Location:        strings.TrimSpace(item.Location),
			TimeOfDay:       strings.TrimSpace(item.TimeOfDay),
			Mood:            strings.TrimSpace(item.Mood),
			Characters:      item.Characters,
			IsAIGenerated:   true,
			GenerationRunID: runID,
			BeatIndex:       item.Sequence,
			BeatPurpose:     item.BeatPurpose,
			ContinuityNote:  item.ContinuityNote,
			ReferenceKeys:   item.ReferenceKeys,
			ImagePrompt:     item.ImagePrompt,
			VisualStateJSON: visualState,
			ComicTexts:      normalizeStoryboardComicTexts(item.ComicTexts),
			LayoutIntent:    strings.TrimSpace(item.LayoutIntent),
			CompositionPlan: strings.TrimSpace(item.CompositionPlan),
			ShotType:        strings.TrimSpace(item.ShotType),
			VisualHierarchy: strings.TrimSpace(item.VisualHierarchy),
			PanelShape:      strings.TrimSpace(item.PanelShape),
			ContextSnapshot: "{}",
		})
	}
	return out
}

func (s *Service) parseStoryboardBiblePlan(text string) (*domain.StoryboardBiblePlan, error) {
	var plan domain.StoryboardBiblePlan
	cleaned := s.fixCommonJSONIssues(s.cleanAIResponseText(text))
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return nil, fmt.Errorf("parse storyboard bible plan: %w (cleaned_prefix=%q)", err, truncateForLog(cleaned, 160))
	}
	return &plan, nil
}

func (s *Service) parseStoryboardScenePlan(text string) (*domain.StoryboardScenePlan, error) {
	var plan domain.StoryboardScenePlan
	cleaned := s.fixCommonJSONIssues(s.cleanAIResponseText(text))
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return nil, fmt.Errorf("parse storyboard scene plan: %w (cleaned_prefix=%q)", err, truncateForLog(cleaned, 160))
	}
	return &plan, nil
}

func (s *Service) storyboardAlignmentSnapshot(ctx context.Context, story *domain.Story, storyboard *domain.Storyboard) (string, string) {
	userPromptPrefs := ""
	if user, err := s.repo.UserByID(ctx, storyboard.UserID); err == nil && user != nil {
		userPromptPrefs = strings.TrimSpace(user.AIPromptPreferences)
	}
	snapshot := map[string]any{
		"storyUseAI":            story.UseAI,
		"aiAssistanceOptions":   story.AIAssistanceOptions,
		"userPromptPreferences": userPromptPrefs,
	}
	prompt := "Human-value and user-alignment snapshot:\n" + mustJSON(snapshot, "{}") + "\nRespect the user's selected values and safety preferences without adding moralizing exposition."
	return mustJSON(snapshot, "{}"), prompt
}

func (s *Service) failStoryboardGenerationRun(ctx context.Context, run *domain.StoryboardGenerationRun, code string, err error) {
	if run == nil {
		return
	}
	run.Status = domain.GenerationStatusFailed
	run.ErrorCode = code
	run.ErrorMessage = err.Error()
	run.UpdatedAt = time.Now().Unix()
	_ = s.repo.UpdateStoryboardGenerationRun(ctx, run)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
