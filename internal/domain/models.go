package domain

// This file re-exports all domain models for backward compatibility.
// All models are now organized into separate files by business domain:
// - common.go: Common errors and base models
// - user_models.go: User, UserFollow, UserSettings
// - story_models.go: Story, Panel, StoryLike, StoryFollow, StoryPublication
// - storyboard_models.go: Storyboard, Scene, StoryboardCharacter, StoryboardLike, StoryComposition, StoryParticipant
// - character_models.go: Character, CharacterFollow
// - group_models.go: Group, GroupMember, GroupMemberInfo, GroupInvitation, GroupActivity, GroupPermission
// - comment_models.go: Comment, CommentLike
// - chat_models.go: ChatThread, ChatMessage
// - notification_models.go: Notification
// - asset_models.go: Asset
// - tag_models.go: Tag, StoryTag, CharacterTag
// - ai_models.go: AITask, AIGenerationRecord, RenderTask, RenderConfig
// - style_models.go: StyleConfig
// - payment_models.go: Membership, SubscriptionPlan, SubscriptionOrder, TokenTransaction
// - search_models.go: SearchHistory, ViewHistory, SearchFilter, SearchResult

// All types are automatically available when importing this package.
