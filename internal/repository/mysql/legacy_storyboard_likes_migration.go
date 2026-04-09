package mysql

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrateLegacyPolymorphicStoryboardLikes copies rows from polymorphic `likes`
// (likeable_type storyboard_node or legacy storyboard) into `storyboard_likes`,
// deletes those `likes` rows, and realigns `storyboards.likes` when any row moved.
// Idempotent: safe on every application startup (MySQL / MariaDB).
func MigrateLegacyPolymorphicStoryboardLikes(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	if !db.Migrator().HasTable("likes") || !db.Migrator().HasTable("storyboard_likes") || !db.Migrator().HasTable("storyboards") {
		if log != nil {
			log.Debug("MigrateLegacyPolymorphicStoryboardLikes: skip (likes / storyboard_likes / storyboards not all present)")
		}
		return nil
	}

	ins := db.WithContext(ctx).Exec(`
INSERT INTO storyboard_likes (id, user_id, storyboard_id, created_at, deleted_at)
SELECT UUID(), l.user_id, l.likeable_id, FROM_UNIXTIME(l.created_at), NULL
FROM likes l
INNER JOIN storyboards s ON s.id = l.likeable_id
WHERE l.likeable_type IN ('storyboard_node', 'storyboard')
  AND l.user_id != ''
  AND l.likeable_id != ''
  AND NOT EXISTS (
    SELECT 1 FROM storyboard_likes sl
    WHERE sl.user_id = l.user_id AND sl.storyboard_id = l.likeable_id
  )
`)
	if ins.Error != nil {
		return fmt.Errorf("backfill storyboard_likes from likes: %w", ins.Error)
	}

	del := db.WithContext(ctx).Exec(`
DELETE FROM likes
WHERE likeable_type IN ('storyboard_node', 'storyboard')
`)
	if del.Error != nil {
		return fmt.Errorf("delete polymorphic storyboard likes: %w", del.Error)
	}

	if ins.RowsAffected == 0 && del.RowsAffected == 0 {
		return nil
	}

	if log != nil {
		log.Info("migrated legacy polymorphic storyboard likes to storyboard_likes",
			zap.Int64("inserted", ins.RowsAffected),
			zap.Int64("deleted_from_likes", del.RowsAffected))
	}

	rec := db.WithContext(ctx).Exec(`
UPDATE storyboards sb
SET sb.likes = (
	SELECT COUNT(*) FROM storyboard_likes sl
	WHERE sl.storyboard_id = sb.id AND sl.deleted_at IS NULL
)
`)
	if rec.Error != nil {
		return fmt.Errorf("reconcile storyboards.likes from storyboard_likes: %w", rec.Error)
	}

	return nil
}
