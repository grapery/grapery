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
	style := strings.TrimSpace(req.Style)
	if style == "" {
		style = inferStoryboardAnalyzeStyle(input)
	}
	useComic := false
	if req.UseComicPagePipeline != nil {
		useComic = *req.UseComicPagePipeline
	}

	intentType := inferStoryboardInputIntent(input)
	resp := &domain.StoryboardAnalyzeResponse{
		AssistantMessage: summarizeStoryboardAnalyzeMessage(input, req.ParentStoryboardID),
		IntentType:       intentType,
		GenerationIntent: domain.StoryboardGenerationIntent{
			UserInput:             input,
			SceneCount:            sceneCount,
			Style:                 style,
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

	if storyID := strings.TrimSpace(req.StoryID); storyID != "" && s.repo != nil {
		if chars, err := s.repo.CharactersByStory(ctx, storyID); err == nil {
			for _, ch := range chars {
				if ch == nil {
					continue
				}
				name := strings.TrimSpace(ch.Name)
				if name == "" {
					continue
				}
				resp.CharacterCandidates = append(resp.CharacterCandidates, domain.StoryboardCharacterCandidate{
					ID:   ch.ID,
					Name: name,
					Hint: "可选入镜角色",
				})
				if len(resp.CharacterCandidates) >= 10 {
					break
				}
			}
		} else {
			s.logger.Debug("analyze: list story characters failed",
				zap.String("storyId", storyID),
				zap.Error(err))
		}
	}

	if draftID := strings.TrimSpace(req.TargetDraftStoryboardID); draftID != "" && s.repo != nil {
		if sb, err := s.repo.StoryboardByID(ctx, draftID); err == nil && sb != nil && sb.UserID == userID {
			s.recordStoryboardAnalyzeConversation(ctx, draftID, userID, input, resp.AssistantMessage)
		}
	}
	return resp, nil
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
	for _, word := range []string{"天气", "你是谁", "代码", "编程", "新闻", "股票", "笑话"} {
		if strings.Contains(lower, word) {
			return "chat_only"
		}
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
