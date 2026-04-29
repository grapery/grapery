package cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	PrefixUser        = "user:"
	PrefixStory       = "story:"
	PrefixStoryboard  = "storyboard:"
	PrefixCharacter   = "character:"
	PrefixComment     = "comment:"
	PrefixChatThread  = "chat_thread:"
	PrefixChatMessage = "chat_message:"

	PrefixUserStories       = "user_stories:"
	PrefixUserCharacters    = "user_characters:"
	PrefixStoryPanels       = "story_panels:"
	PrefixStoryComments     = "story_comments:"
	PrefixUserStoryboards   = "user_storyboards:"

	PrefixStoryLikes    = "story_likes:"
	PrefixStoryViews    = "story_views:"
	PrefixCreatorAnalytics = "creator_analytics:"

	PrefixTrendingStories   = "trending:stories"
	PrefixPopularCharacters = "popular:characters"

	PrefixSession       = "session:"
	PrefixPasswordReset = "pwd_reset:"
	PrefixEmailVerify   = "email_verify:"
	PrefixEmailVerifyCode      = "email_verify_code:"
	PrefixEmailVerifySendLimit = "email_verify_send_limit:"
	PrefixEmailVerifyIPLimit   = "email_verify_ip_limit:"

	PrefixSmsPhoneIPLimit  = "sms:phone:limit:ip:"
	PrefixSmsPhoneUserSend = "sms:phone:send:user:"
	PrefixSmsPhonePhoneSend = "sms:phone:send:phone:"
	PrefixSmsPhoneOTP      = "sms:phone:otp:"

	PrefixSearchStories    = "search:stories:"
	PrefixSearchCharacters = "search:characters:"
	PrefixSearchUsers      = "search:users:"
	PrefixSearchAll        = "search:all:"
	PrefixSearchIndexStories    = "search:index:stories:"
	PrefixSearchIndexCharacters = "search:index:characters:"
	PrefixSearchIndexUsers      = "search:index:users:"

	PrefixStyleConfigByID   = "style_config:id:"
	PrefixStyleConfigByStyle = "style_config:style:"
	PrefixStyleConfigsList   = "style_configs:list:"
)

const (
	PrefixRateLimitAI   = "ratelimit:ai:"
	PrefixRateLimitAuth = "ratelimit:auth:"
	PrefixRateLimitAPI  = "ratelimit:api:"
)

func searchHash(query, searchType string, limit, offset int) string {
	h := md5.Sum([]byte(fmt.Sprintf("%s:%s:%d:%d", query, searchType, limit, offset)))
	return hex.EncodeToString(h[:])
}

// UserKey user profile cache key
func UserKey(userID string) string {
	return PrefixUser + userID
}

func StoryKey(storyID string) string {
	return PrefixStory + storyID
}

func CharacterKey(characterID string) string {
	return PrefixCharacter + characterID
}

func StoryboardKey(boardID string) string {
	return PrefixStoryboard + boardID
}

func CommentKey(commentID string) string {
	return PrefixComment + commentID
}

func UserFollowingKey(userID string) string {
	return PrefixUser + userID + ":following"
}

func UserFollowersKey(userID string) string {
	return PrefixUser + userID + ":followers"
}

func UserStoriesListKey(userID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%d:%d", PrefixUserStories, userID, limit, offset)
}

func UserCharactersListKey(userID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%d:%d", PrefixUserCharacters, userID, limit, offset)
}

func UserStoryboardsListKey(userID string, limit, offset int) string {
	return fmt.Sprintf("%s%s:%d:%d", PrefixUserStoryboards, userID, limit, offset)
}

func StoryboardsListKey(storyID string, limit, offset int) string {
	return fmt.Sprintf("%sstoryboards:%s:%d:%d", PrefixStory, storyID, limit, offset)
}

func StoryCommentsKey(storyID string) string {
	return PrefixStoryComments + storyID
}

func CommentsListKey(targetType, targetID, sort string, limit, offset int) string {
	return fmt.Sprintf("%scomments:%s:%s:%s:%d:%d", PrefixComment, strings.ToLower(targetType), targetID, sort, limit, offset)
}

func SearchStoriesKey(query, searchType string, limit, offset int) string {
	return PrefixSearchStories + searchHash(query, searchType, limit, offset)
}

func SearchCharactersKey(query, searchType string, limit, offset int) string {
	return PrefixSearchCharacters + searchHash(query, searchType, limit, offset)
}

func SearchUsersKey(query, searchType string, limit, offset int) string {
	return PrefixSearchUsers + searchHash(query, searchType, limit, offset)
}

func SearchAllKey(query string, limit int) string {
	h := md5.Sum([]byte(fmt.Sprintf("%s:%d", query, limit)))
	return PrefixSearchAll + hex.EncodeToString(h[:])
}

func SearchIndexStoriesKey(keyword string) string {
	return PrefixSearchIndexStories + strings.ToLower(strings.TrimSpace(keyword))
}

func SearchIndexCharactersKey(keyword string) string {
	return PrefixSearchIndexCharacters + strings.ToLower(strings.TrimSpace(keyword))
}

func SearchIndexUsersKey(keyword string) string {
	return PrefixSearchIndexUsers + strings.ToLower(strings.TrimSpace(keyword))
}

func StyleConfigByIDKey(id string) string {
	return PrefixStyleConfigByID + id
}

func StyleConfigByStyleKey(style string) string {
	return PrefixStyleConfigByStyle + strings.TrimSpace(strings.ToLower(style))
}

func StyleConfigsListKey(limit, offset int) string {
	return fmt.Sprintf("%s%d:%d", PrefixStyleConfigsList, limit, offset)
}

func CreatorAnalyticsKey(userID string, rangeKey string) string {
	return PrefixCreatorAnalytics + userID + ":" + rangeKey
}

func PlazaTopFragmentTopicsKeyV1() string {
	return "plaza:top_fragment_topics:v1"
}

func EmailVerifySendLimitKey(emailLower string) string {
	return PrefixEmailVerifySendLimit + emailLower
}

func EmailVerifyIPLimitKey(ip string) string {
	return PrefixEmailVerifyIPLimit + ip
}

func EmailVerifyCodeKey(emailLower string) string {
	return PrefixEmailVerifyCode + emailLower
}

func PasswordResetKey(token string) string {
	return PrefixPasswordReset + token
}

func SMSPhoneIPLimitKey(ip string) string {
	return PrefixSmsPhoneIPLimit + ip
}

func SMSPhoneSendUserKey(userID string) string {
	return PrefixSmsPhoneUserSend + userID
}

func SMSPhoneSendPhoneKey(phone string) string {
	return PrefixSmsPhonePhoneSend + phone
}

func SMSPhoneOTPKey(userID string, phone string) string {
	return PrefixSmsPhoneOTP + userID + ":" + strings.TrimPrefix(strings.TrimPrefix(phone, "+"), " ")
}

// AITextProviderInflightKey is the Redis string counter used for cluster-wide outbound LLM HTTP text concurrency.
func AITextProviderInflightKey() string {
	return "ai:concurrency:text:inflight:v1"
}

// RateLimitKey generates a Redis key for fixed-window rate limiting.
func RateLimitKey(prefix, identifier string, windowBucket int64) string {
	return fmt.Sprintf("%s%s:%d", prefix, identifier, windowBucket)
}
