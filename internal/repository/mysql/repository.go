package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Repository implements data access with MySQL
type Repository struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewRepository creates a new MySQL repository
func NewRepository(dsn string, log *zap.Logger) (*Repository, error) {
	// 在开发环境下显示错误和警告，生产环境只显示错误
	logLevel := logger.Error
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Info("database connected successfully")
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	repo := &Repository{db: db, log: log}

	// Auto migrate tables
	log.Info("starting database migration...")
	if err := repo.migrate(); err != nil {
		log.Error("database migration failed", zap.Error(err))
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Info("database connected and migrated successfully")
	// 注释：旧版 storyboard 数据迁移（scenes/characters 列）
	// 新数据库无需此迁移，如有旧数据需要迁移可取消注释
	// if err := repo.MigrateStoryboardLegacyData(context.Background(), 100, log); err != nil {
	// 	log.Warn("legacy storyboard migration encountered issues", zap.Error(err))
	// }
	return repo, nil
}

// migrate runs database migrations
func (r *Repository) migrate() error {
	// NOTE:
	// We intentionally avoid running global GORM AutoMigrate here.
	// Reason: if an existing schema has incompatible indexes (e.g. an index on
	// `stories.style`), AutoMigrate may try to change the column type to TEXT and
	// fail with: "BLOB/TEXT column 'style' used in key specification..."
	//
	// Instead, we run a small set of targeted, backwards-compatible schema patches
	// needed by the current code.

	r.log.Info("running targeted schema migrations")

	// 1) Ensure storyboard_video_generations has the new subdivision fields.
	if err := r.ensureStoryboardVideoGenerationSchema(); err != nil {
		return err
	}

	// 1b) Ensure storyboard_scenes has the new subdivision fields.
	if err := r.ensureStoryboardScenesSchema(); err != nil {
		return err
	}

	// 2) Ensure stories.style can store JSON (TEXT) and is not indexed.
	// If an index exists on stories.style from older schemas, we drop it before
	// converting the column to TEXT.
	if err := r.ensureStoriesStyleSchema(); err != nil {
		return err
	}

	// 3) Ensure ai_generation_records prompt fields can store Unicode (utf8mb4).
	if err := r.ensureAIGenerationRecordsSchema(); err != nil {
		return err
	}

	// 4) Ensure group_members has the role_id column for new role system.
	if err := r.ensureGroupMembersSchema(); err != nil {
		return err
	}

	// 5) Ensure group_follows table exists for group following functionality.
	if err := r.ensureGroupFollowsSchema(); err != nil {
		return err
	}

	// 6) Ensure user_devices table exists for push notifications.
	if err := r.ensureUserDevicesSchema(); err != nil {
		return err
	}

	// 6) Ensure storyboard_image_generations has prompt_details_json column.
	if err := r.ensureStoryboardImageGenerationSchema(); err != nil {
		return err
	}

	// 7) Ensure storyboard_video_generations has prompt_details_json column.
	if err := r.ensureStoryboardVideoGenerationPromptDetailsSchema(); err != nil {
		return err
	}

	// 8) Ensure characters table has portrait-related columns.
	if err := r.ensureCharacterPortraitSchema(); err != nil {
		return err
	}

	// 9) Ensure stories table has is_collaboration_open column.
	if err := r.EnsureIsCollaborationOpenColumn(r.log); err != nil {
		return err
	}

	// 10) Ensure users table has groups_count and groups_created columns.
	if err := r.EnsureUserGroupCountColumns(r.log); err != nil {
		return err
	}

	r.log.Info("targeted schema migrations completed successfully")
	return nil
}

type columnInfo struct {
	DataType   string `gorm:"column:data_type"`
	ColumnType string `gorm:"column:column_type"`
}

type columnCharsetInfo struct {
	CharacterSetName string `gorm:"column:character_set_name"`
	CollationName    string `gorm:"column:collation_name"`
	DataType         string `gorm:"column:data_type"`
}

func (r *Repository) columnExists(table, column string) (bool, error) {
	var count int64
	err := r.db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_schema = DATABASE()
		   AND table_name = ?
		   AND column_name = ?`,
		table, column,
	).Scan(&count).Error
	return count > 0, err
}

func (r *Repository) getColumnInfo(table, column string) (*columnInfo, error) {
	var info columnInfo
	err := r.db.Raw(
		`SELECT data_type, column_type FROM information_schema.columns
		 WHERE table_schema = DATABASE()
		   AND table_name = ?
		   AND column_name = ?
		 LIMIT 1`,
		table, column,
	).Scan(&info).Error
	if err != nil {
		return nil, err
	}
	if info.DataType == "" && info.ColumnType == "" {
		return nil, nil
	}
	return &info, nil
}

func (r *Repository) getColumnCharsetInfo(table, column string) (*columnCharsetInfo, error) {
	var info columnCharsetInfo
	err := r.db.Raw(
		`SELECT character_set_name, collation_name, data_type FROM information_schema.columns
		 WHERE table_schema = DATABASE()
		   AND table_name = ?
		   AND column_name = ?
		 LIMIT 1`,
		table, column,
	).Scan(&info).Error
	if err != nil {
		return nil, err
	}
	if info.DataType == "" {
		return nil, nil
	}
	return &info, nil
}

func (r *Repository) ensureColumn(table, column, definition string) error {
	exists, err := r.columnExists(table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", table, column, definition)
	return r.db.Exec(sql).Error
}

type indexRow struct {
	IndexName string `gorm:"column:index_name"`
}

func (r *Repository) listIndexesOnColumn(table, column string) ([]string, error) {
	rows, err := r.db.Raw(
		`SELECT DISTINCT index_name FROM information_schema.statistics
		 WHERE table_schema = DATABASE()
		   AND table_name = ?
		   AND column_name = ?
		   AND index_name <> 'PRIMARY'`,
		table, column,
	).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var idx string
		// MySQL driver may return []uint8 for text columns; scanning into string is safe.
		if err := rows.Scan(&idx); err != nil {
			return nil, err
		}
		if idx != "" {
			out = append(out, idx)
		}
	}
	return out, rows.Err()
}

func (r *Repository) ensureStoryboardVideoGenerationSchema() error {
	// These columns are referenced by inserts in storyboard_generation_impl.go
	// and by converters for keyframe subdivision playback support.
	if err := r.ensureColumn("storyboard_video_generations", "is_subdivided", "TINYINT(1) NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("ensure storyboard_video_generations.is_subdivided: %w", err)
	}
	if err := r.ensureColumn("storyboard_video_generations", "video_segments_json", "TEXT NULL"); err != nil {
		return fmt.Errorf("ensure storyboard_video_generations.video_segments_json: %w", err)
	}
	if err := r.ensureColumn("storyboard_video_generations", "middle_frame_urls_json", "TEXT NULL"); err != nil {
		return fmt.Errorf("ensure storyboard_video_generations.middle_frame_urls_json: %w", err)
	}
	return nil
}

func (r *Repository) ensureStoryboardScenesSchema() error {
	// These columns are referenced by StoryboardScene model for keyframe subdivision support.
	if err := r.ensureColumn("storyboard_scenes", "is_subdivided", "TINYINT(1) NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("ensure storyboard_scenes.is_subdivided: %w", err)
	}
	if err := r.ensureColumn("storyboard_scenes", "video_segments_json", "TEXT NULL"); err != nil {
		return fmt.Errorf("ensure storyboard_scenes.video_segments_json: %w", err)
	}
	if err := r.ensureColumn("storyboard_scenes", "middle_frame_urls", "TEXT NULL"); err != nil {
		return fmt.Errorf("ensure storyboard_scenes.middle_frame_urls: %w", err)
	}
	return nil
}

func (r *Repository) ensureStoriesStyleSchema() error {
	// If stories.style doesn't exist, do nothing here (older schemas may not have it).
	exists, err := r.columnExists("stories", "style")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	info, err := r.getColumnInfo("stories", "style")
	if err != nil {
		return err
	}

	// If it's already TEXT-like, we only need to ensure no indexes remain.
	isTextLike := false
	if info != nil {
		switch info.DataType {
		case "text", "mediumtext", "longtext":
			isTextLike = true
		}
	}

	// Drop any indexes involving stories.style; MySQL disallows TEXT in indexes
	// unless a prefix length is specified.
	indexes, err := r.listIndexesOnColumn("stories", "style")
	if err != nil {
		return err
	}
	for _, idx := range indexes {
		r.log.Warn("dropping incompatible index on stories.style", zap.String("index", idx))
		if err := r.db.Exec(fmt.Sprintf("DROP INDEX `%s` ON `stories`", idx)).Error; err != nil {
			return fmt.Errorf("drop index %s on stories: %w", idx, err)
		}
	}

	// Convert to TEXT if needed.
	if !isTextLike {
		if err := r.db.Exec("ALTER TABLE `stories` MODIFY COLUMN `style` TEXT").Error; err != nil {
			return fmt.Errorf("alter stories.style to TEXT: %w", err)
		}
	}
	return nil
}

func (r *Repository) ensureAIGenerationRecordsSchema() error {
	// If the table doesn't exist (fresh DB), nothing to do.
	exists, err := r.columnExists("ai_generation_records", "original_prompt")
	if err != nil {
		return err
	}
	if !exists {
		r.log.Info("ai_generation_records.original_prompt not found; skipping ai_generation_records charset migration")
		return nil
	}

	// These columns store user/system prompts and can include Chinese/emoji etc.
	cols := []string{"original_prompt", "enhanced_prompt", "system_prompt", "error_message"}
	needAlter := false

	for _, c := range cols {
		info, err := r.getColumnCharsetInfo("ai_generation_records", c)
		if err != nil {
			return err
		}
		// Some columns may not exist in older schemas; skip them.
		if info == nil {
			r.log.Info("ai_generation_records column not found; skipping", zap.String("column", c))
			continue
		}
		r.log.Info("ai_generation_records column charset",
			zap.String("column", c),
			zap.String("charset", info.CharacterSetName),
			zap.String("collation", info.CollationName),
			zap.String("dataType", info.DataType),
		)
		// Only applies to character types; JSON columns report nil charset.
		if info.CharacterSetName != "utf8mb4" {
			needAlter = true
			r.log.Warn("ai_generation_records column not utf8mb4; will convert",
				zap.String("column", c),
				zap.String("charset", info.CharacterSetName),
				zap.String("collation", info.CollationName),
				zap.String("dataType", info.DataType),
			)
		}
	}

	if !needAlter {
		r.log.Info("ai_generation_records prompt columns already utf8mb4; no charset migration needed")
		return nil
	}

	// Convert only the prompt-related TEXT columns to utf8mb4; keep types as TEXT/LONGTEXT.
	// This avoids a full table CONVERT (which can be heavier on large tables).
	for _, c := range cols {
		info, err := r.getColumnInfo("ai_generation_records", c)
		if err != nil {
			return err
		}
		if info == nil {
			continue
		}
		// Preserve existing column_type (e.g. text/mediumtext/longtext) while changing charset/collation.
		sql := fmt.Sprintf(
			"ALTER TABLE `ai_generation_records` MODIFY COLUMN `%s` %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
			c,
			info.ColumnType,
		)
		if err := r.db.Exec(sql).Error; err != nil {
			return fmt.Errorf("alter ai_generation_records.%s to utf8mb4: %w", c, err)
		}
		r.log.Info("ai_generation_records column converted to utf8mb4",
			zap.String("column", c),
			zap.String("columnType", info.ColumnType),
		)
	}
	return nil
}

func (r *Repository) ensureGroupMembersSchema() error {
	// Add role_id column for the new role system.
	// This column references the group_roles table for flexible permission management.
	if err := r.ensureColumn("group_members", "role_id", "VARCHAR(36) NULL"); err != nil {
		return fmt.Errorf("ensure group_members.role_id: %w", err)
	}
	return nil
}

func (r *Repository) ensureGroupFollowsSchema() error {
	// Check if group_follows table exists
	hasTable := r.db.Migrator().HasTable(&GroupFollow{})
	if !hasTable {
		r.log.Info("group_follows table not found, creating it...")
		// Use AutoMigrate to create table with all its columns and indexes
		if err := r.db.AutoMigrate(&GroupFollow{}); err != nil {
			return fmt.Errorf("failed to create group_follows table: %w", err)
		}
		r.log.Info("group_follows table created successfully")
	}
	return nil
}

func (r *Repository) ensureUserDevicesSchema() error {
	// Check if user_devices table exists
	hasTable := r.db.Migrator().HasTable(&UserDevice{})
	if !hasTable {
		r.log.Info("user_devices table not found, creating it...")
		// Use AutoMigrate to create the table with all its columns and indexes
		if err := r.db.AutoMigrate(&UserDevice{}); err != nil {
			return fmt.Errorf("failed to create user_devices table: %w", err)
		}
		r.log.Info("user_devices table created successfully")
	} else {
		r.log.Info("user_devices table already exists")
	}
	return nil
}

func (r *Repository) ensureStoryboardImageGenerationSchema() error {
	// Add prompt_details_json column if it doesn't exist
	// This column stores structured prompt details for client editing
	if err := r.ensureColumn("storyboard_image_generations", "prompt_details_json", "JSON NULL"); err != nil {
		return fmt.Errorf("ensure storyboard_image_generations.prompt_details_json: %w", err)
	}
	return nil
}

func (r *Repository) ensureStoryboardVideoGenerationPromptDetailsSchema() error {
	// Add prompt_details_json column if it doesn't exist
	// This column stores structured prompt details for client editing
	if err := r.ensureColumn("storyboard_video_generations", "prompt_details_json", "JSON NULL"); err != nil {
		return fmt.Errorf("ensure storyboard_video_generations.prompt_details_json: %w", err)
	}
	return nil
}

func (r *Repository) ensureCharacterPortraitSchema() error {
	// Add portrait column if it doesn't exist
	// This column stores the full character portrait image URL (AI-generated)
	if err := r.ensureColumn("characters", "portrait", "VARCHAR(500) DEFAULT NULL COMMENT '完整角色形象图URL（AI生成）'"); err != nil {
		return fmt.Errorf("ensure characters.portrait: %w", err)
	}

	// Add needs_portrait column if it doesn't exist
	// This column indicates whether portrait generation is needed
	if err := r.ensureColumn("characters", "needs_portrait", "TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否需要生成形象'"); err != nil {
		return fmt.Errorf("ensure characters.needs_portrait: %w", err)
	}

	// Add reference_image column if it doesn't exist
	// This column stores reference image URL for portrait generation
	if err := r.ensureColumn("characters", "reference_image", "VARCHAR(500) DEFAULT NULL COMMENT '参考图URL'"); err != nil {
		return fmt.Errorf("ensure characters.reference_image: %w", err)
	}

	// Add portrait_generation_status column if it doesn't exist
	// This column tracks the status of portrait generation: none/pending/generating/generated/failed
	if err := r.ensureColumn("characters", "portrait_generation_status", "VARCHAR(20) DEFAULT 'none' COMMENT '形象生成状态: none/pending/generating/generated/failed'"); err != nil {
		return fmt.Errorf("ensure characters.portrait_generation_status: %w", err)
	}

	// Ensure index exists on portrait_generation_status
	indexExists, err := r.indexExists("characters", "idx_characters_portrait_status")
	if err != nil {
		return fmt.Errorf("check index idx_characters_portrait_status: %w", err)
	}
	if !indexExists {
		if err := r.db.Exec("CREATE INDEX idx_characters_portrait_status ON characters(portrait_generation_status)").Error; err != nil {
			// Index might already exist, ignore error if it's a duplicate key error
			if !strings.Contains(err.Error(), "Duplicate key name") {
				return fmt.Errorf("create index idx_characters_portrait_status: %w", err)
			}
		}
	}

	return nil
}

// indexExists checks if an index exists on a table
func (r *Repository) indexExists(table, indexName string) (bool, error) {
	var count int64
	err := r.db.Raw(
		`SELECT COUNT(*) FROM information_schema.statistics
		 WHERE table_schema = DATABASE()
		   AND table_name = ?
		   AND index_name = ?`,
		table, indexName,
	).Scan(&count).Error
	return count > 0, err
}

// CurrentUser returns the current authenticated user (mock for now)
func (r *Repository) CurrentUser(ctx context.Context) (domain.User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "username = ?", "storyteller_pro").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default user if not exists
			user = User{
				ID:          uuid.New().String(),
				Username:    "storyteller_pro",
				DisplayName: "Alex Morgan",
				Avatar:      "https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=150&h=150&fit=crop",
				Background:  "https://images.unsplash.com/photo-1681230745734-4e59736c3660?w=1200&h=300&fit=crop",
				Bio:         "Passionate storyteller and world builder.",
				Followers:   1247,
				Following:   432,
			}
			if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
				return domain.User{}, err
			}
		} else {
			return domain.User{}, err
		}
	}
	return r.userToDomain(user), nil
}

// GetUser retrieves a user by ID
func (r *Repository) GetUser(ctx context.Context, id string) (domain.User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return domain.User{}, err
	}
	return r.userToDomain(user), nil
}

// ListStories retrieves stories with filters
func (r *Repository) ListStories(ctx context.Context, filter domain.StoryFilter) ([]*domain.Story, int64, error) {
	var stories []Story
	var total int64

	query := r.db.WithContext(ctx).Model(&Story{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.AuthorID != "" {
		query = query.Where("author_id = ?", filter.AuthorID)
	}
	if filter.GroupID != "" {
		query = query.Where("group_id = ?", filter.GroupID)
	}
	if filter.Search != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}
	if filter.Genre != "" {
		query = query.Where("genre = ?", filter.Genre)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	if err := query.Preload("Author").Order("updated_at DESC").Find(&stories).Error; err != nil {
		return nil, 0, err
	}

	// Collect story IDs
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
	for i, s := range stories {
		story := r.storyToDomain(s)
		story.CharacterCount = characterCounts[s.ID]
		result[i] = &story
	}
	return result, total, nil
}

// GetStory retrieves a story by ID
func (r *Repository) GetStory(ctx context.Context, id string) (domain.Story, error) {
	var story Story
	if err := r.db.WithContext(ctx).Preload("Author").First(&story, "id = ?", id).Error; err != nil {
		return domain.Story{}, err
	}
	return r.storyToDomain(story), nil
}

// CreateStory creates a new story
func (r *Repository) CreateStory(ctx context.Context, story *domain.Story) error {
	dbStory := Story{
		ID:                  uuid.New().String(),
		Title:               story.Title,
		Description:         story.Description,
		CoverImage:          story.CoverImage,
		AuthorID:            story.Author.ID,
		Genre:               story.Genre,
		Status:              story.Status,
		Likes:               0,
		Followers:           0,
		Panels:              0,
		IsCollaborationOpen: story.IsCollaborationOpen,
	}

	if err := r.db.WithContext(ctx).Create(&dbStory).Error; err != nil {
		return err
	}

	// 更新传入的 story 对象的 ID
	story.ID = dbStory.ID
	story.CreatedAt = dbStory.CreatedAt.Unix()
	story.UpdatedAt = dbStory.UpdatedAt.Unix()
	return nil
}

// PanelsByStory retrieves panels for a story
func (r *Repository) PanelsByStory(ctx context.Context, storyID string) ([]*domain.Panel, error) {
	var panels []Panel
	if err := r.db.WithContext(ctx).Where("story_id = ?", storyID).Order("sequence ASC").Find(&panels).Error; err != nil {
		return nil, err
	}

	// Load characters for the story once
	characters, err := r.CharactersByStory(ctx, storyID)
	if err != nil {
		r.log.Warn("failed to load characters for story panels", zap.String("storyID", storyID), zap.Error(err))
		characters = []*domain.Character{}
	}

	result := make([]*domain.Panel, len(panels))
	for i, p := range panels {
		panel := r.panelToDomainWithCharacters(p, characters)
		result[i] = &panel
	}
	return result, nil
}

// ListCharacters retrieves all characters
func (r *Repository) ListCharacters(ctx context.Context, limit, offset int) ([]*domain.Character, error) {
	var characters []Character
	query := r.db.WithContext(ctx).Preload("Author").Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&characters).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Character, len(characters))
	for i, c := range characters {
		char := r.characterToDomain(c)
		result[i] = &char
	}
	return result, nil
}

// GetCharacter retrieves a character by ID
func (r *Repository) GetCharacter(ctx context.Context, id string) (domain.Character, error) {
	var character Character
	if err := r.db.WithContext(ctx).Preload("Author").First(&character, "id = ?", id).Error; err != nil {
		return domain.Character{}, err
	}
	return r.characterToDomain(character), nil
}

// ListGroups retrieves all groups
func (r *Repository) ListGroups(ctx context.Context, limit, offset int) ([]*domain.Group, error) {
	var groups []Group
	query := r.db.WithContext(ctx).Preload("Creator").Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&groups).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Group, len(groups))
	for i, g := range groups {
		group := r.groupToDomain(g)
		result[i] = &group
	}
	return result, nil
}

// ListMyGroups retrieves groups that a user is a member of
func (r *Repository) ListMyGroups(ctx context.Context, userID string, limit, offset int) ([]*domain.Group, error) {
	var groups []Group
	query := r.db.WithContext(ctx).
		Joins("INNER JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Preload("Creator").
		Order("groups.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&groups).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Group, len(groups))
	for i, g := range groups {
		group := r.groupToDomain(g)
		result[i] = &group
	}
	return result, nil
}

// ListPublicGroups retrieves public groups that a user is not a member of
func (r *Repository) ListPublicGroups(ctx context.Context, userID string, limit, offset int) ([]*domain.Group, error) {
	var groups []Group
	query := r.db.WithContext(ctx).
		Where("public = ?", true).
		Preload("Creator").
		Order("created_at DESC")

	// Exclude groups the user is already a member of
	if userID != "" {
		query = query.Where("id NOT IN (?)",
			r.db.Table("group_members").
				Select("group_id").
				Where("user_id = ?", userID))
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&groups).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Group, len(groups))
	for i, g := range groups {
		group := r.groupToDomain(g)
		result[i] = &group
	}
	return result, nil
}

// GetGroup retrieves a group by ID
func (r *Repository) GetGroup(ctx context.Context, id string) (domain.Group, error) {
	var group Group
	if err := r.db.WithContext(ctx).Preload("Creator").First(&group, "id = ?", id).Error; err != nil {
		return domain.Group{}, err
	}
	return r.groupToDomain(group), nil
}

// CommentsByStory retrieves comments for a story
func (r *Repository) CommentsByStory(ctx context.Context, storyID string) ([]*domain.Comment, error) {
	var comments []Comment
	if err := r.db.WithContext(ctx).Preload("Author").Where("story_id = ? AND parent_id IS NULL", storyID).Order("created_at DESC").Find(&comments).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Comment, len(comments))
	for i, c := range comments {
		comment := r.commentToDomain(c)
		// Load replies
		comment.Replies = r.loadCommentReplies(ctx, c.ID)
		result[i] = &comment
	}
	return result, nil
}

func (r *Repository) loadCommentReplies(ctx context.Context, parentID string) []domain.Comment {
	var replies []Comment
	if err := r.db.WithContext(ctx).Preload("Author").Where("parent_id = ?", parentID).Order("created_at ASC").Find(&replies).Error; err != nil {
		return []domain.Comment{}
	}

	result := make([]domain.Comment, len(replies))
	for i, reply := range replies {
		result[i] = r.commentToDomain(reply)
	}
	return result
}

// // ChatThreads retrieves chat threads for a user
// func (r *Repository) ChatThreads(ctx context.Context, userID string) ([]*domain.ChatThread, error) {
// 	var threads []ChatThread
// 	if err := r.db.WithContext(ctx).Preload("Character").Where("user_id = ?", userID).Order("last_message_time DESC").Find(&threads).Error; err != nil {
// 		return nil, err
// 	}

// 	result := make([]*domain.ChatThread, len(threads))
// 	for i, t := range threads {
// 		thread := r.chatThreadToDomain(t)
// 		result[i] = &thread
// 	}
// 	return result, nil
// }

// // ChatMessages retrieves messages for a thread
// func (r *Repository) ChatMessages(ctx context.Context, threadID string, limit, offset int) ([]*domain.ChatMessage, error) {
// 	var messages []ChatMessage
// 	query := r.db.WithContext(ctx).Where("thread_id = ?", threadID).Order("created_at ASC")

// 	if limit > 0 {
// 		query = query.Limit(limit).Offset(offset)
// 	}

// 	if err := query.Find(&messages).Error; err != nil {
// 		return nil, err
// 	}

// 	result := make([]*domain.ChatMessage, len(messages))
// 	for i, m := range messages {
// 		msg := r.chatMessageToDomain(m)
// 		result[i] = &msg
// 	}
// 	return result, nil
// }

// AppendChatMessage adds a new message to a thread
func (r *Repository) AppendChatMessage(ctx context.Context, msg domain.ChatMessage) error {
	dbMsg := ChatMessage{
		ID:           uuid.New().String(),
		ThreadID:     msg.ThreadID,
		SenderID:     msg.SenderID,
		SenderName:   msg.SenderName,
		SenderAvatar: msg.SenderAvatar,
		Content:      msg.Content,
		Image:        msg.Image,
		IsUser:       msg.IsUser,
	}

	if err := r.db.WithContext(ctx).Create(&dbMsg).Error; err != nil {
		return err
	}

	// Update thread
	updates := map[string]interface{}{
		"last_message":      msg.Content,
		"last_message_time": time.Now(),
		"message_count":     gorm.Expr("message_count + 1"),
	}
	if !msg.IsUser {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
	}

	return r.db.WithContext(ctx).Model(&ChatThread{}).Where("id = ?", msg.ThreadID).Updates(updates).Error
}

// StoryCompositions retrieves all story compositions
func (r *Repository) StoryCompositions(ctx context.Context) ([]domain.StoryComposition, error) {
	var compositions []StoryComposition
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&compositions).Error; err != nil {
		return nil, err
	}

	result := make([]domain.StoryComposition, len(compositions))
	for i, c := range compositions {
		result[i] = r.compositionToDomain(c, ctx)
	}
	return result, nil
}

// Storyboard retrieves a storyboard by ID
func (r *Repository) Storyboard(ctx context.Context, id string) (domain.Storyboard, error) {
	var storyboard Storyboard
	if err := r.db.WithContext(ctx).Preload("Creator").First(&storyboard, "id = ?", id).Error; err != nil {
		return domain.Storyboard{}, err
	}
	return r.storyboardToDomain(ctx, storyboard)
}

// GroupActivities retrieves activities for a group
func (r *Repository) GroupActivities(ctx context.Context, groupID string, limit int) ([]*domain.GroupActivity, error) {
	var activities []GroupActivity
	query := r.db.WithContext(ctx).Preload("User").Where("group_id = ?", groupID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	} else {
		query = query.Limit(50)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.GroupActivity, len(activities))
	for i, a := range activities {
		activity := r.activityToDomain(a)
		result[i] = &activity
	}
	return result, nil
}

// CreateGroupActivity creates a new group activity record
func (r *Repository) CreateGroupActivity(ctx context.Context, activity *domain.GroupActivity) error {
	model := GroupActivityToModel(activity)
	return r.db.WithContext(ctx).Create(model).Error
}

// GroupActivitiesByTimeRange retrieves activities within a time range
func (r *Repository) GroupActivitiesByTimeRange(ctx context.Context, groupID string, startTime, endTime int64, limit, offset int) ([]*domain.GroupActivity, error) {
	var activities []GroupActivity
	query := r.db.WithContext(ctx).
		Preload("User").
		Preload("Story").
		Where("group_id = ?", groupID).
		Where("created_at >= ?", time.Unix(startTime, 0)).
		Where("created_at <= ?", time.Unix(endTime, 0)).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	} else {
		query = query.Limit(50)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.GroupActivity, len(activities))
	for i, a := range activities {
		activity := r.activityToDomain(a)
		// Add date field for frontend grouping
		activity.Date = a.CreatedAt.Format("2006-01-02")
		result[i] = &activity
	}
	return result, nil
}

// GroupActivitiesByDate retrieves activities for a specific date
func (r *Repository) GroupActivitiesByDate(ctx context.Context, groupID string, date string, limit, offset int) ([]*domain.GroupActivity, error) {
	var activities []GroupActivity

	// Parse date string to get start and end of day
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}

	startOfDay := parsedDate
	endOfDay := parsedDate.Add(24*time.Hour - time.Second)

	query := r.db.WithContext(ctx).
		Preload("User").
		Preload("Story").
		Where("group_id = ?", groupID).
		Where("created_at >= ?", startOfDay).
		Where("created_at <= ?", endOfDay).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	} else {
		query = query.Limit(50)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.GroupActivity, len(activities))
	for i, a := range activities {
		activity := r.activityToDomain(a)
		activity.Date = a.CreatedAt.Format("2006-01-02")
		result[i] = &activity
	}
	return result, nil
}

// GroupActivityHeatmap retrieves activity counts per day for heatmap visualization
// Uses China timezone (UTC+8) for date grouping
func (r *Repository) GroupActivityHeatmap(ctx context.Context, groupID string, startTime, endTime int64) ([]*domain.ActivityHeatmapData, error) {
	type DateCount struct {
		Date  string `gorm:"column:date"`
		Count int    `gorm:"column:count"`
	}

	var dateCounts []DateCount

	startT := time.Unix(startTime, 0)
	endT := time.Unix(endTime, 0)

	// Group activities by date and count using DATE_FORMAT with China timezone conversion
	// CONVERT_TZ converts from UTC ('+00:00') to China Standard Time ('+08:00')
	err := r.db.WithContext(ctx).
		Model(&GroupActivity{}).
		Select("DATE_FORMAT(CONVERT_TZ(created_at, '+00:00', '+08:00'), '%Y-%m-%d') as date, COUNT(*) as count").
		Where("group_id = ?", groupID).
		Where("created_at >= ?", startT).
		Where("created_at <= ?", endT).
		Where("deleted_at IS NULL").
		Group("DATE_FORMAT(CONVERT_TZ(created_at, '+00:00', '+08:00'), '%Y-%m-%d')").
		Order("date ASC").
		Scan(&dateCounts).Error

	if err != nil {
		return nil, err
	}

	result := make([]*domain.ActivityHeatmapData, len(dateCounts))
	for i, dc := range dateCounts {
		result[i] = &domain.ActivityHeatmapData{
			Date:  dc.Date,
			Count: dc.Count,
		}
	}
	return result, nil
}

// Domain conversion helpers

func (r *Repository) userToDomain(u User) domain.User {
	return domain.User{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Avatar:      u.Avatar,
		Background:  u.Background,
		Bio:         u.Bio,
		Followers:   u.Followers,
		Following:   u.Following,
		CreatedAt:   u.CreatedAt,
	}
}

func (r *Repository) storyToDomain(s Story) domain.Story {
	author := r.userToDomain(s.Author)
	groupID := ""
	if s.GroupID != nil {
		groupID = *s.GroupID
	}
	return domain.Story{
		ID:                  s.ID,
		AuthorID:            s.AuthorID,
		GroupID:             groupID,
		Title:               s.Title,
		Description:         s.Description,
		CoverImage:          s.CoverImage,
		Author:              &author,
		Likes:               s.Likes,
		Followers:           s.Followers,
		Panels:              s.Panels,
		StoryboardCount:     s.StoryboardCount,
		Genre:               s.Genre,
		Style:               jsonToStyleConfig(s.Style),
		Status:              s.Status,
		IsCollaborationOpen: s.IsCollaborationOpen,
		CreatedAt:           s.CreatedAt.Unix(),
		UpdatedAt:           s.UpdatedAt.Unix(),
	}
}

func (r *Repository) panelToDomain(p Panel) domain.Panel {
	return r.panelToDomainWithCharacters(p, nil)
}

func (r *Repository) panelToDomainWithCharacters(p Panel, characters []*domain.Character) domain.Panel {
	// Convert character pointers to values
	domainCharacters := make([]domain.Character, 0, len(characters))
	for _, c := range characters {
		if c != nil {
			domainCharacters = append(domainCharacters, *c)
		}
	}

	return domain.Panel{
		ID:         p.ID,
		StoryID:    p.StoryID,
		Sequence:   p.Sequence,
		Title:      p.Title,
		Content:    p.Content,
		Image:      p.Image,
		Characters: domainCharacters,
		Likes:      p.Likes,
		Published:  p.Published,
		CreatedAt:  p.CreatedAt.Unix(),
	}
}

func (r *Repository) characterToDomain(c Character) domain.Character {
	var traits []string
	if c.Traits != "" {
		json.Unmarshal([]byte(c.Traits), &traits)
	}
	var skills []string
	if c.Skills != "" {
		json.Unmarshal([]byte(c.Skills), &skills)
	}

	author := r.userToDomain(c.Author)
	return domain.Character{
		ID:                       c.ID,
		StoryID:                  c.StoryID,
		AuthorID:                 c.AuthorID,
		Name:                     c.Name,
		Description:              c.Description,
		Avatar:                   c.Avatar,
		Poster:                   c.Poster,
		Portrait:                 c.Portrait,
		NeedsPortrait:            c.NeedsPortrait,
		ReferenceImage:           c.ReferenceImage,
		PortraitGenerationStatus: c.PortraitGenerationStatus,
		Author:                   &author,
		Personality:              c.Personality,
		Background:               c.Background,
		ShortTermGoal:            c.ShortTermGoal,
		LongTermGoal:             c.LongTermGoal,
		HandlingStyle:            c.HandlingStyle,
		CognitionRange:           c.CognitionRange,
		AbilityFeatures:          c.AbilityFeatures,
		Appearance:               c.Appearance,
		DressPreference:          c.DressPreference,
		Traits:                   traits,
		Skills:                   skills,
		TraitsJSON:               c.Traits,
		SkillsJSON:               c.Skills,
		IsPublic:                 c.IsPublic,
		SourceType:               c.SourceType,
		SourcePrompt:             c.SourcePrompt,
		SourceImage:              c.SourceImage,
		CreatedBy:                c.CreatedBy,
		LastEditedBy:             c.LastEditedBy,
		GroupID:                  c.GroupID,
		Likes:                    c.Likes,
		Followers:                c.Followers,
		Stories:                  c.Stories,
		CreatedAt:                c.CreatedAt.Unix(),
		UpdatedAt:                c.UpdatedAt.Unix(),
	}
}

func (r *Repository) groupToDomain(g Group) domain.Group {
	creator := r.userToDomain(g.Creator)
	return domain.Group{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		Avatar:      g.Avatar,
		CoverImage:  g.CoverImage,
		Members:     g.Members,
		Stories:     g.Stories,
		Followers:   g.Followers,
		Creator:     &creator,
		Public:      g.Public,
		CreatedAt:   g.CreatedAt.Unix(),
		UpdatedAt:   g.UpdatedAt.Unix(),
	}
}

// // commentToDomain 已移至 comment_impl.go

// func (r *Repository) chatThreadToDomain(t ChatThread) domain.ChatThread {
// 	return domain.ChatThread{
// 		ID:                   t.ID,
// 		CharacterID:          t.CharacterID,
// 		CharacterName:        t.Character.Name,
// 		CharacterAvatar:      t.Character.Avatar,
// 		StoryTitle:           t.StoryTitle,
// 		LastMessage:          t.LastMessage,
// 		LastMessageTime:      t.LastMessageTime,
// 		UnreadCount:          t.UnreadCount,
// 		MessageCount:         t.MessageCount,
// 		InteractionFrequency: t.InteractionFrequency,
// 		CreatedAt: t.CreatedAt.Unix(),
// 	}
// }

// func (r *Repository) chatMessageToDomain(m ChatMessage) domain.ChatMessage {
// 	return domain.ChatMessage{
// 		ID:           m.ID,
// 		ThreadID:     m.ThreadID,
// 		SenderID:     m.SenderID,
// 		SenderName:   m.SenderName,
// 		SenderAvatar: m.SenderAvatar,
// 		Content:      m.Content,
// 		Image:        m.Image,
// 		Timestamp:    m.CreatedAt,
// 		IsUser:       m.IsUser,
// 	}
// }

func (r *Repository) compositionToDomain(c StoryComposition, ctx context.Context) domain.StoryComposition {
	var participants []StoryParticipant
	r.db.WithContext(ctx).Preload("User").Where("composition_id = ?", c.ID).Find(&participants)

	domainParticipants := make([]domain.StoryParticipant, len(participants))
	for i, p := range participants {
		domainParticipants[i] = domain.StoryParticipant{
			ID:       p.ID,
			UserID:   p.UserID,
			Name:     p.User.DisplayName,
			Avatar:   p.User.Avatar,
			Role:     p.Role,
			JoinedAt: p.JoinedAt.Unix(),
		}
	}

	return domain.StoryComposition{
		ID:                    c.ID,
		Title:                 c.Title,
		CoverImage:            c.CoverImage,
		BackgroundDescription: c.Background,
		Theme:                 c.Theme,
		Genre:                 c.Genre,
		RootStoryboardID:      c.RootStoryboardID,
		Participants:          domainParticipants,
		TotalStoryboards:      c.TotalStoryboards,
		TotalForks:            c.TotalForks,
		CreatedAt:             c.CreatedAt.Unix(),
		UpdatedAt:             c.UpdatedAt.Unix(),
	}
}

// storyboardToDomain 已移至 storyboard_impl.go

func (r *Repository) activityToDomain(a GroupActivity) domain.GroupActivity {
	storyTitle := ""
	storyID := ""
	if a.StoryID != nil && a.Story != nil {
		storyTitle = a.Story.Title
		storyID = *a.StoryID
	}

	return domain.GroupActivity{
		ID:         a.ID,
		Type:       a.Type,
		UserID:     a.UserID,
		UserName:   a.User.DisplayName,
		UserAvatar: a.User.Avatar,
		StoryID:    storyID,
		StoryTitle: storyTitle,
		Message:    a.Message,
		Timestamp:  a.CreatedAt.Unix(),
	}
}
