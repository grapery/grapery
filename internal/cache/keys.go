package cache

import "fmt"

// 缓存键前缀定义
const (
	PrefixUser        = "user:"
	PrefixStory       = "story:"
	PrefixStoryboard  = "storyboard:"
	PrefixCharacter   = "character:"
	PrefixComment     = "comment:"
	PrefixChatThread  = "chat_thread:"
	PrefixChatMessage = "chat_message:"

	// 列表缓存
	PrefixUserStories    = "user_stories:"
	PrefixUserCharacters = "user_characters:"
	PrefixStoryPanels    = "story_panels:"
	PrefixStoryComments  = "story_comments:"

	// 统计缓存
	PrefixStoryLikes    = "story_likes:"
	PrefixStoryViews    = "story_views:"
	// 创作者数据分析（设置页；短 TTL 防重复重查询）
	PrefixCreatorAnalytics = "creator_analytics:"
	PrefixUserFollowers = "user_followers:"
	PrefixUserFollowing = "user_following:"

	// 热门排行
	PrefixTrendingStories   = "trending:stories"
	PrefixPopularCharacters = "popular:characters"

	// 会话缓存
	PrefixSession              = "session:"
	PrefixPasswordReset        = "pwd_reset:"
	PrefixEmailVerify          = "email_verify:"
	PrefixEmailVerifyCode      = "email_verify_code:"
	PrefixEmailVerifySendLimit = "email_verify_send_limit:"
	PrefixEmailVerifyIPLimit   = "email_verify_ip_limit:"

	// 手机号短信验证码（登录后绑定）
	PrefixSMSPhoneOTP       = "sms_phone_otp:"
	PrefixSMSPhoneSendUser  = "sms_phone_send_user:"
	PrefixSMSPhoneSendPhone = "sms_phone_send_phone:"
	PrefixSMSPhoneIPLimit   = "sms_phone_ip_limit:"

	// 搜索缓存
	PrefixSearchStories    = "search:stories:"
	PrefixSearchCharacters = "search:characters:"
	PrefixSearchUsers      = "search:users:"
	PrefixSearchAll        = "search:all:"

	// 搜索索引（用于模糊搜索）
	PrefixSearchIndexStories    = "search_index:stories:"
	PrefixSearchIndexCharacters = "search_index:characters:"
	PrefixSearchIndexUsers      = "search_index:users:"

	// 列表缓存（带分页）
	PrefixUserStoriesList    = "user_stories_list:"
	PrefixUserCharactersList = "user_characters_list:"
	PrefixStoryboardsList    = "storyboards_list:"
	PrefixCommentsList       = "comments_list:"
	PrefixUserActivities     = "user_activities:"
	PrefixStyleConfigs       = "style_configs:"
)

// 缓存键生成函数

func UserKey(userID string) string {
	return fmt.Sprintf("%s%s", PrefixUser, userID)
}

func StoryKey(storyID string) string {
	return fmt.Sprintf("%s%s", PrefixStory, storyID)
}

func StoryboardKey(storyboardID string) string {
	return fmt.Sprintf("%s%s", PrefixStoryboard, storyboardID)
}

func CharacterKey(characterID string) string {
	return fmt.Sprintf("%s%s", PrefixCharacter, characterID)
}

func CommentKey(commentID string) string {
	return fmt.Sprintf("%s%s", PrefixComment, commentID)
}

func ChatThreadKey(threadID string) string {
	return fmt.Sprintf("%s%s", PrefixChatThread, threadID)
}

func UserStoriesKey(userID string) string {
	return fmt.Sprintf("%s%s", PrefixUserStories, userID)
}

func UserCharactersKey(userID string) string {
	return fmt.Sprintf("%s%s", PrefixUserCharacters, userID)
}

func StoryPanelsKey(storyID string) string {
	return fmt.Sprintf("%s%s", PrefixStoryPanels, storyID)
}

func StoryCommentsKey(storyID string) string {
	return fmt.Sprintf("%s%s", PrefixStoryComments, storyID)
}

func StoryLikesKey(storyID string) string {
	return fmt.Sprintf("%s%s", PrefixStoryLikes, storyID)
}

func StoryViewsKey(storyID string) string {
	return fmt.Sprintf("%s%s", PrefixStoryViews, storyID)
}

func UserFollowersKey(userID string) string {
	return fmt.Sprintf("%s%s", PrefixUserFollowers, userID)
}

func UserFollowingKey(userID string) string {
	return fmt.Sprintf("%s%s", PrefixUserFollowing, userID)
}

func SessionKey(sessionID string) string {
	return fmt.Sprintf("%s%s", PrefixSession, sessionID)
}

func PasswordResetKey(token string) string {
	return fmt.Sprintf("%s%s", PrefixPasswordReset, token)
}

func EmailVerifyKey(token string) string {
	return fmt.Sprintf("%s%s", PrefixEmailVerify, token)
}

// Email verification code key (per email)
func EmailVerifyCodeKey(email string) string {
	return fmt.Sprintf("%s%s", PrefixEmailVerifyCode, email)
}

// Rate limit keys
func EmailVerifySendLimitKey(email string) string {
	return fmt.Sprintf("%s%s", PrefixEmailVerifySendLimit, email)
}

func EmailVerifyIPLimitKey(ip string) string {
	return fmt.Sprintf("%s%s", PrefixEmailVerifyIPLimit, ip)
}

func SMSPhoneOTPKey(userID, normalizedPhone string) string {
	return fmt.Sprintf("%s%s:%s", PrefixSMSPhoneOTP, userID, normalizedPhone)
}

func SMSPhoneSendUserKey(userID string) string {
	return fmt.Sprintf("%s%s", PrefixSMSPhoneSendUser, userID)
}

func SMSPhoneSendPhoneKey(normalizedPhone string) string {
	return fmt.Sprintf("%s%s", PrefixSMSPhoneSendPhone, normalizedPhone)
}

func SMSPhoneIPLimitKey(ip string) string {
	return fmt.Sprintf("%s%s", PrefixSMSPhoneIPLimit, ip)
}

// 搜索缓存键生成函数
func SearchStoriesKey(query string, searchType string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%s:%d:%d", PrefixSearchStories, query, searchType, limit, offset)
}

func SearchCharactersKey(query string, searchType string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%s:%d:%d", PrefixSearchCharacters, query, searchType, limit, offset)
}

func SearchUsersKey(query string, searchType string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%s:%d:%d", PrefixSearchUsers, query, searchType, limit, offset)
}

func SearchAllKey(query string, limit int) string {
	return fmt.Sprintf("%s%s:%d", PrefixSearchAll, query, limit)
}

// 搜索索引键（用于模糊搜索）
func SearchIndexStoriesKey(keyword string) string {
	return fmt.Sprintf("%s%s", PrefixSearchIndexStories, keyword)
}

func SearchIndexCharactersKey(keyword string) string {
	return fmt.Sprintf("%s%s", PrefixSearchIndexCharacters, keyword)
}

func SearchIndexUsersKey(keyword string) string {
	return fmt.Sprintf("%s%s", PrefixSearchIndexUsers, keyword)
}

// 列表缓存键生成函数（带分页参数）
func UserStoriesListKey(userID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%d:%d", PrefixUserStoriesList, userID, limit, offset)
}

func UserCharactersListKey(userID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%d:%d", PrefixUserCharactersList, userID, limit, offset)
}

func UserStoryboardsListKey(userID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%d:%d", "user_storyboards_list:", userID, limit, offset)
}

func StoryboardsListKey(storyID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%d:%d", PrefixStoryboardsList, storyID, limit, offset)
}

func CommentsListKey(targetType, targetID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%s:%d:%d", PrefixCommentsList, targetType, targetID, limit, offset)
}

func UserActivitiesKey(userID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%d:%d", PrefixUserActivities, userID, limit, offset)
}

func StyleConfigsListKey(limit, offset int) string {
	return fmt.Sprintf("%sall:%d:%d", PrefixStyleConfigs, limit, offset)
}

func StyleConfigByIDKey(id string) string {
	return fmt.Sprintf("%sid:%s", PrefixStyleConfigs, id)
}

func StyleConfigByStyleKey(styleName string) string {
	return fmt.Sprintf("%sstyle:%s", PrefixStyleConfigs, styleName)
}

// CreatorAnalyticsKey caches GET /me/creator-analytics payloads per user and range.
func CreatorAnalyticsKey(userID, rangeKey string) string {
	return fmt.Sprintf("%s%s:%s", PrefixCreatorAnalytics, userID, rangeKey)
}

// PlazaTopFragmentTopicsKeyV1 caches JSON array of top public fragment topic labels for GET /plaza (bump version if semantics change).
func PlazaTopFragmentTopicsKeyV1() string {
	return "plaza:top_fragment_topics:v1"
}

// 限流 key 前缀
const (
	PrefixRateLimitAI   = "ratelimit:ai:"
	PrefixRateLimitAuth = "ratelimit:auth:"
	PrefixRateLimitAPI  = "ratelimit:api:"
)

// RateLimitKey generates a Redis key for fixed-window rate limiting.
// prefix: one of PrefixRateLimitAI/Auth/API
// identifier: userID (authenticated) or client IP (unauthenticated)
// windowBucket: time.Now().Unix() / int64(window.Seconds())
func RateLimitKey(prefix, identifier string, windowBucket int64) string {
	return fmt.Sprintf("%s%s:%d", prefix, identifier, windowBucket)
}
