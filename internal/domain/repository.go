package domain

import (
	"context"
	"time"
)

// StoryFilter holds story query filters
type StoryFilter struct {
	Status string
	UserID string
	Search string
	Genre  string
	Limit  int
	Offset int
}

// Repository defines the data access interface
type Repository interface {
	// ========== Transaction support ==========
	// WithTransaction executes a function within a database transaction
	// If the function returns an error, the transaction is rolled back
	// If the function returns nil, the transaction is committed
	WithTransaction(ctx context.Context, fn func(tx Repository) error) error

	// ========== User operations ==========
	UserByID(ctx context.Context, id string) (*User, error)
	UserByUsername(ctx context.Context, username string) (*User, error)
	UserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, limit, offset int) ([]*User, error)

	// REMOVED: User Activity operations - not in StoryCreationAppUI design

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
	StoriesByUser(ctx context.Context, userID string, limit, offset int) ([]*Story, error)
	TrendingStories(ctx context.Context, limit int) ([]*Story, error)

	// ========== Public Trending ==========
	// REMOVED: DashboardStoryboards - not in StoryCreationAppUI design
	// REMOVED: DashboardCharacterStoryboards - not in StoryCreationAppUI design
	// REMOVED: TrendingStoryboards (authenticated) - not in StoryCreationAppUI design
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

	// ========== Storyboard Panel operations ==========
	PanelsByStoryboard(ctx context.Context, storyboardID string) ([]*StoryboardPanel, error)
	CreateStoryboardPanel(ctx context.Context, panel *StoryboardPanel) error

	// ========== Character operations ==========
	CharacterByID(ctx context.Context, id string) (*Character, error)
	ListCharacters(ctx context.Context, limit, offset int) ([]*Character, error)
	CharactersByUser(ctx context.Context, userID string, limit, offset int) ([]*Character, error)
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

	// REMOVED: Character Poster operations - not in StoryCreationAppUI design
	// REMOVED: Character View operations - not in StoryCreationAppUI design

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
	CountFollowersOfUser(ctx context.Context, followeeID string) (int64, error)
	CountFollowingOfUser(ctx context.Context, followerID string) (int64, error)
	ListUserFollowsByFollower(ctx context.Context, followerID string, limit, offset int) ([]*Follow, error)

	// Block/Unblock
	BlockUser(ctx context.Context, blockerID, blockedID string) error
	UnblockUser(ctx context.Context, blockerID, blockedID string) error
	IsBlocked(ctx context.Context, blockerID, blockedID string) (bool, error)

	// Report
	ReportUser(ctx context.Context, reporterID, reportedID string, reason string) error

	// Get Liked Content IDs
	GetLikedStoryIDs(ctx context.Context, userID string, limit, offset int) ([]string, error)
	GetLikedCharacterIDs(ctx context.Context, userID string, limit, offset int) ([]string, error)
	GetLikedStoryboardIDs(ctx context.Context, userID string, limit, offset int) ([]string, error)

	// Like
	LikeStory(ctx context.Context, userID, storyID string) error
	UnlikeStory(ctx context.Context, userID, storyID string) error
	IsStoryLiked(ctx context.Context, userID, storyID string) (bool, error)

	LikeStoryboard(ctx context.Context, userID, storyboardID string) error
	UnlikeStoryboard(ctx context.Context, userID, storyboardID string) error
	IsStoryboardLiked(ctx context.Context, userID, storyboardID string) (bool, error)
	// BatchIsStoryboardLiked returns liked=true for IDs present in storyboard_likes for this user.
	BatchIsStoryboardLiked(ctx context.Context, userID string, storyboardIDs []string) (map[string]bool, error)
	// ListStoryboardLikers returns users who liked a storyboard (storyboard_likes), newest first.
	ListStoryboardLikers(ctx context.Context, storyboardID string, limit, offset int) ([]*User, int, error)

	// Follow content
	FollowStory(ctx context.Context, userID, storyID string) error
	UnfollowStory(ctx context.Context, userID, storyID string) error
	IsStoryFollowing(ctx context.Context, userID, storyID string) (bool, error)
	CountFollowersOfStory(ctx context.Context, storyID string) (int64, error)
	ListStoryFollowRecordsByStory(ctx context.Context, storyID string, limit, offset int) ([]*Follow, error)
	CountStoriesFollowedByUser(ctx context.Context, userID string) (int64, error)
	ListStoryFollowRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*Follow, error)

	FollowCharacter(ctx context.Context, userID, characterID string) error
	UnfollowCharacter(ctx context.Context, userID, characterID string) error
	IsCharacterFollowing(ctx context.Context, userID, characterID string) (bool, error)
	CountFollowersOfCharacter(ctx context.Context, characterID string) (int64, error)
	ListCharacterFollowRecordsByCharacter(ctx context.Context, characterID string, limit, offset int) ([]*Follow, error)
	CountCharactersFollowedByUser(ctx context.Context, userID string) (int64, error)
	ListCharacterFollowRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*Follow, error)

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
	// StoryboardFeedFromFollowedStories returns the following tab: reader-visible storyboards on followed stories (story_follows),
	// on public stories by followed users (user_follows), and on the viewer’s own stories (author match; any story visibility).
	StoryboardFeedFromFollowedStories(ctx context.Context, userID string, limit, offset int) ([]*Storyboard, int64, error)
	// StoryboardFeedRecommended is the “for you” tab: published storyboards matching onboarding preferred genres, plus a public engagement fallback (guests: trending).
	// excludeStoryboardIDs may be nil; IDs in the set are omitted from the merged list (oversampling fills the page when possible).
	StoryboardFeedRecommended(ctx context.Context, userID string, limit, offset int, excludeStoryboardIDs map[string]struct{}) ([]*Storyboard, int64, error)
	// StoryboardFeedDiscover is the discover tab: only published public storyboards whose story genre is in preferredGenres, by updated_at; guests get trending; empty genres => empty.
	StoryboardFeedDiscover(ctx context.Context, userID string, limit, offset int) ([]*Storyboard, int64, error)
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
	ListStyleConfigs(ctx context.Context, limit, offset int) ([]*StyleConfig, int64, error)
	SearchStyleConfigs(ctx context.Context, keyword string, limit, offset int) ([]*StyleConfig, int64, error)
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

	// ========== Referral System operations (StoryCreationAppUI Design) ==========
	GetUserByReferralCode(ctx context.Context, referralCode string) (*User, error)
	CreateUserReferral(ctx context.Context, referral *UserReferral) error
	GetUserReferralByReferee(ctx context.Context, refereeID string) (*UserReferral, error)
	GetReferralsByUser(ctx context.Context, referrerID string, limit, offset int) ([]*UserReferral, error)
	GetReferralStats(ctx context.Context, userID string) (*ReferralStats, error)
	AddUserPoints(ctx context.Context, userID string, points int) error

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

// BookmarkRepository 收藏/保存相关操作（多态关联）- StoryCreationAppUI Alignment
type BookmarkRepository interface {
	CreateBookmark(ctx context.Context, bookmark *Bookmark) error
	DeleteBookmark(ctx context.Context, id string) error
	GetBookmarkByID(ctx context.Context, id string) (*Bookmark, error)
	GetBookmarksByUser(ctx context.Context, userID string, bookmarkType BookmarkType) ([]*Bookmark, error)
	GetBookmarksByUserPaginated(ctx context.Context, userID string, bookmarkType BookmarkType, limit, offset int) ([]*Bookmark, int64, error)
	GetBookmarksByItem(ctx context.Context, bookmarkType BookmarkType, bookmarkID string) ([]*Bookmark, error)
	CheckBookmarkStatus(ctx context.Context, userID string, bookmarkType BookmarkType, bookmarkID string) (bool, error)
	GetBookmarksCount(ctx context.Context, bookmarkType BookmarkType, bookmarkID string) (int, error)
	UpdateBookmarksCount(ctx context.Context, bookmarkType BookmarkType, bookmarkID string, delta int) error
}

// UserSettingsRepository 用户设置操作
type UserSettingsRepository interface {
	GetUserSettings(userID string) (*UserSettings, error)
	CreateUserSettings(settings *UserSettings) error
	UpdateUserSettings(settings *UserSettings) error
}

// FeedbackRepository 用户反馈
type FeedbackRepository interface {
	CreateFeedback(ctx context.Context, fb *UserFeedback) error
	ListFeedbackByUserID(ctx context.Context, userID string, limit, offset int) ([]*UserFeedback, int64, error)
}
