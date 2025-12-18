package mysql

import (
	"context"
	"encoding/json"
	"fmt"
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
	models := []interface{}{
		// 核心实体
		&User{},
		&Story{},
		&Panel{},
		&StoryScene{},
		&StoryboardCharacterLink{},
		&StoryboardSceneLink{},
		&Character{},
		&Group{},
		&Comment{},
		&ChatThread{},
		&ChatMessage{},
		&StoryComposition{},
		&StoryParticipant{},
		&Storyboard{},
		&StoryboardScene{}, // AI-generated plot scenes within storyboards
		&GroupActivity{},
		&UserActivity{},
		&CharacterPoster{},
		&CharacterAnalytics{},
		// Storyboard AI 生成记录
		&StoryboardContentGeneration{},
		&StoryboardSceneGeneration{},
		&StoryboardImageGeneration{},
		&StoryboardVideoGeneration{},
		// 关系表
		&UserFollow{},
		&StoryLike{},
		&StoryFollow{},
		&StoryContributor{},
		&CharacterFollow{},
		&GroupMember{},
		&GroupRole{},
		&GroupInvitation{},
		&CommentLike{},
		&StoryboardLike{},
		// 系统表
		&Asset{},
		&Notification{},
		&UserSettings{},
		&Membership{},
		&AIGenerationRecord{},
		&Tag{},
		&StoryTag{},
		&CharacterTag{},
		&StyleConfig{},
		&InvitationCode{},
		&SearchHistory{},
		&ViewHistory{},
		&Report{},
		// 任务表
		&AITask{},
		&RenderTask{},
		&StoryPublication{},
		// Agent 系统表
		&Agent{},
		&AgentSkill{},
		&AgentSkillUsage{},
		&AgentInteraction{},
		&AgentMemory{},
		// 支付订阅系统表
		&SubscriptionPlan{},
		&SubscriptionOrder{},
		&TokenTransaction{},
		// Chat enhancements
		&ChatThreadStoryboardBranch{},
		&ChatMessageReaction{},
		&ChatMessageToken{},
		
		// User statistics
		&UserStatistics{},
		
		// User login records
		&UserLoginRecord{},
	}

	r.log.Info("migrating database tables", zap.Int("total_models", len(models)))

	// for _, model := range models {
	// 	modelName := fmt.Sprintf("%T", model)
	// 	r.log.Debug("migrating table", zap.String("model", modelName))
	// 	if err := r.db.AutoMigrate(model); err != nil {
	// 		r.log.Error("failed to migrate table",
	// 			zap.String("model", modelName),
	// 			zap.Error(err))
	// 		return fmt.Errorf("failed to migrate %s: %w", modelName, err)
	// 	}
	// }

	r.log.Info("all tables migrated successfully")
	return nil
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
		ID:          uuid.New().String(),
		Title:       story.Title,
		Description: story.Description,
		CoverImage:  story.CoverImage,
		AuthorID:    story.Author.ID,
		Genre:       story.Genre,
		Status:      story.Status,
		Likes:       0,
		Followers:   0,
		Panels:      0,
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
		ID:          s.ID,
		AuthorID:    s.AuthorID,
		GroupID:     groupID,
		Title:       s.Title,
		Description: s.Description,
		CoverImage:  s.CoverImage,
		Author:      &author,
		Likes:       s.Likes,
		Followers:   s.Followers,
		Panels:      s.Panels,
		Genre:       s.Genre,
		Style:       s.Style,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt.Unix(),
		UpdatedAt:   s.UpdatedAt.Unix(),
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
		ID:              c.ID,
		StoryID:         c.StoryID,
		AuthorID:        c.AuthorID,
		Name:            c.Name,
		Description:     c.Description,
		Avatar:          c.Avatar,
		Poster:          c.Poster,
		Author:          &author,
		Personality:     c.Personality,
		Background:      c.Background,
		ShortTermGoal:   c.ShortTermGoal,
		LongTermGoal:    c.LongTermGoal,
		HandlingStyle:   c.HandlingStyle,
		CognitionRange:  c.CognitionRange,
		AbilityFeatures: c.AbilityFeatures,
		Appearance:      c.Appearance,
		DressPreference: c.DressPreference,
		Traits:          traits,
		Skills:          skills,
		TraitsJSON:      c.Traits,
		SkillsJSON:      c.Skills,
		IsPublic:        c.IsPublic,
		SourceType:      c.SourceType,
		SourcePrompt:    c.SourcePrompt,
		SourceImage:     c.SourceImage,
		CreatedBy:       c.CreatedBy,
		LastEditedBy:    c.LastEditedBy,
		GroupID:         c.GroupID,
		Likes:           c.Likes,
		Followers:       c.Followers,
		Stories:         c.Stories,
		CreatedAt:       c.CreatedAt.Unix(),
		UpdatedAt:       c.UpdatedAt.Unix(),
	}
}

func (r *Repository) groupToDomain(g Group) domain.Group {
	creator := r.userToDomain(g.Creator)
	return domain.Group{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		Avatar:      g.Avatar,
		Members:     g.Members,
		Stories:     g.Stories,
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
