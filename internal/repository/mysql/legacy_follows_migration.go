package mysql

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrateLegacyPolymorphicFollowsThenDrop copies rows from legacy `follows` into story_follows,
// character_follows, and user_follows, then drops `follows`. Safe if the table is already absent.
func MigrateLegacyPolymorphicFollowsThenDrop(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	if !db.Migrator().HasTable("follows") {
		return nil
	}
	steps := []struct {
		sql  string
		desc string
	}{
		{
			desc: "story follows",
			sql: `INSERT IGNORE INTO story_follows (id, user_id, story_id, created_at)
				SELECT id, follower_id, followable_id, FROM_UNIXTIME(created_at) FROM follows WHERE LOWER(followable_type) = 'story' AND followable_id != '' AND follower_id != ''`,
		},
		{
			desc: "character follows",
			sql: `INSERT IGNORE INTO character_follows (id, user_id, character_id, created_at)
				SELECT id, follower_id, followable_id, FROM_UNIXTIME(created_at) FROM follows WHERE LOWER(followable_type) = 'character' AND followable_id != '' AND follower_id != ''`,
		},
		{
			desc: "user follows",
			sql: `INSERT IGNORE INTO user_follows (id, follower_id, followee_id, created_at)
				SELECT id, follower_id, followable_id, FROM_UNIXTIME(created_at) FROM follows WHERE LOWER(followable_type) = 'user' AND followable_id != '' AND follower_id != ''`,
		},
	}
	for _, step := range steps {
		if err := db.WithContext(ctx).Exec(step.sql).Error; err != nil {
			return fmt.Errorf("migrate legacy follows (%s): %w", step.desc, err)
		}
	}
	if err := db.WithContext(ctx).Exec("DROP TABLE IF EXISTS `follows`").Error; err != nil {
		return fmt.Errorf("drop legacy follows table: %w", err)
	}
	if log != nil {
		log.Info("legacy polymorphic follows table dropped after migrating to dedicated follow tables")
	}
	return nil
}
