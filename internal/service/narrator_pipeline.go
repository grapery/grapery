package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ContinueRequest represents a request to continue a storyboard (parallel universe creation)
type ContinueRequest struct {
	ParentStoryboardID string   `json:"parentStoryboardId"`
	UserPrompt         string   `json:"userPrompt"`
	SceneCount         int      `json:"sceneCount"`
	Characters         []string `json:"characters,omitempty"` // Optional: specific character IDs to include
	// GenerateVideo: 为 true 时，每格场景图生成完成后自动基于该图发起视频生成（默认仅图片）
	GenerateVideo bool   `json:"generateVideo"`
	ComicStyle    string `json:"comicStyle,omitempty"` // 漫画风格 slug，持久化到故事板
}

// ContinueResult represents the result of continuing a storyboard
type ContinueResult struct {
	NewStoryboard   *domain.Storyboard            `json:"newStoryboard"`
	GeneratedScenes []domain.StoryboardScene      `json:"generatedScenes"`
	FateSnapshot    map[string]CharacterFateState `json:"fateSnapshot"`
	TokensUsed      int                           `json:"tokensUsed"`
}

// CharacterFateState represents the state of a character at a specific point in time
type CharacterFateState struct {
	CharacterID   string                 `json:"characterId"`
	Name          string                 `json:"name"`
	Health        int                    `json:"health,omitempty"`
	Mood          string                 `json:"mood,omitempty"`
	Location      string                 `json:"location,omitempty"`
	Relationships map[string]string      `json:"relationships,omitempty"`
	Knowledge     []string               `json:"knowledge,omitempty"`
	Goals         []string               `json:"goals,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ContinueStoryboard continues a storyboard by creating a parallel universe
// This is the main entry point for the "AI 辅助平行宇宙续写" feature
func (s *Service) ContinueStoryboard(ctx context.Context, userID string, req *ContinueRequest) (*ContinueResult, error) {
	sceneCount, err := NormalizeStoryboardSceneCount(req.SceneCount)
	if err != nil {
		return nil, err
	}
	req.SceneCount = sceneCount

	s.logger.Info("starting storyboard continuation (parallel universe creation)",
		zap.String("userId", userID),
		zap.String("parentStoryboardId", req.ParentStoryboardID),
		zap.String("userPrompt", truncateLog(req.UserPrompt, 200)),
		zap.Int("requestedSceneCount", req.SceneCount),
		zap.Bool("generateVideoAfterImages", req.GenerateVideo),
		zap.String("comicStyle", strings.TrimSpace(req.ComicStyle)))

	// Step 1: Validate and fetch parent storyboard
	parentStoryboard, err := s.repo.StoryboardByID(ctx, req.ParentStoryboardID)
	if err != nil {
		s.logger.Error("parent storyboard not found",
			zap.String("parentStoryboardId", req.ParentStoryboardID),
			zap.Error(err))
		return nil, fmt.Errorf("parent storyboard not found: %w", err)
	}

	// Verify user has permission to create storyboards in this story
	story, err := s.repo.StoryByID(ctx, parentStoryboard.StoryID)
	if err != nil {
		return nil, fmt.Errorf("story not found: %w", err)
	}

	// Check permissions based on collaboration status
	canContinue := s.canUserContinueStoryboard(ctx, userID, story)
	if !canContinue {
		s.logger.Warn("user does not have permission to continue storyboard",
			zap.String("userId", userID),
			zap.String("storyboardId", req.ParentStoryboardID),
			zap.String("storyId", story.ID))
		return nil, fmt.Errorf("user does not have permission to continue this storyboard")
	}

	// Step 2: Trace the path from root to parent (collect all ancestor scenes)
	pathTracer := &PathTracer{repo: s.repo, logger: s.logger}
	ancestorScenes, err := pathTracer.TracePath(ctx, req.ParentStoryboardID)
	if err != nil {
		s.logger.Error("failed to trace path",
			zap.String("parentStoryboardId", req.ParentStoryboardID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to trace path: %w", err)
	}
	s.logger.Debug("path traced successfully",
		zap.String("parentStoryboardId", req.ParentStoryboardID),
		zap.Int("ancestorScenes", len(ancestorScenes)))

	// Step 3: Synthesize character states (soul + all deltas)
	stateSynthesizer := &StateSynthesizer{repo: s.repo, logger: s.logger}
	characters, fateSnapshot, err := stateSynthesizer.SynthesizeStates(ctx, parentStoryboard, ancestorScenes, req.Characters)
	if err != nil {
		s.logger.Error("failed to synthesize character states",
			zap.String("parentStoryboardId", req.ParentStoryboardID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to synthesize character states: %w", err)
	}
	s.logger.Debug("character states synthesized",
		zap.String("parentStoryboardId", req.ParentStoryboardID),
		zap.Int("characters", len(characters)))

	// Step 4: Serialize fate snapshot
	fateSnapshotJSON, err := json.Marshal(fateSnapshot)
	if err != nil {
		s.logger.Error("failed to marshal fate snapshot",
			zap.String("parentStoryboardId", req.ParentStoryboardID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to serialize fate snapshot: %w", err)
	}
	fateSnapshotStr := string(fateSnapshotJSON)

	// Generate hash for fate snapshot (for quick comparison)
	hash := sha256.Sum256(fateSnapshotJSON)
	fateSnapshotHash := hex.EncodeToString(hash[:])

	s.logger.Debug("fate snapshot serialized",
		zap.String("fateSnapshotSize", fmt.Sprintf("%d bytes", len(fateSnapshotJSON))),
		zap.String("fateSnapshotHash", fateSnapshotHash))

	// Step 5: Create new storyboard
	newStoryboardID := uuid.New().String()
	now := currentTimeMillis()
	fateSnapshotStrRef := fateSnapshotStr
	fateSnapshotHashRef := fateSnapshotHash
	newStoryboard := &domain.Storyboard{
		BaseModel: common.BaseModel{
			ID:        newStoryboardID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		StoryID:                  parentStoryboard.StoryID,
		ParentID:                 req.ParentStoryboardID,
		UserID:                   userID,
		Title:                    generateContinuationTitle(parentStoryboard),
		Content:                  "", // Will be filled by AI
		RawInput:                 req.UserPrompt,
		IsStandalone:             false,
		IsAIGenerated:            true,
		SceneCount:               req.SceneCount,
		WorkflowStatus:           domain.WorkflowStatusDraft,
		CurrentStep:              1,
		GenerateVideoAfterImages: req.GenerateVideo,
		ContinuationComicStyle:   strings.TrimSpace(req.ComicStyle),
		EngagementStats: common.EngagementStats{
			Likes:    0,
			Comments: 0,
			Shares:   0,
			Views:    0,
		},
		FateSnapshot:     &fateSnapshotStrRef,
		FateSnapshotHash: &fateSnapshotHashRef,
	}

	// Step 6: Generate storyboard content and scenes using AI
	generatedScenes, tokensUsed, err := s.generateContinuationStoryboard(
		ctx,
		newStoryboard,
		req.UserPrompt,
		ancestorScenes,
		fateSnapshot,
	)
	if err != nil {
		s.logger.Error("failed to generate continuation storyboard",
			zap.String("newStoryboardId", newStoryboardID),
			zap.Error(err))
		if nerr := s.NotifyStoryboardGenerationFailed(ctx, userID, parentStoryboard.StoryID, req.ParentStoryboardID, err.Error()); nerr != nil {
			s.logger.Warn("continuation storyboard generation failure notify failed", zap.Error(nerr))
		}
		return nil, fmt.Errorf("failed to generate storyboard: %w", err)
	}

	// Attach scenes for response payload (same slice as persisted in transaction)
	newStoryboard.StoryboardScenes = generatedScenes

	// Step 7: Persist storyboard + scenes in one transaction, then start async image jobs
	if err := s.repo.WithTransaction(ctx, func(tx domain.Repository) error {
		if err := tx.CreateStoryboard(ctx, newStoryboard); err != nil {
			s.logger.Error("failed to create new storyboard in transaction",
				zap.String("newStoryboardId", newStoryboardID),
				zap.Error(err))
			return fmt.Errorf("failed to create storyboard: %w", err)
		}
		if len(generatedScenes) > 0 {
			scenes := make([]*domain.StoryboardScene, len(generatedScenes))
			for i := range generatedScenes {
				scenes[i] = &generatedScenes[i]
			}
			if err := tx.CreateStoryboardScenes(ctx, newStoryboard.ID, scenes); err != nil {
				s.logger.Error("failed to persist continuation storyboard scenes",
					zap.String("newStoryboardId", newStoryboardID),
					zap.Int("sceneCount", len(scenes)),
					zap.Error(err))
				return fmt.Errorf("failed to persist storyboard scenes: %w", err)
			}
			s.logger.Info("continuation storyboard scenes persisted",
				zap.String("newStoryboardId", newStoryboardID),
				zap.Int("sceneCount", len(scenes)))
		}
		return nil
	}); err != nil {
		if nerr := s.NotifyStoryboardGenerationFailed(ctx, userID, parentStoryboard.StoryID, req.ParentStoryboardID, "续写保存失败："+err.Error()); nerr != nil {
			s.logger.Warn("continuation persist failure notify failed", zap.Error(nerr))
		}
		return nil, err
	}

	if len(generatedScenes) > 0 || strings.TrimSpace(newStoryboard.Content) != "" {
		if err := s.NotifyStoryboardInitialGenerationCompleted(ctx, newStoryboard.UserID, newStoryboard.ID, newStoryboard.StoryID, tokensUsed); err != nil {
			s.logger.Warn("continuation storyboard completion notify failed",
				zap.Error(err),
				zap.String("storyboardId", newStoryboard.ID))
		}
	}

	s.startContinuationSceneImageGenerations(newStoryboard.ID, newStoryboard.ContinuationComicStyle, generatedScenes)

	// Update parent storyboard's fork count
	parentStoryboard.ForkCount++
	if err := s.repo.UpdateStoryboard(ctx, parentStoryboard); err != nil {
		s.logger.Warn("failed to update parent storyboard fork count",
			zap.String("parentStoryboardId", req.ParentStoryboardID),
			zap.Error(err))
	}

	s.logger.Info("storyboard continuation completed successfully",
		zap.String("newStoryboardId", newStoryboardID),
		zap.String("parentStoryboardId", req.ParentStoryboardID),
		zap.Int("scenesGenerated", len(generatedScenes)),
		zap.Int("tokensUsed", tokensUsed))

	return &ContinueResult{
		NewStoryboard:   newStoryboard,
		GeneratedScenes: generatedScenes,
		FateSnapshot:    fateSnapshot,
		TokensUsed:      tokensUsed,
	}, nil
}

// canUserContinueStoryboard checks if user can continue the storyboard based on story permissions
func (s *Service) canUserContinueStoryboard(ctx context.Context, userID string, story *domain.Story) bool {
	// Story creator can always continue
	if story.UserID == userID {
		return true
	}

	// Check if collaboration is open
	if story.IsCollaborationOpen {
		return true
	}

	// Check if user is a contributor
	contributors, err := s.repo.GetStoryContributors(ctx, story.ID, 0, 100)
	if err != nil {
		s.logger.Warn("failed to check story contributors",
			zap.String("storyId", story.ID),
			zap.Error(err))
		return false
	}

	for _, contributor := range contributors {
		if contributor.UserID == userID {
			return true
		}
	}

	return false
}

// generateContinuationTitle generates a title for the continuation
func generateContinuationTitle(parent *domain.Storyboard) string {
	return parent.Title + " (续)"
}

// generateContinuationStoryboard 生成续写故事板的内容和场景
// 这是平行宇宙续写的核心生成逻辑：
// 1. 生成 Storyboard.Content（AI 叙述文本）
// 2. 基于 Content 生成多个 Scene（场景细分）
// 生图在事务落库之后由 startContinuationSceneImageGenerations 启动。
func (s *Service) generateContinuationStoryboard(
	ctx context.Context,
	newStoryboard *domain.Storyboard,
	userPrompt string,
	ancestorScenes []*domain.StoryboardScene,
	fateSnapshot map[string]CharacterFateState,
) ([]domain.StoryboardScene, int, error) {
	s.logger.Info("generating continuation storyboard content and scenes",
		zap.String("storyboardId", newStoryboard.ID),
		zap.Int("requestedSceneCount", newStoryboard.SceneCount))

	// Step 1: Generate storyboard content (narrative)
	// 使用祖先场景上下文 + 用户输入 + 角色状态生成叙述文本
	content, tokensContent, err := s.generateStoryboardContent(
		ctx,
		newStoryboard,
		userPrompt,
		ancestorScenes,
		fateSnapshot,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to generate storyboard content: %w", err)
	}

	// Update storyboard with generated content
	newStoryboard.Content = content

	// Step 2: Generate scenes based on the content
	// 基于生成的叙述文本，将其细分为多个场景
	scenes, tokensScenes, err := s.generateScenesFromContent(
		ctx,
		newStoryboard.ID,
		content,
		newStoryboard.SceneCount,
		newStoryboard.ContinuationComicStyle,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to generate scenes from content: %w", err)
	}

	totalTokens := tokensContent + tokensScenes

	s.logger.Info("continuation storyboard generated successfully",
		zap.String("storyboardId", newStoryboard.ID),
		zap.Int("scenesGenerated", len(scenes)),
		zap.Int("totalTokens", totalTokens))

	return scenes, totalTokens, nil
}

// startContinuationSceneImageGenerations 在故事板与分镜行已提交后，为每个分镜异步拉起 GenerateSceneImage。
func (s *Service) startContinuationSceneImageGenerations(storyboardID, comicStyle string, scenes []domain.StoryboardScene) {
	if len(scenes) == 0 {
		return
	}
	comicStyle = strings.TrimSpace(comicStyle)
	s.logger.Info("starting async image generation for continuation scenes",
		zap.String("storyboardId", storyboardID),
		zap.Int("sceneCount", len(scenes)),
		zap.String("comicStyle", comicStyle))

	for _, scene := range scenes {
		scene := scene
		description := scene.Description
		if strings.TrimSpace(scene.ImagePrompt) != "" {
			description = strings.TrimSpace(scene.ImagePrompt) + "\n\nContinuity note: " + strings.TrimSpace(scene.ContinuityNote)
		}
		imageReq := &ImageGenerationRequest{
			StoryboardID:     storyboardID,
			SceneID:          scene.ID,
			SceneTitle:       scene.Title,
			SceneDescription: description,
			SceneCharacters:  scene.Characters,
			ReferenceImages:  s.storyboardSceneReferenceImages(context.Background(), scene),
			ComicStyle:       comicStyle,
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("panic in continuation image generation",
						zap.String("storyboardId", storyboardID),
						zap.String("sceneId", scene.ID),
						zap.Any("panic", r))
				}
			}()
			gen, err := s.GenerateSceneImage(context.Background(), imageReq)
			if err != nil {
				s.logger.Warn("failed to start image generation for continuation scene",
					zap.String("storyboardId", storyboardID),
					zap.String("sceneId", scene.ID),
					zap.Error(err))
				return
			}
			s.logger.Debug("continuation image generation started",
				zap.String("storyboardId", storyboardID),
				zap.String("sceneId", scene.ID),
				zap.String("generationId", gen.ID))
		}()
	}
}

func (s *Service) storyboardSceneReferenceImages(ctx context.Context, scene domain.StoryboardScene) []string {
	if strings.TrimSpace(scene.GenerationRunID) == "" || len(scene.ReferenceKeys) == 0 {
		return nil
	}
	assets, err := s.repo.ListStoryboardGenerationAssets(ctx, scene.GenerationRunID)
	if err != nil || len(assets) == 0 {
		return nil
	}
	keyAllowed := make(map[string]struct{}, len(scene.ReferenceKeys))
	for _, key := range scene.ReferenceKeys {
		keyAllowed[strings.TrimSpace(key)] = struct{}{}
	}
	priority := map[string]int{
		domain.StoryboardGenerationAssetCharacterTurnaround: 0,
		domain.StoryboardGenerationAssetCharacterAnchor:     1,
		domain.StoryboardGenerationAssetPreviousScene:       2,
		domain.StoryboardGenerationAssetParentTailScene:     3,
		domain.StoryboardGenerationAssetLocationAnchor:      4,
		domain.StoryboardGenerationAssetPropAnchor:          5,
	}
	filtered := make([]*domain.StoryboardGenerationAsset, 0, len(assets))
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		if _, ok := keyAllowed[asset.AssetKey]; ok {
			filtered = append(filtered, asset)
			continue
		}
		if asset.Kind == domain.StoryboardGenerationAssetParentTailScene {
			filtered = append(filtered, asset)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		pi, ok := priority[filtered[i].Kind]
		if !ok {
			pi = 99
		}
		pj, ok := priority[filtered[j].Kind]
		if !ok {
			pj = 99
		}
		if pi != pj {
			return pi < pj
		}
		return filtered[i].Source < filtered[j].Source
	})
	out := make([]string, 0, len(filtered))
	seen := map[string]struct{}{}
	for _, asset := range filtered {
		u := strings.TrimSpace(asset.ImageURL)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
		if len(out) >= 6 {
			break
		}
	}
	return out
}

// generateStoryboardContent 生成故事板的叙述内容
func (s *Service) generateStoryboardContent(
	ctx context.Context,
	newStoryboard *domain.Storyboard,
	userPrompt string,
	ancestorScenes []*domain.StoryboardScene,
	fateSnapshot map[string]CharacterFateState,
) (string, int, error) {
	s.logger.Info("generating storyboard content",
		zap.String("storyboardId", newStoryboard.ID),
		zap.Int("ancestorScenes", len(ancestorScenes)),
		zap.Int("characterCount", len(fateSnapshot)))

	// Check text LLM availability — 火山优先，失败再 Gemini
	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	geminiOK := s.geminiClient != nil
	if !huoshanOK && !geminiOK {
		s.logger.Warn("no AI client available, using placeholder content",
			zap.String("storyboardId", newStoryboard.ID))
		placeholderContent := fmt.Sprintf("基于用户输入「%s」生成的故事叙述内容。\n\n"+
			"这是一个平行宇宙的续写，包含了 %d 个角色的命运状态。",
			truncateLog(userPrompt, 100), len(fateSnapshot))
		return placeholderContent, 0, nil
	}

	// Build context from ancestor scenes and character states
	contextBuilder := &strings.Builder{}
	contextBuilder.WriteString("# 平行宇宙续写上下文\n\n")

	// Add ancestor scenes summary (limited to avoid token overflow)
	if len(ancestorScenes) > 0 {
		contextBuilder.WriteString("## 前情回顾\n\n")
		maxScenes := 10 // Limit to prevent token overflow
		if len(ancestorScenes) > maxScenes {
			contextBuilder.WriteString(fmt.Sprintf("(最近 %d 个场景的摘要)\n\n", maxScenes))
		}
		for i, scene := range ancestorScenes {
			if i >= maxScenes {
				break
			}
			contextBuilder.WriteString(fmt.Sprintf("### 场景 %d: %s\n", scene.Sequence, scene.Title))
			contextBuilder.WriteString(fmt.Sprintf("%s\n\n", truncateLog(scene.Description, 200)))
		}
	}

	// Add character fate states
	if len(fateSnapshot) > 0 {
		contextBuilder.WriteString("## 角色当前状态\n\n")
		for charID, state := range fateSnapshot {
			contextBuilder.WriteString(fmt.Sprintf("### %s (ID: %s)\n", state.Name, charID))
			if state.Health > 0 {
				contextBuilder.WriteString(fmt.Sprintf("- 生命值: %d\n", state.Health))
			}
			if state.Mood != "" {
				contextBuilder.WriteString(fmt.Sprintf("- 情绪: %s\n", state.Mood))
			}
			if state.Location != "" {
				contextBuilder.WriteString(fmt.Sprintf("- 位置: %s\n", state.Location))
			}
			if len(state.Knowledge) > 0 {
				contextBuilder.WriteString(fmt.Sprintf("- 知识: %s\n", strings.Join(state.Knowledge, ", ")))
			}
			if len(state.Goals) > 0 {
				contextBuilder.WriteString(fmt.Sprintf("- 目标: %s\n", strings.Join(state.Goals, ", ")))
			}
			contextBuilder.WriteString("\n")
		}
	}

	// Build the prompt
	comicStyleBlock := ""
	if cs := strings.TrimSpace(newStoryboard.ContinuationComicStyle); cs != "" {
		comicStyleBlock = fmt.Sprintf(`
## 漫画/视觉风格
用户选择的漫画风格 slug：%s
叙述与节奏应贴合该风格的气质与画面感（为后续分镜图生成一致）。

`, cs)
	}
	prompt := fmt.Sprintf(`你是一位专业的小说作家。请根据以下上下文和用户输入，续写一个引人入胜的故事章节。

%s
%s
## 用户输入
%s

## 要求
1. 保持叙事连贯性，自然地延续前文
2. 基于角色当前状态进行合理发展
3. 创造引人入胜的情节转折
4. 篇幅控制在 800-1500 字
5. 只输出故事内容，不要输出其他解释或说明

请开始创作:`, contextBuilder.String(), comicStyleBlock, userPrompt)

	s.logger.Debug("calling storyboard continuation content LLM (huoshan then gemini)",
		zap.String("storyboardId", newStoryboard.ID),
		zap.Int("promptLength", len(prompt)))

	content, _, _, totalTokens, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, prompt, "narrator_content", 8192, 0.7, false, 0.7, 8192)
	if err != nil {
		s.logger.Error("AI content generation failed",
			zap.String("storyboardId", newStoryboard.ID),
			zap.String("lastProvider", prov),
			zap.Error(err))
		return "", 0, fmt.Errorf("AI content generation failed: %w", err)
	}

	s.logger.Info("storyboard content generated successfully",
		zap.String("storyboardId", newStoryboard.ID),
		zap.String("provider", prov),
		zap.Int("contentLength", len(content)),
		zap.Int("totalTokens", totalTokens))

	return content, totalTokens, nil
}

// generateScenesFromContent 基于叙述内容生成场景
func (s *Service) generateScenesFromContent(
	ctx context.Context,
	storyboardID string,
	content string,
	sceneCount int,
	comicStyle string,
) ([]domain.StoryboardScene, int, error) {
	s.logger.Info("generating scenes from content",
		zap.String("storyboardId", storyboardID),
		zap.Int("sceneCount", sceneCount))

	// Truncate content if too long (prevent token overflow)
	maxContentLength := 3000
	contentForPrompt := content
	if len(content) > maxContentLength {
		contentForPrompt = truncateLog(content, maxContentLength) + "\n\n(内容已截断，仅供参考)"
	}

	// Text LLM：火山优先，失败再 Gemini
	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	geminiOK := s.geminiClient != nil
	if !huoshanOK && !geminiOK {
		s.logger.Warn("no AI client available, using placeholder scenes",
			zap.String("storyboardId", storyboardID))
		return s.generatePlaceholderScenes(storyboardID, sceneCount), 0, nil
	}

	comicHint := ""
	if cs := strings.TrimSpace(comicStyle); cs != "" {
		comicHint = fmt.Sprintf(`
## 漫画/视觉风格
分镜描述需在构图、氛围上与漫画风格「%s」一致，便于后续生图。

`, cs)
	}
	// Build the prompt for structured scene generation
	prompt := fmt.Sprintf(`你是一位专业的编剧。请将以下故事内容细分为 %d 个场景，并为每个场景生成详细信息。

## 故事内容
%s
%s
## 要求
1. 将故事自然地划分为 %d 个场景
2. 每个场景包含：标题、描述、地点、时间段
3. 场景之间要有逻辑连贯性
4. 输出 JSON 格式，包含 scenes 数组

## 输出格式
请严格按照以下 JSON 格式输出：
{
  "scenes": [
    {
      "sequence": 1,
      "title": "场景标题",
      "description": "场景详细描述（100-200字）",
      "location": "地点",
      "timeOfDay": "时间段（如：清晨、中午、傍晚、深夜）",
      "mood": "氛围（如：紧张、温馨、悲伤、欢快）"
    }
  ]
}

请生成场景信息:`, sceneCount, contentForPrompt, comicHint, sceneCount)

	s.logger.Debug("calling narrator scene plan LLM (huoshan then gemini)",
		zap.String("storyboardId", storyboardID),
		zap.Int("sceneCount", sceneCount),
		zap.Int("promptLength", len(prompt)))

	generatedText, _, _, totalTokens, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, prompt, "narrator_scenes", 8192, 0.35, true, 0.35, 8192)
	if err != nil {
		s.logger.Warn("AI scene generation failed, using placeholder scenes",
			zap.String("storyboardId", storyboardID),
			zap.String("lastProvider", prov),
			zap.Error(err))
		return s.generatePlaceholderScenes(storyboardID, sceneCount), 0, nil
	}

	// Parse the generated scenes from JSON
	scenes, err := s.parseGeneratedScenes(storyboardID, generatedText, sceneCount)
	if err != nil {
		s.logger.Warn("failed to parse generated scenes, using placeholder scenes",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return s.generatePlaceholderScenes(storyboardID, sceneCount), 0, nil
	}

	s.logger.Info("scenes generated successfully",
		zap.String("storyboardId", storyboardID),
		zap.String("provider", prov),
		zap.Int("sceneCount", len(scenes)),
		zap.Int("totalTokens", totalTokens))

	return scenes, totalTokens, nil
}

// generatePlaceholderScenes generates placeholder scenes when AI is not available
func (s *Service) generatePlaceholderScenes(storyboardID string, sceneCount int) []domain.StoryboardScene {
	scenes := make([]domain.StoryboardScene, sceneCount)
	now := currentTimeMillis()

	for i := 0; i < sceneCount; i++ {
		sceneID := uuid.New().String()
		scenes[i] = domain.StoryboardScene{
			BaseModel: common.BaseModel{
				ID:        sceneID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			StoryboardID:  storyboardID,
			Sequence:      i + 1,
			Title:         fmt.Sprintf("场景 %d", i+1),
			Description:   fmt.Sprintf("这是基于 AI 生成内容创建的场景 %d。完整的场景描述将在后续步骤中生成。", i+1),
			Location:      "待定",
			TimeOfDay:     "待定",
			Mood:          "待定",
			IsAIGenerated: true,
		}
	}

	return scenes
}

// parseGeneratedScenes parses AI-generated scene data from JSON text
func (s *Service) parseGeneratedScenes(storyboardID, generatedText string, expectedCount int) ([]domain.StoryboardScene, error) {
	// Extract JSON from the response (handle markdown code blocks)
	jsonStart := strings.Index(generatedText, "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON found in generated text")
	}
	jsonEnd := strings.LastIndex(generatedText, "}")
	if jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("invalid JSON format in generated text")
	}
	jsonStr := generatedText[jsonStart : jsonEnd+1]

	// Parse JSON
	var result struct {
		Scenes []struct {
			Sequence    int    `json:"sequence"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Location    string `json:"location"`
			TimeOfDay   string `json:"timeOfDay"`
			Mood        string `json:"mood"`
		} `json:"scenes"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse scene JSON: %w", err)
	}

	// Convert to domain scenes
	now := currentTimeMillis()
	scenes := make([]domain.StoryboardScene, len(result.Scenes))

	for i, sceneData := range result.Scenes {
		sceneID := uuid.New().String()
		scenes[i] = domain.StoryboardScene{
			BaseModel: common.BaseModel{
				ID:        sceneID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			StoryboardID:  storyboardID,
			Sequence:      sceneData.Sequence,
			Title:         sceneData.Title,
			Description:   sceneData.Description,
			Location:      sceneData.Location,
			TimeOfDay:     sceneData.TimeOfDay,
			Mood:          sceneData.Mood,
			IsAIGenerated: true,
		}
	}

	// Fill missing scenes if AI didn't generate enough
	for i := len(scenes); i < expectedCount; i++ {
		sceneID := uuid.New().String()
		scenes = append(scenes, domain.StoryboardScene{
			BaseModel: common.BaseModel{
				ID:        sceneID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			StoryboardID:  storyboardID,
			Sequence:      i + 1,
			Title:         fmt.Sprintf("场景 %d", i+1),
			Description:   "补充场景",
			IsAIGenerated: true,
		})
	}

	return scenes, nil
}

func currentTimeMillis() int64 {
	return time.Now().UnixMilli()
}

// truncateLog truncates a string for logging purposes
func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
