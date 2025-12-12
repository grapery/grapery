package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func (r *Repository) CreateTag(ctx context.Context, tag *domain.Tag) error {
	dbTag := Tag{
		ID:   uuid.New().String(),
		Name: tag.Name,
	}
	if err := r.db.WithContext(ctx).Create(&dbTag).Error; err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}
	tag.ID = dbTag.ID
	tag.CreatedAt = dbTag.CreatedAt.Unix()
	return nil
}

func (r *Repository) GetOrCreateTag(ctx context.Context, name string) (*domain.Tag, error) {
	var dbTag Tag
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&dbTag).Error; err == nil {
		tag := r.tagToDomain(dbTag)
		return &tag, nil
	}

	tag := &domain.Tag{Name: name}
	if err := r.CreateTag(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (r *Repository) AddStoryTag(ctx context.Context, storyID, tagID string) error {
	storyTag := StoryTag{
		ID:      uuid.New().String(),
		StoryID: storyID,
		TagID:   tagID,
	}
	return r.db.WithContext(ctx).Create(&storyTag).Error
}

func (r *Repository) RemoveStoryTag(ctx context.Context, storyID, tagID string) error {
	return r.db.WithContext(ctx).Where("story_id = ? AND tag_id = ?", storyID, tagID).Delete(&StoryTag{}).Error
}

func (r *Repository) StoryTags(ctx context.Context, storyID string) ([]*domain.Tag, error) {
	var storyTags []StoryTag
	if err := r.db.WithContext(ctx).Preload("Tag").Where("story_id = ?", storyID).Find(&storyTags).Error; err != nil {
		return nil, fmt.Errorf("failed to get story tags: %w", err)
	}

	result := make([]*domain.Tag, len(storyTags))
	for i, st := range storyTags {
		tag := r.tagToDomain(st.Tag)
		result[i] = &tag
	}
	return result, nil
}

func (r *Repository) StoriesByTag(ctx context.Context, tagID string, limit, offset int) ([]*domain.Story, error) {
	var storyTags []StoryTag
	query := r.db.WithContext(ctx).Preload("Story").Preload("Story.Author").Where("tag_id = ?", tagID)
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&storyTags).Error; err != nil {
		return nil, fmt.Errorf("failed to get stories by tag: %w", err)
	}

	result := make([]*domain.Story, len(storyTags))
	for i, st := range storyTags {
		story := r.storyToDomain(st.Story)
		result[i] = &story
	}
	return result, nil
}

func (r *Repository) PopularTags(ctx context.Context, limit int) ([]*domain.Tag, error) {
	var tags []Tag
	if err := r.db.WithContext(ctx).Order("usage_count DESC").Limit(limit).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to get popular tags: %w", err)
	}

	result := make([]*domain.Tag, len(tags))
	for i, t := range tags {
		tag := r.tagToDomain(t)
		result[i] = &tag
	}
	return result, nil
}

func (r *Repository) tagToDomain(t Tag) domain.Tag {
	return domain.Tag{
		ID:         t.ID,
		Name:       t.Name,
		UsageCount: t.UsageCount,
		CreatedAt:  t.CreatedAt.Unix(),
	}
}
