package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

func (s *Service) isStoryboardContinuation(storyboard *domain.Storyboard) bool {
	return !storyboard.IsStandalone && storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker
}

func (s *Service) canGenerateStoryboardText() bool {
	return s.aiGenService != nil
}

func (s *Service) canGenerateStoryboardSceneImages() bool {
	return s.aiGenService != nil && s.genAPI != nil
}

// invalidateStoryboardDetailAndListCaches drops the storyboard detail key and list keys for the story
// so continuation_summary and other row fields are not served stale from cache.
func (s *Service) invalidateStoryboardDetailAndListCaches(ctx context.Context, storyboardID, storyID string) {
	c := s.getCache()
	if c == nil || storyboardID == "" {
		return
	}
	if err := c.Delete(ctx, cache.StoryboardKey(storyboardID)); err != nil {
		s.logger.Warn("failed to invalidate storyboard cache",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	}
	if storyID == "" {
		return
	}
	for limit := 20; limit <= 100; limit += 20 {
		for offset := 0; offset < 200; offset += limit {
			_ = c.Delete(ctx, cache.StoryboardsListKey(storyID, limit, offset))
		}
	}
}

// invalidateParentStoryboardCaches busts parent detail, children list, and by-parent list caches after a child publishes.
func (s *Service) invalidateParentStoryboardCaches(ctx context.Context, parentID, storyID string) {
	c := s.getCache()
	if c == nil {
		return
	}
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return
	}
	_ = c.Delete(ctx, cache.StoryboardKey(parentID))
	_ = c.Delete(ctx, cache.StoryboardKey(parentID)+":children")
	if storyID == "" {
		return
	}
	for limit := 20; limit <= 100; limit += 20 {
		for offset := 0; offset < 200; offset += limit {
			_ = c.Delete(ctx, cache.StoryboardsListKey(storyID+"_parent_"+parentID, limit, offset))
		}
	}
}

// storyboardParentIDForReparent normalizes a storyboard's parent for child reparenting on delete (root → empty).
func storyboardParentIDForReparent(parentID string) string {
	p := strings.TrimSpace(parentID)
	if p == domain.StoryboardRootMarker {
		return ""
	}
	return p
}

// invalidateStoryboardCachesAfterDelete busts caches after delete + child reparent.
func (s *Service) invalidateStoryboardCachesAfterDelete(ctx context.Context, deletedID, storyID, grandparentID string, reparentedChildIDs []string) {
	c := s.getCache()
	deletedID = strings.TrimSpace(deletedID)
	storyID = strings.TrimSpace(storyID)
	grandparentID = strings.TrimSpace(grandparentID)

	if c != nil && deletedID != "" {
		_ = c.Delete(ctx, cache.StoryboardKey(deletedID))
		_ = c.Delete(ctx, cache.StoryboardKey(deletedID)+":children")
		_ = c.Delete(ctx, cache.StoryboardKey(deletedID)+":tree")
	}
	if c != nil && storyID != "" {
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.StoryboardsListKey(storyID, limit, offset))
				if deletedID != "" {
					_ = c.Delete(ctx, cache.StoryboardsListKey(storyID+"_parent_"+deletedID, limit, offset))
				}
				if grandparentID != "" {
					_ = c.Delete(ctx, cache.StoryboardsListKey(storyID+"_parent_"+grandparentID, limit, offset))
				}
			}
		}
	}
	if grandparentID != "" {
		if err := s.repo.RecountParentPublishedForkCount(ctx, grandparentID); err != nil {
			s.logger.Warn("failed to recount grandparent fork count after storyboard delete",
				zap.String("grandparentId", grandparentID),
				zap.String("deletedId", deletedID),
				zap.Error(err))
		}
		s.invalidateParentStoryboardCaches(ctx, grandparentID, storyID)
		s.invalidateStoryboardDetailAndListCaches(ctx, grandparentID, storyID)
	}
	for _, childID := range reparentedChildIDs {
		childID = strings.TrimSpace(childID)
		if childID == "" {
			continue
		}
		s.invalidateStoryboardDetailAndListCaches(ctx, childID, storyID)
	}
	if c := s.getCache(); c != nil && deletedID != "" {
		deleteCommentsListCacheForTarget(ctx, c, "storyboard", deletedID)
	}
}

// removeStoryboardFromDefaultPath drops a deleted node from the story's default path and clears path marks.
func (s *Service) removeStoryboardFromDefaultPath(ctx context.Context, storyID, deletedStoryboardID string) {
	storyID = strings.TrimSpace(storyID)
	deletedStoryboardID = strings.TrimSpace(deletedStoryboardID)
	if storyID == "" || deletedStoryboardID == "" {
		return
	}
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil || story == nil || len(story.DefaultPathNodeIDs) == 0 {
		return
	}
	found := false
	newIDs := make([]string, 0, len(story.DefaultPathNodeIDs))
	for _, nodeID := range story.DefaultPathNodeIDs {
		if nodeID == deletedStoryboardID {
			found = true
			continue
		}
		newIDs = append(newIDs, nodeID)
	}
	if !found {
		return
	}
	story.DefaultPathNodeIDs = newIDs
	now := time.Now().Unix()
	story.DefaultPathUpdatedAt = &now
	if err := s.repo.UpdateStory(ctx, story); err != nil {
		s.logger.Warn("failed to update story default path after storyboard delete",
			zap.String("storyId", storyID),
			zap.String("deletedStoryboardId", deletedStoryboardID),
			zap.Error(err))
		return
	}
	s.syncStoryboardDefaultPathMarks(ctx, storyID, newIDs)
}

func (s *Service) syncStoryboardDefaultPathMarks(ctx context.Context, storyID string, nodeIDs []string) {
	storyboards, err := s.repo.StoryboardsByStory(ctx, storyID, 1000, 0)
	if err != nil {
		s.logger.Warn("failed to load storyboards for default path mark sync",
			zap.String("storyId", storyID),
			zap.Error(err))
		return
	}
	orderMap := make(map[string]int, len(nodeIDs))
	for i, nodeID := range nodeIDs {
		orderMap[nodeID] = i + 1
	}
	for _, sb := range storyboards {
		if sb == nil {
			continue
		}
		if order, ok := orderMap[sb.ID]; ok {
			sb.IsInDefaultPath = true
			sb.DefaultPathOrder = order
		} else {
			sb.IsInDefaultPath = false
			sb.DefaultPathOrder = 0
		}
		if err := s.repo.UpdateStoryboard(ctx, sb); err != nil {
			s.logger.Warn("failed to sync storyboard default path mark",
				zap.String("storyboardId", sb.ID),
				zap.Error(err))
		}
	}
}

// onChildStoryboardPublished updates parent fork stats, caches, and optional fork notification when a child is first published.
func (s *Service) onChildStoryboardPublished(ctx context.Context, child *domain.Storyboard) {
	if child == nil {
		return
	}
	parentID := strings.TrimSpace(child.ParentID)
	if parentID == "" || parentID == domain.StoryboardRootMarker {
		return
	}
	if err := s.repo.RecountParentPublishedForkCount(ctx, parentID); err != nil {
		s.logger.Warn("failed to recount parent fork count on child publish",
			zap.String("parentId", parentID),
			zap.String("childId", child.ID),
			zap.Error(err))
	}
	s.invalidateParentStoryboardCaches(ctx, parentID, child.StoryID)
	s.invalidateStoryboardDetailAndListCaches(ctx, parentID, child.StoryID)

	if s.metrics != nil {
		parent, err := s.repo.StoryboardByID(ctx, parentID)
		if err == nil && parent != nil {
			s.metrics.RecordStoryboardChildCount(parentID, float64(parent.ForkCount))
		}
	}

	parent, err := s.repo.StoryboardByID(ctx, parentID)
	if err != nil || parent == nil || parent.UserID == child.UserID {
		return
	}
	creator, err := s.repo.UserByID(ctx, child.UserID)
	if err != nil {
		return
	}
	if err := s.NotifyStoryboardForked(ctx,
		parent.UserID,
		child.UserID,
		creator.DisplayName,
		creator.Avatar,
		child.StoryID,
		parentID,
		child.ID); err != nil {
		s.logger.Warn("failed to send storyboard forked notification on publish",
			zap.Error(err),
			zap.String("parentId", parentID),
			zap.String("childId", child.ID))
	}
}

// refreshContinuationSummaryAfterScenesChanged clears persisted summary, busts caches, and asynchronously
// regenerates summary when storyboard has body text and text AI is available.
func (s *Service) refreshContinuationSummaryAfterScenesChanged(ctx context.Context, storyboardID string) {
	if err := s.repo.UpdateStoryboardContinuationSummary(ctx, storyboardID, ""); err != nil {
		s.logger.Warn("failed to clear continuation summary after scene change",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	}
	sb, err := s.repo.StoryboardByID(ctx, storyboardID)
	storyID := ""
	if err == nil && sb != nil {
		storyID = sb.StoryID
	}
	s.invalidateStoryboardDetailAndListCaches(ctx, storyboardID, storyID)
	if err == nil && sb != nil && strings.TrimSpace(sb.Content) != "" && s.canGenerateStoryboardText() {
		s.regenerateStoryboardContinuationSummaryAsync(storyboardID)
	}
}

func tailRunes(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(text)
	if len(r) <= maxRunes {
		return text
	}
	return string(r[len(r)-maxRunes:])
}

func (s *Service) buildStoryboardContext(ctx context.Context, storyboard *domain.Storyboard, story *domain.Story) string {
	continuation := s.isStoryboardContinuation(storyboard)
	s.logger.Debug("building storyboard context",
		zap.String("storyboardId", storyboard.ID),
		zap.String("storyId", story.ID),
		zap.Bool("isStandalone", storyboard.IsStandalone),
		zap.Bool("continuation", continuation))

	var b strings.Builder
	b.WriteString(fmt.Sprintf("故事标题: %s\n", story.Title))
	b.WriteString(fmt.Sprintf("故事简介: %s\n\n", story.Description))

	if storyboard.IsStandalone {
		b.WriteString("（独立故事线，不参考前情）\n\n")
		s.logger.Debug("storyboard is standalone, skipping ancestor context",
			zap.String("storyboardId", storyboard.ID))
	} else if continuation {
		ancestors := s.getAncestorStoryboards(ctx, storyboard, 5)
		var parent *domain.Storyboard
		if len(ancestors) > 0 {
			parent = ancestors[len(ancestors)-1]
		}
		if parent != nil && parent.StoryID != storyboard.StoryID {
			s.logger.Warn("parent storyboard storyId mismatch, skipping continuation anchor",
				zap.String("storyboardId", storyboard.ID),
				zap.String("parentId", parent.ID))
			parent = nil
		}

		hasParentSummary := parent != nil && strings.TrimSpace(parent.ContinuationSummary) != ""
		hasParentTail := parent != nil && strings.TrimSpace(parent.Content) != ""
		tailSceneCount := 0
		hasFateParent := parent != nil && parent.FateSnapshot != nil && strings.TrimSpace(*parent.FateSnapshot) != ""
		hasFateSelf := storyboard.FateSnapshot != nil && strings.TrimSpace(*storyboard.FateSnapshot) != ""

		if hasFateParent {
			snap := truncateStringToMaxRunes(strings.TrimSpace(*parent.FateSnapshot), fateSnapshotMaxRunes)
			b.WriteString("【分叉时刻角色状态（父节点快照）】\n")
			b.WriteString(snap)
			b.WriteString("\n\n")
		}
		if hasFateSelf {
			snap := truncateStringToMaxRunes(strings.TrimSpace(*storyboard.FateSnapshot), fateSnapshotMaxRunes)
			b.WriteString("【分叉时刻角色状态（当前续写节点）】\n")
			b.WriteString(snap)
			b.WriteString("\n\n")
		}

		if len(storyboard.CharacterRefs) > 0 {
			b.WriteString("【用户本次指定优先角色】\n")
			for _, ref := range storyboard.CharacterRefs {
				line := fmt.Sprintf("- 角色ID: %s", ref.CharacterID)
				if ch, err := s.repo.CharacterByID(ctx, ref.CharacterID); err == nil && ch != nil {
					line = fmt.Sprintf("- %s [角色ID: %s]", ch.Name, ch.ID)
					if ch.Description != "" {
						line += fmt.Sprintf("：%s", truncateStringToMaxRunes(ch.Description, userRefDescriptionMaxRunes))
					}
				}
				b.WriteString(line)
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}

		if len(storyboard.SceneRefs) > 0 {
			b.WriteString("【用户本次指定优先场景（故事地点）】\n")
			for _, ref := range storyboard.SceneRefs {
				line := fmt.Sprintf("- 场景ID: %s", ref.StorySceneID)
				if sc, err := s.repo.StorySceneByID(ctx, story.ID, ref.StorySceneID); err == nil && sc != nil {
					line = fmt.Sprintf("- %s [场景ID: %s]", sc.Title, sc.ID)
					if sc.Description != "" {
						line += fmt.Sprintf("：%s", truncateStringToMaxRunes(sc.Description, userRefDescriptionMaxRunes))
					}
					if sc.Location != "" {
						line += fmt.Sprintf(" (地点: %s)", sc.Location)
					}
				}
				b.WriteString(line)
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}

		if parent != nil {
			summaryRunes := utf8.RuneCountInString(strings.TrimSpace(parent.ContinuationSummary))
			tailBudget := parentContentTailMaxRunes
			if summaryRunes > 400 {
				tailBudget = parentContentTailMaxRunes / 2
			}
			b.WriteString("【直接父节点续写锚点】\n")
			if s := strings.TrimSpace(parent.ContinuationSummary); s != "" {
				b.WriteString("前情摘要：\n")
				b.WriteString(truncateStringToMaxRunes(s, parentSummaryBlockMaxRunes))
				b.WriteString("\n\n")
			}
			if tail := strings.TrimSpace(parent.Content); tail != "" && tailBudget > 0 {
				b.WriteString("父正文尾部：\n")
				b.WriteString(tailRunes(tail, tailBudget))
				b.WriteString("\n\n")
			}
			scenes, err := s.repo.StoryboardScenes(ctx, parent.ID)
			if err == nil && len(scenes) > 0 {
				sort.Slice(scenes, func(i, j int) bool {
					return scenes[i].Sequence < scenes[j].Sequence
				})
				k := parentTailSceneCount
				if k > len(scenes) {
					k = len(scenes)
				}
				tailScenes := scenes[len(scenes)-k:]
				tailSceneCount = len(tailScenes)
				b.WriteString("父节点末尾分镜：\n")
				for _, sc := range tailScenes {
					if sc == nil {
						continue
					}
					desc := truncateStringToMaxRunes(strings.TrimSpace(sc.Description), 180)
					b.WriteString(fmt.Sprintf("- %s：%s\n", sc.Title, desc))
				}
				b.WriteString("\n")
			}
		}

		if len(ancestors) > 1 {
			b.WriteString("【更早的前情（压缩）】\n")
			for i, a := range ancestors[:len(ancestors)-1] {
				if sum := strings.TrimSpace(a.ContinuationSummary); sum != "" {
					b.WriteString(fmt.Sprintf("- 第%d段摘要《%s》：%s\n",
						i+1, a.Title, truncateStringToMaxRunes(sum, ancestorSummaryMaxRunes)))
				} else {
					excerpt := truncateStringToMaxRunes(strings.TrimSpace(a.Content), ancestorSummaryMaxRunes)
					b.WriteString(fmt.Sprintf("- 第%d段《%s》：%s\n", i+1, a.Title, excerpt))
				}
			}
			b.WriteString("\n")
		}

		s.logger.Info("continuation storyboard context blocks",
			zap.String("storyboardId", storyboard.ID),
			zap.Bool("parentSummary", hasParentSummary),
			zap.Bool("parentContentTail", hasParentTail),
			zap.Int("parentTailScenes", tailSceneCount),
			zap.Bool("fateParent", hasFateParent),
			zap.Bool("fateSelf", hasFateSelf),
			zap.Int("userCharacterRefs", len(storyboard.CharacterRefs)),
			zap.Int("userSceneRefs", len(storyboard.SceneRefs)),
			zap.Int("contextRunes", utf8.RuneCountInString(b.String())))
	} else if storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker {
		ancestors := s.getAncestorStoryboards(ctx, storyboard, 5)
		if len(ancestors) > 0 {
			s.logger.Debug("adding ancestor context (non-continuation branch)",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("ancestorCount", len(ancestors)))
			b.WriteString("前情提要（按时间顺序）：\n")
			for i, ancestor := range ancestors {
				ancestorContent := truncateForLog(ancestor.Content, 300)
				b.WriteString(fmt.Sprintf("\n【第%d章 - %s】\n%s\n", i+1, ancestor.Title, ancestorContent))
			}
			b.WriteString("\n")
		}
	}

	characters, err := s.repo.CharactersByStory(ctx, story.ID)
	if err == nil && len(characters) > 0 {
		s.logger.Debug("adding all story characters to context",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("characterCount", len(characters)))
		b.WriteString("故事中的可用角色（AI 请根据剧情需要智能选择）：\n")
		for _, char := range characters {
			b.WriteString(fmt.Sprintf("- %s [角色ID: %s]", char.Name, char.ID))
			if char.Description != "" {
				b.WriteString(fmt.Sprintf(": %s", truncateForLog(char.Description, 100)))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString("重要提示：AI 应根据用户描述的故事情节，智能选择合适的角色。只有确实参与场景的角色才需要包含在characters数组中，每个角色对象必须包含name（角色完整名称）和id（对应的角色ID）。\n\n")
	}

	scenes, err := s.repo.StoryScenes(ctx, story.ID, 100, 0)
	if err == nil && len(scenes) > 0 {
		s.logger.Debug("adding all story scenes to context",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneCount", len(scenes)))
		b.WriteString("故事中的可用场景地点（AI 请根据剧情需要智能选择）：\n")
		for _, scene := range scenes {
			b.WriteString(fmt.Sprintf("- %s [场景ID: %s]", scene.Title, scene.ID))
			if scene.Description != "" {
				b.WriteString(fmt.Sprintf(": %s", truncateForLog(scene.Description, 100)))
			}
			if scene.Location != "" {
				b.WriteString(fmt.Sprintf(" (地点: %s)", scene.Location))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString("重要提示：AI 应根据用户描述的故事情节，智能选择合适的场景。如果场景地点与剧情相关，应在storySceneId字段中提供对应的场景ID。\n\n")
	}

	out := b.String()
	s.logger.Debug("storyboard context built",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("contextLength", len(out)),
		zap.Int("contextRunes", utf8.RuneCountInString(out)))
	return out
}

func (s *Service) generateOrRefreshStoryboardSummary(ctx context.Context, storyboardID string) {
	if s.aiGenService == nil {
		return
	}
	sb, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		s.logger.Warn("continuation summary skipped: storyboard load failed",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return
	}
	if strings.TrimSpace(sb.Content) == "" {
		if err := s.repo.UpdateStoryboardContinuationSummary(ctx, storyboardID, ""); err != nil {
			s.logger.Warn("failed to clear continuation summary",
				zap.String("storyboardId", storyboardID),
				zap.Error(err))
		}
		s.invalidateStoryboardDetailAndListCaches(ctx, storyboardID, sb.StoryID)
		return
	}

	scenes, err := s.repo.StoryboardScenes(ctx, storyboardID)
	if err != nil {
		s.logger.Warn("continuation summary: scenes load failed, using content only",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		scenes = nil
	}
	if len(scenes) > 1 {
		sort.Slice(scenes, func(i, j int) bool {
			return scenes[i].Sequence < scenes[j].Sequence
		})
	}
	var sceneExcerpt strings.Builder
	n := len(scenes)
	start := 0
	if n > 6 {
		start = n - 6
	}
	for i := start; i < n; i++ {
		sc := scenes[i]
		if sc == nil {
			continue
		}
		sceneExcerpt.WriteString(fmt.Sprintf("- %s: %s\n", sc.Title, truncateForLog(sc.Description, 240)))
	}

	systemPrompt := `你是专业叙事编辑。只输出一段连续的中文摘要正文，不要标题、不要引号包裹、不要 Markdown。`
	userPrompt := fmt.Sprintf(`以下是一则故事板已落库的正文与部分分镜标题/描述。请用第三人称写一段客观、克制的叙事摘要（中文）。

要求：
- 只概括已发生的关键事实与结局走向，不要引入新剧情或新角色关系。
- 不要列表式道德评价；不要预测下一章。
- 输出严格控制在 %d 个 Unicode 字符（标量）以内。

【正文】
%s

【分镜摘录】
%s`, continuationSummaryMaxOutputRunes, sb.Content, sceneExcerpt.String())

	genReq := &GenerateTextRequest{
		UserID:            sb.UserID,
		OriginalPrompt:    userPrompt,
		SystemPrompt:      systemPrompt,
		Model:             "gemini-2.5-flash",
		Temperature:       0.25,
		MaxTokens:         1024,
		RelatedEntityID:   sb.ID,
		RelatedEntityType: "storyboard",
		Metadata: map[string]interface{}{
			"step":         "storyboard_continuation_summary",
			"storyboardId": sb.ID,
			"storyId":      sb.StoryID,
		},
	}
	res, err := s.aiGenService.GenerateText(ctx, genReq)
	if err != nil {
		s.logger.Warn("continuation summary generation failed",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return
	}
	summary := strings.TrimSpace(res.Text)
	summary = truncateStringToMaxRunes(summary, continuationSummaryMaxOutputRunes)
	if summary == "" {
		s.logger.Warn("continuation summary empty after generation",
			zap.String("storyboardId", storyboardID))
		return
	}
	if err := s.repo.UpdateStoryboardContinuationSummary(ctx, storyboardID, summary); err != nil {
		s.logger.Warn("failed to persist continuation summary",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return
	}
	s.invalidateStoryboardDetailAndListCaches(ctx, storyboardID, sb.StoryID)
	s.logger.Info("continuation summary persisted",
		zap.String("storyboardId", storyboardID),
		zap.Int("summaryRunes", utf8.RuneCountInString(summary)))
}

func (s *Service) regenerateStoryboardContinuationSummaryAsync(storyboardID string) {
	go func() {
		ctx := context.Background()
		s.generateOrRefreshStoryboardSummary(ctx, storyboardID)
	}()
}

func truncateStoryboardContentToMaxRunes(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(text)
	if len(r) <= maxRunes {
		return text
	}
	slice := r[:maxRunes]
	best := -1
	for i := len(slice) - 1; i >= maxRunes/2; i-- {
		ch := slice[i]
		if ch == '。' || ch == '！' || ch == '？' || ch == '.' || ch == '!' || ch == '?' {
			best = i
			break
		}
	}
	if best >= maxRunes/2 {
		return string(slice[:best+1])
	}
	return string(slice) + "…"
}
