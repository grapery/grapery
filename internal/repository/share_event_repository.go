package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ShareEventRepository persists share issue/open funnel events.
type ShareEventRepository struct {
	db *gorm.DB
}

func NewShareEventRepository(db *gorm.DB) *ShareEventRepository {
	return &ShareEventRepository{db: db}
}

// Create inserts a share event. Empty ID is filled with a UUID.
func (r *ShareEventRepository) Create(ctx context.Context, ev *domain.ShareEvent) error {
	if r == nil || r.db == nil || ev == nil {
		return nil
	}
	if ev.ID == "" {
		ev.ID = uuid.New().String()
	}
	if ev.CreatedAt == 0 {
		ev.CreatedAt = time.Now().Unix()
	}
	return r.db.WithContext(ctx).Create(ev).Error
}
