package cache

import "fmt"

// 缓存键前缀定义
const (
	PrefixUser        = "user:"
	PrefixStory       = "story:"
	PrefixStoryboard  = "storyboard:"
	PrefixCharacter   = "character:"
	PrefixGroup       = "group:"
	PrefixComment     = "comment:"
	PrefixChatThread  = "chat_thread:"
	PrefixChatMessage = "chat_message:"

	// 列表缓存
	PrefixUserStories    = "user_stories:"
	PrefixUserCharacters = "user_characters:"
	PrefixUserGroups     = "user_groups:"
	PrefixStoryPanels    = "story_panels:"
	PrefixStoryComments  = "story_comments:"
	PrefixGroupMembers   = "group_members:"

	// 统计缓存
	PrefixStoryLikes    = "story_likes:"
	PrefixStoryViews    = "story_views:"
	PrefixUserFollowers = "user_followers:"
	PrefixUserFollowing = "user_following:"

	// 热门排行
	PrefixTrendingStories   = "trending:stories"
	PrefixPopularCharacters = "popular:characters"
	PrefixActiveGroups      = "active:groups"

	// 会话缓存
	PrefixSession       = "session:"
	PrefixPasswordReset = "pwd_reset:"
	PrefixEmailVerify   = "email_verify:"
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

func GroupKey(groupID string) string {
	return fmt.Sprintf("%s%s", PrefixGroup, groupID)
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

func UserGroupsKey(userID string) string {
	return fmt.Sprintf("%s%s", PrefixUserGroups, userID)
}

func StoryPanelsKey(storyID string) string {
	return fmt.Sprintf("%s%s", PrefixStoryPanels, storyID)
}

func StoryCommentsKey(storyID string) string {
	return fmt.Sprintf("%s%s", PrefixStoryComments, storyID)
}

func GroupMembersKey(groupID string) string {
	return fmt.Sprintf("%s%s", PrefixGroupMembers, groupID)
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
