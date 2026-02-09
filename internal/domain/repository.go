package domain

import (
	"context"
	"time"
)

// StoryFilter holds story query filters
type StoryFilter struct {
	Status   string
	AuthorID string
	GroupID  string
	Search   string
	Genre    string
	Limit    int
	Offset   int
}

// Repository defines the data access interface
type Repository interface {
	// ========== User operations ==========
	UserByID(ctx context.Context, id string) (*User, error)
	UserByUsername(ctx context.Context, username string) (*User, error)
	UserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, limit, offset int) ([]*User, error)

	// ========== User Activity operations ==========
	CreateUserActivity(ctx context.Context, activity *UserActivity) error
	UserActivitiesByUserID(ctx context.Context, userID string, limit, offset int) ([]*UserActivity, error)
	UserActivitiesByTimeRange(ctx context.Context, userID string, startTime, endTime int64, limit, offset int) ([]*UserActivity, error)
	UserActivitiesByDate(ctx context.Context, userID string, date string, limit, offset int) ([]*UserActivity, error)
	UserActivityHeatmap(ctx context.Context, userID string, startTime, endTime int64) ([]*ActivityHeatmapData, error)
	DeleteUserActivity(ctx context.Context, id string) error

	// ========== User Settings ==========
	UserSettings(ctx context.Context, userID string) (*UserSettings, error)
	CreateUserSettings(ctx context.Context, settings *UserSettings) error
	UpdateUserSettings(ctx context.Context, settings *UserSettings) error

	// ========== Membership ==========
	Membership(ctx context.Context, userID string) (*Membership, error)
	CreateMembership(ctx context.Context, membership *Membership) error
	UpdateMembership(ctx context.Context, membership *Membership) error

	// ========== Story operations ==========
	StoryByID(ctx context.Context, id string) (*Story, error)
	ListStories(ctx context.Context, filter StoryFilter) ([]*Story, int64, error)
	CreateStory(ctx context.Context, story *Story) error
	UpdateStory(ctx context.Context, story *Story) error
	DeleteStory(ctx context.Context, id string) error
	StoriesByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*Story, error)
	TrendingStories(ctx context.Context, limit int) ([]*Story, error)

	// ========== Dashboard feeds ==========
	// DashboardStoryboards returns storyboards from stories the user created OR follows.
	DashboardStoryboards(ctx context.Context, userID string, limit, offset int) ([]*Storyboard, int64, error)
	// DashboardGroupStoryboards returns storyboards from stories that belong to groups the user joined.
	DashboardGroupStoryboards(ctx context.Context, userID string, limit, offset int) ([]*Storyboard, int64, error)
	// DashboardCharacterStoryboards returns storyboards that followed characters participate in.
	DashboardCharacterStoryboards(ctx context.Context, userID string, limit, offset int) ([]*Storyboard, int64, error)
	// TrendingStoryboards returns published storyboards from trending stories:
	// - Stories the user contributed to
	// - Stories with high likes
	// - Stories with high storyboard count
	// - Stories with high followers
	TrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*Storyboard, int64, error)
	// GetPublicTrendingStoryboards returns published trending storyboards accessible to all users.
	// If userID is empty (guest), returns globally trending storyboards.
	// If userID is provided (authenticated), returns personalized trending storyboards.
	GetPublicTrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*Storyboard, int64, error)

	// ========== Story Contributor operations ==========
	AddStoryContributor(ctx context.Context, contributor *StoryContributor) error
	RemoveStoryContributor(ctx context.Context, storyID, userID string) error
	GetStoryContributors(ctx context.Context, storyID string, limit, offset int) ([]*StoryContributor, error)
	GetStoryContributor(ctx context.Context, storyID, userID string) (*StoryContributor, error)
	IsStoryContributor(ctx context.Context, storyID, userID string) (bool, error)
	UpdateStoryContributorRole(ctx context.Context, storyID, userID string, role StoryContributorRole) error

	// ========== Panel operations ==========
	PanelByID(ctx context.Context, id string) (*Panel, error)
	PanelsByStory(ctx context.Context, storyID string) ([]*Panel, error)
	CreatePanel(ctx context.Context, panel *Panel) error
	UpdatePanel(ctx context.Context, panel *Panel) error
	DeletePanel(ctx context.Context, id string) error

	// ========== Character operations ==========
	CharacterByID(ctx context.Context, id string) (*Character, error)
	ListCharacters(ctx context.Context, limit, offset int) ([]*Character, error)
	CharactersByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*Character, error)
	CharactersByStory(ctx context.Context, storyID string) ([]*Character, error)
	CreateCharacter(ctx context.Context, character *Character) error
	UpdateCharacter(ctx context.Context, character *Character) error
	DeleteCharacter(ctx context.Context, id string) error
	PopularCharacters(ctx context.Context, limit int) ([]*Character, error)
	StoryboardsByCharacter(ctx context.Context, characterID string, limit, offset int) ([]*Storyboard, int64, error)

	// ========== Story asset operations ==========
	StorySceneByID(ctx context.Context, storyID, sceneID string) (*StoryScene, error)
	StoryScenes(ctx context.Context, storyID string, limit, offset int) ([]*StoryScene, error)
	CreateStoryScene(ctx context.Context, scene *StoryScene) error
	UpdateStoryScene(ctx context.Context, scene *StoryScene) error
	DeleteStoryScene(ctx context.Context, storyID, sceneID string) error

	// ========== Character Analytics operations ==========
	GetOrCreateCharacterAnalytics(ctx context.Context, characterID string) (*CharacterAnalytics, error)
	CharacterAnalyticsByCharacterID(ctx context.Context, characterID string) (*CharacterAnalytics, error)
	UpdateCharacterAnalytics(ctx context.Context, analytics *CharacterAnalytics) error
	IncrementCharacterMessages(ctx context.Context, characterID string, count int) error
	IncrementCharacterTokens(ctx context.Context, characterID string, tokens int64) error
	IncrementCharacterChatters(ctx context.Context, characterID string) error

	// ========== Character Poster operations ==========
	CreateCharacterPoster(ctx context.Context, poster *CharacterPoster) error
	CharacterPosterByID(ctx context.Context, id string) (*CharacterPoster, error)
	CharacterPostersByCharacterID(ctx context.Context, characterID string, limit, offset int) ([]*CharacterPoster, error)
	UpdateCharacterPoster(ctx context.Context, poster *CharacterPoster) error
	DeleteCharacterPoster(ctx context.Context, id string) error
	IncrementPosterLikes(ctx context.Context, posterID string) error
	IncrementPosterShares(ctx context.Context, posterID string) error

	// ========== Group basic operations ==========
	ListGroups(ctx context.Context, limit, offset int) ([]*Group, error)
	ListMyGroups(ctx context.Context, userID string, limit, offset int) ([]*Group, error)
	ListPublicGroups(ctx context.Context, userID string, limit, offset int) ([]*Group, error)
	GroupsByUser(ctx context.Context, userID string) ([]*Group, error)
	GroupActivities(ctx context.Context, groupID string, limit int) ([]*GroupActivity, error)
	GroupActivitiesByTimeRange(ctx context.Context, groupID string, startTime, endTime int64, limit, offset int) ([]*GroupActivity, error)
	GroupActivitiesByDate(ctx context.Context, groupID string, date string, limit, offset int) ([]*GroupActivity, error)
	GroupActivityHeatmap(ctx context.Context, groupID string, startTime, endTime int64) ([]*ActivityHeatmapData, error)
	CreateGroupActivity(ctx context.Context, activity *GroupActivity) error

	// ========== Comment operations (旧版本，已废弃) ==========
	// 保留用于兼容性，实际使用下面的新版本
	CommentsByStory(ctx context.Context, storyID string) ([]*Comment, error)
	CommentsByParent(ctx context.Context, parentID string) ([]*Comment, error)

	// ========== Storyboard operations (旧版本，已废弃) ==========
	// 保留用于兼容性，实际使用下面的新版本

	StoryCompositionByID(ctx context.Context, id string) (*StoryComposition, error)
	ListStoryCompositions(ctx context.Context, limit, offset int) ([]*StoryComposition, error)
	CreateStoryComposition(ctx context.Context, composition *StoryComposition) error
	UpdateStoryComposition(ctx context.Context, composition *StoryComposition) error

	// ========== Relationship operations ==========
	// Follow
	FollowUser(ctx context.Context, followerID, followeeID string) error
	UnfollowUser(ctx context.Context, followerID, followeeID string) error
	IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error)
	Followers(ctx context.Context, userID string, limit, offset int) ([]*User, error)
	Following(ctx context.Context, userID string, limit, offset int) ([]*User, error)

	// Like
	LikeStory(ctx context.Context, userID, storyID string) error
	UnlikeStory(ctx context.Context, userID, storyID string) error
	IsStoryLiked(ctx context.Context, userID, storyID string) (bool, error)

	LikeStoryboard(ctx context.Context, userID, storyboardID string) error
	UnlikeStoryboard(ctx context.Context, userID, storyboardID string) error

	// Follow content
	FollowStory(ctx context.Context, userID, storyID string) error
	UnfollowStory(ctx context.Context, userID, storyID string) error

	FollowCharacter(ctx context.Context, userID, characterID string) error
	UnfollowCharacter(ctx context.Context, userID, characterID string) error
	IsCharacterFollowing(ctx context.Context, userID, characterID string) (bool, error)

	// Liked content
	LikedStories(ctx context.Context, userID string, limit, offset int) ([]*Story, error)
	LikedCharacters(ctx context.Context, userID string, limit, offset int) ([]*Character, error)
	LikedStoryboards(ctx context.Context, userID string, limit, offset int) ([]*Storyboard, error)

	// ========== Group operations ==========
	GroupByID(ctx context.Context, id string) (*Group, error)
	CreateGroup(ctx context.Context, group *Group) error
	UpdateGroup(ctx context.Context, group *Group) error
	DeleteGroup(ctx context.Context, id string) error

	// Group membership
	AddGroupMember(ctx context.Context, groupID, userID string, role GroupMemberRole, invitedBy string) error
	RemoveGroupMember(ctx context.Context, groupID, userID string) error
	GetGroupMembers(ctx context.Context, groupID string, limit, offset int) ([]*GroupMemberInfo, error)
	GetMemberRole(ctx context.Context, groupID, userID string) (GroupMemberRole, error)
	UpdateMemberRole(ctx context.Context, groupID, userID string, role GroupMemberRole) error
	IsGroupMember(ctx context.Context, groupID, userID string) (bool, error)

	// Group roles
	CreateGroupRole(ctx context.Context, role *GroupRole) error
	GetGroupRoleByCode(ctx context.Context, code string) (*GroupRole, error)
	GetGroupRoleByID(ctx context.Context, id string) (*GroupRole, error)
	ListGroupRoles(ctx context.Context) ([]*GroupRole, error)
	InitializeGroupRoles(ctx context.Context) error
	UpdateMemberRoleID(ctx context.Context, groupID, userID, roleID string) error
	GetMemberRoleID(ctx context.Context, groupID, userID string) (string, error)

	// Group invitations
	CreateGroupInvitation(ctx context.Context, groupID, inviterID, inviteeID, message string) (*GroupInvitation, error)
	GetInvitationByID(ctx context.Context, id string) (*GroupInvitation, error)
	GetPendingInvitation(ctx context.Context, groupID, inviteeID string) (*GroupInvitation, error)
	GetPendingInvitationsForUser(ctx context.Context, userID string, limit, offset int) ([]*GroupInvitation, error)
	UpdateInvitationStatus(ctx context.Context, id, status string) error
	ExpirePendingInvitations(ctx context.Context) (int64, error)

	// ========== Group Showcase operations ==========
	// 添加展示内容到小组
	AddGroupShowcase(ctx context.Context, showcase *GroupShowcase) error
	// 移除小组展示内容
	RemoveGroupShowcase(ctx context.Context, showcaseID string) error
	// 获取小组展示列表
	GetGroupShowcases(ctx context.Context, groupID string, contentType GroupShowcaseRelationType, limit, offset int) ([]*GroupShowcase, int64, error)
	// 获取小组展示详情
	GetGroupShowcaseByID(ctx context.Context, showcaseID string) (*GroupShowcase, error)
	// 更新小组展示排序
	UpdateGroupShowcaseOrder(ctx context.Context, showcaseID string, sortOrder int) error

	// Group follow
	FollowGroup(ctx context.Context, userID, groupID string) error
	UnfollowGroup(ctx context.Context, userID, groupID string) error
	IsFollowingGroup(ctx context.Context, userID, groupID string) (bool, error)
	ListFollowedGroups(ctx context.Context, userID string, limit, offset int) ([]*Group, error)

	// Group story count
	IncrementGroupStoryCount(ctx context.Context, groupID string) error
	DecrementGroupStoryCount(ctx context.Context, groupID string) error

	// ========== User Statistics operations ==========
	CountAllUsers(ctx context.Context) (int, error)
	CountNewUsersByDate(ctx context.Context, date time.Time) (int, error)
	GetUserStatisticsByDate(ctx context.Context, date time.Time) (*UserStatistics, error)
	SaveUserStatistics(ctx context.Context, stats *UserStatistics) error

	// ========== User Login Record operations ==========
	CreateUserLoginRecord(ctx context.Context, record *UserLoginRecord) error
	GetUserLoginRecords(ctx context.Context, userID string, limit, offset int) ([]*UserLoginRecord, error)
	GetLatestUserLoginRecord(ctx context.Context, userID string) (*UserLoginRecord, error)

	// ========== Storyboard operations ==========
	StoryboardByID(ctx context.Context, id string) (*Storyboard, error)
	CreateStoryboard(ctx context.Context, storyboard *Storyboard) error
	UpdateStoryboard(ctx context.Context, storyboard *Storyboard) error
	DeleteStoryboard(ctx context.Context, id string) error
	StoryboardsByStory(ctx context.Context, storyID string, limit, offset int) ([]*Storyboard, error)
	RootStoryboardsByStory(ctx context.Context, storyID string, limit, offset int) ([]*Storyboard, error)        // 获取故事的根故事板（ParentID 为空或 "__root__"）
	StoryboardsByParent(ctx context.Context, storyID, parentID string, limit, offset int) ([]*Storyboard, error) // 按 ParentID 过滤
	StoryboardsByCreator(ctx context.Context, creatorID string, limit, offset int) ([]*Storyboard, error)
	DraftStoryboardsByCreator(ctx context.Context, creatorID string, limit, offset int) ([]*Storyboard, error)
	CountStoryboardsByCreator(ctx context.Context, creatorID string) (int64, error)
	CountStoryboardsByStory(ctx context.Context, storyID string) (int64, error)
	// CharacterStoryboardCountsByStory returns participation counts keyed by characterID, counting distinct storyboard IDs within the given story.
	CharacterStoryboardCountsByStory(ctx context.Context, storyID string) (map[string]int64, error)
	StoryboardChildren(ctx context.Context, parentID string) ([]*Storyboard, error)
	StoryboardTree(ctx context.Context, rootID string) ([]*Storyboard, error)
	StoryboardFeed(ctx context.Context, limit, offset int) ([]*Storyboard, int64, error) // Community feed of published storyboards
	ForkStoryboard(ctx context.Context, parentID, creatorID string, storyboard *Storyboard) error
	IncrementStoryboardViews(ctx context.Context, id string) error
	IncrementStoryStoryboardCount(ctx context.Context, storyID string) error
	DecrementStoryStoryboardCount(ctx context.Context, storyID string) error
	UpdateStoryboardTokens(ctx context.Context, storyboardID string, tokens int) error
	UpdateStoryboardWorkflow(ctx context.Context, storyboardID string, status string, step int) error

	// ========== StoryboardScene operations (AI-generated plot scenes) ==========
	CreateStoryboardScenes(ctx context.Context, storyboardID string, scenes []*StoryboardScene) error
	StoryboardScenes(ctx context.Context, storyboardID string) ([]*StoryboardScene, error)
	DeleteStoryboardScenes(ctx context.Context, storyboardID string) error
	UpdateStoryboardScene(ctx context.Context, scene *StoryboardScene) error
	UpdateStoryboardSceneImage(ctx context.Context, sceneID, imageURL string) error
	UpdateStoryboardSceneVideo(ctx context.Context, sceneID, videoURL string) error
	UpdateStoryboardSceneVideoWithSubdivision(ctx context.Context, sceneID, videoURL string, isSubdivided bool, videoSegmentsJSON, middleFrameURLsJSON string) error

	// ========== Storyboard Generation operations ==========
	// Content generation (Step 1)
	CreateContentGeneration(ctx context.Context, gen *StoryboardContentGeneration) error
	GetContentGeneration(ctx context.Context, id string) (*StoryboardContentGeneration, error)
	GetContentGenerationByStoryboard(ctx context.Context, storyboardID string) (*StoryboardContentGeneration, error)
	UpdateContentGeneration(ctx context.Context, gen *StoryboardContentGeneration) error

	// Scene generation (Step 2)
	CreateSceneGeneration(ctx context.Context, gen *StoryboardSceneGeneration) error
	GetSceneGeneration(ctx context.Context, id string) (*StoryboardSceneGeneration, error)
	ListSceneGenerations(ctx context.Context, storyboardID string) ([]*StoryboardSceneGeneration, error)
	UpdateSceneGeneration(ctx context.Context, gen *StoryboardSceneGeneration) error

	// Image generation (Step 3)
	CreateImageGeneration(ctx context.Context, gen *StoryboardImageGeneration) error
	GetImageGeneration(ctx context.Context, id string) (*StoryboardImageGeneration, error)
	ListImageGenerations(ctx context.Context, storyboardID string) ([]*StoryboardImageGeneration, error)
	UpdateImageGeneration(ctx context.Context, gen *StoryboardImageGeneration) error

	// Video generation (Step 4)
	CreateVideoGeneration(ctx context.Context, gen *StoryboardVideoGeneration) error
	GetVideoGeneration(ctx context.Context, id string) (*StoryboardVideoGeneration, error)
	ListVideoGenerations(ctx context.Context, storyboardID string) ([]*StoryboardVideoGeneration, error)
	UpdateVideoGeneration(ctx context.Context, gen *StoryboardVideoGeneration) error
	// ListPendingVideoGenerations returns all video generations that are processing and have a provider task ID (for recovery)
	ListPendingVideoGenerations(ctx context.Context) ([]*StoryboardVideoGeneration, error)

	// Aggregate token consumption
	GetStoryboardTotalTokens(ctx context.Context, storyboardID string) (int, error)

	// ========== Comment operations ==========
	CommentByID(ctx context.Context, id string) (*Comment, error)
	CreateComment(ctx context.Context, comment *Comment) error
	UpdateComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, id string) error
	CommentsByTarget(ctx context.Context, targetType, targetID string, limit, offset int) ([]*Comment, int64, error)
	CommentReplies(ctx context.Context, parentID string, limit, offset int) ([]*Comment, error)
	CommentTree(ctx context.Context, rootID string) ([]*Comment, error)
	LikeComment(ctx context.Context, userID, commentID string, isLike bool) error
	UnlikeComment(ctx context.Context, userID, commentID string) error
	IsCommentLiked(ctx context.Context, userID, commentID string) (bool, bool, error) // (exists, isLike, error)

	// ========== Notification operations ==========
	NotificationsByUser(ctx context.Context, userID string, limit, offset int) ([]*Notification, error)
	UnreadNotificationCount(ctx context.Context, userID string) (int, error)
	CreateNotification(ctx context.Context, notification *Notification) error
	MarkNotificationRead(ctx context.Context, id string) error
	MarkAllNotificationsRead(ctx context.Context, userID string) error
	DeleteNotification(ctx context.Context, id string) error

	// ========== Tag operations ==========
	CreateTag(ctx context.Context, tag *Tag) error
	GetTagByID(ctx context.Context, tagID string) (*Tag, error)
	GetTagByName(ctx context.Context, name string) (*Tag, error)
	GetOrCreateTag(ctx context.Context, name string) (*Tag, error)
	UpdateTag(ctx context.Context, tag *Tag) error
	DeleteTag(ctx context.Context, tagID string) error
	ListTags(ctx context.Context, category string, limit, offset int) ([]*Tag, int64, error)
	AddStoryTag(ctx context.Context, storyID, tagID string) error
	RemoveStoryTag(ctx context.Context, storyID, tagID string) error
	StoryTags(ctx context.Context, storyID string) ([]*Tag, error)
	StoriesByTag(ctx context.Context, tagID string, limit, offset int) ([]*Story, error)
	AddCharacterTag(ctx context.Context, characterID, tagID string) error
	RemoveCharacterTag(ctx context.Context, characterID, tagID string) error
	CharacterTags(ctx context.Context, characterID string) ([]*Tag, error)
	PopularTags(ctx context.Context, limit int) ([]*Tag, error)
	SearchTags(ctx context.Context, query string, limit int) ([]*Tag, error)

	// ========== AI Generation operations ==========
	// AI生成记录管理 - 用于记录AI能力使用数据（任务管理、Token计费）
	CreateAIGenerationRecord(ctx context.Context, record *AIGenerationRecord) error
	GetAIGenerationRecord(ctx context.Context, recordID string) (*AIGenerationRecord, error)
	UpdateAIGenerationRecord(ctx context.Context, record *AIGenerationRecord) error
	DeleteAIGenerationRecord(ctx context.Context, recordID string) error
	ListAIGenerationRecords(ctx context.Context, userID string, limit, offset int) ([]*AIGenerationRecord, error)
	ListAIGenerationRecordsByTimeRange(ctx context.Context, userID string, startTime, endTime int64) ([]*AIGenerationRecord, error)
	ListAIGenerationRecordsByEntity(ctx context.Context, entityType, entityID string, limit, offset int) ([]*AIGenerationRecord, error)
	AIGenerationRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*AIGenerationRecord, error)
	GetUserTokenStats(ctx context.Context, userID string, startTime, endTime int64) (map[string]interface{}, error)
	GetPendingAIGenerationRecords(ctx context.Context, statuses []AITaskStatus, limit int) ([]*AIGenerationRecord, error)

	// ========== Asset operations ==========
	AssetByID(ctx context.Context, id string) (*Asset, error)
	AssetsByUser(ctx context.Context, userID string, assetType string, limit, offset int) ([]*Asset, error)
	CreateAsset(ctx context.Context, asset *Asset) error
	UpdateAsset(ctx context.Context, asset *Asset) error
	DeleteAsset(ctx context.Context, id string) error

	// ========== Search operations ==========
	SearchStories(ctx context.Context, query string, limit, offset int) ([]*Story, error)
	SearchCharacters(ctx context.Context, query string, limit, offset int) ([]*Character, error)
	SearchUsers(ctx context.Context, query string, limit, offset int) ([]*User, error)
	SearchGroups(ctx context.Context, query string, limit, offset int) ([]*Group, error)
	CreateSearchHistory(ctx context.Context, history *SearchHistory) error

	// Advanced search
	AdvancedSearch(ctx context.Context, filter *SearchFilter) ([]*SearchResult, int64, error)
	SearchByCategory(ctx context.Context, category, searchType string, limit, offset int) ([]*SearchResult, error)
	SearchByTags(ctx context.Context, tags []string, searchType string, limit, offset int) ([]*SearchResult, error)
	GetSearchSuggestions(ctx context.Context, query string, limit int) ([]string, error)
	GetTrendingSearches(ctx context.Context, limit int) ([]string, error)
	GetUserSearchHistory(ctx context.Context, userID string, limit int) ([]*SearchHistory, error)

	// ========== View History ==========
	CreateViewHistory(ctx context.Context, history *ViewHistory) error
	ViewHistoryByUser(ctx context.Context, userID string, limit, offset int) ([]*ViewHistory, error)

	// ========== AI Task operations ==========
	CreateAITask(ctx context.Context, task *AITask) error
	GetAITask(ctx context.Context, taskID string) (*AITask, error)
	UpdateAITask(ctx context.Context, task *AITask) error
	UpdateAITaskStatus(ctx context.Context, taskID string, status AITaskStatus, progress int) error
	UpdateAITaskProgress(ctx context.Context, taskID string, progress int) error
	ListAITasks(ctx context.Context, userID string, limit, offset int) ([]*AITask, error)
	ListPendingAITasks(ctx context.Context, limit int) ([]*AITask, error)
	DeleteAITask(ctx context.Context, taskID string) error

	// ========== Render Task operations ==========
	CreateRenderTask(ctx context.Context, task *RenderTask) error
	GetRenderTask(ctx context.Context, taskID string) (*RenderTask, error)
	GetRenderTaskByStoryID(ctx context.Context, storyID string) (*RenderTask, error)
	UpdateRenderTask(ctx context.Context, task *RenderTask) error
	UpdateRenderTaskStatus(ctx context.Context, taskID string, status RenderTaskStatus, progress int) error
	UpdateRenderTaskProgress(ctx context.Context, taskID string, progress int) error
	ListRenderTasks(ctx context.Context, userID string, limit, offset int) ([]*RenderTask, error)
	ListPendingRenderTasks(ctx context.Context, limit int) ([]*RenderTask, error)
	DeleteRenderTask(ctx context.Context, taskID string) error

	// ========== Story Publication operations ==========
	CreateStoryPublication(ctx context.Context, publication *StoryPublication) error
	GetStoryPublication(ctx context.Context, publicationID string) (*StoryPublication, error)
	GetLatestStoryPublication(ctx context.Context, storyID string) (*StoryPublication, error)
	UpdateStoryPublication(ctx context.Context, publication *StoryPublication) error
	ListStoryPublications(ctx context.Context, storyID string) ([]*StoryPublication, error)

	// ========== Token & Subscription operations ==========
	// Token transactions
	CreateTokenTransaction(ctx context.Context, transaction *TokenTransaction) error
	GetTokenTransaction(ctx context.Context, transactionID string) (*TokenTransaction, error)
	ListTokenTransactions(ctx context.Context, userID string, limit, offset int) ([]*TokenTransaction, int64, error)
	GetTokenBalance(ctx context.Context, userID string) (int, error)
	UpdateTokenBalance(ctx context.Context, userID string, amount int, source, description string) (*TokenTransaction, error)

	// Subscription plans
	CreateSubscriptionPlan(ctx context.Context, plan *SubscriptionPlan) error
	GetSubscriptionPlan(ctx context.Context, planID string) (*SubscriptionPlan, error)
	UpdateSubscriptionPlan(ctx context.Context, plan *SubscriptionPlan) error
	DeleteSubscriptionPlan(ctx context.Context, planID string) error
	ListSubscriptionPlans(ctx context.Context, activeOnly bool) ([]*SubscriptionPlan, error)

	// Subscription orders
	CreateSubscriptionOrder(ctx context.Context, order *SubscriptionOrder) error
	GetSubscriptionOrder(ctx context.Context, orderID string) (*SubscriptionOrder, error)
	UpdateSubscriptionOrder(ctx context.Context, order *SubscriptionOrder) error
	ListSubscriptionOrders(ctx context.Context, userID string, limit, offset int) ([]*SubscriptionOrder, int64, error)
	GetActiveSubscription(ctx context.Context, userID string) (*SubscriptionOrder, error)

	// User settings (additional methods)
	GetUserNotificationSettings(ctx context.Context, userID string) (map[string]bool, error)
	UpdateUserNotificationSettings(ctx context.Context, userID string, settings map[string]bool) error
	GetUserPrivacySettings(ctx context.Context, userID string) (map[string]interface{}, error)
	UpdateUserPrivacySettings(ctx context.Context, userID string, settings map[string]interface{}) error

	// ========== StyleConfig operations ==========
	CreateStyleConfig(ctx context.Context, styleConfig *StyleConfig) error
	GetStyleConfigByID(ctx context.Context, id string) (*StyleConfig, error)
	GetStyleConfigByStyle(ctx context.Context, styleName string) (*StyleConfig, error)
	ListStyleConfigs(ctx context.Context, groupID string, limit, offset int) ([]*StyleConfig, int64, error)
	SearchStyleConfigs(ctx context.Context, keyword, groupID string, limit, offset int) ([]*StyleConfig, int64, error)
	UpdateStyleConfig(ctx context.Context, styleConfig *StyleConfig) error
	DeleteStyleConfig(ctx context.Context, id string) error
	BatchCreateStyleConfigs(ctx context.Context, styleConfigs []*StyleConfig) error

	// ========== Agent operations ==========
	CreateAgent(ctx context.Context, agent *Agent) error
	GetAgentByID(ctx context.Context, id string) (*Agent, error)
	GetAgentByCharacterID(ctx context.Context, characterID string) (*Agent, error)
	UpdateAgent(ctx context.Context, agent *Agent) error
	DeleteAgent(ctx context.Context, id string) error
	ListAgents(ctx context.Context, limit, offset int) ([]*Agent, error)
	IncrementAgentInteractionCount(ctx context.Context, agentID string) error

	// ========== Agent Skill operations ==========
	CreateAgentSkill(ctx context.Context, skill *AgentSkill) error
	GetAgentSkillByID(ctx context.Context, id string) (*AgentSkill, error)
	ListAgentSkills(ctx context.Context, agentID string, filter *SkillFilter) ([]*AgentSkill, error)
	UpdateAgentSkill(ctx context.Context, skill *AgentSkill) error
	DeleteAgentSkill(ctx context.Context, id string) error
	IncrementSkillUsage(ctx context.Context, skillID string, success bool, executionTime int) error

	// ========== Agent Interaction operations ==========
	CreateAgentInteraction(ctx context.Context, interaction *AgentInteraction) error
	GetAgentInteraction(ctx context.Context, id string) (*AgentInteraction, error)
	ListAgentInteractions(ctx context.Context, filter *InteractionFilter) ([]*AgentInteraction, error)
	GetInteractionStats(ctx context.Context, agentID string) (map[string]interface{}, error)

	// ========== Agent Memory operations ==========
	CreateAgentMemory(ctx context.Context, memory *AgentMemory) error
	GetAgentMemory(ctx context.Context, id string) (*AgentMemory, error)
	ListAgentMemories(ctx context.Context, filter *MemoryFilter) ([]*AgentMemory, error)
	UpdateAgentMemory(ctx context.Context, memory *AgentMemory) error
	DeleteAgentMemory(ctx context.Context, id string) error
	IncrementMemoryAccess(ctx context.Context, memoryID string) error

	// ========== Invitation Code operations ==========
	CreateInvitationCode(ctx context.Context, code *InvitationCode) error
	GetInvitationCodeByCode(ctx context.Context, code string) (*InvitationCode, error)
	GetInvitationCodeByID(ctx context.Context, id string) (*InvitationCode, error)
	ListInvitationCodes(ctx context.Context, createdBy string, limit, offset int) ([]*InvitationCode, error)
	UpdateInvitationCode(ctx context.Context, code *InvitationCode) error
	DeleteInvitationCode(ctx context.Context, id string) error
	UseInvitationCode(ctx context.Context, code string, userID string) error
	ValidateInvitationCode(ctx context.Context, code string) error

	// ========== Writers Room operations ==========
	WritersRoomByStoryID(ctx context.Context, storyID string) (*WritersRoom, error)
	CreateWritersRoom(ctx context.Context, room *WritersRoom) error
	UpdateWritersRoom(ctx context.Context, room *WritersRoom) error
	DeleteWritersRoom(ctx context.Context, roomID string) error

	WritersRoomParticipants(ctx context.Context, roomID string) ([]*WritersRoomParticipant, error)
	AddWritersRoomParticipant(ctx context.Context, participant *WritersRoomParticipant) error
	RemoveWritersRoomParticipant(ctx context.Context, roomID, userID string) error
	IsWritersRoomParticipant(ctx context.Context, roomID, userID string) (bool, error)
	UpdateParticipantLastRead(ctx context.Context, roomID, userID string) error
	IncrementParticipantCount(ctx context.Context, roomID string) error
	DecrementParticipantCount(ctx context.Context, roomID string) error

	WritersRoomMessages(ctx context.Context, roomID string, limit, offset int) ([]*WritersRoomMessage, error)
	CreateWritersRoomMessage(ctx context.Context, msg *WritersRoomMessage) error
	DeleteWritersRoomMessage(ctx context.Context, messageID string) error
	WritersRoomMessageByID(ctx context.Context, messageID string) (*WritersRoomMessage, error)
	IncrementMessageCount(ctx context.Context, roomID string) error
	UpdateRoomLastMessage(ctx context.Context, roomID string, lastMessage string, lastMessageTime int64) error

	WritersRoomMessageReactionByID(ctx context.Context, reactionID string) (*WritersRoomMessageReaction, error)
	CreateWritersRoomMessageReaction(ctx context.Context, reaction *WritersRoomMessageReaction) error
	DeleteWritersRoomMessageReaction(ctx context.Context, messageID, userID string) error
	WritersRoomMessageReactions(ctx context.Context, messageID string) ([]*WritersRoomMessageReaction, error)

	MessageReadReceipts(ctx context.Context, messageID string) ([]*MessageReadReceipt, error)
	CreateMessageReadReceipt(ctx context.Context, receipt *MessageReadReceipt) error
	UpdateMessageReadReceipt(ctx context.Context, messageID, userID string) error
	MarkMessageAsRead(ctx context.Context, messageID, userID string) error

	WritersRoomUnreadCount(ctx context.Context, roomID, userID string) (int, error)

	// ========== Third Party Login operations ==========
	// 第三方登录账户关联（支持 Google/Apple 跨设备登录）
	CreateThirdPartyLogin(ctx context.Context, login *ThirdPartyLogin) error
	GetThirdPartyLogin(ctx context.Context, id string) (*ThirdPartyLogin, error)
	GetThirdPartyLoginByProviderUserID(ctx context.Context, provider ThirdPartyProvider, providerUserID string) (*ThirdPartyLogin, error)
	GetThirdPartyLoginByEmail(ctx context.Context, provider ThirdPartyProvider, email string) (*ThirdPartyLogin, error)
	GetThirdPartyLoginsByUserID(ctx context.Context, userID string) ([]*ThirdPartyLogin, error)
	UpdateThirdPartyLogin(ctx context.Context, login *ThirdPartyLogin) error
	DeleteThirdPartyLogin(ctx context.Context, id string) error
	// 通过任意第三方登录（Google 或 Apple）的 email 查找关联的用户
	GetUserByThirdPartyEmail(ctx context.Context, email string) (*User, error)

	// ========== Fragment operations ==========
	// FragmentByID retrieves a fragment by ID
	FragmentByID(ctx context.Context, id string) (*Fragment, error)
	// ListFragments retrieves fragments with pagination
	ListFragments(ctx context.Context, limit, offset int, visibility string) ([]*Fragment, int64, error)
	// CreateFragment creates a new fragment
	CreateFragment(ctx context.Context, fragment *Fragment) error
	// UpdateFragment updates a fragment
	UpdateFragment(ctx context.Context, fragment *Fragment) error
	// DeleteFragment deletes a fragment
	DeleteFragment(ctx context.Context, id string) error

	// ========== User Device operations ==========
	// 用户设备管理（APNs/FCM 推送通知）
	CreateUserDevice(ctx context.Context, device *UserDevice) error
	GetUserDevice(ctx context.Context, id string) (*UserDevice, error)
	GetUserDeviceByToken(ctx context.Context, deviceToken string) (*UserDevice, error)
	GetUserDevicesByUserID(ctx context.Context, userID string) ([]*UserDevice, error)
	GetActiveUserDevicesByUserID(ctx context.Context, userID string) ([]*UserDevice, error)
	GetUserDevicesByPlatform(ctx context.Context, userID string, platform DevicePlatform) ([]*UserDevice, error)
	UpdateUserDevice(ctx context.Context, device *UserDevice) error
	DeleteUserDevice(ctx context.Context, id string) error
	DeleteUserDeviceByToken(ctx context.Context, deviceToken string) error
	DeactivateUserDevice(ctx context.Context, deviceToken string) error
	UpdateUserDeviceLastActive(ctx context.Context, deviceToken string, lastActiveAt int64) error
}

// FollowRepository 关注相关操作（多态关联）
type FollowRepository interface {
	CreateFollow(ctx context.Context, follow *Follow) error
	DeleteFollow(ctx context.Context, id string) error
	GetFollowByID(ctx context.Context, id string) (*Follow, error)
	GetFollowsByFollower(ctx context.Context, userID string, followableType FollowableType) ([]*Follow, error)
	GetFollowsByFollowable(ctx context.Context, followableType FollowableType, followableID string) ([]*Follow, error)
	CheckFollowStatus(ctx context.Context, followerID string, followableType FollowableType, followableID string) (bool, error)
	GetFollowersCount(ctx context.Context, followableType FollowableType, followableID string) (int, error)
}

// LikeRepository 点赞相关操作（多态关联）
type LikeRepository interface {
	CreateLike(ctx context.Context, like *Like) error
	DeleteLike(ctx context.Context, id string) error
	GetLikeByID(ctx context.Context, id string) (*Like, error)
	GetLikesByUser(ctx context.Context, userID string, likeableType LikeableType) ([]*Like, error)
	GetLikesByLikeable(ctx context.Context, likeableType LikeableType, likeableID string) ([]*Like, error)
	CheckLikeStatus(ctx context.Context, userID string, likeableType LikeableType, likeableID string) (bool, error)
	GetLikesCount(ctx context.Context, likeableType LikeableType, likeableID string) (int, error)
}

// UserSettingsRepository 用户设置操作
type UserSettingsRepository interface {
	GetUserSettings(userID string) (*UserSettings, error)
	CreateUserSettings(settings *UserSettings) error
	UpdateUserSettings(settings *UserSettings) error
}


