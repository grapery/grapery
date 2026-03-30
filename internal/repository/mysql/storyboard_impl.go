package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// StoryboardByID retrieves a storyboard by ID
func (r *Repository) StoryboardByID(ctx context.Context, id string) (*domain.Storyboard, error) {
	var sb Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Where("id = ?", id).
		First(&sb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get storyboard: %w", err)
	}

	domainSb, err := r.storyboardToDomain(ctx, sb)
	if err != nil {
		return nil, err
	}
	return &domainSb, nil
}

// fateSnapshotForMySQLColumn returns JSON acceptable for MySQL JSON columns.
// An empty Go string maps to invalid SQL JSON ("document is empty"); use "{}" when unset.
func fateSnapshotForMySQLColumn(p *string) (string, error) {
	if p == nil {
		return "{}", nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return "{}", nil
	}
	if !json.Valid([]byte(t)) {
		return "", fmt.Errorf("fate_snapshot is not valid JSON")
	}
	return t, nil
}

// CreateStoryboard creates a new storyboard
func (r *Repository) CreateStoryboard(ctx context.Context, storyboard *domain.Storyboard) error {
	// Set parentID to NULL for root storyboards (empty or "__root__" marker)
	var parentID *string
	if storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker {
		parentID = &storyboard.ParentID
	}

	fateSnap, err := fateSnapshotForMySQLColumn(storyboard.FateSnapshot)
	if err != nil {
		return fmt.Errorf("invalid fate_snapshot: %w", err)
	}

	dbStoryboard := Storyboard{
		ID:               uuid.New().String(),
		StoryID:          storyboard.StoryID,
		ParentID:         parentID,
		UserID:           storyboard.UserID,
		Title:            storyboard.Title,
		Content:          storyboard.Content,
		RawInput:         storyboard.RawInput,
		Likes:            0,
		Comments:         0,
		Shares:           0,
		ForkCount:        0,
		Views:            0,
		TokenConsumption: storyboard.TokenConsumption,
		FateSnapshot:     fateSnap,
		FateSnapshotHash: storyboard.FateSnapshotHash,
	}

	if err := r.db.WithContext(ctx).Create(&dbStoryboard).Error; err != nil {
		return fmt.Errorf("failed to create storyboard: %w", err)
	}

	// 更新传入的 storyboard 对象
	storyboard.ID = dbStoryboard.ID
	storyboard.CreatedAt = dbStoryboard.CreatedAt.Unix()
	storyboard.UpdatedAt = dbStoryboard.UpdatedAt.Unix()

	// 如果有父节点（非root），更新父节点的 fork 计数
	if storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker {
		if err := r.db.WithContext(ctx).
			Model(&Storyboard{}).
			Where("id = ?", storyboard.ParentID).
			UpdateColumn("fork_count", gorm.Expr("fork_count + ?", 1)).Error; err != nil {
			r.log.Warn("failed to update parent fork count", zap.Error(err))
		}
	}

	// Persist associations
	if err := r.AttachCharactersToStoryboard(ctx, dbStoryboard.ID, storyboard.CharacterRefs); err != nil {
		return fmt.Errorf("attach storyboard characters: %w", err)
	}
	if err := r.AttachScenesToStoryboard(ctx, dbStoryboard.ID, storyboard.SceneRefs); err != nil {
		return fmt.Errorf("attach storyboard scenes: %w", err)
	}

	return nil
}

// UpdateStoryboard updates an existing storyboard
func (r *Repository) UpdateStoryboard(ctx context.Context, storyboard *domain.Storyboard) error {
	updates := map[string]interface{}{
		"title":      storyboard.Title,
		"content":    storyboard.Content,
		"raw_input":  storyboard.RawInput,
		"updated_at": time.Now().UTC(),
	}

	if err := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Where("id = ?", storyboard.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update storyboard: %w", err)
	}

	if storyboard.CharacterRefs != nil {
		if err := r.AttachCharactersToStoryboard(ctx, storyboard.ID, storyboard.CharacterRefs); err != nil {
			return fmt.Errorf("attach storyboard characters: %w", err)
		}
	}
	if storyboard.SceneRefs != nil {
		if err := r.AttachScenesToStoryboard(ctx, storyboard.ID, storyboard.SceneRefs); err != nil {
			return fmt.Errorf("attach storyboard scenes: %w", err)
		}
	}

	return nil
}

// DeleteStoryboard deletes a storyboard
func (r *Repository) DeleteStoryboard(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&Storyboard{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete storyboard: %w", err)
	}
	return nil
}

// StoryboardsByStory retrieves storyboards for a story
func (r *Repository) StoryboardsByStory(ctx context.Context, storyID string, limit, offset int) ([]*domain.Storyboard, error) {
	var storyboards []Storyboard
	query := r.db.WithContext(ctx).
		Preload("Creator").
		Where("story_id = ?", storyID).
		Order("created_at ASC") // 按创建时间升序，第一个故事板排在最前

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&storyboards).Error; err != nil {
		return nil, fmt.Errorf("failed to get storyboards: %w", err)
	}

	result := make([]*domain.Storyboard, 0, len(storyboards))
	for _, sb := range storyboards {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, err
		}
		result = append(result, &domainSb)
	}

	return result, nil
}

// RootStoryboardsByStory retrieves all published storyboards for a story
// Returns storyboards ordered by created_at DESC (newest first)
func (r *Repository) RootStoryboardsByStory(ctx context.Context, storyID string, limit, offset int) ([]*domain.Storyboard, error) {
	var storyboards []Storyboard
	query := r.db.WithContext(ctx).
		Preload("Creator").
		Where("story_id = ?", storyID).
		Where("workflow_status = ?", domain.WorkflowStatusPublished). // Only return published storyboards
		Order("created_at DESC")                                      // 最新的在前

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	r.log.Info("RootStoryboardsByStory query",
		zap.String("storyId", storyID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if err := query.Find(&storyboards).Error; err != nil {
		return nil, fmt.Errorf("failed to get root storyboards: %w", err)
	}

	r.log.Info("RootStoryboardsByStory result",
		zap.String("storyId", storyID),
		zap.Int("count", len(storyboards)))

	result := make([]*domain.Storyboard, 0, len(storyboards))
	for _, sb := range storyboards {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, err
		}
		result = append(result, &domainSb)
	}

	return result, nil
}

// StoryboardsByParent retrieves storyboards by parent ID.
// For branch continuation flow, return all children regardless of workflow status.
func (r *Repository) StoryboardsByParent(ctx context.Context, storyID, parentID string, limit, offset int) ([]*domain.Storyboard, error) {
	var storyboards []Storyboard
	query := r.db.WithContext(ctx).
		Preload("Creator").
		Where("story_id = ?", storyID).
		Where("parent_id = ?", parentID).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&storyboards).Error; err != nil {
		return nil, fmt.Errorf("failed to get storyboards by parent: %w", err)
	}

	result := make([]*domain.Storyboard, 0, len(storyboards))
	for _, sb := range storyboards {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, err
		}
		result = append(result, &domainSb)
	}

	return result, nil
}

// StoryboardsByCreator retrieves storyboards created by a specific user
func (r *Repository) StoryboardsByCreator(ctx context.Context, creatorID string, limit, offset int) ([]*domain.Storyboard, error) {
	var storyboards []Storyboard
	query := r.db.WithContext(ctx).
		Preload("Creator").
		Where("creator_id = ?", creatorID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&storyboards).Error; err != nil {
		return nil, fmt.Errorf("failed to get storyboards by creator: %w", err)
	}

	result := make([]*domain.Storyboard, 0, len(storyboards))
	for _, sb := range storyboards {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, err
		}
		result = append(result, &domainSb)
	}

	return result, nil
}

// DraftStoryboardsByCreator retrieves draft (unpublished) storyboards created by a specific user
func (r *Repository) DraftStoryboardsByCreator(ctx context.Context, creatorID string, limit, offset int) ([]*domain.Storyboard, error) {
	var storyboards []Storyboard
	query := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Where("creator_id = ?", creatorID).
		Where("workflow_status != ?", domain.WorkflowStatusPublished).
		Order("updated_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&storyboards).Error; err != nil {
		return nil, fmt.Errorf("failed to get draft storyboards by creator: %w", err)
	}

	result := make([]*domain.Storyboard, 0, len(storyboards))
	for _, sb := range storyboards {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, err
		}
		result = append(result, &domainSb)
	}

	return result, nil
}

// CountStoryboardsByCreator counts storyboards created by a specific user
func (r *Repository) CountStoryboardsByCreator(ctx context.Context, creatorID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Where("creator_id = ?", creatorID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count storyboards by creator: %w", err)
	}
	return count, nil
}

// CountStoryboardsByStory counts storyboards that belong to a given story.
func (r *Repository) CountStoryboardsByStory(ctx context.Context, storyID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Where("story_id = ?", storyID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count storyboards by story: %w", err)
	}
	return count, nil
}

// CharacterStoryboardCountsByStory returns participation counts keyed by characterID,
// counting distinct storyboard IDs within the given story.
func (r *Repository) CharacterStoryboardCountsByStory(ctx context.Context, storyID string) (map[string]int64, error) {
	type row struct {
		CharacterID string `gorm:"column:character_id"`
		Cnt         int64  `gorm:"column:cnt"`
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Table("storyboard_character_links").
		Select("storyboard_character_links.character_id as character_id, COUNT(DISTINCT storyboard_character_links.storyboard_id) as cnt").
		Joins("JOIN storyboards ON storyboards.id = storyboard_character_links.storyboard_id AND storyboards.deleted_at IS NULL").
		Where("storyboards.story_id = ?", storyID).
		Where("storyboard_character_links.character_id IS NOT NULL").
		Group("storyboard_character_links.character_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to count character storyboard participation by story: %w", err)
	}

	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		if r.CharacterID == "" {
			continue
		}
		out[r.CharacterID] = r.Cnt
	}
	return out, nil
}

// StoryboardChildren retrieves child storyboards (forks/continuations)
func (r *Repository) StoryboardChildren(ctx context.Context, parentID string) ([]*domain.Storyboard, error) {
	var storyboards []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Where("parent_id = ?", parentID).
		Where("workflow_status = ?", domain.WorkflowStatusPublished). // Only return published storyboards
		Order("created_at DESC").
		Find(&storyboards).Error; err != nil {
		return nil, fmt.Errorf("failed to get child storyboards: %w", err)
	}

	result := make([]*domain.Storyboard, 0, len(storyboards))
	for _, sb := range storyboards {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, err
		}
		result = append(result, &domainSb)
	}

	return result, nil
}

// StoryboardTree retrieves the entire tree starting from a root
func (r *Repository) StoryboardTree(ctx context.Context, rootID string) ([]*domain.Storyboard, error) {
	// 递归查询所有子节点
	var result []*domain.Storyboard
	visited := make(map[string]bool)

	var fetchTree func(parentID string) error
	fetchTree = func(parentID string) error {
		if visited[parentID] {
			return nil // 防止循环引用
		}
		visited[parentID] = true

		// 获取当前节点
		sb, err := r.StoryboardByID(ctx, parentID)
		if err != nil {
			return err
		}
		result = append(result, sb)

		// 获取子节点
		children, err := r.StoryboardChildren(ctx, parentID)
		if err != nil {
			return err
		}

		// 递归获取每个子节点的树
		for _, child := range children {
			if err := fetchTree(child.ID); err != nil {
				return err
			}
		}

		return nil
	}

	if err := fetchTree(rootID); err != nil {
		return nil, fmt.Errorf("failed to get storyboard tree: %w", err)
	}

	return result, nil
}

// ForkStoryboard creates a fork of a storyboard
func (r *Repository) ForkStoryboard(ctx context.Context, parentID, creatorID string, storyboard *domain.Storyboard) error {
	// 设置父节点
	storyboard.ParentID = parentID
	storyboard.UserID = creatorID

	// 创建新的 storyboard
	return r.CreateStoryboard(ctx, storyboard)
}

// IncrementStoryboardViews increments the view count
func (r *Repository) IncrementStoryboardViews(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + ?", 1)).Error; err != nil {
		return fmt.Errorf("failed to increment views: %w", err)
	}
	return nil
}

// IncrementStoryStoryboardCount increments the storyboard count for a story
func (r *Repository) IncrementStoryStoryboardCount(ctx context.Context, storyID string) error {
	if err := r.db.WithContext(ctx).
		Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("storyboard_count", gorm.Expr("storyboard_count + ?", 1)).Error; err != nil {
		return fmt.Errorf("failed to increment story storyboard count: %w", err)
	}
	return nil
}

// DecrementStoryStoryboardCount decrements the storyboard count for a story
func (r *Repository) DecrementStoryStoryboardCount(ctx context.Context, storyID string) error {
	if err := r.db.WithContext(ctx).
		Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("storyboard_count", gorm.Expr("GREATEST(storyboard_count - ?, 0)", 1)).Error; err != nil {
		return fmt.Errorf("failed to decrement story storyboard count: %w", err)
	}
	return nil
}

// storyboardToDomain converts database model to domain model
func (r *Repository) storyboardToDomain(ctx context.Context, sb Storyboard) (domain.Storyboard, error) {
	// 获取子节点 IDs
	var childrenIds []string
	r.db.Model(&Storyboard{}).
		Where("parent_id = ?", sb.ID).
		Pluck("id", &childrenIds)

	charRefs, err := r.storyboardCharacterLinks(ctx, sb.ID)
	if err != nil {
		return domain.Storyboard{}, fmt.Errorf("load storyboard character links: %w", err)
	}
	sceneRefs, err := r.storyboardSceneLinks(ctx, sb.ID)
	if err != nil {
		return domain.Storyboard{}, fmt.Errorf("load storyboard scene links: %w", err)
	}

	// 获取 AI 生成的场景（包含图片）
	storyboardScenes, err := r.StoryboardScenes(ctx, sb.ID)
	if err != nil {
		return domain.Storyboard{}, fmt.Errorf("load storyboard scenes: %w", err)
	}

	r.log.Debug("storyboardToDomain: loaded storyboard scenes",
		zap.String("storyboardId", sb.ID),
		zap.Int("sceneCount", len(storyboardScenes)))

	// 转换指针切片为值切片
	scenes := make([]domain.StoryboardScene, len(storyboardScenes))
	for i, scene := range storyboardScenes {
		if scene != nil {
			scenes[i] = *scene
			r.log.Debug("storyboardToDomain: scene detail",
				zap.String("sceneId", scene.ID),
				zap.String("image", scene.Image),
				zap.String("title", scene.Title))
		}
	}

	var parentID string
	if sb.ParentID != nil {
		parentID = *sb.ParentID
	}

	var storyRel *domain.Story
	if sb.Story.ID != "" {
		s := r.storyToDomain(sb.Story)
		storyRel = &s
	}

	return domain.Storyboard{
		BaseModel: common.BaseModel{
			ID:        sb.ID,
			CreatedAt: sb.CreatedAt.Unix(),
			UpdatedAt: sb.UpdatedAt.Unix(),
		},
		StoryID:        sb.StoryID,
		ParentID:       parentID,
		UserID:         sb.UserID,
		CreatorName:    sb.Creator.DisplayName,
		CreatorAvatar:  sb.Creator.Avatar,
		Title:          sb.Title,
		Content:        sb.Content,
		RawInput:       sb.RawInput,
		IsStandalone:   sb.IsStandalone,
		IsAIGenerated:  sb.IsAIGenerated,
		SceneCount:     sb.SceneCount,
		WorkflowStatus: sb.WorkflowStatus,
		CurrentStep:    sb.CurrentStep,
		EngagementStats: common.EngagementStats{
			Likes:    sb.Likes,
			Comments: sb.Comments,
			Shares:   sb.Shares,
			Views:    sb.Views,
		},
		ForkCount:        sb.ForkCount,
		TokenConsumption: sb.TokenConsumption,
		ChildrenIds:      childrenIds,
		StoryboardScenes: scenes,
		CharacterRefs:    charRefs,
		SceneRefs:        sceneRefs,
		Story:            storyRel,
	}, nil
}

// ========== StoryboardScene operations (AI-generated plot scenes) ==========

// CreateStoryboardScenes creates multiple AI-generated plot scenes for a storyboard
func (r *Repository) CreateStoryboardScenes(ctx context.Context, storyboardID string, scenes []*domain.StoryboardScene) error {
	if len(scenes) == 0 {
		return nil
	}

	dbScenes := make([]StoryboardScene, len(scenes))
	for i, scene := range scenes {
		var storySceneID *string
		if scene.StorySceneID != "" {
			storySceneID = &scene.StorySceneID
		}

		// Convert characters slice to JSON
		charactersJSON := "[]"
		if len(scene.Characters) > 0 {
			if jsonBytes, err := json.Marshal(scene.Characters); err == nil {
				charactersJSON = string(jsonBytes)
			}
		}

		// MySQL JSON columns reject "" ("The document is empty"); use minimal valid JSON.
		contextSnapshot := scene.ContextSnapshot
		if contextSnapshot == "" || !json.Valid([]byte(contextSnapshot)) {
			contextSnapshot = "{}"
		}

		dbScenes[i] = StoryboardScene{
			ID:              uuid.NewString(),
			StoryboardID:    storyboardID,
			StorySceneID:    storySceneID,
			Sequence:        scene.Sequence,
			Title:           scene.Title,
			Description:     scene.Description,
			Image:           scene.Image,
			Location:        scene.Location,
			TimeOfDay:       scene.TimeOfDay,
			Characters:      charactersJSON,
			Mood:            scene.Mood,
			IsAIGenerated:   scene.IsAIGenerated,
			ContextSnapshot: contextSnapshot,
		}
		// Also update the domain object with the generated ID
		scenes[i].ID = dbScenes[i].ID
		scenes[i].StoryboardID = storyboardID
	}

	if err := r.db.WithContext(ctx).Create(&dbScenes).Error; err != nil {
		return fmt.Errorf("failed to create storyboard scenes: %w", err)
	}

	r.log.Info("storyboard scenes created",
		zap.String("storyboardId", storyboardID),
		zap.Int("count", len(scenes)))

	return nil
}

// StoryboardScenes retrieves all AI-generated plot scenes for a storyboard
func (r *Repository) StoryboardScenes(ctx context.Context, storyboardID string) ([]*domain.StoryboardScene, error) {
	var dbScenes []StoryboardScene
	if err := r.db.WithContext(ctx).
		Preload("StoryScene").
		Where("storyboard_id = ?", storyboardID).
		Order("sequence ASC").
		Find(&dbScenes).Error; err != nil {
		return nil, fmt.Errorf("failed to get storyboard scenes: %w", err)
	}

	scenes := make([]*domain.StoryboardScene, len(dbScenes))
	for i, dbScene := range dbScenes {
		scenes[i] = r.storyboardSceneToDomain(dbScene)
	}

	return scenes, nil
}

// DeleteStoryboardScenes deletes all AI-generated plot scenes for a storyboard
func (r *Repository) DeleteStoryboardScenes(ctx context.Context, storyboardID string) error {
	if err := r.db.WithContext(ctx).
		Where("storyboard_id = ?", storyboardID).
		Delete(&StoryboardScene{}).Error; err != nil {
		return fmt.Errorf("failed to delete storyboard scenes: %w", err)
	}

	r.log.Info("storyboard scenes deleted", zap.String("storyboardId", storyboardID))
	return nil
}

// UpdateStoryboardScene updates a single storyboard scene
func (r *Repository) UpdateStoryboardScene(ctx context.Context, scene *domain.StoryboardScene) error {
	var storySceneID *string
	if scene.StorySceneID != "" {
		storySceneID = &scene.StorySceneID
	}

	charactersJSON := "[]"
	if len(scene.Characters) > 0 {
		if jsonBytes, err := json.Marshal(scene.Characters); err == nil {
			charactersJSON = string(jsonBytes)
		}
	}

	videoSegmentsJSON := ""
	if len(scene.VideoSegments) > 0 {
		if jsonBytes, err := json.Marshal(scene.VideoSegments); err == nil {
			videoSegmentsJSON = string(jsonBytes)
		}
	}

	middleFrameURLsJSON := ""
	if len(scene.MiddleFrameURLs) > 0 {
		if jsonBytes, err := json.Marshal(scene.MiddleFrameURLs); err == nil {
			middleFrameURLsJSON = string(jsonBytes)
		}
	}

	updates := map[string]interface{}{
		"story_scene_id":      storySceneID,
		"sequence":            scene.Sequence,
		"title":               scene.Title,
		"description":         scene.Description,
		"image":               scene.Image,
		"location":            scene.Location,
		"time_of_day":         scene.TimeOfDay,
		"characters":          charactersJSON,
		"mood":                scene.Mood,
		"is_subdivided":       scene.IsSubdivided,
		"video_segments_json": videoSegmentsJSON,
		"middle_frame_urls":   middleFrameURLsJSON,
	}

	if err := r.db.WithContext(ctx).
		Model(&StoryboardScene{}).
		Where("id = ?", scene.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update storyboard scene: %w", err)
	}

	return nil
}

// UpdateStoryboardSceneImage updates only the image field of a storyboard scene
func (r *Repository) UpdateStoryboardSceneImage(ctx context.Context, sceneID, imageURL string) error {
	if err := r.db.WithContext(ctx).
		Model(&StoryboardScene{}).
		Where("id = ?", sceneID).
		Update("image", imageURL).Error; err != nil {
		return fmt.Errorf("failed to update storyboard scene image: %w", err)
	}

	r.log.Info("storyboard scene image updated",
		zap.String("sceneId", sceneID),
		zap.String("imageUrl", imageURL))
	return nil
}

// UpdateStoryboardSceneVideo updates only the video_url field of a storyboard scene
func (r *Repository) UpdateStoryboardSceneVideo(ctx context.Context, sceneID, videoURL string) error {
	if err := r.db.WithContext(ctx).
		Model(&StoryboardScene{}).
		Where("id = ?", sceneID).
		Update("video_url", videoURL).Error; err != nil {
		return fmt.Errorf("failed to update storyboard scene video: %w", err)
	}

	r.log.Info("storyboard scene video updated",
		zap.String("sceneId", sceneID),
		zap.String("videoUrl", videoURL))
	return nil
}

// UpdateStoryboardSceneVideoWithSubdivision updates video URL and subdivision info for a scene
func (r *Repository) UpdateStoryboardSceneVideoWithSubdivision(ctx context.Context, sceneID, videoURL string, isSubdivided bool, videoSegmentsJSON, middleFrameURLsJSON string) error {
	updates := map[string]interface{}{
		"video_url":           videoURL,
		"is_subdivided":       isSubdivided,
		"video_segments_json": videoSegmentsJSON,
		"middle_frame_urls":   middleFrameURLsJSON,
	}

	if err := r.db.WithContext(ctx).
		Model(&StoryboardScene{}).
		Where("id = ?", sceneID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update storyboard scene video with subdivision: %w", err)
	}

	r.log.Info("storyboard scene video with subdivision updated",
		zap.String("sceneId", sceneID),
		zap.String("videoUrl", videoURL),
		zap.Bool("isSubdivided", isSubdivided))
	return nil
}

// storyboardSceneToDomain converts database StoryboardScene to domain StoryboardScene
func (r *Repository) storyboardSceneToDomain(dbScene StoryboardScene) *domain.StoryboardScene {
	scene := &domain.StoryboardScene{
		BaseModel: common.BaseModel{
			ID:        dbScene.ID,
			CreatedAt: dbScene.CreatedAt.Unix(),
			UpdatedAt: dbScene.UpdatedAt.Unix(),
		},
		StoryboardID:  dbScene.StoryboardID,
		Sequence:      dbScene.Sequence,
		Title:         dbScene.Title,
		Description:   dbScene.Description,
		Image:         dbScene.Image,
		VideoUrl:      dbScene.VideoUrl,
		Location:      dbScene.Location,
		TimeOfDay:     dbScene.TimeOfDay,
		Mood:          dbScene.Mood,
		IsAIGenerated: dbScene.IsAIGenerated,
		IsSubdivided:      dbScene.IsSubdivided,
		ContextSnapshot:   dbScene.ContextSnapshot,
	}

	if dbScene.StorySceneID != nil {
		scene.StorySceneID = *dbScene.StorySceneID
	}

	// Parse characters JSON
	if dbScene.Characters != "" && dbScene.Characters != "[]" {
		var characters []string
		if err := json.Unmarshal([]byte(dbScene.Characters), &characters); err == nil {
			scene.Characters = characters
		}
	}

	// Parse video segments JSON
	if dbScene.VideoSegmentsJSON != "" && dbScene.VideoSegmentsJSON != "[]" {
		var segments []domain.VideoSegmentInfo
		if err := json.Unmarshal([]byte(dbScene.VideoSegmentsJSON), &segments); err == nil {
			scene.VideoSegments = segments
		}
	}

	// Parse middle frame URLs JSON
	if dbScene.MiddleFrameURLs != "" && dbScene.MiddleFrameURLs != "[]" {
		var urls []string
		if err := json.Unmarshal([]byte(dbScene.MiddleFrameURLs), &urls); err == nil {
			scene.MiddleFrameURLs = urls
		}
	}

	// Convert related StoryScene if preloaded
	if dbScene.StoryScene != nil {
		scene.StoryScene = &domain.StoryScene{
			ID:          dbScene.StoryScene.ID,
			StoryID:     dbScene.StoryScene.StoryID,
			Title:       dbScene.StoryScene.Title,
			Description: dbScene.StoryScene.Description,
			Image:       dbScene.StoryScene.Image,
			Location:    dbScene.StoryScene.Location,
			TimeOfDay:   dbScene.StoryScene.TimeOfDay,
		}
	}

	return scene
}

// StoryboardFeed retrieves published storyboards from the community feed, ordered by created_at DESC
func (r *Repository) StoryboardFeed(ctx context.Context, limit, offset int) ([]*domain.Storyboard, int64, error) {
	var storyboards []Storyboard
	var total int64

	// Count total published storyboards
	if err := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Where("workflow_status = ?", domain.WorkflowStatusPublished).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count storyboard feed: %w", err)
	}

	// Fetch published storyboards ordered by created_at DESC
	query := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Where("workflow_status = ?", domain.WorkflowStatusPublished).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&storyboards).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get storyboard feed: %w", err)
	}

	r.log.Info("StoryboardFeed result",
		zap.Int("count", len(storyboards)),
		zap.Int64("total", total),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	result := make([]*domain.Storyboard, 0, len(storyboards))
	for _, sb := range storyboards {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, &domainSb)
	}

	return result, total, nil
}

// StoryboardFeedFromFollowedStories returns published storyboards for stories the user follows,
// ordered by storyboard activity (updated_at) descending.
func (r *Repository) StoryboardFeedFromFollowedStories(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if userID == "" {
		return []*domain.Storyboard{}, 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	base := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Joins("JOIN story_follows sf ON sf.story_id = storyboards.story_id AND sf.user_id = ?", userID).
		Joins("JOIN stories ON stories.id = storyboards.story_id").
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("storyboard feed following count: %w", err)
	}

	var rows []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Preload("Story.Author").
		Model(&Storyboard{}).
		Joins("JOIN story_follows sf ON sf.story_id = storyboards.story_id AND sf.user_id = ?", userID).
		Joins("JOIN stories ON stories.id = storyboards.story_id").
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished).
		Order("storyboards.updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("storyboard feed following: %w", err)
	}

	out := make([]*domain.Storyboard, 0, len(rows))
	for _, sb := range rows {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, &domainSb)
	}
	return out, total, nil
}
