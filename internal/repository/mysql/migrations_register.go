package mysql

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/repository/migrations"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// init 自动注册 mysql 包的迁移步骤
func init() {
	registry := migrations.GetRegistry()

	// ========== 核心实体表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_users",
		Description: "Create and migrate users table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&User{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_login_records",
		Description: "Create and migrate user_login_records table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserLoginRecord{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_settings",
		Description: "Create and migrate user_settings table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserSettings{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_devices",
		Description: "Create and migrate user_devices table for push notifications",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserDevice{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_statistics",
		Description: "Create and migrate user_statistics table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserStatistics{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_activities",
		Description: "Create and migrate user_activities table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserActivity{})
		},
		Required: true,
	})

	// ========== 故事相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_stories",
		Description: "Create and migrate stories table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Story{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_panels",
		Description: "Create and migrate panels table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Panel{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_contributors",
		Description: "Create and migrate story_contributors table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryContributor{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_compositions",
		Description: "Create and migrate story_compositions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryComposition{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_participants",
		Description: "Create and migrate story_participants table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryParticipant{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_tags",
		Description: "Create and migrate story_tags table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryTag{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_likes",
		Description: "Create and migrate story_likes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryLike{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_follows",
		Description: "Create and migrate story_follows table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryFollow{})
		},
		Required: true,
	})

	// ========== Storyboard 相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboards",
		Description: "Create and migrate storyboards table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Storyboard{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_content_generations",
		Description: "Create and migrate storyboard_content_generations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryboardContentGeneration{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_scene_generations",
		Description: "Create and migrate storyboard_scene_generations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryboardSceneGeneration{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_image_generations",
		Description: "Create and migrate storyboard_image_generations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryboardImageGeneration{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_video_generations",
		Description: "Create and migrate storyboard_video_generations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryboardVideoGeneration{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_likes",
		Description: "Create and migrate storyboard_likes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryboardLike{})
		},
		Required: true,
	})

	// ========== 角色相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_characters",
		Description: "Create and migrate characters table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Character{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_character_posters",
		Description: "Create and migrate character_posters table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&CharacterPoster{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_character_analytics",
		Description: "Create and migrate character_analytics table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&CharacterAnalytics{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_character_tags",
		Description: "Create and migrate character_tags table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&CharacterTag{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_character_follows",
		Description: "Create and migrate character_follows table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&CharacterFollow{})
		},
		Required: true,
	})

	// ========== 场景相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_scenes",
		Description: "Create and migrate story_scenes table (story-scoped locations)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryScene{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_scenes",
		Description: "Create and migrate storyboard_scenes table (AI-generated plot scenes)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryboardScene{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_character_links",
		Description: "Create and migrate storyboard_character_links table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryboardCharacterLink{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_storyboard_scene_links",
		Description: "Create and migrate storyboard_scene_links table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryboardSceneLink{})
		},
		Required: true,
	})

	// ========== 群组相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_groups",
		Description: "Create and migrate groups table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Group{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_group_members",
		Description: "Create and migrate group_members table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GroupMember{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_group_roles",
		Description: "Create and migrate group_roles table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GroupRole{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_group_invitations",
		Description: "Create and migrate group_invitations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GroupInvitation{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_group_follows",
		Description: "Create and migrate group_follows table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GroupFollow{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_group_blacklists",
		Description: "Create and migrate group_blacklists table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GroupBlacklist{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_group_activities",
		Description: "Create and migrate group_activities table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GroupActivity{})
		},
		Required: true,
	})

	// ========== 评论表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_comments",
		Description: "Create and migrate comments table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Comment{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_comment_likes",
		Description: "Create and migrate comment_likes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&CommentLike{})
		},
		Required: true,
	})

	// ========== Writers Room 表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_writers_rooms",
		Description: "Create and migrate writers_rooms table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&WritersRoomDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_writers_room_participants",
		Description: "Create and migrate writers_room_participants table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&WritersRoomParticipantDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_writers_room_messages",
		Description: "Create and migrate writers_room_messages table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&WritersRoomMessageDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_writers_room_message_reactions",
		Description: "Create and migrate writers_room_message_reactions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&WritersRoomMessageReactionDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_message_read_receipts",
		Description: "Create and migrate message_read_receipts table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&MessageReadReceiptDB{})
		},
		Required: true,
	})

	// ========== 标签系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_tags",
		Description: "Create and migrate tags table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Tag{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_style_configs",
		Description: "Create and migrate style_configs table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StyleConfig{})
		},
		Required: true,
	})

	// ========== 搜索和浏览表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_search_histories",
		Description: "Create and migrate search_histories table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&SearchHistory{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_view_histories",
		Description: "Create and migrate view_histories table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&ViewHistory{})
		},
		Required: true,
	})

	// ========== 举报系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_reports",
		Description: "Create and migrate reports table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Report{})
		},
		Required: true,
	})

	// ========== Agent 系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agents",
		Description: "Create and migrate agents table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Agent{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agent_skills",
		Description: "Create and migrate agent_skills table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AgentSkill{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agent_skill_usages",
		Description: "Create and migrate agent_skill_usages table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AgentSkillUsage{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agent_interactions",
		Description: "Create and migrate agent_interactions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AgentInteraction{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_agent_memories",
		Description: "Create and migrate agent_memories table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AgentMemory{})
		},
		Required: true,
	})

	// ========== AI 任务系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_ai_tasks",
		Description: "Create and migrate ai_tasks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AITask{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_render_tasks",
		Description: "Create and migrate render_tasks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&RenderTask{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_story_publications",
		Description: "Create and migrate story_publications table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&StoryPublication{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_ai_generation_records",
		Description: "Create and migrate ai_generation_records table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AIGenerationRecord{})
		},
		Required: true,
	})

	// ========== 通知系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_notifications",
		Description: "Create and migrate notifications table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Notification{})
		},
		Required: true,
	})

	// ========== 资产管理表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_assets",
		Description: "Create and migrate assets table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Asset{})
		},
		Required: true,
	})

	// ========== 支付订阅表（核心） ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_memberships",
		Description: "Create and migrate memberships table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Membership{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_subscription_plans",
		Description: "Create and migrate subscription_plans table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&SubscriptionPlan{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_subscription_orders",
		Description: "Create and migrate subscription_orders table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&SubscriptionOrder{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_token_transactions",
		Description: "Create and migrate token_transactions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&TokenTransaction{})
		},
		Required: true,
	})

	// ========== 邀请码系统表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_invitation_codes",
		Description: "Create and migrate invitation_codes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&InvitationCode{})
		},
		Required: true,
	})

	// ========== 第三方登录表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_third_party_logins",
		Description: "Create and migrate third_party_logins table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&ThirdPartyLogin{})
		},
		Required: true,
	})

	// ========== 关系表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_user_follows",
		Description: "Create and migrate user_follows table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserFollow{})
		},
		Required: true,
	})

	// ========== Fragment 相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragments",
		Description: "Create and migrate fragments table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&FragmentDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_generation_tasks",
		Description: "Create and migrate fragment_generation_tasks table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&FragmentGenerationTaskDB{})
		},
		Required: true,
	})

	// ========== Fragment Interaction 相关表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_likes",
		Description: "Create and migrate fragment_likes table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&FragmentLikeDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_comments",
		Description: "Create and migrate fragment_comments table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&FragmentCommentDB{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_fragment_shares",
		Description: "Create and migrate fragment_shares table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&FragmentShareDB{})
		},
		Required: true,
	})

	// ========== 多态关注/点赞表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_follows",
		Description: "Create and migrate follows table (polymorphic follow)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Follow{})
		},
		Required: true,
	})

	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_likes",
		Description: "Create and migrate likes table (polymorphic like)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Like{})
		},
		Required: true,
	})

	// ========== Group Showcase 表 ==========
	registry.RegisterCoreStep(migrations.MigrationStep{
		Name:        "migrate_group_showcases",
		Description: "Create and migrate group_showcases table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GroupShowcase{})
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
		Description: "Ensure ai_generation_records prompt fields support Unicode (utf8mb4)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureAIGenerationRecordsSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_group_members_schema",
		Description: "Ensure group_members has role_id column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureGroupMembersSchema()
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_group_follows_schema",
		Description: "Ensure group_follows table exists",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureGroupFollowsSchema()
		},
		Required: false,
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
		Description: "Ensure storyboard_image_generations has prompt_details_json column",
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
		Name:        "ensure_character_portrait_schema",
		Description: "Ensure characters has portrait-related columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureCharacterPortraitSchema()
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
		Name:        "ensure_user_group_count_columns",
		Description: "Ensure users has groups_count and groups_created columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.EnsureUserGroupCountColumns(log)
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_groups_blocked_count_column",
		Description: "Ensure groups has blocked_count column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureGroupsBlockedCountColumn()
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

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_characters_poster_creation_permission_column",
		Description: "Ensure characters has poster_creation_permission column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			repo := &Repository{db: db, log: log}
			return repo.ensureCharactersPosterCreationPermissionColumn()
		},
		Required: false,
	})

	// ========== Migration 008: Add fragment and story source tracking columns ==========
	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_stories_source_fragment_id_column",
		Description: "Ensure stories has source_fragment_id column",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Story{})
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_stories_use_ai_columns",
		Description: "Ensure stories has use_ai and ai_assistance_options columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Story{})
		},
		Required: false,
	})

	registry.RegisterSchemaFixStep(migrations.MigrationStep{
		Name:        "ensure_fragments_converted_columns",
		Description: "Ensure fragments has converted_to_story_id and is_converted columns",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&FragmentDB{})
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
