package mysql

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/repository/migrations"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// storiesSourceFragmentUniqueIndex is the GORM default name for Story.SourceFragmentID uniqueIndex.
const storiesSourceFragmentUniqueIndex = "idx_stories_source_fragment_id"

// mysqlTableOptionsUTF8MB4 is appended to every CREATE TABLE from AutoMigrate so new tables default to
// utf8mb4; varchar FK columns then match users.id and avoid MySQL ER 3780 (incompatible referencing columns).
const mysqlTableOptionsUTF8MB4 = "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"

func isDuplicateMySQLIndex(err error, indexName string) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	if !strings.Contains(s, indexName) {
		return false
	}
	return strings.Contains(s, "Duplicate key name") || strings.Contains(s, "1061")
}

// autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex wraps db.AutoMigrate. GORM may re-migrate
// Story when migrating models that declare a Story relation (FK), repeating CREATE UNIQUE INDEX on
// source_fragment_id; MySQL ER_DUP_KEYNAME (1061) is then treated as success for idempotent startup.
func autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db *gorm.DB, log *zap.Logger, dst ...interface{}) error {
	session := db.Session(&gorm.Session{}).Set("gorm:table_options", mysqlTableOptionsUTF8MB4)
	err := session.AutoMigrate(dst...)
	if err == nil {
		return nil
	}
	if isDuplicateMySQLIndex(err, storiesSourceFragmentUniqueIndex) {
		if log != nil {
			log.Warn("AutoMigrate: stories source_fragment unique index already exists, skipping duplicate create",
				zap.String("index", storiesSourceFragmentUniqueIndex),
				zap.Error(err))
		}
		return nil
	}
	return err
}

func autoMigrateStories(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	_ = ctx
	return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Story{})
}

// init 自动注册 mysql 包的迁移步骤
func init() {
	registry := migrations.GetRegistry()

	// ========== 核心实体表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_users",
		Description: "Create and migrate users table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &User{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_login_records",
		Description: "Create and migrate user_login_records table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserLoginRecord{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_account_deletion_blocks",
		Description: "Create account_deletion_blocks for post-deletion registration cooldown",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AccountDeletionBlock{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_account_deletion_requests",
		Description: "Create account_deletion_requests for phased account closure",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AccountDeletionRequest{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "seed_system_anonymous_deleted_user_placeholder",
		Description: "Ensure system anonymous user exists for orphaned public content (SYSTEM_ANONYMOUS_USER_ID)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			_ = ctx
			raw := ""
			if log != nil {
				raw = os.Getenv("SYSTEM_ANONYMOUS_USER_ID")
				log.Info("seeding system anonymous user if absent", zap.String("configuredID", strings.TrimSpace(raw)))
			}
			systemID := config.EffectiveSystemAnonymousUserID(strings.TrimSpace(os.Getenv("SYSTEM_ANONYMOUS_USER_ID")))
			return SeedSystemAnonymousUser(db, log, systemID)
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_settings",
		Description: "Create and migrate user_settings table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserSettings{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_genre_catalog_entries",
		Description: "Create genre_catalog_entries for discovery feed genre preferences",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &GenreCatalogEntry{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "seed_genre_catalog_page0",
		Description: "Seed default discovery genres (page 0) when genre_catalog_entries is empty",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			_ = ctx
			return SeedGenreCatalogPage0IfEmpty(db, func(msg string, args ...interface{}) {
				if log != nil {
					log.Info(fmt.Sprintf(msg, args...))
				}
			})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_feedback",
		Description: "Create user_feedback table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserFeedback{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_devices",
		Description: "Create and migrate user_devices table for push notifications",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserDevice{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_statistics",
		Description: "Create and migrate user_statistics table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserStatistics{})
		},
		Required: true,
	})

	// REMOVED: migrate_user_activities - not in StoryCreationAppUI design

	// ========== 故事相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_stories",
		Description: "Create and migrate stories table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateStories(ctx, db, log)
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_panels",
		Description: "Create and migrate panels table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Panel{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_contributors",
		Description: "Create and migrate story_contributors table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryContributor{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_compositions",
		Description: "Create and migrate story_compositions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryComposition{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_participants",
		Description: "Create and migrate story_participants table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryParticipant{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_tags",
		Description: "Create and migrate story_tags table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryTag{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_likes",
		Description: "Create and migrate story_likes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryLike{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_follows",
		Description: "Create and migrate story_follows table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryFollow{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_bookmarks",
		Description: "Create and migrate bookmarks table (story / fragment / storyboard saves)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Bookmark{})
		},
		Required: true,
	})

	// ========== Storyboard 相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboards",
		Description: "Create and migrate storyboards table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Storyboard{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_panels",
		Description: "Create and migrate storyboard_panels table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardPanel{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_content_generations",
		Description: "Create and migrate storyboard_content_generations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardContentGeneration{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_scene_generations",
		Description: "Create and migrate storyboard_scene_generations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardSceneGeneration{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_image_generations",
		Description: "Create and migrate storyboard_image_generations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardImageGeneration{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_video_generations",
		Description: "Create and migrate storyboard_video_generations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardVideoGeneration{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_generation_runs",
		Description: "Create and migrate storyboard_generation_runs table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardGenerationRun{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_generation_assets",
		Description: "Create and migrate storyboard_generation_assets table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardGenerationAsset{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_ai_prompt_audit_records",
		Description: "Create and migrate ai_prompt_audit_records table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AIPromptAuditRecord{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_likes",
		Description: "Create and migrate storyboard_likes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardLike{})
		},
		Required: true,
	})

	// ========== 角色相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_characters",
		Description: "Create and migrate characters table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Character{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_character_generation_tasks",
		Description: "Create and migrate character_generation_tasks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &CharacterGenerationTask{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "fix_character_generation_tasks_empty_character_id",
		Description: "Set character_id to NULL where empty so FK fk_character_generation_tasks_character allows pending tasks",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			mig := db.WithContext(ctx).Migrator()
			if !mig.HasTable(&CharacterGenerationTask{}) {
				// 部分环境仅执行到本步或前置 AutoMigrate 未落表时，先补建表再执行数据修正。
				log.Warn("character_generation_tasks table missing before fix step; running AutoMigrate for CharacterGenerationTask")
				if err := autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &CharacterGenerationTask{}); err != nil {
					return fmt.Errorf("ensure character_generation_tasks exists: %w", err)
				}
			}
			if err := db.WithContext(ctx).Exec(
				"UPDATE character_generation_tasks SET character_id = NULL WHERE character_id = ''",
			).Error; err != nil {
				return err
			}
			if err := db.WithContext(ctx).Exec(
				"ALTER TABLE character_generation_tasks MODIFY COLUMN character_id VARCHAR(36) NULL",
			).Error; err != nil {
				log.Warn("character_generation_tasks.character_id nullable alter skipped", zap.Error(err))
			}
			return nil
		},
		Required: true,
	})

	// REMOVED: migrate_character_posters - not in StoryCreationAppUI design

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_character_analytics",
		Description: "Create and migrate character_analytics table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &CharacterAnalytics{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_character_tags",
		Description: "Create and migrate character_tags table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &CharacterTag{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_character_follows",
		Description: "Create and migrate character_follows table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &CharacterFollow{})
		},
		Required: true,
	})

	// ========== 场景相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_scenes",
		Description: "Create and migrate story_scenes table (story-scoped locations)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryScene{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_scenes",
		Description: "Create and migrate storyboard_scenes table (AI-generated plot scenes)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardScene{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_character_links",
		Description: "Create and migrate storyboard_character_links table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardCharacterLink{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_scene_links",
		Description: "Create and migrate storyboard_scene_links table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryboardSceneLink{})
		},
		Required: true,
	})

	// ========== 群组相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_comments",
		Description: "Create and migrate comments table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Comment{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_comment_likes",
		Description: "Create and migrate comment_likes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &CommentLike{})
		},
		Required: true,
	})

	// ========== 标签系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_tags",
		Description: "Create and migrate tags table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Tag{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_style_configs",
		Description: "Create and migrate style_configs table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StyleConfig{})
		},
		Required: true,
	})

	// ========== 搜索和浏览表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_search_histories",
		Description: "Create and migrate search_histories table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &SearchHistory{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_view_histories",
		Description: "Create and migrate view_histories table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &ViewHistory{})
		},
		Required: true,
	})

	// ========== Agent 系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agents",
		Description: "Create and migrate agents table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Agent{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agent_skills",
		Description: "Create and migrate agent_skills table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AgentSkill{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agent_skill_usages",
		Description: "Create and migrate agent_skill_usages table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AgentSkillUsage{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agent_interactions",
		Description: "Create and migrate agent_interactions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AgentInteraction{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agent_memories",
		Description: "Create and migrate agent_memories table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AgentMemory{})
		},
		Required: true,
	})

	// ========== AI 任务系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_ai_tasks",
		Description: "Create and migrate ai_tasks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AITask{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_render_tasks",
		Description: "Create and migrate render_tasks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &RenderTask{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_publications",
		Description: "Create and migrate story_publications table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &StoryPublication{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_ai_generation_records",
		Description: "Create and migrate ai_generation_records table; force utf8mb4 on prompt columns (fixes MySQL 1366)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			if err := autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &AIGenerationRecord{}); err != nil {
				return err
			}
			repo := &Repository{db: db, log: log}
			return repo.ensureAIGenerationRecordsSchema()
		},
		Required: true,
	})

	// ========== 通知系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_notifications",
		Description: "Create and migrate notifications table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Notification{})
		},
		Required: true,
	})

	// ========== 资产管理表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_assets",
		Description: "Create and migrate assets table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Asset{})
		},
		Required: true,
	})

	// ========== 支付订阅表（核心） ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_memberships",
		Description: "Create and migrate memberships table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Membership{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_subscription_plans",
		Description: "Create and migrate subscription_plans table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &SubscriptionPlan{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_subscription_orders",
		Description: "Create and migrate subscription_orders table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &SubscriptionOrder{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_token_transactions",
		Description: "Create and migrate token_transactions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &TokenTransaction{})
		},
		Required: true,
	})

	// ========== 邀请码系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_invitation_codes",
		Description: "Create and migrate invitation_codes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &InvitationCode{})
		},
		Required: true,
	})

	// ========== 邀请推荐系统表 (StoryCreationAppUI Design) ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_referrals",
		Description: "Create and migrate user_referrals table for referral system",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserReferral{})
		},
		Required: true,
	})

	// ========== 第三方登录表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_third_party_logins",
		Description: "Create and migrate third_party_logins table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &ThirdPartyLogin{})
		},
		Required: true,
	})

	// ========== 关系表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_follows",
		Description: "Create and migrate user_follows table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserFollow{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_blocks",
		Description: "Create and migrate user_blocks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserBlock{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_reports",
		Description: "Create and migrate user_reports table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserReport{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_content_reports",
		Description: "Create and migrate content_reports table (UGC moderation, App Store guideline 1.2)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &ContentReport{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_report_review_metadata",
		Description: "Add review_remarks/reviewed_by/reviewed_at to user_reports and content_reports (Forge moderation)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &UserReport{}, &ContentReport{})
		},
		Required: true,
	})

	// ========== Fragment 相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragments",
		Description: "Create and migrate fragments table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_generation_tasks",
		Description: "Create and migrate fragment_generation_tasks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentGenerationTaskDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_panel_generation_tasks",
		Description: "Create and migrate fragment_panel_generation_tasks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentPanelGenerationTaskDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_generation_step_audit_records",
		Description: "Create generation_step_audit_records for agent generation audit",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &GenerationStepAuditRecordDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_generation_assets",
		Description: "Create and migrate fragment_generation_assets table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentGenerationAssetDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_generation_image_slots",
		Description: "Create fragment_generation_image_slots for durable per-image generation state",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentGenerationImageSlotDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_conversation_messages",
		Description: "Create and migrate fragment_conversation_messages for creator AI chat history",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentConversationMessageDB{})
		},
		Required: true,
	})

	// ========== Fragment Interaction 相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_likes",
		Description: "Create and migrate fragment_likes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentLikeDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_comments",
		Description: "Create and migrate fragment_comments table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentCommentDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_shares",
		Description: "Create and migrate fragment_shares table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentShareDB{})
		},
		Required: true,
	})

	// ========== 多态关注/点赞表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_legacy_polymorphic_follows_then_drop",
		Description: "Copy legacy follows rows into story_follows, character_follows, user_follows; drop follows",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return MigrateLegacyPolymorphicFollowsThenDrop(ctx, db, log)
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_likes",
		Description: "Create and migrate likes table (polymorphic like)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &Like{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_comic_styles",
		Description: "Create fragment_comic_styles and seed defaults",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			_ = ctx
			if err := autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentComicStyle{}); err != nil {
				return err
			}
			return SeedFragmentComicStylesIfEmpty(db, log)
		},
		Required: true,
	})

	// 注册 Schema 修复步骤
	registerSchemaFixSteps(registry)

	// 注册索引创建步骤
	registerIndexSteps(registry)
}

// registerSchemaFixSteps 注册 Schema 修复步骤
func registerSchemaFixSteps(registry *migrations.MigrationRegistry) {
	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_storyboard_video_generation_schema",
		Description: "Ensure storyboard_video_generations has subdivision fields",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoryboardVideoGenerationSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_storyboard_scenes_schema",
		Description: "Ensure storyboard_scenes has subdivision fields",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoryboardScenesSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_stories_style_schema",
		Description: "Ensure stories.style can store JSON (TEXT) without index",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoriesStyleSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_ai_generation_records_schema",
		Description: "Ensure ai_generation_records (incl. original_prompt) uses utf8mb4 for Unicode",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureAIGenerationRecordsSchema()
		},
		// Required: old DBs may have passed core migrate when ALTER failures were ignored; re-run must fail loud if still broken.
		Required: true,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_user_devices_schema",
		Description: "Ensure user_devices table exists",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			// 注意：这里不能调用 ensureUserDevicesSchema，因为它是 AutoMigrate
			// 用户已经在前面迁移了，这里可以跳过
			return nil
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_storyboard_image_generation_schema",
		Description: "Ensure storyboard_image_generations has prompt_details_json and pipeline_kind columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoryboardImageGenerationSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_storyboard_video_generation_prompt_details_schema",
		Description: "Ensure storyboard_video_generations has prompt_details_json column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoryboardVideoGenerationPromptDetailsSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_storyboard_continuation_generation_options_schema",
		Description: "Ensure storyboards has generate_video_after_images and continuation_comic_style",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoryboardContinuationGenerationOptionsSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_storyboard_continuation_summary_schema",
		Description: "Ensure storyboards has continuation_summary for fork/continuation context",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoryboardContinuationSummarySchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_storyboard_use_comic_page_pipeline_schema",
		Description: "Ensure storyboards has use_comic_page_pipeline for comic-page vs auto scene image",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoryboardUseComicPagePipelineSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_character_portrait_schema",
		Description: "Ensure characters has portrait-related columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureCharacterPortraitSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_characters_role_column",
		Description: "Ensure characters has role column (与 API role / 编辑器角色定位对齐)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			m := migrations.NewMigrationManager(db, log)
			return m.AddColumn("characters", "role", "VARCHAR(100) NULL COMMENT '故事内定位'")
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_is_collaboration_open_column",
		Description: "Ensure stories has is_collaboration_open column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.EnsureIsCollaborationOpenColumn(log)
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_user_fragments_count_column",
		Description: "Ensure users has fragments_count column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureUserFragmentsCountColumn()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_user_settings_preferred_genres_column",
		Description: "Ensure user_settings has preferred_genres_json column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureUserSettingsPreferredGenresColumn()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_genre_catalog_title_ja_column",
		Description: "Add title_ja to genre_catalog_entries for Japanese UI",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			_ = ctx
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &GenreCatalogEntry{})
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "backfill_genre_catalog_title_ja",
		Description: "Populate title_ja for genre catalog (seeds + fallback from title_zh)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			_ = ctx
			_ = log
			return BackfillGenreCatalogTitleJa(db)
		},
		Required: false,
	})

	// ========== Migration 007: Add follows, likes tables and update settings ==========
	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_stories_ai_enabled_column",
		Description: "Ensure stories has ai_enabled column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureStoriesAIEnabledColumn()
		},
		Required: false,
	})

	// REMOVED: ensure_characters_poster_creation_permission_column - not in StoryCreationAppUI design

	// ========== Migration 008: Add fragment and story source tracking columns ==========
	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_stories_source_fragment_id_column",
		Description: "Ensure stories has source_fragment_id column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateStories(ctx, db, log)
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_stories_source_fragment_id_unique",
		Description: "Unique index: one story per source fragment (MySQL allows multiple NULLs)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateStories(ctx, db, log)
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_stories_use_ai_columns",
		Description: "Ensure stories has use_ai and ai_assistance_options columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateStories(ctx, db, log)
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_fragments_converted_columns",
		Description: "Ensure fragments has converted_to_story_id and is_converted columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentDB{})
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_fragments_style_column",
		Description: "Ensure fragments has style column (comic/image slug for convert-to-story inheritance)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentDB{})
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_fragment_generation_trace_schema",
		Description: "Ensure fragments, generation tasks, slots, and assets can store generation trace metadata",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &FragmentDB{}, &FragmentGenerationTaskDB{}, &FragmentGenerationImageSlotDB{}, &FragmentGenerationAssetDB{})
		},
		Required: false,
	})

	// ========== Migration 009: Add user points and referral system (StoryCreationAppUI) ==========
	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_user_points_referral_columns",
		Description: "Ensure users has points and referral_code columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, log, &User{})
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_user_phone_oauth_columns",
		Description: "Ensure users has phone, phone_verified_at, pending_oauth_phone_sms",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureUserPhoneOAuthColumns()
		},
		Required: true,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "migrate_legacy_polymorphic_storyboard_likes",
		Description: "Copy likes(storyboard_node|storyboard) into storyboard_likes; drop those likes rows; reconcile storyboards.likes",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return MigrateLegacyPolymorphicStoryboardLikes(ctx, db, log)
		},
		Required: false,
	})
}

// registerIndexSteps 注册索引创建步骤
func registerIndexSteps(registry *migrations.MigrationRegistry) {
	registry.RegisterIndexStep(migrations.MigrationStep{
		Name:        "create_web_payments_indexes",
		Description: "Create web_payments composite indexes",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			// 注意：web_payments 表在 pay 包中，这里会由 pay 包处理
			// 如果需要在这里创建索引，确保表已存在
			return nil
		},
		Required: false,
	})

	registry.RegisterIndexStep(migrations.MigrationStep{
		Name:        "create_characters_portrait_status_index",
		Description: "Create index on characters.portrait_generation_status",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureCharacterPortraitSchema() // 这个方法包含了索引创建
		},
		Required: false,
	})
}
