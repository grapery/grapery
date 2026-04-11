package mysql

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// CreatorAnalyticsAggregate rolls up views/likes/saves/comments/shares/forks for stories authored by userID.
func (r *Repository) CreatorAnalyticsAggregate(ctx context.Context, userID string) (*domain.CreatorAnalyticsAggregate, error) {
	var row struct {
		TotalViews           int64 `gorm:"column:total_views"`
		TotalStoryLikes      int64 `gorm:"column:story_likes"`
		TotalStoryboardLikes int64 `gorm:"column:sb_likes"`
		TotalSaves           int64 `gorm:"column:saves"`
		TotalComments        int64 `gorm:"column:sb_comments"`
		TotalShares          int64 `gorm:"column:sb_shares"`
		TotalForks           int64 `gorm:"column:forks"`
	}

	q := `
SELECT
  COALESCE(SUM(sb.views), 0) AS total_views,
  COALESCE(SUM(s.likes), 0) AS story_likes,
  COALESCE(SUM(sb.likes), 0) AS sb_likes,
  COALESCE(SUM(s.saves), 0) AS saves,
  COALESCE(SUM(sb.comments), 0) AS sb_comments,
  COALESCE(SUM(sb.shares), 0) AS sb_shares,
  COALESCE(SUM(sb.fork_count), 0) AS forks
FROM stories s
LEFT JOIN storyboards sb ON sb.story_id = s.id AND sb.deleted_at IS NULL
WHERE s.author_id = ? AND s.deleted_at IS NULL
`
	if err := r.db.WithContext(ctx).Raw(q, userID).Scan(&row).Error; err != nil {
		return nil, err
	}

	return &domain.CreatorAnalyticsAggregate{
		TotalViews:           row.TotalViews,
		TotalStoryLikes:      row.TotalStoryLikes,
		TotalStoryboardLikes: row.TotalStoryboardLikes,
		TotalSaves:           row.TotalSaves,
		TotalComments:        row.TotalComments,
		TotalShares:          row.TotalShares,
		TotalForks:           row.TotalForks,
	}, nil
}

// TopCreatorStoryboards returns top storyboards by views for stories owned by userID.
func (r *Repository) TopCreatorStoryboards(ctx context.Context, userID string, limit int) ([]*domain.CreatorAnalyticsStoryboardRow, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	var rows []struct {
		ID      string `gorm:"column:id"`
		Title   string `gorm:"column:title"`
		Views   int    `gorm:"column:views"`
		Likes   int    `gorm:"column:likes"`
		StoryID string `gorm:"column:story_id"`
	}

	q := `
SELECT sb.id, sb.title, sb.views, sb.likes, sb.story_id
FROM storyboards sb
INNER JOIN stories s ON s.id = sb.story_id AND s.deleted_at IS NULL
WHERE s.author_id = ? AND sb.deleted_at IS NULL
ORDER BY sb.views DESC, sb.likes DESC, sb.updated_at DESC
LIMIT ?
`
	if err := r.db.WithContext(ctx).Raw(q, userID, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]*domain.CreatorAnalyticsStoryboardRow, 0, len(rows))
	for _, rw := range rows {
		out = append(out, &domain.CreatorAnalyticsStoryboardRow{
			ID:      rw.ID,
			Title:   rw.Title,
			Views:   rw.Views,
			Likes:   rw.Likes,
			StoryID: rw.StoryID,
		})
	}
	return out, nil
}
