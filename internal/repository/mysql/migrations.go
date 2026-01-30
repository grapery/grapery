package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// MigrateStoryboardLegacyData converts legacy inline storyboard scenes/characters
// into story-scoped assets and association links.
func (r *Repository) MigrateStoryboardLegacyData(ctx context.Context, batchSize int, logger *zap.Logger) error {
	if batchSize <= 0 {
		batchSize = 50
	}

	type legacyStoryboard struct {
		ID         string
		StoryID    string
		CreatorID  string
		ScenesJSON string
		CharsJSON  string
	}

	offset := 0
	for {
		var rows []legacyStoryboard
		err := r.db.WithContext(ctx).
			Table("storyboards").
			Select("id, story_id, creator_id, scenes as scenes_json, characters as chars_json").
			Where("(scenes IS NOT NULL AND scenes <> '') OR (characters IS NOT NULL AND characters <> '')").
			Limit(batchSize).
			Offset(offset).
			Scan(&rows).Error
		if err != nil {
			return fmt.Errorf("scan legacy storyboard rows: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			if err := r.migrateSingleStoryboard(ctx, row, logger); err != nil {
				return err
			}
		}

		offset += len(rows)
	}

	if err := r.dropLegacyStoryboardColumns(logger); err != nil {
		return err
	}

	return nil
}

func (r *Repository) migrateSingleStoryboard(ctx context.Context, row struct {
	ID         string
	StoryID    string
	CreatorID  string
	ScenesJSON string
	CharsJSON  string
}, logger *zap.Logger) error {
	// LegacyScene for parsing old scene data in migrations
	type LegacyScene struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Image       string `json:"image"`
		Location    string `json:"location,omitempty"`
		TimeOfDay   string `json:"timeOfDay,omitempty"`
	}

	var sceneRefs []domain.StoryboardSceneRef
	if row.ScenesJSON != "" {
		var scenes []LegacyScene
		if err := json.Unmarshal([]byte(row.ScenesJSON), &scenes); err != nil {
			logger.Warn("skip legacy storyboard scenes due to unmarshal failure",
				zap.String("storyboardId", row.ID),
				zap.Error(err))
		} else {
			for i := range scenes {
				scene := scenes[i]
				now := time.Now().Unix()
				storyScene := &domain.StoryScene{
					ID:           "",
					StoryID:      row.StoryID,
					Title:        scene.Title,
					Description:  scene.Description,
					Image:        scene.Image,
					Location:     scene.Location,
					TimeOfDay:    scene.TimeOfDay,
					SourceType:   "legacy",
					SourcePrompt: "",
					SourceImage:  scene.Image,
					CreatedBy:    row.CreatorID,
					LastEditedBy: row.CreatorID,
					IsPublic:     false,
					CreatedAt:    now,
					UpdatedAt:    now,
				}

				if err := r.CreateStoryScene(ctx, storyScene); err != nil {
					return fmt.Errorf("create migrated story scene: %w", err)
				}

				sceneRefs = append(sceneRefs, domain.StoryboardSceneRef{
					StorySceneID:   storyScene.ID,
					Sequence:       i,
					IsPrimaryScene: i == 0,
				})
			}
		}
	}

	var charRefs []domain.StoryboardCharacterRef
	if row.CharsJSON != "" {
		var characters []domain.StoryboardCharacter
		if err := json.Unmarshal([]byte(row.CharsJSON), &characters); err != nil {
			logger.Warn("skip legacy storyboard characters due to unmarshal failure",
				zap.String("storyboardId", row.ID),
				zap.Error(err))
		} else {
			for i := range characters {
				char := characters[i]
				character := &domain.Character{
					StoryID:      row.StoryID,
					Name:         char.Name,
					Avatar:       char.Avatar,
					Description:  "",
					SourceType:   "legacy",
					SourcePrompt: "",
					SourceImage:  "",
					CreatedBy:    row.CreatorID,
					LastEditedBy: row.CreatorID,
					IsPublic:     false,
				}

				if err := r.CreateCharacter(ctx, character); err != nil {
					return fmt.Errorf("create migrated character: %w", err)
				}

				charRefs = append(charRefs, domain.StoryboardCharacterRef{
					CharacterID: character.ID,
					Role:        char.Role,
					Order:       i,
					Notes:       "",
				})
			}
		}
	}

	if len(sceneRefs) > 0 {
		if err := r.AttachScenesToStoryboard(ctx, row.ID, sceneRefs); err != nil {
			return fmt.Errorf("attach migrated scenes: %w", err)
		}
	}

	if len(charRefs) > 0 {
		if err := r.AttachCharactersToStoryboard(ctx, row.ID, charRefs); err != nil {
			return fmt.Errorf("attach migrated characters: %w", err)
		}
	}

	// 清空旧字段，避免重复迁移
	if row.ScenesJSON != "" || row.CharsJSON != "" {
		if err := r.db.WithContext(ctx).
			Model(&Storyboard{}).
			Where("id = ?", row.ID).
			Updates(map[string]interface{}{
				"scenes":     "",
				"characters": "",
			}).Error; err != nil {
			logger.Warn("failed to clear legacy storyboard JSON columns",
				zap.String("storyboardId", row.ID),
				zap.Error(err))
		}
	}

	return nil
}

func (r *Repository) dropLegacyStoryboardColumns(logger *zap.Logger) error {
	migrator := r.db.Migrator()
	type tableName struct{}

	if migrator.HasColumn(&Storyboard{}, "scenes") {
		if err := r.db.Exec("ALTER TABLE storyboards DROP COLUMN scenes").Error; err != nil {
			logger.Warn("failed to drop legacy column storyboards.scenes", zap.Error(err))
			return err
		}
		logger.Info("Dropped legacy column storyboards.scenes")
	}

	if migrator.HasColumn(&Storyboard{}, "characters") {
		if err := r.db.Exec("ALTER TABLE storyboards DROP COLUMN characters").Error; err != nil {
			logger.Warn("failed to drop legacy column storyboards.characters", zap.Error(err))
			return err
		}
		logger.Info("Dropped legacy column storyboards.characters")
	}

	if migrator.HasColumn(&Storyboard{}, "images") {
		if err := r.db.Exec("ALTER TABLE storyboards DROP COLUMN images").Error; err != nil {
			logger.Warn("failed to drop legacy column storyboards.images", zap.Error(err))
			return err
		}
		logger.Info("Dropped legacy column storyboards.images")
	}

	return nil
}

// EnsureIsCollaborationOpenColumn ensures is_collaboration_open column exists in stories table
func (r *Repository) EnsureIsCollaborationOpenColumn(logger *zap.Logger) error {
	migrator := r.db.Migrator()
	type Story struct{}

	if !migrator.HasColumn(&Story{}, "is_collaboration_open") {
		logger.Info("Adding is_collaboration_open column to stories table")
		if err := r.db.Exec("ALTER TABLE stories ADD COLUMN is_collaboration_open BOOLEAN DEFAULT FALSE NOT NULL COMMENT 'Whether collaboration is open: true=anyone can edit, false=only author and group members can edit'").Error; err != nil {
			logger.Error("failed to add is_collaboration_open column", zap.Error(err))
			return err
		}

		// Create index for query performance
		if err := r.db.Exec("CREATE INDEX idx_stories_is_collaboration_open ON stories(is_collaboration_open)").Error; err != nil {
			logger.Warn("failed to create index on is_collaboration_open", zap.Error(err))
			// Don't return error - index creation is not critical
		}

		logger.Info("Successfully added is_collaboration_open column to stories table")
	} else {
		logger.Debug("is_collaboration_open column already exists in stories table")
	}

	return nil
}

// EnsureUserGroupCountColumns ensures groups_count and groups_created columns exist in users table
func (r *Repository) EnsureUserGroupCountColumns(logger *zap.Logger) error {
	migrator := r.db.Migrator()
	type User struct{}

	if !migrator.HasColumn(&User{}, "groups_count") {
		logger.Info("Adding groups_count column to users table")
		if err := r.db.Exec("ALTER TABLE users ADD COLUMN groups_count INT DEFAULT 0 NOT NULL COMMENT 'Number of groups the user has joined'").Error; err != nil {
			logger.Error("failed to add groups_count column", zap.Error(err))
			return err
		}
		logger.Info("Successfully added groups_count column to users table")
	} else {
		logger.Debug("groups_count column already exists in users table")
	}

	if !migrator.HasColumn(&User{}, "groups_created") {
		logger.Info("Adding groups_created column to users table")
		if err := r.db.Exec("ALTER TABLE users ADD COLUMN groups_created INT DEFAULT 0 NOT NULL COMMENT 'Number of groups created by this user'").Error; err != nil {
			logger.Error("failed to add groups_created column", zap.Error(err))
			return err
		}
		logger.Info("Successfully added groups_created column to users table")
	} else {
		logger.Debug("groups_created column already exists in users table")
	}

	return nil
}

// ensureGroupsBlockedCountColumn ensures the groups table has the blocked_count column
func (r *Repository) ensureGroupsBlockedCountColumn() error {
	migrator := r.db.Migrator()
	type Group struct{}

	if !migrator.HasColumn(&Group{}, "blocked_count") {
		r.log.Info("Adding blocked_count column to groups table")
		if err := r.db.Exec("ALTER TABLE groups ADD COLUMN blocked_count INT DEFAULT 0 NOT NULL COMMENT 'Number of blocked users in this group'").Error; err != nil {
			r.log.Error("failed to add blocked_count column", zap.Error(err))
			return err
		}
		r.log.Info("Successfully added blocked_count column to groups table")
	} else {
		r.log.Debug("blocked_count column already exists in groups table")
	}

	return nil
}

// ensureUserFragmentsCountColumn ensures the users table has the fragments_count column
func (r *Repository) ensureUserFragmentsCountColumn() error {
	migrator := r.db.Migrator()
	type User struct{}

	if !migrator.HasColumn(&User{}, "fragments_count") {
		r.log.Info("Adding fragments_count column to users table")
		if err := r.db.Exec("ALTER TABLE users ADD COLUMN fragments_count INT DEFAULT 0 NOT NULL COMMENT 'Number of fragments created by this user'").Error; err != nil {
			r.log.Error("failed to add fragments_count column", zap.Error(err))
			return err
		}
		r.log.Info("Successfully added fragments_count column to users table")
	} else {
		r.log.Debug("fragments_count column already exists in users table")
	}

	return nil
}

