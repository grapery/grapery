package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
		if err := r.db.Exec("ALTER TABLE stories ADD COLUMN is_collaboration_open BOOLEAN DEFAULT FALSE NOT NULL COMMENT 'Whether collaboration is open: true=anyone can edit, false=only author can edit'").Error; err != nil {
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

// ensureUserSettingsPreferredGenresColumn ensures the user_settings table has preferred_genres_json column
func (r *Repository) ensureUserSettingsPreferredGenresColumn() error {
	migrator := r.db.Migrator()
	type UserSettings struct{}
	if !migrator.HasColumn(&UserSettings{}, "preferred_genres_json") {
		r.log.Info("Adding preferred_genres_json column to user_settings table")
		if err := r.db.Exec("ALTER TABLE user_settings ADD COLUMN preferred_genres_json JSON NULL COMMENT 'onboarding multi-select genres'").Error; err != nil {
			r.log.Error("failed to add preferred_genres_json column", zap.Error(err))
			return err
		}
		r.log.Info("Successfully added preferred_genres_json column to user_settings table")
	}
	return nil
}

// ensureStoriesAIEnabledColumn ensures the stories table has the ai_enabled column
func (r *Repository) ensureStoriesAIEnabledColumn() error {
	migrator := r.db.Migrator()
	type Story struct{}

	if !migrator.HasColumn(&Story{}, "ai_enabled") {
		r.log.Info("Adding ai_enabled column to stories table")
		if err := r.db.Exec("ALTER TABLE stories ADD COLUMN ai_enabled BOOLEAN DEFAULT TRUE NOT NULL COMMENT '是否允许AI辅助'").Error; err != nil {
			r.log.Error("failed to add ai_enabled column", zap.Error(err))
			return err
		}
		r.log.Info("Successfully added ai_enabled column to stories table")
	} else {
		r.log.Debug("ai_enabled column already exists in stories table")
	}

	return nil
}

// ensureCharactersPosterCreationPermissionColumn ensures the characters table has the poster_creation_permission column
func (r *Repository) ensureCharactersPosterCreationPermissionColumn() error {
	migrator := r.db.Migrator()
	type Character struct{}

	if !migrator.HasColumn(&Character{}, "poster_creation_permission") {
		r.log.Info("Adding poster_creation_permission column to characters table")
		if err := r.db.Exec("ALTER TABLE characters ADD COLUMN poster_creation_permission VARCHAR(50) DEFAULT 'creator_only' NOT NULL COMMENT '海报创建权限: creator_only, anyone'").Error; err != nil {
			r.log.Error("failed to add poster_creation_permission column", zap.Error(err))
			return err
		}
		r.log.Info("Successfully added poster_creation_permission column to characters table")
	} else {
		r.log.Debug("poster_creation_permission column already exists in characters table")
	}

	return nil
}

// ensureStoryboardVideoGenerationSchema ensures storyboard_video_generations has subdivision fields
func (r *Repository) ensureStoryboardVideoGenerationSchema() error {
	migrator := r.db.Migrator()
	type StoryboardVideoGeneration struct{}

	columns := []struct {
		name    string
		def     string
		comment string
	}{
		{"target_audience", "VARCHAR(100) DEFAULT ''", "目标受众"},
		{"narrative_style", "VARCHAR(100) DEFAULT ''", "叙事风格"},
		{"visual_references", "TEXT", "视觉参考"},
	}

	for _, col := range columns {
		if !migrator.HasColumn(&StoryboardVideoGeneration{}, col.name) {
			r.log.Info("Adding column to storyboard_video_generations", zap.String("column", col.name))
			if err := r.db.Exec(fmt.Sprintf("ALTER TABLE storyboard_video_generations ADD COLUMN %s %s COMMENT '%s'", col.name, col.def, col.comment)).Error; err != nil {
				r.log.Error("failed to add column", zap.String("column", col.name), zap.Error(err))
				return err
			}
		}
	}

	return nil
}

// ensureStoryboardScenesSchema ensures storyboard_scenes has subdivision fields
func (r *Repository) ensureStoryboardScenesSchema() error {
	migrator := r.db.Migrator()
	type StoryboardScene struct{}

	columns := []struct {
		name    string
		def     string
		comment string
	}{
		{"camera_angle", "VARCHAR(100) DEFAULT ''", "镜头角度"},
		{"lighting", "VARCHAR(100) DEFAULT ''", "光照"},
		{"color_palette", "VARCHAR(100) DEFAULT ''", "色彩方案"},
	}

	for _, col := range columns {
		if !migrator.HasColumn(&StoryboardScene{}, col.name) {
			r.log.Info("Adding column to storyboard_scenes", zap.String("column", col.name))
			if err := r.db.Exec(fmt.Sprintf("ALTER TABLE storyboard_scenes ADD COLUMN %s %s COMMENT '%s'", col.name, col.def, col.comment)).Error; err != nil {
				r.log.Error("failed to add column", zap.String("column", col.name), zap.Error(err))
				return err
			}
		}
	}

	return nil
}

// ensureStoriesStyleSchema ensures stories.style can store JSON (TEXT) without index
func (r *Repository) ensureStoriesStyleSchema() error {
	migrator := r.db.Migrator()
	type Story struct{}

	// Check if style column exists and is VARCHAR - convert to TEXT
	if migrator.HasColumn(&Story{}, "style") {
		// Check column type - if it's VARCHAR, alter to TEXT
		var columnType string
		row := r.db.Raw("SELECT DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'stories' AND COLUMN_NAME = 'style'").Row()
		if err := row.Scan(&columnType); err == nil && columnType == "varchar" {
			r.log.Info("Converting stories.style from VARCHAR to TEXT")
			if err := r.db.Exec("ALTER TABLE stories MODIFY COLUMN style TEXT COMMENT '故事风格 (JSON)'").Error; err != nil {
				r.log.Error("failed to convert style column to TEXT", zap.Error(err))
				return err
			}
		}
	} else {
		r.log.Info("Adding style column to stories table")
		if err := r.db.Exec("ALTER TABLE stories ADD COLUMN style TEXT COMMENT '故事风格 (JSON)'").Error; err != nil {
			r.log.Error("failed to add style column", zap.Error(err))
			return err
		}
	}

	return nil
}

func aiGenerationRecordsUTF8PriorityColumns() []struct {
	column string
	def    string
} {
	return []struct {
		column string
		def    string
	}{
		{"original_prompt", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
		{"enhanced_prompt", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
		{"system_prompt", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
		{"error_message", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
		// 含中文的 JSON 若落在 latin1 TEXT 上会 1366；改为 utf8mb4 LONGTEXT（仍存 JSON 字符串，与 GORM type:json 兼容）
		{"input_params", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
		{"output_result", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
		{"metadata", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
	}
}

// ForceAIGenerationRecordsUTF8MB4Columns 对已知列逐个执行 MODIFY（幂等）。在仍出现 1366 时调用，避免 information_schema 与真实表不一致。
func ForceAIGenerationRecordsUTF8MB4Columns(db *gorm.DB, log *zap.Logger) {
	for _, p := range aiGenerationRecordsUTF8PriorityColumns() {
		q := fmt.Sprintf("ALTER TABLE `ai_generation_records` MODIFY COLUMN `%s` %s", p.column, p.def)
		if err := db.Exec(q).Error; err != nil {
			if log != nil {
				log.Warn("ai_generation_records utf8mb4 ALTER failed or column missing",
					zap.String("column", p.column), zap.Error(err))
			}
			continue
		}
		if log != nil {
			log.Info("ai_generation_records column coerced to utf8mb4", zap.String("column", p.column))
		}
	}
}

// ApplyAIGenerationRecordsUTF8MB4IfNeeded runs before migrations on each server start. If the table already exists
// but original_prompt is not utf8mb4 (common after legacy latin1 schemas), ALTER fixes MySQL 1366 on Chinese text.
func ApplyAIGenerationRecordsUTF8MB4IfNeeded(db *gorm.DB, log *zap.Logger) error {
	var tableCount int64
	if err := db.Raw(`
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = 'ai_generation_records'
`).Scan(&tableCount).Error; err != nil {
		return fmt.Errorf("check ai_generation_records table: %w", err)
	}
	if tableCount == 0 {
		return nil
	}

	var rows []struct {
		ColumnName       string         `gorm:"column:column_name"`
		CharacterSetName sql.NullString `gorm:"column:character_set_name"`
		DataType         string         `gorm:"column:data_type"`
	}
	if err := db.Raw(`
SELECT COLUMN_NAME AS column_name, CHARACTER_SET_NAME AS character_set_name, DATA_TYPE AS data_type
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = 'ai_generation_records'
  AND COLUMN_NAME IN (
    'original_prompt','enhanced_prompt','system_prompt','error_message',
    'input_params','output_result','metadata'
  )
`).Scan(&rows).Error; err != nil {
		return fmt.Errorf("list ai_generation_records text column charsets: %w", err)
	}
	meta := make(map[string]struct {
		Charset  sql.NullString
		DataType string
	}, len(rows))
	for _, r := range rows {
		meta[r.ColumnName] = struct {
			Charset  sql.NullString
			DataType string
		}{r.CharacterSetName, strings.ToLower(strings.TrimSpace(r.DataType))}
	}

	for _, p := range aiGenerationRecordsUTF8PriorityColumns() {
		m, ok := meta[p.column]
		if !ok {
			continue
		}
		if m.Charset.Valid && strings.EqualFold(strings.TrimSpace(m.Charset.String), "utf8mb4") {
			continue
		}
		// 原生 JSON 类型在 information_schema 中常无 CHARACTER_SET_NAME，按 MySQL 语义已是 utf8mb4，跳过以免无谓改类型
		if !m.Charset.Valid && m.DataType == "json" {
			continue
		}

		if log != nil {
			log.Warn("ai_generation_records column is not utf8mb4; applying ALTER (fixes MySQL 1366 for Chinese/emoji)",
				zap.String("column", p.column),
				zap.Any("character_set", m.Charset),
				zap.String("data_type", m.DataType))
		}
		q := fmt.Sprintf("ALTER TABLE `ai_generation_records` MODIFY COLUMN `%s` %s", p.column, p.def)
		if err := db.Exec(q).Error; err != nil {
			return fmt.Errorf("ai_generation_records: MODIFY %s to utf8mb4: %w (grant ALTER or run SQL manually)", p.column, err)
		}
		if log != nil {
			log.Info("ai_generation_records column set to utf8mb4", zap.String("column", p.column))
		}
	}
	return nil
}

// ensureAIGenerationRecordsSchema ensures ai_generation_records supports Unicode (utf8mb4) for all
// prompt and error text, including original_prompt / enhanced_prompt / system_prompt (not only legacy prompt columns).
// MySQL error 1366 on UTF-8 bytes indicates a latin1 (or non-utf8mb4) column charset.
//
// We MODIFY critical LONGTEXT columns first (fixes 1366 even if CONVERT fails on some hosts), then CONVERT the table.
func (r *Repository) ensureAIGenerationRecordsSchema() error {
	const table = "`ai_generation_records`"

	if err := ApplyAIGenerationRecordsUTF8MB4IfNeeded(r.db, r.log); err != nil {
		return err
	}

	const convertSQL = "ALTER TABLE `ai_generation_records` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
	convertErr := r.db.Exec(convertSQL).Error
	if convertErr == nil {
		r.log.Info("ai_generation_records table converted to utf8mb4")
		return nil
	}
	r.log.Warn("ai_generation_records CONVERT TO utf8mb4 failed, applying remaining per-column MODIFY",
		zap.Error(convertErr))

	// 仅旧版表存在 prompt / negative_prompt；当前模型已用 original_prompt 等，缺列则跳过以免 1054 噪音
	legacy := []struct {
		column string
		def    string
	}{
		{"prompt", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
		{"negative_prompt", "LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"},
	}
	for _, col := range legacy {
		var n int64
		if err := r.db.Raw(`
SELECT COUNT(*) FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'ai_generation_records' AND COLUMN_NAME = ?
`, col.column).Scan(&n).Error; err != nil {
			r.log.Warn("legacy ai_generation_records column check failed",
				zap.String("column", col.column), zap.Error(err))
			continue
		}
		if n == 0 {
			continue
		}
		q := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN `%s` %s", table, col.column, col.def)
		if err := r.db.Exec(q).Error; err != nil {
			r.log.Warn("failed to modify column to utf8mb4",
				zap.String("column", col.column), zap.Error(err))
		}
	}

	return nil
}

// ensureStoryboardImageGenerationSchema ensures storyboard_image_generations has prompt_details_json column
func (r *Repository) ensureStoryboardImageGenerationSchema() error {
	migrator := r.db.Migrator()
	type StoryboardImageGeneration struct{}

	if !migrator.HasColumn(&StoryboardImageGeneration{}, "prompt_details_json") {
		r.log.Info("Adding prompt_details_json column to storyboard_image_generations")
		if err := r.db.Exec("ALTER TABLE storyboard_image_generations ADD COLUMN prompt_details_json TEXT COMMENT 'Prompt details JSON'").Error; err != nil {
			r.log.Error("failed to add prompt_details_json column", zap.Error(err))
			return err
		}
	}
	if !migrator.HasColumn(&StoryboardImageGeneration{}, "PipelineKind") {
		r.log.Info("Adding pipeline_kind column to storyboard_image_generations")
		if err := r.db.Exec("ALTER TABLE storyboard_image_generations ADD COLUMN pipeline_kind VARCHAR(24) NOT NULL DEFAULT '' COMMENT 'scene | comic_page'").Error; err != nil {
			r.log.Error("failed to add pipeline_kind column", zap.Error(err))
			return err
		}
	}

	return nil
}

// ensureStoryboardVideoGenerationPromptDetailsSchema ensures storyboard_video_generations has prompt_details_json column
func (r *Repository) ensureStoryboardVideoGenerationPromptDetailsSchema() error {
	migrator := r.db.Migrator()
	type StoryboardVideoGeneration struct{}

	if !migrator.HasColumn(&StoryboardVideoGeneration{}, "prompt_details_json") {
		r.log.Info("Adding prompt_details_json column to storyboard_video_generations")
		if err := r.db.Exec("ALTER TABLE storyboard_video_generations ADD COLUMN prompt_details_json TEXT COMMENT 'Prompt details JSON'").Error; err != nil {
			r.log.Error("failed to add prompt_details_json column", zap.Error(err))
			return err
		}
	}

	return nil
}

// ensureStoryboardContinuationGenerationOptionsSchema adds continuation generation option columns on storyboards.
func (r *Repository) ensureStoryboardContinuationGenerationOptionsSchema() error {
	migrator := r.db.Migrator()
	if !migrator.HasColumn(&Storyboard{}, "GenerateVideoAfterImages") {
		r.log.Info("Adding generate_video_after_images to storyboards")
		if err := r.db.Exec(`ALTER TABLE storyboards ADD COLUMN generate_video_after_images TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'After scene image: also generate video from image'`).Error; err != nil {
			r.log.Error("failed to add generate_video_after_images", zap.Error(err))
			return err
		}
	}
	if !migrator.HasColumn(&Storyboard{}, "ContinuationComicStyle") {
		r.log.Info("Adding continuation_comic_style to storyboards")
		if err := r.db.Exec(`ALTER TABLE storyboards ADD COLUMN continuation_comic_style VARCHAR(80) NOT NULL DEFAULT '' COMMENT 'Comic/manga style slug for AI continuation'`).Error; err != nil {
			r.log.Error("failed to add continuation_comic_style", zap.Error(err))
			return err
		}
	}
	return nil
}

// ensureStoryboardUseComicPagePipelineSchema adds use_comic_page_pipeline (向导「多格漫画页」配菜策略).
func (r *Repository) ensureStoryboardUseComicPagePipelineSchema() error {
	migrator := r.db.Migrator()
	if !migrator.HasColumn(&Storyboard{}, "UseComicPagePipeline") {
		r.log.Info("Adding use_comic_page_pipeline to storyboards")
		if err := r.db.Exec(`ALTER TABLE storyboards ADD COLUMN use_comic_page_pipeline TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Skip auto GenerateSceneImage; client uses comic-page pipeline'`).Error; err != nil {
			r.log.Error("failed to add use_comic_page_pipeline", zap.Error(err))
			return err
		}
	}
	return nil
}

// ensureStoryboardContinuationSummarySchema adds continuation_summary for fork/continuation context compression.
func (r *Repository) ensureStoryboardContinuationSummarySchema() error {
	migrator := r.db.Migrator()
	if !migrator.HasColumn(&Storyboard{}, "ContinuationSummary") {
		r.log.Info("Adding continuation_summary to storyboards")
		if err := r.db.Exec(`ALTER TABLE storyboards ADD COLUMN continuation_summary TEXT NULL COMMENT 'Lossy narrative summary for continuation/fork context'`).Error; err != nil {
			r.log.Error("failed to add continuation_summary", zap.Error(err))
			return err
		}
	}
	return nil
}

// ensureCharacterPortraitSchema ensures characters has portrait-related columns
func (r *Repository) ensureCharacterPortraitSchema() error {
	migrator := r.db.Migrator()
	type Character struct{}

	columns := []struct {
		name    string
		def     string
		comment string
	}{
		{"portrait_style", "VARCHAR(100) DEFAULT ''", "Portrait style"},
		{"portrait_background", "VARCHAR(100) DEFAULT ''", "Portrait background"},
		{"portrait_lighting", "VARCHAR(100) DEFAULT ''", "Portrait lighting"},
		{"portrait_angle", "VARCHAR(100) DEFAULT ''", "Portrait camera angle"},
		{"portrait_expression", "VARCHAR(100) DEFAULT ''", "Portrait expression"},
	}

	for _, col := range columns {
		if !migrator.HasColumn(&Character{}, col.name) {
			r.log.Info("Adding column to characters", zap.String("column", col.name))
			if err := r.db.Exec(fmt.Sprintf("ALTER TABLE characters ADD COLUMN %s %s COMMENT '%s'", col.name, col.def, col.comment)).Error; err != nil {
				r.log.Error("failed to add column", zap.String("column", col.name), zap.Error(err))
				return err
			}
		}
	}

	return nil
}
