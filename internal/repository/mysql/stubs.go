package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// 这个文件包含所有其他repository方法的实现
// 这些方法补充了其他impl文件中未实现的方法


// ========== Story operations ==========

func (r *Repository) StoriesByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*domain.Story, error) {
	var stories []Story
	err := r.db.WithContext(ctx).
		Preload("Author").
		Where("author_id = ?", authorID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&stories).Error
	if err != nil {
		return nil, err
	}

	// Collect story IDs for character count query
	storyIDs := make([]string, len(stories))
	for i, s := range stories {
		storyIDs[i] = s.ID
	}

	// Get character counts for all stories
	characterCounts := make(map[string]int)
	if len(storyIDs) > 0 {
		type CharacterCountResult struct {
			StoryID string
			Count   int
		}
		var counts []CharacterCountResult
		r.db.WithContext(ctx).Model(&Character{}).
			Select("story_id, COUNT(*) as count").
			Where("story_id IN ?", storyIDs).
			Group("story_id").
			Scan(&counts)
		for _, c := range counts {
			characterCounts[c.StoryID] = c.Count
		}
	}

	result := make([]*domain.Story, len(stories))
	for i := range stories {
		story := ModelToStory(&stories[i])
		story.CharacterCount = characterCounts[stories[i].ID]
		result[i] = story
	}
	return result, nil
}

func (r *Repository) TrendingStories(ctx context.Context, limit int) ([]*domain.Story, error) {
	var stories []Story

	// Default/cap: trending is non-paginated and should not exceed 20.
	if limit <= 0 {
		limit = 20
	}
	if limit > 20 {
		limit = 20
	}

	// Simple hotness ordering: followers > likes > updated recency.
	// No time range limit - includes all published stories.
	err := r.db.WithContext(ctx).
		Preload("Author").
		Where("status = ?", "published").
		Order("followers DESC, likes DESC, updated_at DESC").
		Limit(limit).
		Find(&stories).Error
	if err != nil {
		return nil, err
	}

	// Collect story IDs for character count query
	storyIDs := make([]string, len(stories))
	for i, s := range stories {
		storyIDs[i] = s.ID
	}

	// Get character counts for all stories
	characterCounts := make(map[string]int)
	if len(storyIDs) > 0 {
		type CharacterCountResult struct {
			StoryID string
			Count   int
		}
		var counts []CharacterCountResult
		r.db.WithContext(ctx).Model(&Character{}).
			Select("story_id, COUNT(*) as count").
			Where("story_id IN ?", storyIDs).
			Group("story_id").
			Scan(&counts)
		for _, c := range counts {
			characterCounts[c.StoryID] = c.Count
		}
	}

	result := make([]*domain.Story, len(stories))
	for i := range stories {
		story := ModelToStory(&stories[i])
		story.CharacterCount = characterCounts[stories[i].ID]
		result[i] = story
	}
	return result, nil
}

// ========== Panel operations ==========

func (r *Repository) PanelByID(ctx context.Context, id string) (*domain.Panel, error) {
	var panel Panel
	err := r.db.WithContext(ctx).
		Preload("Story").
		Preload("Story.Author").
		First(&panel, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToPanel(&panel), nil
}

func (r *Repository) CreatePanel(ctx context.Context, panel *domain.Panel) error {
	dbPanel := PanelToModel(panel)
	if dbPanel.ID == "" {
		dbPanel.ID = uuid.New().String()
	}
	dbPanel.CreatedAt = time.Now()
	dbPanel.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Create(dbPanel).Error; err != nil {
		return err
	}

	// 更新故事的面板数
	if err := r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", panel.StoryID).
		UpdateColumn("panels", gorm.Expr("panels + ?", 1)).Error; err != nil {
		return err
	}

	panel.ID = dbPanel.ID
	panel.CreatedAt = timeToUnix(dbPanel.CreatedAt)
	return nil
}

func (r *Repository) UpdatePanel(ctx context.Context, panel *domain.Panel) error {
	dbPanel := PanelToModel(panel)
	dbPanel.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).
		Model(&Panel{}).
		Where("id = ?", panel.ID).
		Updates(dbPanel)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) DeletePanel(ctx context.Context, id string) error {
	// 获取面板信息以更新故事的面板数
	var panel Panel
	if err := r.db.WithContext(ctx).First(&panel, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}

	// 软删除面板
	if err := r.db.WithContext(ctx).Delete(&Panel{}, "id = ?", id).Error; err != nil {
		return err
	}

	// 更新故事的面板数
	if err := r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", panel.StoryID).
		UpdateColumn("panels", gorm.Expr("GREATEST(panels - ?, 0)", 1)).Error; err != nil {
		return err
	}

	return nil
}

// ========== Character operations ==========

func (r *Repository) PopularCharacters(ctx context.Context, limit int) ([]*domain.Character, error) {
	var characters []Character
	err := r.db.WithContext(ctx).
		Preload("Author").
		Where("is_public = ?", true).
		Order("likes DESC, followers DESC").
		Limit(limit).
		Find(&characters).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Character, len(characters))
	for i := range characters {
		result[i] = ModelToCharacter(&characters[i])
	}
	return result, nil
}


// ========== AI Generation operations ==========

func (r *Repository) CreateAIGenerationRecord(ctx context.Context, record *domain.AIGenerationRecord) error {
	dbRecord := AIGenerationRecordToModel(record)

	if dbRecord.ID == "" {
		dbRecord.ID = uuid.New().String()
	}

	if dbRecord.CreatedAt.IsZero() {
		dbRecord.CreatedAt = time.Now()
	}

	if err := r.db.WithContext(ctx).Create(dbRecord).Error; err != nil {
		return err
	}

	// 更新 record ID
	record.ID = dbRecord.ID
	return nil
}

func (r *Repository) GetAIGenerationRecord(ctx context.Context, recordID string) (*domain.AIGenerationRecord, error) {
	var record AIGenerationRecord
	err := r.db.WithContext(ctx).
		Preload("User").
		First(&record, "id = ?", recordID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToAIGenerationRecord(&record), nil
}

func (r *Repository) UpdateAIGenerationRecord(ctx context.Context, record *domain.AIGenerationRecord) error {
	dbRecord := AIGenerationRecordToModel(record)

	result := r.db.WithContext(ctx).
		Model(&AIGenerationRecord{}).
		Where("id = ?", record.ID).
		Updates(dbRecord)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteAIGenerationRecord(ctx context.Context, recordID string) error {
	result := r.db.WithContext(ctx).Delete(&AIGenerationRecord{}, "id = ?", recordID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListAIGenerationRecords(ctx context.Context, userID string, limit, offset int) ([]*domain.AIGenerationRecord, error) {
	var records []AIGenerationRecord
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.AIGenerationRecord, len(records))
	for i := range records {
		result[i] = ModelToAIGenerationRecord(&records[i])
	}
	return result, nil
}

func (r *Repository) ListAIGenerationRecordsByTimeRange(ctx context.Context, userID string, startTime, endTime int64) ([]*domain.AIGenerationRecord, error) {
	var records []AIGenerationRecord
	start := unixToTime(startTime)
	end := unixToTime(endTime)

	err := r.db.WithContext(ctx).
		Preload("User").
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end).
		Order("created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.AIGenerationRecord, len(records))
	for i := range records {
		result[i] = ModelToAIGenerationRecord(&records[i])
	}
	return result, nil
}

func (r *Repository) ListAIGenerationRecordsByEntity(ctx context.Context, entityType, entityID string, limit, offset int) ([]*domain.AIGenerationRecord, error) {
	var records []AIGenerationRecord
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("related_entity_type = ? AND related_entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.AIGenerationRecord, len(records))
	for i := range records {
		result[i] = ModelToAIGenerationRecord(&records[i])
	}
	return result, nil
}

func (r *Repository) AIGenerationRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.AIGenerationRecord, error) {
	// 调用新的方法实现
	return r.ListAIGenerationRecords(ctx, userID, limit, offset)
}

func (r *Repository) GetUserTokenStats(ctx context.Context, userID string, startTime, endTime int64) (map[string]interface{}, error) {
	records, err := r.ListAIGenerationRecordsByTimeRange(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// 统计数据
	totalTokens := 0
	totalImages := 0
	totalVideos := 0
	byProvider := make(map[string]int)
	byType := make(map[string]int)

	for _, record := range records {
		totalTokens += record.TotalTokens
		totalImages += record.ImageCount
		totalVideos += record.VideoCount
		byProvider[record.Provider] += record.TotalTokens
		byType[record.Type] += record.TotalTokens
	}

	return map[string]interface{}{
		"totalTokens":   totalTokens,
		"totalImages":   totalImages,
		"totalVideos":   totalVideos,
		"totalRequests": len(records),
		"byProvider":    byProvider,
		"byType":        byType,
	}, nil
}

// GetPendingAIGenerationRecords 获取待处理的AI生成记录（用于服务重启恢复）
func (r *Repository) GetPendingAIGenerationRecords(ctx context.Context, statuses []domain.AITaskStatus, limit int) ([]*domain.AIGenerationRecord, error) {
	var records []AIGenerationRecord

	// 将状态转换为字符串数组
	statusStrings := make([]string, len(statuses))
	for i, status := range statuses {
		statusStrings[i] = string(status)
	}

	query := r.db.WithContext(ctx).
		Preload("User").
		Where("status IN ?", statusStrings).
		Where("type = ?", domain.AITaskGenerateVideo). // 只获取视频生成任务
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&records).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.AIGenerationRecord, len(records))
	for i := range records {
		result[i] = ModelToAIGenerationRecord(&records[i])
	}
	return result, nil
}

// ========== Asset operations ==========

func (r *Repository) AssetByID(ctx context.Context, id string) (*domain.Asset, error) {
	var asset Asset
	err := r.db.WithContext(ctx).
		Preload("User").
		First(&asset, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToAsset(&asset), nil
}

func (r *Repository) AssetsByUser(ctx context.Context, userID string, assetType string, limit, offset int) ([]*domain.Asset, error) {
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if assetType != "" {
		query = query.Where("type = ?", assetType)
	}

	var assets []Asset
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&assets).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Asset, len(assets))
	for i := range assets {
		result[i] = ModelToAsset(&assets[i])
	}
	return result, nil
}

func (r *Repository) CreateAsset(ctx context.Context, asset *domain.Asset) error {
	dbAsset := AssetToModel(asset)
	if dbAsset.ID == "" {
		dbAsset.ID = uuid.New().String()
	}
	dbAsset.CreatedAt = time.Now()

	if err := r.db.WithContext(ctx).Create(dbAsset).Error; err != nil {
		return err
	}

	asset.ID = dbAsset.ID
	asset.CreatedAt = timeToUnix(dbAsset.CreatedAt)
	return nil
}

func (r *Repository) UpdateAsset(ctx context.Context, asset *domain.Asset) error {
	dbAsset := AssetToModel(asset)

	result := r.db.WithContext(ctx).
		Model(&Asset{}).
		Where("id = ?", asset.ID).
		Updates(dbAsset)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteAsset(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Asset{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ========== Tag operations ==========

func (r *Repository) AddCharacterTag(ctx context.Context, characterID, tagID string) error {
	// 检查是否已存在
	var count int64
	if err := r.db.WithContext(ctx).Model(&CharacterTag{}).
		Where("character_id = ? AND tag_id = ?", characterID, tagID).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil // 已存在，不报错
	}

	tag := &CharacterTag{
		ID:          uuid.New().String(),
		CharacterID: characterID,
		TagID:       tagID,
		CreatedAt:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(tag).Error; err != nil {
		return err
	}

	// 更新标签使用次数
	if err := r.db.WithContext(ctx).Model(&Tag{}).
		Where("id = ?", tagID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

func (r *Repository) RemoveCharacterTag(ctx context.Context, characterID, tagID string) error {
	result := r.db.WithContext(ctx).
		Where("character_id = ? AND tag_id = ?", characterID, tagID).
		Delete(&CharacterTag{})

	if result.Error != nil {
		return result.Error
	}

	// 更新标签使用次数
	if result.RowsAffected > 0 {
		if err := r.db.WithContext(ctx).Model(&Tag{}).
			Where("id = ?", tagID).
			UpdateColumn("usage_count", gorm.Expr("GREATEST(usage_count - ?, 0)", 1)).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) CharacterTags(ctx context.Context, characterID string) ([]*domain.Tag, error) {
	var characterTags []CharacterTag
	err := r.db.WithContext(ctx).
		Preload("Tag").
		Where("character_id = ?", characterID).
		Find(&characterTags).Error
	if err != nil {
		return nil, err
	}

	tags := make([]*domain.Tag, len(characterTags))
	for i := range characterTags {
		tags[i] = ModelToTag(&characterTags[i].Tag)
	}
	return tags, nil
}

func (r *Repository) GetTagByID(ctx context.Context, tagID string) (*domain.Tag, error) {
	var tag Tag
	err := r.db.WithContext(ctx).First(&tag, "id = ?", tagID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToTag(&tag), nil
}

func (r *Repository) GetTagByName(ctx context.Context, name string) (*domain.Tag, error) {
	var tag Tag
	err := r.db.WithContext(ctx).First(&tag, "name = ?", name).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToTag(&tag), nil
}

func (r *Repository) UpdateTag(ctx context.Context, tag *domain.Tag) error {
	dbTag := TagToModel(tag)

	result := r.db.WithContext(ctx).
		Model(&Tag{}).
		Where("id = ?", tag.ID).
		Updates(dbTag)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteTag(ctx context.Context, tagID string) error {
	// 删除标签及其关联
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除故事标签关联
		if err := tx.Where("tag_id = ?", tagID).Delete(&StoryTag{}).Error; err != nil {
			return err
		}

		// 删除角色标签关联
		if err := tx.Where("tag_id = ?", tagID).Delete(&CharacterTag{}).Error; err != nil {
			return err
		}

		// 删除标签本身
		result := tx.Delete(&Tag{}, "id = ?", tagID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}

		return nil
	})
}

func (r *Repository) ListTags(ctx context.Context, category string, limit, offset int) ([]*domain.Tag, int64, error) {
	query := r.db.WithContext(ctx).Model(&Tag{})
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tags []Tag
	err := query.Order("usage_count DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tags).Error
	if err != nil {
		return nil, 0, err
	}

	result := make([]*domain.Tag, len(tags))
	for i := range tags {
		result[i] = ModelToTag(&tags[i])
	}
	return result, total, nil
}

func (r *Repository) SearchTags(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	var tags []Tag
	err := r.db.WithContext(ctx).
		Where("name LIKE ?", "%"+query+"%").
		Order("usage_count DESC").
		Limit(limit).
		Find(&tags).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Tag, len(tags))
	for i := range tags {
		result[i] = ModelToTag(&tags[i])
	}
	return result, nil
}

// ========== Search operations ==========

func (r *Repository) AdvancedSearch(ctx context.Context, filter *domain.SearchFilter) ([]*domain.SearchResult, int64, error) {
	var results []*domain.SearchResult
	var total int64

	// 根据类型搜索不同的实体
	switch filter.Type {
	case "story", "all":
		storyResults, storyTotal, err := r.searchStories(ctx, filter)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, storyResults...)
		total += storyTotal

	case "character":
		charResults, charTotal, err := r.searchCharacters(ctx, filter)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, charResults...)
		total += charTotal

	case "user":
		userResults, userTotal, err := r.searchUsers(ctx, filter)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, userResults...)
		total += userTotal
	}

	// 如果有限制，截取结果
	if filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start > len(results) {
			return []*domain.SearchResult{}, total, nil
		}
		if end > len(results) {
			end = len(results)
		}
		results = results[start:end]
	}

	return results, total, nil
}

func (r *Repository) searchStories(ctx context.Context, filter *domain.SearchFilter) ([]*domain.SearchResult, int64, error) {
	query := r.db.WithContext(ctx).Model(&Story{}).Preload("Author")

	if filter.Query != "" {
		query = query.Where("title LIKE ? OR description LIKE ?",
			"%"+filter.Query+"%", "%"+filter.Query+"%")
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.AuthorID != "" {
		query = query.Where("author_id = ?", filter.AuthorID)
	}

	if filter.MinLikes > 0 {
		query = query.Where("likes >= ?", filter.MinLikes)
	}

	if len(filter.Categories) > 0 {
		query = query.Where("genre IN ?", filter.Categories)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	orderBy := "created_at DESC"
	if filter.SortBy == "likes" {
		orderBy = "likes DESC"
	} else if filter.SortBy == "followers" {
		orderBy = "followers DESC"
	}
	query = query.Order(orderBy)

	var stories []Story
	if err := query.Find(&stories).Error; err != nil {
		return nil, 0, err
	}

	results := make([]*domain.SearchResult, len(stories))
	for i, story := range stories {
		results[i] = &domain.SearchResult{
			Type:        "story",
			ID:          story.ID,
			Title:       story.Title,
			Description: story.Description,
			Cover:       story.CoverImage,
			AuthorID:    story.AuthorID,
			Likes:       story.Likes,
			CreatedAt:   timeToUnix(story.CreatedAt),
			Relevance:   1.0,
		}
		if story.Author.ID != "" {
			results[i].Author = story.Author.DisplayName
		}
	}

	return results, total, nil
}

func (r *Repository) searchCharacters(ctx context.Context, filter *domain.SearchFilter) ([]*domain.SearchResult, int64, error) {
	query := r.db.WithContext(ctx).Model(&Character{}).Preload("Author")

	if filter.Query != "" {
		query = query.Where("name LIKE ? OR description LIKE ?",
			"%"+filter.Query+"%", "%"+filter.Query+"%")
	}

	if filter.AuthorID != "" {
		query = query.Where("author_id = ?", filter.AuthorID)
	}

	if filter.MinLikes > 0 {
		query = query.Where("likes >= ?", filter.MinLikes)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := "created_at DESC"
	if filter.SortBy == "likes" {
		orderBy = "likes DESC"
	}
	query = query.Order(orderBy)

	var characters []Character
	if err := query.Find(&characters).Error; err != nil {
		return nil, 0, err
	}

	results := make([]*domain.SearchResult, len(characters))
	for i, char := range characters {
		results[i] = &domain.SearchResult{
			Type:        "character",
			ID:          char.ID,
			Title:       char.Name,
			Description: char.Description,
			Cover:       char.Avatar,
			AuthorID:    char.AuthorID,
			Likes:       char.Likes,
			CreatedAt:   timeToUnix(char.CreatedAt),
			Relevance:   1.0,
		}
		if char.Author.ID != "" {
			results[i].Author = char.Author.DisplayName
		}
	}

	return results, total, nil
}

func (r *Repository) searchUsers(ctx context.Context, filter *domain.SearchFilter) ([]*domain.SearchResult, int64, error) {
	query := r.db.WithContext(ctx).Model(&User{})

	if filter.Query != "" {
		query = query.Where("username LIKE ? OR display_name LIKE ?",
			"%"+filter.Query+"%", "%"+filter.Query+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Order("followers DESC")

	var users []User
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	results := make([]*domain.SearchResult, len(users))
	for i, user := range users {
		results[i] = &domain.SearchResult{
			Type:        "user",
			ID:          user.ID,
			Title:       user.DisplayName,
			Description: user.Bio,
			Cover:       user.Avatar,
			Likes:       user.Followers,
			CreatedAt:   user.CreatedAt,
			Relevance:   1.0,
		}
	}

	return results, total, nil
}

func (r *Repository) SearchByCategory(ctx context.Context, category, searchType string, limit, offset int) ([]*domain.SearchResult, error) {
	filter := &domain.SearchFilter{
		Type:       searchType,
		Categories: []string{category},
		Limit:      limit,
		Offset:     offset,
	}
	results, _, err := r.AdvancedSearch(ctx, filter)
	return results, err
}

func (r *Repository) SearchByTags(ctx context.Context, tags []string, searchType string, limit, offset int) ([]*domain.SearchResult, error) {
	var results []*domain.SearchResult

	if searchType == "story" || searchType == "all" {
		// 查找包含这些标签的故事
		var storyTags []StoryTag
		err := r.db.WithContext(ctx).
			Preload("Story").
			Preload("Story.Author").
			Preload("Tag").
			Joins("JOIN tags ON tags.id = story_tags.tag_id").
			Where("tags.name IN ?", tags).
			Find(&storyTags).Error
		if err != nil {
			return nil, err
		}

		// 去重并转换为搜索结果
		storyMap := make(map[string]*Story)
		for _, st := range storyTags {
			storyMap[st.StoryID] = &st.Story
		}

		for _, story := range storyMap {
			result := &domain.SearchResult{
				Type:        "story",
				ID:          story.ID,
				Title:       story.Title,
				Description: story.Description,
				Cover:       story.CoverImage,
				AuthorID:    story.AuthorID,
				Likes:       story.Likes,
				CreatedAt:   timeToUnix(story.CreatedAt),
				Relevance:   1.0,
			}
			if story.Author.ID != "" {
				result.Author = story.Author.DisplayName
			}
			results = append(results, result)
		}
	}

	// 应用分页
	if offset < len(results) {
		end := offset + limit
		if end > len(results) {
			end = len(results)
		}
		results = results[offset:end]
	} else {
		results = []*domain.SearchResult{}
	}

	return results, nil
}

func (r *Repository) GetSearchSuggestions(ctx context.Context, query string, limit int) ([]string, error) {
	var suggestions []string

	// 从故事标题中获取建议
	var stories []Story
	err := r.db.WithContext(ctx).
		Select("title").
		Where("title LIKE ?", "%"+query+"%").
		Limit(limit / 2).
		Find(&stories).Error
	if err != nil {
		return nil, err
	}

	for _, story := range stories {
		suggestions = append(suggestions, story.Title)
	}

	// 从角色名称中获取建议
	var characters []Character
	err = r.db.WithContext(ctx).
		Select("name").
		Where("name LIKE ?", "%"+query+"%").
		Limit(limit / 2).
		Find(&characters).Error
	if err != nil {
		return nil, err
	}

	for _, char := range characters {
		suggestions = append(suggestions, char.Name)
	}

	return suggestions, nil
}

func (r *Repository) GetTrendingSearches(ctx context.Context, limit int) ([]string, error) {
	var searches []SearchHistory
	err := r.db.WithContext(ctx).
		Select("query, COUNT(*) as count").
		Group("query").
		Order("count DESC").
		Limit(limit).
		Find(&searches).Error
	if err != nil {
		return nil, err
	}

	queries := make([]string, len(searches))
	for i, search := range searches {
		queries[i] = search.Query
	}
	return queries, nil
}

func (r *Repository) GetUserSearchHistory(ctx context.Context, userID string, limit int) ([]*domain.SearchHistory, error) {
	var histories []SearchHistory
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&histories).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.SearchHistory, len(histories))
	for i := range histories {
		result[i] = ModelToSearchHistory(&histories[i])
	}
	return result, nil
}

// ========== Token & Subscription ==========

func (r *Repository) CreateTokenTransaction(ctx context.Context, transaction *domain.TokenTransaction) error {
	dbTx := TokenTransactionToModel(transaction)
	if dbTx.ID == "" {
		dbTx.ID = uuid.New().String()
	}
	dbTx.CreatedAt = time.Now()

	if err := r.db.WithContext(ctx).Create(dbTx).Error; err != nil {
		return err
	}

	transaction.ID = dbTx.ID
	transaction.CreatedAt = timeToUnix(dbTx.CreatedAt)
	return nil
}

func (r *Repository) GetTokenTransaction(ctx context.Context, transactionID string) (*domain.TokenTransaction, error) {
	var tx TokenTransaction
	err := r.db.WithContext(ctx).
		Preload("User").
		First(&tx, "id = ?", transactionID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToTokenTransaction(&tx), nil
}

func (r *Repository) ListTokenTransactions(ctx context.Context, userID string, limit, offset int) ([]*domain.TokenTransaction, int64, error) {
	query := r.db.WithContext(ctx).Model(&TokenTransaction{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var transactions []TokenTransaction
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error
	if err != nil {
		return nil, 0, err
	}

	result := make([]*domain.TokenTransaction, len(transactions))
	for i := range transactions {
		result[i] = ModelToTokenTransaction(&transactions[i])
	}
	return result, total, nil
}

func (r *Repository) GetTokenBalance(ctx context.Context, userID string) (int, error) {
	var membership Membership
	err := r.db.WithContext(ctx).First(&membership, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil // 未找到会员信息，返回0
		}
		return 0, err
	}
	return membership.TokenQuota - membership.TokenUsed, nil
}

func (r *Repository) UpdateTokenBalance(ctx context.Context, userID string, amount int, source, description string) (*domain.TokenTransaction, error) {
	var resultTx *domain.TokenTransaction
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取当前余额
		var membership Membership
		err := tx.Where("user_id = ?", userID).First(&membership).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新的会员记录
				membership = Membership{
					ID:         uuid.New().String(),
					UserID:     userID,
					Tier:       "free",
					Status:     "active",
					StartDate:  time.Now(),
					TokenQuota: 0,
					TokenUsed:  0,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
				if err := tx.Create(&membership).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// 计算新余额
		currentBalance := membership.TokenQuota - membership.TokenUsed
		newBalance := currentBalance + amount

		if newBalance < 0 {
			return errors.New("insufficient token balance")
		}

		// 更新会员信息
		if amount > 0 {
			// 充值，增加配额
			membership.TokenQuota += amount
		} else {
			// 消费，增加使用量
			membership.TokenUsed += -amount
		}
		membership.UpdatedAt = time.Now()

		if err := tx.Save(&membership).Error; err != nil {
			return err
		}

		// 创建交易记录
		transaction := &TokenTransaction{
			ID:          uuid.New().String(),
			UserID:      userID,
			Type:        source,
			Amount:      amount,
			Balance:     newBalance,
			Source:      source,
			Description: description,
			CreatedAt:   time.Now(),
		}

		if err := tx.Create(transaction).Error; err != nil {
			return err
		}

		resultTx = ModelToTokenTransaction(transaction)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resultTx, nil
}

func (r *Repository) CreateSubscriptionPlan(ctx context.Context, plan *domain.SubscriptionPlan) error {
	dbPlan := SubscriptionPlanToModel(plan)
	if dbPlan.ID == "" {
		dbPlan.ID = uuid.New().String()
	}
	dbPlan.CreatedAt = time.Now()
	dbPlan.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Create(dbPlan).Error; err != nil {
		return err
	}

	plan.ID = dbPlan.ID
	plan.CreatedAt = timeToUnix(dbPlan.CreatedAt)
	plan.UpdatedAt = timeToUnix(dbPlan.UpdatedAt)
	return nil
}

func (r *Repository) GetSubscriptionPlan(ctx context.Context, planID string) (*domain.SubscriptionPlan, error) {
	var plan SubscriptionPlan
	err := r.db.WithContext(ctx).First(&plan, "id = ?", planID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToSubscriptionPlan(&plan), nil
}

func (r *Repository) UpdateSubscriptionPlan(ctx context.Context, plan *domain.SubscriptionPlan) error {
	dbPlan := SubscriptionPlanToModel(plan)
	dbPlan.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).
		Model(&SubscriptionPlan{}).
		Where("id = ?", plan.ID).
		Updates(dbPlan)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	plan.UpdatedAt = timeToUnix(dbPlan.UpdatedAt)
	return nil
}

func (r *Repository) DeleteSubscriptionPlan(ctx context.Context, planID string) error {
	result := r.db.WithContext(ctx).Delete(&SubscriptionPlan{}, "id = ?", planID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListSubscriptionPlans(ctx context.Context, activeOnly bool) ([]*domain.SubscriptionPlan, error) {
	query := r.db.WithContext(ctx).Model(&SubscriptionPlan{})
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	var plans []SubscriptionPlan
	err := query.Order("sort_order ASC, price ASC").Find(&plans).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.SubscriptionPlan, len(plans))
	for i := range plans {
		result[i] = ModelToSubscriptionPlan(&plans[i])
	}
	return result, nil
}

func (r *Repository) CreateSubscriptionOrder(ctx context.Context, order *domain.SubscriptionOrder) error {
	dbOrder := SubscriptionOrderToModel(order)
	if dbOrder.ID == "" {
		dbOrder.ID = uuid.New().String()
	}
	dbOrder.CreatedAt = time.Now()
	dbOrder.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Create(dbOrder).Error; err != nil {
		return err
	}

	order.ID = dbOrder.ID
	order.CreatedAt = timeToUnix(dbOrder.CreatedAt)
	order.UpdatedAt = timeToUnix(dbOrder.UpdatedAt)
	return nil
}

func (r *Repository) GetSubscriptionOrder(ctx context.Context, orderID string) (*domain.SubscriptionOrder, error) {
	var order SubscriptionOrder
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Plan").
		First(&order, "id = ?", orderID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToSubscriptionOrder(&order), nil
}

func (r *Repository) UpdateSubscriptionOrder(ctx context.Context, order *domain.SubscriptionOrder) error {
	dbOrder := SubscriptionOrderToModel(order)
	dbOrder.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).
		Model(&SubscriptionOrder{}).
		Where("id = ?", order.ID).
		Updates(dbOrder)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	order.UpdatedAt = timeToUnix(dbOrder.UpdatedAt)
	return nil
}

func (r *Repository) ListSubscriptionOrders(ctx context.Context, userID string, limit, offset int) ([]*domain.SubscriptionOrder, int64, error) {
	query := r.db.WithContext(ctx).Model(&SubscriptionOrder{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orders []SubscriptionOrder
	err := query.Preload("Plan").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	result := make([]*domain.SubscriptionOrder, len(orders))
	for i := range orders {
		result[i] = ModelToSubscriptionOrder(&orders[i])
	}
	return result, total, nil
}

func (r *Repository) GetActiveSubscription(ctx context.Context, userID string) (*domain.SubscriptionOrder, error) {
	var order SubscriptionOrder
	now := time.Now()
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("user_id = ? AND status = ? AND end_date > ?", userID, "paid", now).
		Order("end_date DESC").
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToSubscriptionOrder(&order), nil
}

// ========== User Settings ==========

func (r *Repository) GetUserNotificationSettings(ctx context.Context, userID string) (map[string]bool, error) {
	var settings UserSettings
	err := r.db.WithContext(ctx).First(&settings, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回默认设置
			return map[string]bool{
				"email":   true,
				"push":    true,
				"comment": true,
				"like":    true,
				"follow":  true,
				"mention": true,
				"system":  true,
			}, nil
		}
		return nil, err
	}

	// 从 JSON 字段解析通知设置
	notificationSettings := map[string]bool{
		"email": true,
		"push":  true,
	}
	// TODO: 从 settings.NotificationSettings 解析更多通知选项

	return notificationSettings, nil
}

func (r *Repository) UpdateUserNotificationSettings(ctx context.Context, userID string, settings map[string]bool) error {
	var userSettings UserSettings
	err := r.db.WithContext(ctx).First(&userSettings, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新设置
			userSettings = UserSettings{
				ID:        uuid.New().String(),
				UserID:    userID,
				UpdatedAt: time.Now().Unix(),
			}
		} else {
			return err
		}
	}

	// 更新通知设置 JSON
	userSettings.UpdatedAt = time.Now().Unix()

	return r.db.WithContext(ctx).Save(&userSettings).Error
}

func (r *Repository) GetUserPrivacySettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	var settings UserSettings
	err := r.db.WithContext(ctx).First(&settings, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回默认设置
			return map[string]interface{}{
				"profile_visibility":  "public",
				"allow_comments_from": "everyone",
				"allow_messages_from": "followers_only",
				"show_online_status":  true,
			}, nil
		}
		return nil, err
	}

	return map[string]interface{}{
		"profile_visibility":  settings.ProfileVisibility,
		"allow_comments_from": settings.AllowCommentsFrom,
		"allow_messages_from": settings.AllowMessagesFrom,
		"show_online_status":  settings.ShowOnlineStatus,
	}, nil
}

func (r *Repository) UpdateUserPrivacySettings(ctx context.Context, userID string, settings map[string]interface{}) error {
	var userSettings UserSettings
	err := r.db.WithContext(ctx).First(&userSettings, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新设置
			userSettings = UserSettings{
				ID:                uuid.New().String(),
				UserID:            userID,
				ProfileVisibility: "public",
				ShowOnlineStatus:  true,
				UpdatedAt:         time.Now().Unix(),
			}
		} else {
			return err
		}
	}

	// 更新设置
	if profileVisibility, ok := settings["profile_visibility"].(string); ok {
		userSettings.ProfileVisibility = profileVisibility
	}
	if allowCommentsFrom, ok := settings["allow_comments_from"].(string); ok {
		userSettings.AllowCommentsFrom = allowCommentsFrom
	}
	if allowMessagesFrom, ok := settings["allow_messages_from"].(string); ok {
		userSettings.AllowMessagesFrom = allowMessagesFrom
	}
	if showOnlineStatus, ok := settings["show_online_status"].(bool); ok {
		userSettings.ShowOnlineStatus = showOnlineStatus
	}
	userSettings.UpdatedAt = time.Now().Unix()

	return r.db.WithContext(ctx).Save(&userSettings).Error
}

// ========== Helper functions ==========

// 辅助函数：构建搜索查询的 LIKE 模式
func buildLikePattern(query string) string {
	return "%" + strings.TrimSpace(query) + "%"
}

// 辅助函数：计算相关度分数
func calculateRelevance(title, description, query string) float64 {
	query = strings.ToLower(query)
	title = strings.ToLower(title)
	description = strings.ToLower(description)

	score := 0.0
	if strings.Contains(title, query) {
		score += 1.0
	}
	if strings.Contains(description, query) {
		score += 0.5
	}

	return score
}

// 辅助函数：将任意值序列化为 JSON 字符串
func toJSONString(v interface{}) string {
	if v == nil {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// 辅助函数：从 JSON 字符串反序列化
func fromJSONString(s string, v interface{}) error {
	if s == "" {
		return nil
	}
	return json.Unmarshal([]byte(s), v)
}

// 辅助函数：格式化错误信息
func formatError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s failed: %w", operation, err)
}

// ========== Story Composition operations ==========

// StoryCompositionByID retrieves a story composition by ID
func (r *Repository) StoryCompositionByID(ctx context.Context, id string) (*domain.StoryComposition, error) {
	var composition domain.StoryComposition
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&composition).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &composition, nil
}

// ListStoryCompositions retrieves all story compositions
func (r *Repository) ListStoryCompositions(ctx context.Context, limit, offset int) ([]*domain.StoryComposition, error) {
	var compositions []*domain.StoryComposition
	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&compositions).Error; err != nil {
		return nil, err
	}
	return compositions, nil
}

// CreateStoryComposition creates a new story composition
func (r *Repository) CreateStoryComposition(ctx context.Context, composition *domain.StoryComposition) error {
	if composition.ID == "" {
		composition.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(composition).Error
}

// UpdateStoryComposition updates an existing story composition
func (r *Repository) UpdateStoryComposition(ctx context.Context, composition *domain.StoryComposition) error {
	return r.db.WithContext(ctx).Save(composition).Error
}

// IsStoryLiked checks if a user has liked a story
func (r *Repository) IsStoryLiked(ctx context.Context, userID, storyID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&StoryLike{}).
		Where("user_id = ? AND story_id = ?", userID, storyID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// LikeStoryboard likes a storyboard
func (r *Repository) LikeStoryboard(ctx context.Context, userID, storyboardID string) error {
	// Check if already liked
	var count int64
	if err := r.db.WithContext(ctx).Model(&StoryboardLike{}).
		Where("user_id = ? AND storyboard_id = ?", userID, storyboardID).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return domain.ErrAlreadyLiked
	}

	// Create like record
	like := StoryboardLike{
		ID:           uuid.New().String(),
		UserID:       userID,
		StoryboardID: storyboardID,
		CreatedAt:    time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&like).Error; err != nil {
		return err
	}

	// Update storyboard likes count
	return r.db.WithContext(ctx).Model(&Storyboard{}).
		Where("id = ?", storyboardID).
		UpdateColumn("likes", gorm.Expr("likes + ?", 1)).Error
}

// UnlikeStoryboard removes a like from a storyboard
func (r *Repository) UnlikeStoryboard(ctx context.Context, userID, storyboardID string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND storyboard_id = ?", userID, storyboardID).
		Delete(&StoryboardLike{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	// Update storyboard likes count
	return r.db.WithContext(ctx).Model(&Storyboard{}).
		Where("id = ?", storyboardID).
		UpdateColumn("likes", gorm.Expr("GREATEST(likes - ?, 0)", 1)).Error
}

// ListStories retrieves stories with filtering
func (r *Repository) ListStories(ctx context.Context, filter domain.StoryFilter) ([]*domain.Story, int64, error) {
	var stories []Story
	var total int64

	query := r.db.WithContext(ctx).Model(&Story{})

	// Apply filters
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.AuthorID != "" {
		query = query.Where("author_id = ?", filter.AuthorID)
	}
	if filter.Search != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}
	if filter.Genre != "" {
		query = query.Where("genre = ?", filter.Genre)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and fetch data
	query = query.Preload("Author").Order("created_at DESC")
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	if err := query.Find(&stories).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*domain.Story, len(stories))
	for i := range stories {
		result[i] = ModelToStory(&stories[i])
	}

	return result, total, nil
}

// PanelsByStory retrieves all panels for a story
func (r *Repository) PanelsByStory(ctx context.Context, storyID string) ([]*domain.Panel, error) {
	var panels []Panel
	if err := r.db.WithContext(ctx).
		Where("story_id = ?", storyID).
		Order("sequence ASC").
		Find(&panels).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Panel, len(panels))
	for i := range panels {
		result[i] = ModelToPanel(&panels[i])
	}

	return result, nil
}
