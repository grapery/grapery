package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// Bookmark MySQL model
type Bookmark struct {
	ID             string    `gorm:"primaryKey;size:36"`
	UserID         string    `gorm:"index;size:36;not null"`
	BookmarkType   string    `gorm:"index;size:20;not null"` // story, fragment, storyboard
	BookmarkID     string    `gorm:"index;size:36;not null"` // The ID of the bookmarked item
	CollectionName string    `gorm:"size:100"`               // Optional collection/folder name
	CreatedAt      time.Time `gorm:"autoCreateTime;index"`
}

// TableName specifies the table name for Bookmark
func (Bookmark) TableName() string {
	return "bookmarks"
}

// BookmarkRepositoryImpl implements domain.BookmarkRepository
type BookmarkRepositoryImpl struct {
	db *gorm.DB
}

// NewBookmarkRepository creates a new BookmarkRepository instance
func NewBookmarkRepository(db *gorm.DB) domain.BookmarkRepository {
	return &BookmarkRepositoryImpl{db: db}
}

// CreateBookmark creates a bookmark
func (r *BookmarkRepositoryImpl) CreateBookmark(ctx context.Context, bookmark *domain.Bookmark) error {
	if bookmark.ID == "" {
		bookmark.ID = uuid.New().String()
	}
	bookmark.CreatedAt = time.Now().Unix()

	model := &Bookmark{
		ID:             bookmark.ID,
		UserID:         bookmark.UserID,
		BookmarkType:   string(bookmark.BookmarkType),
		BookmarkID:     bookmark.BookmarkID,
		CollectionName: bookmark.CollectionName,
		CreatedAt:      time.Unix(bookmark.CreatedAt, 0),
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create bookmark: %w", err)
	}
	return nil
}

// DeleteBookmark deletes a bookmark by ID
func (r *BookmarkRepositoryImpl) DeleteBookmark(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Bookmark{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete bookmark: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("bookmark not found")
	}
	return nil
}

// GetBookmarkByID gets a bookmark by ID
func (r *BookmarkRepositoryImpl) GetBookmarkByID(ctx context.Context, id string) (*domain.Bookmark, error) {
	var model Bookmark
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("bookmark not found")
		}
		return nil, fmt.Errorf("failed to get bookmark: %w", err)
	}
	return modelToDomainBookmark(&model), nil
}

// GetBookmarksByUser gets all bookmarks by a user
func (r *BookmarkRepositoryImpl) GetBookmarksByUser(ctx context.Context, userID string, bookmarkType domain.BookmarkType) ([]*domain.Bookmark, error) {
	var models []Bookmark
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if bookmarkType != "" {
		query = query.Where("bookmark_type = ?", string(bookmarkType))
	}
	if err := query.Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get bookmarks by user: %w", err)
	}
	return modelsToDomainBookmarks(models), nil
}

// GetBookmarksByUserPaginated gets bookmarks by user with pagination.
func (r *BookmarkRepositoryImpl) GetBookmarksByUserPaginated(ctx context.Context, userID string, bookmarkType domain.BookmarkType, limit, offset int) ([]*domain.Bookmark, int64, error) {
	var models []Bookmark
	var total int64

	query := r.db.WithContext(ctx).Model(&Bookmark{}).Where("user_id = ?", userID)
	if bookmarkType != "" {
		query = query.Where("bookmark_type = ?", string(bookmarkType))
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count bookmarks by user: %w", err)
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get paginated bookmarks by user: %w", err)
	}
	return modelsToDomainBookmarks(models), total, nil
}

// GetBookmarksByItem gets all bookmarks for a specific item
func (r *BookmarkRepositoryImpl) GetBookmarksByItem(ctx context.Context, bookmarkType domain.BookmarkType, bookmarkID string) ([]*domain.Bookmark, error) {
	var models []Bookmark
	if err := r.db.WithContext(ctx).
		Where("bookmark_type = ? AND bookmark_id = ?", string(bookmarkType), bookmarkID).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get bookmarks by item: %w", err)
	}
	return modelsToDomainBookmarks(models), nil
}

// CheckBookmarkStatus checks if a user has bookmarked a specific item
func (r *BookmarkRepositoryImpl) CheckBookmarkStatus(ctx context.Context, userID string, bookmarkType domain.BookmarkType, bookmarkID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Bookmark{}).
		Where("user_id = ? AND bookmark_type = ? AND bookmark_id = ?",
			userID, string(bookmarkType), bookmarkID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check bookmark status: %w", err)
	}
	return count > 0, nil
}

// GetBookmarksCount gets the count of bookmarks for a specific item
func (r *BookmarkRepositoryImpl) GetBookmarksCount(ctx context.Context, bookmarkType domain.BookmarkType, bookmarkID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Bookmark{}).
		Where("bookmark_type = ? AND bookmark_id = ?", string(bookmarkType), bookmarkID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get bookmarks count: %w", err)
	}
	return int(count), nil
}

// UpdateBookmarksCount updates the saves count on the target entity
func (r *BookmarkRepositoryImpl) UpdateBookmarksCount(ctx context.Context, bookmarkType domain.BookmarkType, bookmarkID string, delta int) error {
	switch bookmarkType {
	case domain.BookmarkTypeStory:
		expr := gorm.Expr("saves + ?", delta)
		if delta < 0 {
			expr = gorm.Expr("GREATEST(saves + ?, 0)", delta)
		}
		return r.db.WithContext(ctx).Model(&Story{}).
			Where("id = ?", bookmarkID).
			Update("saves", expr).Error
	case domain.BookmarkTypeFragment:
		expr := gorm.Expr("saves + ?", delta)
		if delta < 0 {
			expr = gorm.Expr("GREATEST(saves + ?, 0)", delta)
		}
		return r.db.WithContext(ctx).Model(&Fragment{}).
			Where("id = ?", bookmarkID).
			Update("saves", expr).Error
	}
	return nil
}

// Helper functions

func modelToDomainBookmark(m *Bookmark) *domain.Bookmark {
	return &domain.Bookmark{
		ID:             m.ID,
		UserID:         m.UserID,
		BookmarkType:   domain.BookmarkType(m.BookmarkType),
		BookmarkID:     m.BookmarkID,
		CollectionName: m.CollectionName,
		CreatedAt:      m.CreatedAt.Unix(),
	}
}

func modelsToDomainBookmarks(models []Bookmark) []*domain.Bookmark {
	result := make([]*domain.Bookmark, len(models))
	for i, m := range models {
		result[i] = modelToDomainBookmark(&m)
	}
	return result
}
