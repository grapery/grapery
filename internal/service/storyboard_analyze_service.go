package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// AnalyzeStoryboardDirection performs lightweight phase-one planning for conversational storyboard creation.
func (s *Service) AnalyzeStoryboardDirection(ctx context.Context, userID string, req domain.StoryboardAnalyzeRequest) (*domain.StoryboardAnalyzeResponse, error) {
	input := strings.TrimSpace(req.UserInput)
	if input == "" {
		return nil, fmt.Errorf("userInput is required")
	}

	language := strings.TrimSpace(req.Language)
	if language == "" {
		language = "zh-Hans"
	}
	sceneCount := req.SceneCount
	if sceneCount <= 0 {
		sceneCount = 3
	}
	sceneCount = normalizeStoryboardSceneCount(sceneCount)
	if explicit := explicitCreativeCount(input); explicit > 0 {
		sceneCount = normalizeStoryboardSceneCount(explicit)
	}
	style := strings.TrimSpace(req.Style)
	if explicit := explicitCreativeStyle(input); explicit != "" {
		style = explicit
	} else if style == "" {
		style = inferStoryboardAnalyzeStyle(input)
	}
	useComic := false
	if req.UseComicPagePipeline != nil {
		useComic = *req.UseComicPagePipeline
	}
	aspectRatio := domain.NormalizeFragmentAspectRatio(req.AspectRatio)
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	if explicit := explicitCreativeAspectRatio(input); explicit != "" {
		aspectRatio = explicit
	}

	hasExisting := strings.TrimSpace(req.ParentStoryboardID) != "" || strings.TrimSpace(req.TargetDraftStoryboardID) != ""
	editPlan := planCreativeEdit(input, hasExisting, "格")
	intentType := inferStoryboardInputIntent(input)
	if editPlan.NeedsClarification {
		intentType = "ask_clarification"
	} else if editPlan.Operation == "replace" {
		intentType = "revise_current"
	} else if editPlan.Operation == "adjust_options" {
		intentType = "adjust_options"
	}
	resp := &domain.StoryboardAnalyzeResponse{
		AssistantMessage: summarizeStoryboardAnalyzeMessage(input, req.ParentStoryboardID),
		IntentType:       intentType,
		EditPlan:         editPlan,
		GenerationIntent: domain.StoryboardGenerationIntent{
			UserInput:             input,
			SceneCount:            sceneCount,
			Style:                 style,
			AspectRatio:           aspectRatio,
			Language:              language,
			StoryID:               strings.TrimSpace(req.StoryID),
			ParentStoryboardID:    strings.TrimSpace(req.ParentStoryboardID),
			UseComicPagePipeline:  useComic,
			TargetDraftStoryboard: strings.TrimSpace(req.TargetDraftStoryboardID),
		},
		RecommendedOptions: domain.StoryboardRecommendedOptions{
			StyleCandidates: []string{style, "smart_recommend"},
			CanStart:        intentType != "chat_only" && intentType != "ask_clarification",
		},
	}
	if editPlan.NeedsClarification {
		resp.AssistantMessage = editPlan.ClarificationQuestion
	}

	if storyID := strings.TrimSpace(req.StoryID); storyID != "" && s.repo != nil {
		story, err := s.repo.StoryByID(ctx, storyID)
		if err != nil {
			return nil, fmt.Errorf("load story framework: %w", err)
		}
		if story == nil {
			return nil, fmt.Errorf("load story framework: story not found")
		}
		alignment := storyboardFrameworkAlignment(story)
		var parent *domain.Storyboard
		if parentID := strings.TrimSpace(req.ParentStoryboardID); parentID != "" {
			parent, err = s.repo.StoryboardByID(ctx, parentID)
			if err != nil {
				return nil, fmt.Errorf("load parent storyboard framework: %w", err)
			}
			if parent == nil {
				return nil, fmt.Errorf("load parent storyboard framework: storyboard not found")
			}
			if parent.StoryID != storyID {
				return nil, fmt.Errorf("parent storyboard belongs to different story")
			}
			applyParentStoryboardFramework(alignment, parent)
		}

		if chars, err := s.repo.CharactersByStory(ctx, storyID); err == nil {
			for _, ch := range chars {
				if ch == nil {
					continue
				}
				name := strings.TrimSpace(ch.Name)
				if name == "" {
					continue
				}
				if len(resp.CharacterCandidates) < 10 {
					resp.CharacterCandidates = append(resp.CharacterCandidates, domain.StoryboardCharacterCandidate{
						ID:   ch.ID,
						Name: name,
						Hint: "可选入镜角色",
					})
				}
				if storyboardInputMentions(input, ch.Name) {
					resp.GenerationIntent.CharacterIDs = appendUniqueString(resp.GenerationIntent.CharacterIDs, ch.ID)
				}
			}
		} else {
			s.logger.Debug("analyze: list story characters failed",
				zap.String("storyId", storyID),
				zap.Error(err))
		}
		if len(resp.GenerationIntent.CharacterIDs) == 0 && parent != nil {
			for _, ref := range parent.CharacterRefs {
				resp.GenerationIntent.CharacterIDs = appendUniqueString(resp.GenerationIntent.CharacterIDs, ref.CharacterID)
			}
		}
		if scenes, err := s.repo.StoryScenes(ctx, storyID, 100, 0); err == nil {
			for _, scene := range scenes {
				if scene != nil && (storyboardInputMentions(input, scene.Title) || storyboardInputMentions(input, scene.Location)) {
					resp.GenerationIntent.SceneIDs = appendUniqueString(resp.GenerationIntent.SceneIDs, scene.ID)
				}
			}
		}
		if len(resp.GenerationIntent.SceneIDs) == 0 && parent != nil {
			for _, ref := range parent.SceneRefs {
				resp.GenerationIntent.SceneIDs = appendUniqueString(resp.GenerationIntent.SceneIDs, ref.StorySceneID)
			}
		}
		alignment.Warnings = append(alignment.Warnings, detectStoryboardFrameworkWarnings(input)...)
		if len(alignment.Warnings) > 0 {
			alignment.Status = "needs_confirmation"
			alignment.RequiresConfirmation = true
		}
		resp.FrameworkAlignment = alignment
		resp.RecommendedOptions.CanStart = resp.RecommendedOptions.CanStart && !alignment.Blocking
	}

	if draftID := strings.TrimSpace(req.TargetDraftStoryboardID); draftID != "" && s.repo != nil {
		if sb, err := s.repo.StoryboardByID(ctx, draftID); err == nil && sb != nil && sb.UserID == userID {
			s.recordStoryboardAnalyzeConversation(ctx, draftID, userID, input, resp.AssistantMessage)
		}
	}
	return resp, nil
}

func storyboardFrameworkAlignment(story *domain.Story) *domain.StoryboardFrameworkAlignment {
	alignment := &domain.StoryboardFrameworkAlignment{Status: "aligned"}
	if story == nil {
		return alignment
	}
	alignment.StoryTitle = strings.TrimSpace(story.Title)
	alignment.StoryGenre = strings.TrimSpace(story.Genre)
	alignment.StoryPremise = truncateStringToMaxRunes(strings.TrimSpace(story.Description), 180)
	if alignment.StoryPremise != "" {
		alignment.InheritedFacts = append(alignment.InheritedFacts, "遵循故事核心前提与世界设定")
	}
	if alignment.StoryGenre != "" {
		alignment.InheritedFacts = append(alignment.InheritedFacts, "保持「"+alignment.StoryGenre+"」题材基调")
	}
	return alignment
}

func applyParentStoryboardFramework(alignment *domain.StoryboardFrameworkAlignment, parent *domain.Storyboard) {
	if alignment == nil || parent == nil {
		return
	}
	alignment.ParentStoryboardTitle = strings.TrimSpace(parent.Title)
	alignment.ParentEnding = truncateStringToMaxRunes(strings.TrimSpace(firstNonEmpty(parent.ContinuationSummary, tailRunes(parent.Content, 240))), 180)
	alignment.InheritedFacts = append(alignment.InheritedFacts, "承接父故事板结尾与人物当前状态")
	if alignment.ParentEnding == "" {
		alignment.Warnings = append(alignment.Warnings, "父故事板暂时没有可用的结尾摘要，请确认本次走向能够自然承接")
	}
}

func detectStoryboardFrameworkWarnings(input string) []string {
	lower := strings.ToLower(strings.TrimSpace(input))
	if lower == "" {
		return nil
	}
	for _, marker := range []string{"推翻设定", "重置世界", "重启世界", "全部重来", "其实没死", "复活", "retcon", "不再是"} {
		if strings.Contains(lower, marker) {
			return []string{"这次描述可能改写已有设定或人物状态；继续后会把它视为你明确允许的分支变化"}
		}
	}
	return nil
}

func storyboardInputMentions(input, candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	return candidate != "" && strings.Contains(strings.ToLower(input), strings.ToLower(candidate))
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func normalizeStoryboardSceneCount(count int) int {
	if count < 2 {
		return 2
	}
	if count > 8 {
		return 8
	}
	return count
}

func inferStoryboardInputIntent(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	if lower == "" {
		return "ask_clarification"
	}
	if isCreativeInputChatOnly(lower) {
		return "chat_only"
	}
	for _, word := range []string{"换个故事板", "新故事板", "重新开始", "另一个故事板", "新建故事板", "另开分支"} {
		if strings.Contains(lower, word) {
			return "new_storyboard"
		}
	}
	for _, word := range []string{"画风", "格数", "分镜", "角色", "多格", "漫画", "续写", "分支", "走向"} {
		if strings.Contains(lower, word) {
			return "adjust_options"
		}
	}
	for _, word := range []string{"改一下", "修改", "重写", "调整", "换个", "重新生成"} {
		if strings.Contains(lower, word) {
			return "revise_current"
		}
	}
	return "new_storyboard"
}

func inferStoryboardAnalyzeStyle(input string) string {
	if strings.Contains(input, "水墨") {
		return "ink_wash"
	}
	if strings.Contains(input, "动漫") || strings.Contains(strings.ToLower(input), "anime") {
		return "anime"
	}
	if strings.Contains(input, "写实") || strings.Contains(strings.ToLower(input), "realistic") {
		return "realistic"
	}
	if strings.Contains(input, "漫画") || strings.Contains(strings.ToLower(input), "comic") {
		return "comic"
	}
	return "fantasy"
}

func inferStoryboardTopic(input string) string {
	runes := []rune(strings.TrimSpace(input))
	if len(runes) == 0 {
		return ""
	}
	if len(runes) <= 12 {
		return string(runes)
	}
	return string(runes[:12])
}

func summarizeStoryboardAnalyzeMessage(input, parentStoryboardID string) string {
	topic := inferStoryboardTopic(input)
	if strings.TrimSpace(parentStoryboardID) != "" {
		if topic == "" {
			return "好的，我会基于当前分支帮你续写下一幕。"
		}
		return fmt.Sprintf("好的，我会基于当前分支续写「%s」。", topic)
	}
	if topic == "" {
		return "我会先帮你整理故事走向和分镜规划。"
	}
	return fmt.Sprintf("我理解了，这是关于「%s」的故事板走向。", topic)
}
