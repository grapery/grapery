package mysql

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func (r *Repository) SearchStories(ctx context.Context, query string, limit, offset int) ([]*domain.Story, error) {
	var stories []Story
	q := r.db.WithContext(ctx).Preload("Author").
		Where("title LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%").
		Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&stories).Error; err != nil {
		return nil, fmt.Errorf("failed to search stories: %w", err)
	}
	result := make([]*domain.Story, len(stories))
	for i, s := range stories {
		ds := r.storyToDomain(s)
		result[i] = &ds
	}
	return result, nil
}

func (r *Repository) SearchCharacters(ctx context.Context, query string, limit, offset int) ([]*domain.Character, error) {
	var characters []Character
	q := r.db.WithContext(ctx).Preload("Author").
		Where("name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%").
		Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&characters).Error; err != nil {
		return nil, fmt.Errorf("failed to search characters: %w", err)
	}
	result := make([]*domain.Character, len(characters))
	for i, c := range characters {
		dc := r.characterToDomain(c)
		result[i] = &dc
	}
	return result, nil
}

func (r *Repository) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*domain.User, error) {
	var users []User
	q := r.db.WithContext(ctx).
		Where("username LIKE ? OR display_name LIKE ?", "%"+query+"%", "%"+query+"%").
		Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	result := make([]*domain.User, len(users))
	for i, u := range users {
		du := r.userToDomain(u)
		result[i] = &du
	}
	return result, nil
}

func (r *Repository) SearchGroups(ctx context.Context, query string, limit, offset int) ([]*domain.Group, error) {
	var groups []Group
	q := r.db.WithContext(ctx).
		Where("name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%").
		Where("public = ?", true). // Only search public groups
		Order("members DESC, created_at DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to search groups: %w", err)
	}
	result := make([]*domain.Group, len(groups))
	for i, g := range groups {
		dg := r.groupToDomain(g)
		result[i] = &dg
	}
	return result, nil
}

func (r *Repository) CreateSearchHistory(ctx context.Context, history *domain.SearchHistory) error {
	dbHistory := &SearchHistory{
		ID:          history.ID,
		UserID:      history.UserID,
		Query:       history.Query,
		Type:        history.Type,
		ResultCount: history.ResultCount,
		CreatedAt:   unixToTime(history.CreatedAt),
	}
	return r.db.WithContext(ctx).Create(dbHistory).Error
}

func (r *Repository) ViewHistoryByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.ViewHistory, error) {
	var histories []ViewHistory
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("viewed_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&histories).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ViewHistory, len(histories))
	for i, h := range histories {
		result[i] = ModelToViewHistory(&h)
	}
	return result, nil
}

func (r *Repository) CreateViewHistory(ctx context.Context, history *domain.ViewHistory) error {
	dbHistory := &ViewHistory{
		ID:         history.ID,
		UserID:     history.UserID,
		EntityType: history.EntityType,
		EntityID:   history.EntityID,
		Duration:   history.Duration,
		ViewedAt:   unixToTime(history.ViewedAt),
	}
	return r.db.WithContext(ctx).Create(dbHistory).Error
}
