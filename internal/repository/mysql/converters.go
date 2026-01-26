package mysql

import (
	"encoding/json"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ========== 时间转换辅助函数 ==========

// timeToUnix 将 time.Time 转换为 Unix 时间戳（秒）
func timeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// unixToTime 将 Unix 时间戳（秒）转换为 time.Time
func unixToTime(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0)
}

// int64PtrToTime 将 *int64 转换为 time.Time
func int64PtrToTime(ptr *int64) time.Time {
	if ptr == nil || *ptr == 0 {
		return time.Time{}
	}
	return time.Unix(*ptr, 0)
}

// timeToInt64Ptr 将 time.Time 转换为 *int64
func timeToInt64Ptr(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	unix := t.Unix()
	return &unix
}

// int64ToInt64Ptr 将 int64 转换为 *int64
func int64ToInt64Ptr(val int64) *int64 {
	if val == 0 {
		return nil
	}
	return &val
}

// int64PtrToInt64 将 *int64 转换为 int64
func int64PtrToInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// stringToStringPtr 将 string 转换为 *string
func stringToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// stringPtrToString 将 *string 转换为 string
func stringPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// ========== StyleConfig JSON 转换 ==========

// styleConfigToJSON 将 *domain.StyleConfig 序列化为 JSON 字符串
func styleConfigToJSON(style *domain.StyleConfig) string {
	if style == nil {
		return ""
	}
	data, err := json.Marshal(style)
	if err != nil {
		return ""
	}
	return string(data)
}

// jsonToStyleConfig 将 JSON 字符串反序列化为 *domain.StyleConfig
func jsonToStyleConfig(jsonStr string) *domain.StyleConfig {
	if jsonStr == "" {
		return nil
	}
	var style domain.StyleConfig
	if err := json.Unmarshal([]byte(jsonStr), &style); err != nil {
		return nil
	}
	return &style
}

// ========== User 转换 ==========

// UserToModel 将 domain.User 转换为 MySQL User 模型
func UserToModel(d *domain.User) *User {
	if d == nil {
		return nil
	}
	return &User{
		ID:                  d.ID,
		Username:            d.Username,
		Email:               d.Email,
		PasswordHash:        d.PasswordHash,
		DisplayName:         d.DisplayName,
		Avatar:              d.Avatar,
		Background:          d.Background,
		Bio:                 d.Bio,
		Location:            d.Location,
		Website:             d.Website,
		AIPromptPreferences: d.AIPromptPreferences,
		DateOfBirth:         int64PtrToInt64(d.DateOfBirth),
		Followers:           d.Followers,
		Following:           d.Following,
		StoryboardCount:     d.StoryboardCount,
		GroupsCount:         d.GroupsCount,
		GroupsCreated:       d.GroupsCreated,
		Status:              d.Status,
		EmailVerified:       d.EmailVerified,
		LastLoginAt:         int64PtrToInt64(d.LastLoginAt),
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
	}
}

// ModelToUser 将 MySQL User 模型转换为 domain.User
func ModelToUser(m *User) *domain.User {
	if m == nil {
		return nil
	}
	return &domain.User{
		ID:                  m.ID,
		Username:            m.Username,
		Email:               m.Email,
		PasswordHash:        m.PasswordHash,
		DisplayName:         m.DisplayName,
		Avatar:              m.Avatar,
		Background:          m.Background,
		Bio:                 m.Bio,
		Location:            m.Location,
		Website:             m.Website,
		AIPromptPreferences: m.AIPromptPreferences,
		DateOfBirth:         int64ToInt64Ptr(m.DateOfBirth),
		Followers:           m.Followers,
		Following:           m.Following,
		StoryboardCount:     m.StoryboardCount,
		GroupsCount:         m.GroupsCount,
		GroupsCreated:       m.GroupsCreated,
		Status:              m.Status,
		EmailVerified:       m.EmailVerified,
		LastLoginAt:         int64ToInt64Ptr(m.LastLoginAt),
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}

// ========== Story 转换 ==========

// StoryToModel 将 domain.Story 转换为 MySQL Story 模型
func StoryToModel(d *domain.Story) *Story {
	if d == nil {
		return nil
	}
	return &Story{
		ID:                d.ID,
		Title:             d.Title,
		Description:       d.Description,
		CoverImage:        d.CoverImage,
		AuthorID:          d.AuthorID,
		GroupID:           stringToStringPtr(d.GroupID),
		Likes:             d.Likes,
		Followers:         d.Followers,
		Panels:            d.Panels,
		StoryboardCount:   d.StoryboardCount,
		DefaultSceneCount: d.DefaultSceneCount,
		Genre:             d.Genre,
		Style:             styleConfigToJSON(d.Style),
		Status:            d.Status,
		CreatedAt:         unixToTime(d.CreatedAt),
		UpdatedAt:         unixToTime(d.UpdatedAt),
	}
}

// ModelToStory 将 MySQL Story 模型转换为 domain.Story
func ModelToStory(m *Story) *domain.Story {
	if m == nil {
		return nil
	}
	d := &domain.Story{
		ID:                m.ID,
		Title:             m.Title,
		Description:       m.Description,
		CoverImage:        m.CoverImage,
		AuthorID:          m.AuthorID,
		GroupID:           stringPtrToString(m.GroupID),
		Likes:             m.Likes,
		Followers:         m.Followers,
		Panels:            m.Panels,
		StoryboardCount:   m.StoryboardCount,
		DefaultSceneCount: m.DefaultSceneCount,
		Genre:             m.Genre,
		Style:             jsonToStyleConfig(m.Style),
		Status:            m.Status,
		CreatedAt:         timeToUnix(m.CreatedAt),
		UpdatedAt:         timeToUnix(m.UpdatedAt),
	}
	if m.Author.ID != "" {
		d.Author = ModelToUser(&m.Author)
	}
	return d
}

// ========== Panel 转换 ==========

// PanelToModel 将 domain.Panel 转换为 MySQL Panel 模型
func PanelToModel(d *domain.Panel) *Panel {
	if d == nil {
		return nil
	}
	return &Panel{
		ID:        d.ID,
		StoryID:   d.StoryID,
		Sequence:  d.Sequence,
		Title:     d.Title,
		Content:   d.Content,
		Image:     d.Image,
		Likes:     d.Likes,
		Published: d.Published,
		CreatedAt: unixToTime(d.CreatedAt),
	}
}

// ModelToPanel 将 MySQL Panel 模型转换为 domain.Panel
func ModelToPanel(m *Panel) *domain.Panel {
	if m == nil {
		return nil
	}
	return &domain.Panel{
		ID:        m.ID,
		StoryID:   m.StoryID,
		Sequence:  m.Sequence,
		Title:     m.Title,
		Content:   m.Content,
		Image:     m.Image,
		Likes:     m.Likes,
		Published: m.Published,
		CreatedAt: timeToUnix(m.CreatedAt),
	}
}

// ========== Storyboard 转换 ==========

// StoryboardToModel 将 domain.Storyboard 转换为 MySQL Storyboard 模型
func StoryboardToModel(d *domain.Storyboard) *Storyboard {
	if d == nil {
		return nil
	}
	return &Storyboard{
		ID:               d.ID,
		StoryID:          d.StoryID,
		ParentID:         stringToStringPtr(d.ParentID),
		CreatorID:        d.CreatorID,
		Title:            d.Title,
		Content:          d.Content,
		RawInput:         d.RawInput,
		IsStandalone:     d.IsStandalone,
		IsAIGenerated:    d.IsAIGenerated,
		SceneCount:       d.SceneCount,
		WorkflowStatus:   d.WorkflowStatus,
		CurrentStep:      d.CurrentStep,
		Likes:            d.Likes,
		Comments:         d.Comments,
		Shares:           d.Shares,
		ForkCount:        d.ForkCount,
		Views:            d.Views,
		TokenConsumption: d.TokenConsumption,
		CreatedAt:        unixToTime(d.CreatedAt),
		UpdatedAt:        unixToTime(d.UpdatedAt),
	}
}

// ModelToStoryboard 将 MySQL Storyboard 模型转换为 domain.Storyboard
func ModelToStoryboard(m *Storyboard) *domain.Storyboard {
	if m == nil {
		return nil
	}
	d := &domain.Storyboard{
		ID:               m.ID,
		StoryID:          m.StoryID,
		ParentID:         stringPtrToString(m.ParentID),
		CreatorID:        m.CreatorID,
		CreatorName:      m.Creator.DisplayName,
		CreatorAvatar:    m.Creator.Avatar,
		Title:            m.Title,
		Content:          m.Content,
		RawInput:         m.RawInput,
		IsStandalone:     m.IsStandalone,
		IsAIGenerated:    m.IsAIGenerated,
		SceneCount:       m.SceneCount,
		WorkflowStatus:   m.WorkflowStatus,
		CurrentStep:      m.CurrentStep,
		Likes:            m.Likes,
		Comments:         m.Comments,
		Shares:           m.Shares,
		ForkCount:        m.ForkCount,
		Views:            m.Views,
		TokenConsumption: m.TokenConsumption,
		CreatedAt:        timeToUnix(m.CreatedAt),
		UpdatedAt:        timeToUnix(m.UpdatedAt),
	}
	// AfterFind hook will populate transient fields
	return d
}

func modelToStoryScene(m *StoryScene) *domain.StoryScene {
	if m == nil {
		return nil
	}
	return &domain.StoryScene{
		ID:           m.ID,
		StoryID:      m.StoryID,
		Title:        m.Title,
		Description:  m.Description,
		Image:        m.Image,
		Location:     m.Location,
		TimeOfDay:    m.TimeOfDay,
		SourceType:   m.SourceType,
		SourcePrompt: m.SourcePrompt,
		SourceImage:  m.SourceImage,
		CreatedBy:    m.CreatedBy,
		LastEditedBy: m.LastEditedBy,
		IsPublic:     m.IsPublic,
		CreatedAt:    timeToUnix(m.CreatedAt),
		UpdatedAt:    timeToUnix(m.UpdatedAt),
	}
}

func modelToStoryboardCharacterRef(m *StoryboardCharacterLink) domain.StoryboardCharacterRef {
	var character *domain.Character
	if m.Character != nil {
		c := characterLinkToDomain(m.Character)
		character = &c
	}
	var characterID string
	if m.CharacterID != nil {
		characterID = *m.CharacterID
	}
	return domain.StoryboardCharacterRef{
		ID:           m.ID,
		StoryboardID: m.StoryboardID,
		CharacterID:  characterID,
		Role:         m.Role,
		Order:        m.Ordering,
		Notes:        m.Notes,
		CreatedAt:    timeToUnix(m.CreatedAt),
		UpdatedAt:    timeToUnix(m.UpdatedAt),
		Character:    character,
	}
}

// characterLinkToDomain converts a Character from StoryboardCharacterLink to domain.Character
func characterLinkToDomain(c *Character) domain.Character {
	if c == nil {
		return domain.Character{}
	}
	return domain.Character{
		ID:          c.ID,
		StoryID:     c.StoryID,
		Name:        c.Name,
		Description: c.Description,
		Avatar:      c.Avatar,
		Poster:      c.Poster,
		IsPublic:    c.IsPublic,
		CreatedAt:   c.CreatedAt.Unix(),
		UpdatedAt:   c.UpdatedAt.Unix(),
	}
}

func modelToStoryboardSceneRef(m *StoryboardSceneLink) domain.StoryboardSceneRef {
	var scene *domain.StoryScene
	if m.StoryScene != nil {
		scene = modelToStoryScene(m.StoryScene)
	}
	return domain.StoryboardSceneRef{
		ID:             m.ID,
		StoryboardID:   m.StoryboardID,
		StorySceneID:   m.StorySceneID,
		Sequence:       m.Sequence,
		IsPrimaryScene: m.IsPrimaryScene,
		CreatedAt:      timeToUnix(m.CreatedAt),
		UpdatedAt:      timeToUnix(m.UpdatedAt),
		StoryScene:     scene,
	}
}

// ========== Character 转换 ==========

// CharacterToModel 将 domain.Character 转换为 MySQL Character 模型
func CharacterToModel(d *domain.Character) *Character {
	if d == nil {
		return nil
	}
	return &Character{
		ID:           d.ID,
		StoryID:      d.StoryID,
		Name:         d.Name,
		Description:  d.Description,
		Avatar:       d.Avatar,
		Poster:       d.Poster,
		AuthorID:     d.AuthorID,
		Personality:  d.Personality,
		Background:   d.Background,
		SourceType:   d.SourceType,
		SourcePrompt: d.SourcePrompt,
		SourceImage:  d.SourceImage,
		CreatedBy:    d.CreatedBy,
		LastEditedBy: d.LastEditedBy,
		Likes:        d.Likes,
		Followers:    d.Followers,
		Stories:      d.Stories,
		Traits:       d.TraitsJSON,
		Skills:       d.SkillsJSON,
		IsPublic:     d.IsPublic,
		GroupID:      d.GroupID,
		CreatedAt:    unixToTime(d.CreatedAt),
		UpdatedAt:    unixToTime(d.UpdatedAt),
	}
}

// ModelToCharacter 将 MySQL Character 模型转换为 domain.Character
func ModelToCharacter(m *Character) *domain.Character {
	if m == nil {
		return nil
	}
	d := &domain.Character{
		ID:           m.ID,
		StoryID:      m.StoryID,
		AuthorID:     m.AuthorID,
		Name:         m.Name,
		Description:  m.Description,
		Avatar:       m.Avatar,
		Poster:       m.Poster,
		Personality:  m.Personality,
		Background:   m.Background,
		TraitsJSON:   m.Traits,
		SkillsJSON:   m.Skills,
		IsPublic:     m.IsPublic,
		SourceType:   m.SourceType,
		SourcePrompt: m.SourcePrompt,
		SourceImage:  m.SourceImage,
		CreatedBy:    m.CreatedBy,
		LastEditedBy: m.LastEditedBy,
		GroupID:      m.GroupID,
		Likes:        m.Likes,
		Followers:    m.Followers,
		Stories:      m.Stories,
		CreatedAt:    timeToUnix(m.CreatedAt),
		UpdatedAt:    timeToUnix(m.UpdatedAt),
	}
	if m.Author.ID != "" {
		d.Author = ModelToUser(&m.Author)
	}
	// AfterFind hook will populate Traits and Skills
	return d
}

// ========== Group 转换 ==========

// GroupToModel 将 domain.Group 转换为 MySQL Group 模型
func GroupToModel(d *domain.Group) *Group {
	if d == nil {
		return nil
	}
	return &Group{
		ID:           d.ID,
		Name:         d.Name,
		Description:  d.Description,
		Avatar:       d.Avatar,
		CoverImage:   d.CoverImage,
		Members:      d.Members,
		Stories:      d.Stories,
		Followers:    d.Followers,
		BlockedCount: d.BlockedCount,
		CreatorID:    d.CreatorID,
		Public:       d.Public,
		CreatedAt:    unixToTime(d.CreatedAt),
		UpdatedAt:    unixToTime(d.UpdatedAt),
	}
}

// ModelToGroup 将 MySQL Group 模型转换为 domain.Group
func ModelToGroup(m *Group) *domain.Group {
	if m == nil {
		return nil
	}
	d := &domain.Group{
		ID:           m.ID,
		CreatorID:    m.CreatorID,
		Name:         m.Name,
		Description:  m.Description,
		Avatar:       m.Avatar,
		CoverImage:   m.CoverImage,
		Members:      m.Members,
		Stories:      m.Stories,
		Followers:    m.Followers,
		BlockedCount: m.BlockedCount,
		Public:       m.Public,
		CreatedAt:    timeToUnix(m.CreatedAt),
		UpdatedAt:    timeToUnix(m.UpdatedAt),
	}
	if m.Creator.ID != "" {
		d.Creator = ModelToUser(&m.Creator)
	}
	return d
}

// ========== Comment 转换 ==========

// CommentToModel 将 domain.Comment 转换为 MySQL Comment 模型
func CommentToModel(d *domain.Comment) *Comment {
	if d == nil {
		return nil
	}
	return &Comment{
		ID:         d.ID,
		AuthorID:   d.AuthorID,
		Content:    d.Content,
		TargetType: d.TargetType,
		TargetID:   d.TargetID,
		ParentID:   stringToStringPtr(d.ParentID),
		RootID:     stringToStringPtr(d.RootID),
		Likes:      d.Likes,
		Dislikes:   d.Dislikes,
		ReplyCount: d.ReplyCount,
		CreatedAt:  unixToTime(d.CreatedAt),
		UpdatedAt:  unixToTime(d.UpdatedAt),
	}
}

// ModelToComment 将 MySQL Comment 模型转换为 domain.Comment
func ModelToComment(m *Comment) *domain.Comment {
	if m == nil {
		return nil
	}
	d := &domain.Comment{
		ID:         m.ID,
		AuthorID:   m.AuthorID,
		Content:    m.Content,
		TargetType: m.TargetType,
		TargetID:   m.TargetID,
		ParentID:   stringPtrToString(m.ParentID),
		RootID:     stringPtrToString(m.RootID),
		Likes:      m.Likes,
		Dislikes:   m.Dislikes,
		ReplyCount: m.ReplyCount,
		CreatedAt:  timeToUnix(m.CreatedAt),
		UpdatedAt:  timeToUnix(m.UpdatedAt),
	}
	if m.Author.ID != "" {
		d.Author = ModelToUser(&m.Author)
	}
	return d
}

// ========== ChatThread 转换 ==========

// ChatThreadToModel 将 domain.ChatThread 转换为 MySQL ChatThread 模型
func ChatThreadToModel(d *domain.ChatThread) *ChatThread {
	if d == nil {
		return nil
	}
	return &ChatThread{
		ID:                   d.ID,
		CharacterID:          d.CharacterID,
		UserID:               d.UserID,
		StoryTitle:           d.StoryTitle,
		LastMessage:          d.LastMessage,
		LastMessageTime:      unixToTime(d.LastMessageTime),
		UnreadCount:          d.UnreadCount,
		MessageCount:         d.MessageCount,
		InteractionFrequency: d.InteractionFrequency,
		CreatedAt:            unixToTime(d.CreatedAt),
	}
}

// ModelToChatThread 将 MySQL ChatThread 模型转换为 domain.ChatThread
func ModelToChatThread(m *ChatThread) *domain.ChatThread {
	if m == nil {
		return nil
	}
	d := &domain.ChatThread{
		ID:                   m.ID,
		UserID:               m.UserID,
		CharacterID:          m.CharacterID,
		CharacterName:        m.Character.Name,
		CharacterAvatar:      m.Character.Avatar,
		StoryTitle:           m.StoryTitle,
		LastMessage:          m.LastMessage,
		LastMessageTime:      timeToUnix(m.LastMessageTime),
		UnreadCount:          m.UnreadCount,
		MessageCount:         m.MessageCount,
		InteractionFrequency: m.InteractionFrequency,
		CreatedAt:            timeToUnix(m.CreatedAt),
	}
	if m.User.ID != "" {
		d.User = ModelToUser(&m.User)
	}
	if m.Character.ID != "" {
		d.Character = ModelToCharacter(&m.Character)
	}
	return d
}

// ========== ChatMessage 转换 ==========

// ChatMessageToModel 将 domain.ChatMessage 转换为 MySQL ChatMessage 模型
func ChatMessageToModel(d *domain.ChatMessage) *ChatMessage {
	if d == nil {
		return nil
	}
	return &ChatMessage{
		ID:           d.ID,
		ThreadID:     d.ThreadID,
		SenderID:     d.SenderID,
		SenderName:   d.SenderName,
		SenderAvatar: d.SenderAvatar,
		Content:      d.Content,
		Image:        d.Image,
		IsUser:       d.IsUser,
		CreatedAt:    unixToTime(d.Timestamp),
	}
}

// ModelToChatMessage 将 MySQL ChatMessage 模型转换为 domain.ChatMessage
func ModelToChatMessage(m *ChatMessage) *domain.ChatMessage {
	if m == nil {
		return nil
	}
	return &domain.ChatMessage{
		ID:           m.ID,
		ThreadID:     m.ThreadID,
		SenderID:     m.SenderID,
		SenderName:   m.SenderName,
		SenderAvatar: m.SenderAvatar,
		Content:      m.Content,
		Image:        m.Image,
		Timestamp:    timeToUnix(m.CreatedAt),
		IsUser:       m.IsUser,
	}
}

// ========== 关系表转换 ==========

// UserFollowToModel 将 domain.UserFollow 转换为 MySQL UserFollow 模型
func UserFollowToModel(d *domain.UserFollow) *UserFollow {
	if d == nil {
		return nil
	}
	return &UserFollow{
		ID:         d.ID,
		FollowerID: d.FollowerID,
		FolloweeID: d.FolloweeID,
		CreatedAt:  unixToTime(d.CreatedAt),
	}
}

// ModelToUserFollow 将 MySQL UserFollow 模型转换为 domain.UserFollow
func ModelToUserFollow(m *UserFollow) *domain.UserFollow {
	if m == nil {
		return nil
	}
	return &domain.UserFollow{
		ID:         m.ID,
		FollowerID: m.FollowerID,
		FolloweeID: m.FolloweeID,
		CreatedAt:  timeToUnix(m.CreatedAt),
	}
}

// StoryLikeToModel 将 domain.StoryLike 转换为 MySQL StoryLike 模型
func StoryLikeToModel(d *domain.StoryLike) *StoryLike {
	if d == nil {
		return nil
	}
	return &StoryLike{
		ID:        d.ID,
		UserID:    d.UserID,
		StoryID:   d.StoryID,
		CreatedAt: unixToTime(d.CreatedAt),
	}
}

// ModelToStoryLike 将 MySQL StoryLike 模型转换为 domain.StoryLike
func ModelToStoryLike(m *StoryLike) *domain.StoryLike {
	if m == nil {
		return nil
	}
	return &domain.StoryLike{
		ID:        m.ID,
		UserID:    m.UserID,
		StoryID:   m.StoryID,
		CreatedAt: timeToUnix(m.CreatedAt),
	}
}

// StoryFollowToModel 将 domain.StoryFollow 转换为 MySQL StoryFollow 模型
func StoryFollowToModel(d *domain.StoryFollow) *StoryFollow {
	if d == nil {
		return nil
	}
	return &StoryFollow{
		ID:        d.ID,
		UserID:    d.UserID,
		StoryID:   d.StoryID,
		CreatedAt: unixToTime(d.CreatedAt),
	}
}

// ModelToStoryFollow 将 MySQL StoryFollow 模型转换为 domain.StoryFollow
func ModelToStoryFollow(m *StoryFollow) *domain.StoryFollow {
	if m == nil {
		return nil
	}
	return &domain.StoryFollow{
		ID:        m.ID,
		UserID:    m.UserID,
		StoryID:   m.StoryID,
		CreatedAt: timeToUnix(m.CreatedAt),
	}
}

// CharacterFollowToModel 将 domain.CharacterFollow 转换为 MySQL CharacterFollow 模型
func CharacterFollowToModel(d *domain.CharacterFollow) *CharacterFollow {
	if d == nil {
		return nil
	}
	return &CharacterFollow{
		ID:          d.ID,
		UserID:      d.UserID,
		CharacterID: d.CharacterID,
		CreatedAt:   unixToTime(d.CreatedAt),
	}
}

// ModelToCharacterFollow 将 MySQL CharacterFollow 模型转换为 domain.CharacterFollow
func ModelToCharacterFollow(m *CharacterFollow) *domain.CharacterFollow {
	if m == nil {
		return nil
	}
	return &domain.CharacterFollow{
		ID:          m.ID,
		UserID:      m.UserID,
		CharacterID: m.CharacterID,
		CreatedAt:   timeToUnix(m.CreatedAt),
	}
}

// GroupMemberToModel 将 domain.GroupMember 转换为 MySQL GroupMember 模型
func GroupMemberToModel(d *domain.GroupMember) *GroupMember {
	if d == nil {
		return nil
	}
	return &GroupMember{
		ID:       d.ID,
		GroupID:  d.GroupID,
		UserID:   d.UserID,
		Role:     string(d.Role),
		JoinedAt: unixToTime(d.JoinedAt),
	}
}

// ModelToGroupMember 将 MySQL GroupMember 模型转换为 domain.GroupMember
func ModelToGroupMember(m *GroupMember) *domain.GroupMember {
	if m == nil {
		return nil
	}
	return &domain.GroupMember{
		ID:       m.ID,
		GroupID:  m.GroupID,
		UserID:   m.UserID,
		Role:     domain.GroupMemberRole(m.Role),
		JoinedAt: timeToUnix(m.JoinedAt),
	}
}

// GroupInvitationToModel 将 domain.GroupInvitation 转换为 MySQL GroupInvitation 模型
func GroupInvitationToModel(d *domain.GroupInvitation) *GroupInvitation {
	if d == nil {
		return nil
	}
	return &GroupInvitation{
		ID:        d.ID,
		GroupID:   d.GroupID,
		InviterID: d.InviterID,
		InviteeID: d.InviteeID,
		Status:    d.Status,
		Message:   d.Message,
		CreatedAt: unixToTime(d.CreatedAt),
		ExpiresAt: unixToTime(d.ExpiresAt),
	}
}

// ModelToGroupInvitation 将 MySQL GroupInvitation 模型转换为 domain.GroupInvitation
func ModelToGroupInvitation(m *GroupInvitation) *domain.GroupInvitation {
	if m == nil {
		return nil
	}
	return &domain.GroupInvitation{
		ID:        m.ID,
		GroupID:   m.GroupID,
		InviterID: m.InviterID,
		InviteeID: m.InviteeID,
		Status:    m.Status,
		Message:   m.Message,
		CreatedAt: timeToUnix(m.CreatedAt),
		ExpiresAt: timeToUnix(m.ExpiresAt),
	}
}

// CommentLikeToModel 将 domain.CommentLike 转换为 MySQL CommentLike 模型
func CommentLikeToModel(d *domain.CommentLike) *CommentLike {
	if d == nil {
		return nil
	}
	return &CommentLike{
		ID:        d.ID,
		UserID:    d.UserID,
		CommentID: d.CommentID,
		IsLike:    d.IsLike,
		CreatedAt: unixToTime(d.CreatedAt),
	}
}

// ModelToCommentLike 将 MySQL CommentLike 模型转换为 domain.CommentLike
func ModelToCommentLike(m *CommentLike) *domain.CommentLike {
	if m == nil {
		return nil
	}
	return &domain.CommentLike{
		ID:        m.ID,
		UserID:    m.UserID,
		CommentID: m.CommentID,
		IsLike:    m.IsLike,
		CreatedAt: timeToUnix(m.CreatedAt),
	}
}

// StoryboardLikeToModel 将 domain.StoryboardLike 转换为 MySQL StoryboardLike 模型
func StoryboardLikeToModel(d *domain.StoryboardLike) *StoryboardLike {
	if d == nil {
		return nil
	}
	return &StoryboardLike{
		ID:           d.ID,
		UserID:       d.UserID,
		StoryboardID: d.StoryboardID,
		CreatedAt:    unixToTime(d.CreatedAt),
	}
}

// ModelToStoryboardLike 将 MySQL StoryboardLike 模型转换为 domain.StoryboardLike
func ModelToStoryboardLike(m *StoryboardLike) *domain.StoryboardLike {
	if m == nil {
		return nil
	}
	return &domain.StoryboardLike{
		ID:           m.ID,
		UserID:       m.UserID,
		StoryboardID: m.StoryboardID,
		CreatedAt:    timeToUnix(m.CreatedAt),
	}
}

// ========== Storyboard Generation 转换 ==========

// StoryboardContentGenerationToModel 将 domain 转换为 MySQL 模型
func StoryboardContentGenerationToModel(d *domain.StoryboardContentGeneration) *StoryboardContentGeneration {
	if d == nil {
		return nil
	}
	characterIDsJSON, _ := json.Marshal(d.CharacterIDs)
	sceneIDsJSON, _ := json.Marshal(d.SceneIDs)

	var completedAt *time.Time
	if d.CompletedAt != nil {
		t := unixToTime(*d.CompletedAt)
		completedAt = &t
	}

	return &StoryboardContentGeneration{
		ID:               d.ID,
		StoryboardID:     d.StoryboardID,
		RawInput:         d.RawInput,
		CharacterIDsJSON: string(characterIDsJSON),
		SceneIDsJSON:     string(sceneIDsJSON),
		Style:            d.Style,
		GeneratedContent: d.GeneratedContent,
		Status:           d.Status,
		InputTokens:      d.InputTokens,
		OutputTokens:     d.OutputTokens,
		TotalTokens:      d.TotalTokens,
		ErrorMessage:     d.ErrorMessage,
		CreatedAt:        unixToTime(d.CreatedAt),
		CompletedAt:      completedAt,
	}
}

// ModelToStoryboardContentGeneration 将 MySQL 模型转换为 domain
func ModelToStoryboardContentGeneration(m *StoryboardContentGeneration) *domain.StoryboardContentGeneration {
	if m == nil {
		return nil
	}
	var characterIDs []string
	var sceneIDs []string
	_ = json.Unmarshal([]byte(m.CharacterIDsJSON), &characterIDs)
	_ = json.Unmarshal([]byte(m.SceneIDsJSON), &sceneIDs)

	var completedAt *int64
	if m.CompletedAt != nil {
		unix := m.CompletedAt.Unix()
		completedAt = &unix
	}

	return &domain.StoryboardContentGeneration{
		ID:               m.ID,
		StoryboardID:     m.StoryboardID,
		RawInput:         m.RawInput,
		CharacterIDs:     characterIDs,
		SceneIDs:         sceneIDs,
		Style:            m.Style,
		GeneratedContent: m.GeneratedContent,
		Status:           m.Status,
		InputTokens:      m.InputTokens,
		OutputTokens:     m.OutputTokens,
		TotalTokens:      m.TotalTokens,
		ErrorMessage:     m.ErrorMessage,
		CreatedAt:        timeToUnix(m.CreatedAt),
		CompletedAt:      completedAt,
	}
}

// StoryboardSceneGenerationToModel 将 domain 转换为 MySQL 模型
func StoryboardSceneGenerationToModel(d *domain.StoryboardSceneGeneration) *StoryboardSceneGeneration {
	if d == nil {
		return nil
	}
	var completedAt *time.Time
	if d.CompletedAt != nil {
		t := unixToTime(*d.CompletedAt)
		completedAt = &t
	}

	return &StoryboardSceneGeneration{
		ID:               d.ID,
		StoryboardID:     d.StoryboardID,
		SceneID:          d.SceneID,
		SceneTitle:       d.SceneTitle,
		SceneLocation:    d.SceneLocation,
		InputDescription: d.InputDescription,
		GeneratedDetail:  d.GeneratedDetail,
		Status:           d.Status,
		InputTokens:      d.InputTokens,
		OutputTokens:     d.OutputTokens,
		TotalTokens:      d.TotalTokens,
		ErrorMessage:     d.ErrorMessage,
		CreatedAt:        unixToTime(d.CreatedAt),
		CompletedAt:      completedAt,
	}
}

// ModelToStoryboardSceneGeneration 将 MySQL 模型转换为 domain
func ModelToStoryboardSceneGeneration(m *StoryboardSceneGeneration) *domain.StoryboardSceneGeneration {
	if m == nil {
		return nil
	}
	var completedAt *int64
	if m.CompletedAt != nil {
		unix := m.CompletedAt.Unix()
		completedAt = &unix
	}

	return &domain.StoryboardSceneGeneration{
		ID:               m.ID,
		StoryboardID:     m.StoryboardID,
		SceneID:          m.SceneID,
		SceneTitle:       m.SceneTitle,
		SceneLocation:    m.SceneLocation,
		InputDescription: m.InputDescription,
		GeneratedDetail:  m.GeneratedDetail,
		Status:           m.Status,
		InputTokens:      m.InputTokens,
		OutputTokens:     m.OutputTokens,
		TotalTokens:      m.TotalTokens,
		ErrorMessage:     m.ErrorMessage,
		CreatedAt:        timeToUnix(m.CreatedAt),
		CompletedAt:      completedAt,
	}
}

// StoryboardImageGenerationToModel 将 domain 转换为 MySQL 模型
func StoryboardImageGenerationToModel(d *domain.StoryboardImageGeneration) *StoryboardImageGeneration {
	if d == nil {
		return nil
	}
	referenceImagesJSON, _ := json.Marshal(d.ReferenceImages)

	var completedAt *time.Time
	if d.CompletedAt != nil {
		t := unixToTime(*d.CompletedAt)
		completedAt = &t
	}

	return &StoryboardImageGeneration{
		ID:                  d.ID,
		StoryboardID:        d.StoryboardID,
		SceneID:             d.SceneID,
		SceneTitle:          d.SceneTitle,
		SceneDescription:    d.SceneDescription,
		ReferenceImagesJSON: string(referenceImagesJSON),
		GeneratedPrompt:     d.GeneratedPrompt,
		PromptDetailsJSON:   promptDetailsToJSON(d.PromptDetails),
		GeneratedImageURL:   d.GeneratedImageURL,
		ImageWidth:          d.ImageWidth,
		ImageHeight:         d.ImageHeight,
		Status:              d.Status,
		InputTokens:         d.InputTokens,
		OutputTokens:        d.OutputTokens,
		TotalTokens:         d.TotalTokens,
		ErrorMessage:        d.ErrorMessage,
		CreatedAt:           unixToTime(d.CreatedAt),
		CompletedAt:         completedAt,
	}
}

// ModelToStoryboardImageGeneration 将 MySQL 模型转换为 domain
func ModelToStoryboardImageGeneration(m *StoryboardImageGeneration) *domain.StoryboardImageGeneration {
	if m == nil {
		return nil
	}
	var referenceImages []string
	_ = json.Unmarshal([]byte(m.ReferenceImagesJSON), &referenceImages)

	var completedAt *int64
	if m.CompletedAt != nil {
		unix := m.CompletedAt.Unix()
		completedAt = &unix
	}

	var promptDetails *domain.ImagePromptDetails
	if m.PromptDetailsJSON != "" {
		promptDetails = jsonToPromptDetails(m.PromptDetailsJSON)
	}

	return &domain.StoryboardImageGeneration{
		ID:                m.ID,
		StoryboardID:      m.StoryboardID,
		SceneID:           m.SceneID,
		SceneTitle:        m.SceneTitle,
		SceneDescription:  m.SceneDescription,
		ReferenceImages:   referenceImages,
		GeneratedPrompt:   m.GeneratedPrompt,
		PromptDetails:     promptDetails,
		GeneratedImageURL: m.GeneratedImageURL,
		ImageWidth:        m.ImageWidth,
		ImageHeight:       m.ImageHeight,
		Status:            m.Status,
		InputTokens:       m.InputTokens,
		OutputTokens:      m.OutputTokens,
		TotalTokens:       m.TotalTokens,
		ErrorMessage:      m.ErrorMessage,
		CreatedAt:         timeToUnix(m.CreatedAt),
		CompletedAt:       completedAt,
	}
}

// StoryboardVideoGenerationToModel 将 domain 转换为 MySQL 模型
func StoryboardVideoGenerationToModel(d *domain.StoryboardVideoGeneration) *StoryboardVideoGeneration {
	if d == nil {
		return nil
	}
	var completedAt *time.Time
	if d.CompletedAt != nil {
		t := unixToTime(*d.CompletedAt)
		completedAt = &t
	}

	return &StoryboardVideoGeneration{
		ID:                  d.ID,
		StoryboardID:        d.StoryboardID,
		SceneID:             d.SceneID,
		SceneTitle:          d.SceneTitle,
		InputDescription:    d.InputDescription,
		ReferenceImageURL:   d.ReferenceImageURL,
		EndFrameURL:         d.EndFrameURL,
		GeneratedPrompt:     d.GeneratedPrompt,
		PromptDetailsJSON:   videoPromptDetailsToJSON(d.PromptDetails),
		GeneratedVideoURL:   d.GeneratedVideoURL,
		ProviderTaskID:      d.ProviderTaskID,
		ProviderName:        d.ProviderName,
		Duration:            d.Duration,
		Status:              d.Status,
		InputTokens:         d.InputTokens,
		OutputTokens:        d.OutputTokens,
		TotalTokens:         d.TotalTokens,
		ErrorMessage:        d.ErrorMessage,
		IsSubdivided:        d.IsSubdivided,
		VideoSegmentsJSON:   d.VideoSegmentsJSON,
		MiddleFrameURLsJSON: d.MiddleFrameURLsJSON,
		CreatedAt:           unixToTime(d.CreatedAt),
		CompletedAt:         completedAt,
	}
}

// ModelToStoryboardVideoGeneration 将 MySQL 模型转换为 domain
func ModelToStoryboardVideoGeneration(m *StoryboardVideoGeneration) *domain.StoryboardVideoGeneration {
	if m == nil {
		return nil
	}
	var completedAt *int64
	if m.CompletedAt != nil {
		unix := m.CompletedAt.Unix()
		completedAt = &unix
	}

	var videoPromptDetails *domain.VideoPromptDetails
	if m.PromptDetailsJSON != "" {
		videoPromptDetails = jsonToVideoPromptDetails(m.PromptDetailsJSON)
	}

	result := &domain.StoryboardVideoGeneration{
		ID:                  m.ID,
		StoryboardID:        m.StoryboardID,
		SceneID:             m.SceneID,
		SceneTitle:          m.SceneTitle,
		InputDescription:    m.InputDescription,
		ReferenceImageURL:   m.ReferenceImageURL,
		EndFrameURL:         m.EndFrameURL,
		GeneratedPrompt:     m.GeneratedPrompt,
		PromptDetails:       videoPromptDetails,
		GeneratedVideoURL:   m.GeneratedVideoURL,
		ProviderTaskID:      m.ProviderTaskID,
		ProviderName:        m.ProviderName,
		Duration:            m.Duration,
		Status:              m.Status,
		InputTokens:         m.InputTokens,
		OutputTokens:        m.OutputTokens,
		TotalTokens:         m.TotalTokens,
		ErrorMessage:        m.ErrorMessage,
		IsSubdivided:        m.IsSubdivided,
		VideoSegmentsJSON:   m.VideoSegmentsJSON,
		MiddleFrameURLsJSON: m.MiddleFrameURLsJSON,
		CreatedAt:           timeToUnix(m.CreatedAt),
		CompletedAt:         completedAt,
	}

	// Parse video segments from JSON
	if m.VideoSegmentsJSON != "" {
		var segments []domain.VideoSegmentInfo
		if err := json.Unmarshal([]byte(m.VideoSegmentsJSON), &segments); err == nil {
			result.VideoSegments = segments
		}
	}

	// Parse middle frame URLs from JSON
	if m.MiddleFrameURLsJSON != "" {
		var urls []string
		if err := json.Unmarshal([]byte(m.MiddleFrameURLsJSON), &urls); err == nil {
			result.MiddleFrameURLs = urls
		}
	}

	return result
}

// ========== Asset 转换 ==========

// AssetToModel 将 domain.Asset 转换为 MySQL Asset 模型
func AssetToModel(d *domain.Asset) *Asset {
	if d == nil {
		return nil
	}
	return &Asset{
		ID:         d.ID,
		UserID:     d.UserID,
		Type:       d.Type,
		Name:       d.Name,
		URL:        d.URL,
		Thumbnail:  d.Thumbnail,
		Size:       d.Size,
		MimeType:   d.MimeType,
		Width:      d.Width,
		Height:     d.Height,
		Duration:   d.Duration,
		Tags:       d.TagsJSON,
		UsageCount: d.UsageCount,
		CreatedAt:  unixToTime(d.CreatedAt),
	}
}

// ModelToAsset 将 MySQL Asset 模型转换为 domain.Asset
func ModelToAsset(m *Asset) *domain.Asset {
	if m == nil {
		return nil
	}
	d := &domain.Asset{
		ID:         m.ID,
		UserID:     m.UserID,
		Type:       m.Type,
		Name:       m.Name,
		URL:        m.URL,
		Thumbnail:  m.Thumbnail,
		Size:       m.Size,
		MimeType:   m.MimeType,
		Width:      m.Width,
		Height:     m.Height,
		Duration:   m.Duration,
		TagsJSON:   m.Tags,
		UsageCount: m.UsageCount,
		CreatedAt:  timeToUnix(m.CreatedAt),
	}
	// AfterFind hook will populate Tags
	return d
}

// ========== Notification 转换 ==========

// NotificationToModel 将 domain.Notification 转换为 MySQL Notification 模型
func NotificationToModel(d *domain.Notification) *Notification {
	if d == nil {
		return nil
	}
	return &Notification{
		ID:          d.ID,
		UserID:      d.UserID,
		Type:        d.Type,
		Title:       d.Title,
		Content:     d.Content,
		Link:        d.Link,
		Read:        d.Read,
		ActorID:     d.ActorID,
		ActorName:   d.ActorName,
		ActorAvatar: d.ActorAvatar,
		CreatedAt:   unixToTime(d.CreatedAt),
	}
}

// ModelToNotification 将 MySQL Notification 模型转换为 domain.Notification
func ModelToNotification(m *Notification) *domain.Notification {
	if m == nil {
		return nil
	}
	return &domain.Notification{
		ID:          m.ID,
		UserID:      m.UserID,
		Type:        m.Type,
		Title:       m.Title,
		Content:     m.Content,
		Link:        m.Link,
		Read:        m.Read,
		ActorID:     m.ActorID,
		ActorName:   m.ActorName,
		ActorAvatar: m.ActorAvatar,
		CreatedAt:   timeToUnix(m.CreatedAt),
	}
}

// ========== UserSettings 转换 ==========

// UserSettingsToModel 将 domain.UserSettings 转换为 MySQL UserSettings 模型
func UserSettingsToModel(d *domain.UserSettings) *UserSettings {
	if d == nil {
		return nil
	}
	return &UserSettings{
		ID:                 d.ID,
		UserID:             d.UserID,
		Language:           d.Language,
		Theme:              d.Theme,
		EmailNotifications: d.EmailNotifications,
		PushNotifications:  d.PushNotifications,
		ShowAdultContent:   d.ShowAdultContent,
		ProfileVisibility:  d.ProfileVisibility,
		AllowComments:      d.AllowComments,
		AllowMessages:      d.AllowMessages,
		ShowOnlineStatus:   d.ShowOnlineStatus,
		UpdatedAt:          unixToTime(d.UpdatedAt),
	}
}

// ModelToUserSettings 将 MySQL UserSettings 模型转换为 domain.UserSettings
func ModelToUserSettings(m *UserSettings) *domain.UserSettings {
	if m == nil {
		return nil
	}
	return &domain.UserSettings{
		ID:                 m.ID,
		UserID:             m.UserID,
		Language:           m.Language,
		Theme:              m.Theme,
		EmailNotifications: m.EmailNotifications,
		PushNotifications:  m.PushNotifications,
		ShowAdultContent:   m.ShowAdultContent,
		ProfileVisibility:  m.ProfileVisibility,
		AllowComments:      m.AllowComments,
		AllowMessages:      m.AllowMessages,
		ShowOnlineStatus:   m.ShowOnlineStatus,
		UpdatedAt:          timeToUnix(m.UpdatedAt),
	}
}

// ========== Membership 转换 ==========

// MembershipToModel 将 domain.Membership 转换为 MySQL Membership 模型
func MembershipToModel(d *domain.Membership) *Membership {
	if d == nil {
		return nil
	}
	var endDate *time.Time
	if d.EndDate != nil && *d.EndDate != 0 {
		t := unixToTime(*d.EndDate)
		endDate = &t
	}
	return &Membership{
		ID:           d.ID,
		UserID:       d.UserID,
		Tier:         d.Tier,
		Status:       d.Status,
		StartDate:    unixToTime(d.StartDate),
		EndDate:      endDate,
		AutoRenew:    d.AutoRenew,
		TokenQuota:   d.TokenQuota,
		TokenUsed:    d.TokenUsed,
		StorageQuota: d.StorageQuota,
		StorageUsed:  d.StorageUsed,
		CreatedAt:    unixToTime(d.CreatedAt),
		UpdatedAt:    unixToTime(d.UpdatedAt),
	}
}

// ModelToMembership 将 MySQL Membership 模型转换为 domain.Membership
func ModelToMembership(m *Membership) *domain.Membership {
	if m == nil {
		return nil
	}
	var endDate *int64
	if m.EndDate != nil && !m.EndDate.IsZero() {
		unix := timeToUnix(*m.EndDate)
		endDate = &unix
	}
	return &domain.Membership{
		ID:           m.ID,
		UserID:       m.UserID,
		Tier:         m.Tier,
		Status:       m.Status,
		StartDate:    timeToUnix(m.StartDate),
		EndDate:      endDate,
		AutoRenew:    m.AutoRenew,
		TokenQuota:   m.TokenQuota,
		TokenUsed:    m.TokenUsed,
		StorageQuota: m.StorageQuota,
		StorageUsed:  m.StorageUsed,
		CreatedAt:    timeToUnix(m.CreatedAt),
		UpdatedAt:    timeToUnix(m.UpdatedAt),
	}
}

// ========== AITask 转换 ==========

// AITaskToModel 将 domain.AITask 转换为 MySQL AITask 模型
func AITaskToModel(d *domain.AITask) *AITask {
	if d == nil {
		return nil
	}
	var startedAt, completedAt *time.Time
	if d.StartedAt != nil && *d.StartedAt != 0 {
		t := unixToTime(*d.StartedAt)
		startedAt = &t
	}
	if d.CompletedAt != nil && *d.CompletedAt != 0 {
		t := unixToTime(*d.CompletedAt)
		completedAt = &t
	}
	return &AITask{
		ID:                d.ID,
		UserID:            d.UserID,
		Type:              string(d.Type),
		Status:            string(d.Status),
		Provider:          d.Provider,
		Model:             d.Model,
		Input:             d.Input,
		Output:            d.Output,
		TokensUsed:        d.TokensUsed,
		Progress:          d.Progress,
		ErrorMessage:      d.ErrorMessage,
		RelatedEntityID:   d.RelatedEntityID,
		RelatedEntityType: d.RelatedEntityType,
		CreatedAt:         unixToTime(d.CreatedAt),
		StartedAt:         startedAt,
		CompletedAt:       completedAt,
		UpdatedAt:         unixToTime(d.UpdatedAt),
	}
}

// ModelToAITask 将 MySQL AITask 模型转换为 domain.AITask
func ModelToAITask(m *AITask) *domain.AITask {
	if m == nil {
		return nil
	}
	var startedAt, completedAt *int64
	if m.StartedAt != nil && !m.StartedAt.IsZero() {
		unix := timeToUnix(*m.StartedAt)
		startedAt = &unix
	}
	if m.CompletedAt != nil && !m.CompletedAt.IsZero() {
		unix := timeToUnix(*m.CompletedAt)
		completedAt = &unix
	}
	return &domain.AITask{
		ID:                m.ID,
		UserID:            m.UserID,
		Type:              domain.AITaskType(m.Type),
		Status:            domain.AITaskStatus(m.Status),
		Provider:          m.Provider,
		Model:             m.Model,
		Input:             m.Input,
		Output:            m.Output,
		TokensUsed:        m.TokensUsed,
		Progress:          m.Progress,
		ErrorMessage:      m.ErrorMessage,
		RelatedEntityID:   m.RelatedEntityID,
		RelatedEntityType: m.RelatedEntityType,
		CreatedAt:         timeToUnix(m.CreatedAt),
		StartedAt:         startedAt,
		CompletedAt:       completedAt,
		UpdatedAt:         timeToUnix(m.UpdatedAt),
	}
}

// ========== AIGenerationRecord 转换 ==========

// AIGenerationRecordToModel 将 domain 模型转换为 MySQL 模型
func AIGenerationRecordToModel(d *domain.AIGenerationRecord) *AIGenerationRecord {
	if d == nil {
		return nil
	}

	// 转换 Metadata
	var metadataJSON string
	if d.Metadata != nil {
		if data, err := json.Marshal(d.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	m := &AIGenerationRecord{
		ID:       d.ID,
		UserID:   d.UserID,
		Type:     d.Type,
		Provider: d.Provider,
		Model:    d.Model,

		// 提示词信息
		OriginalPrompt: d.OriginalPrompt,
		EnhancedPrompt: d.EnhancedPrompt,
		SystemPrompt:   d.SystemPrompt,
		InputParams:    d.InputParams,
		OutputResult:   d.OutputResult,

		// Token 消耗统计
		InputTokens:  d.InputTokens,
		OutputTokens: d.OutputTokens,
		TotalTokens:  d.TotalTokens,
		ImageCount:   d.ImageCount,
		VideoCount:   d.VideoCount,

		// 任务状态
		Status:       string(d.Status),
		Progress:     d.Progress,
		ErrorMessage: d.ErrorMessage,
		ErrorCode:    d.ErrorCode,

		// 时间统计
		DurationMs:    d.DurationMs,
		QueueTimeMs:   d.QueueTimeMs,
		ProcessTimeMs: d.ProcessTimeMs,

		// 关联的业务实体
		RelatedEntityID:   d.RelatedEntityID,
		RelatedEntityType: d.RelatedEntityType,

		// 时间戳
		CreatedAt: unixToTime(d.CreatedAt),

		// 扩展元数据
		Metadata: metadataJSON,
	}

	// 转换可选时间戳
	if d.StartedAt != nil && *d.StartedAt != 0 {
		t := unixToTime(*d.StartedAt)
		m.StartedAt = &t
	}
	if d.CompletedAt != nil && *d.CompletedAt != 0 {
		t := unixToTime(*d.CompletedAt)
		m.CompletedAt = &t
	}

	return m
}

// ModelToAIGenerationRecord 将 MySQL 模型转换为 domain 模型
func ModelToAIGenerationRecord(m *AIGenerationRecord) *domain.AIGenerationRecord {
	if m == nil {
		return nil
	}

	// 解析 Metadata
	var metadata map[string]interface{}
	if m.Metadata != "" {
		_ = json.Unmarshal([]byte(m.Metadata), &metadata)
	}

	d := &domain.AIGenerationRecord{
		ID:       m.ID,
		UserID:   m.UserID,
		Type:     m.Type,
		Provider: m.Provider,
		Model:    m.Model,

		// 提示词信息
		OriginalPrompt: m.OriginalPrompt,
		EnhancedPrompt: m.EnhancedPrompt,
		SystemPrompt:   m.SystemPrompt,
		InputParams:    m.InputParams,
		OutputResult:   m.OutputResult,

		// Token 消耗统计
		InputTokens:  m.InputTokens,
		OutputTokens: m.OutputTokens,
		TotalTokens:  m.TotalTokens,
		ImageCount:   m.ImageCount,
		VideoCount:   m.VideoCount,

		// 任务状态
		Status:       domain.AITaskStatus(m.Status),
		Progress:     m.Progress,
		ErrorMessage: m.ErrorMessage,
		ErrorCode:    m.ErrorCode,

		// 时间统计
		DurationMs:    m.DurationMs,
		QueueTimeMs:   m.QueueTimeMs,
		ProcessTimeMs: m.ProcessTimeMs,

		// 关联的业务实体
		RelatedEntityID:   m.RelatedEntityID,
		RelatedEntityType: m.RelatedEntityType,

		// 时间戳
		CreatedAt: timeToUnix(m.CreatedAt),

		// 扩展元数据
		Metadata: metadata,
	}

	// 转换可选时间戳
	if m.StartedAt != nil && !m.StartedAt.IsZero() {
		unix := timeToUnix(*m.StartedAt)
		d.StartedAt = &unix
	}
	if m.CompletedAt != nil && !m.CompletedAt.IsZero() {
		unix := timeToUnix(*m.CompletedAt)
		d.CompletedAt = &unix
	}

	// Relations
	if m.User.ID != "" {
		d.User = ModelToUser(&m.User)
	}

	return d
}

// ========== RenderTask 转换 ==========

// RenderTaskToModel 将 domain.RenderTask 转换为 MySQL RenderTask 模型
func RenderTaskToModel(d *domain.RenderTask) *RenderTask {
	if d == nil {
		return nil
	}
	var startedAt, completedAt *time.Time
	if d.StartedAt != nil && *d.StartedAt != 0 {
		t := unixToTime(*d.StartedAt)
		startedAt = &t
	}
	if d.CompletedAt != nil && *d.CompletedAt != 0 {
		t := unixToTime(*d.CompletedAt)
		completedAt = &t
	}
	return &RenderTask{
		ID:           d.ID,
		UserID:       d.UserID,
		StoryID:      d.StoryID,
		Type:         string(d.Type),
		Status:       string(d.Status),
		Config:       d.Config,
		OutputURL:    d.OutputURL,
		ThumbnailURL: d.ThumbnailURL,
		Progress:     d.Progress,
		ErrorMessage: d.ErrorMessage,
		FileSize:     d.FileSize,
		Duration:     d.Duration,
		Resolution:   d.Resolution,
		CreatedAt:    unixToTime(d.CreatedAt),
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
		UpdatedAt:    unixToTime(d.UpdatedAt),
	}
}

// ModelToRenderTask 将 MySQL RenderTask 模型转换为 domain.RenderTask
func ModelToRenderTask(m *RenderTask) *domain.RenderTask {
	if m == nil {
		return nil
	}
	var startedAt, completedAt *int64
	if m.StartedAt != nil && !m.StartedAt.IsZero() {
		unix := timeToUnix(*m.StartedAt)
		startedAt = &unix
	}
	if m.CompletedAt != nil && !m.CompletedAt.IsZero() {
		unix := timeToUnix(*m.CompletedAt)
		completedAt = &unix
	}
	return &domain.RenderTask{
		ID:           m.ID,
		UserID:       m.UserID,
		StoryID:      m.StoryID,
		Type:         domain.RenderTaskType(m.Type),
		Status:       domain.RenderTaskStatus(m.Status),
		Config:       m.Config,
		OutputURL:    m.OutputURL,
		ThumbnailURL: m.ThumbnailURL,
		Progress:     m.Progress,
		ErrorMessage: m.ErrorMessage,
		FileSize:     m.FileSize,
		Duration:     m.Duration,
		Resolution:   m.Resolution,
		CreatedAt:    timeToUnix(m.CreatedAt),
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
		UpdatedAt:    timeToUnix(m.UpdatedAt),
	}
}

// ========== Tag 转换 ==========

// TagToModel 将 domain.Tag 转换为 MySQL Tag 模型
func TagToModel(d *domain.Tag) *Tag {
	if d == nil {
		return nil
	}
	return &Tag{
		ID:         d.ID,
		Name:       d.Name,
		Category:   d.Category,
		UsageCount: d.UsageCount,
		CreatedAt:  unixToTime(d.CreatedAt),
	}
}

// ModelToTag 将 MySQL Tag 模型转换为 domain.Tag
func ModelToTag(m *Tag) *domain.Tag {
	if m == nil {
		return nil
	}
	return &domain.Tag{
		ID:         m.ID,
		Name:       m.Name,
		Category:   m.Category,
		UsageCount: m.UsageCount,
		CreatedAt:  timeToUnix(m.CreatedAt),
	}
}

// StoryTagToModel 将 domain.StoryTag 转换为 MySQL StoryTag 模型
func StoryTagToModel(d *domain.StoryTag) *StoryTag {
	if d == nil {
		return nil
	}
	return &StoryTag{
		ID:        d.ID,
		StoryID:   d.StoryID,
		TagID:     d.TagID,
		CreatedAt: unixToTime(d.CreatedAt),
	}
}

// ModelToStoryTag 将 MySQL StoryTag 模型转换为 domain.StoryTag
func ModelToStoryTag(m *StoryTag) *domain.StoryTag {
	if m == nil {
		return nil
	}
	return &domain.StoryTag{
		ID:        m.ID,
		StoryID:   m.StoryID,
		TagID:     m.TagID,
		CreatedAt: timeToUnix(m.CreatedAt),
	}
}

// CharacterTagToModel 将 domain.CharacterTag 转换为 MySQL CharacterTag 模型
func CharacterTagToModel(d *domain.CharacterTag) *CharacterTag {
	if d == nil {
		return nil
	}
	return &CharacterTag{
		ID:          d.ID,
		CharacterID: d.CharacterID,
		TagID:       d.TagID,
		CreatedAt:   unixToTime(d.CreatedAt),
	}
}

// ModelToCharacterTag 将 MySQL CharacterTag 模型转换为 domain.CharacterTag
func ModelToCharacterTag(m *CharacterTag) *domain.CharacterTag {
	if m == nil {
		return nil
	}
	return &domain.CharacterTag{
		ID:          m.ID,
		CharacterID: m.CharacterID,
		TagID:       m.TagID,
		CreatedAt:   timeToUnix(m.CreatedAt),
	}
}

// ========== StyleConfig 转换 ==========

// StyleConfigToModel 将 domain.StyleConfig 转换为 MySQL StyleConfig 模型
func StyleConfigToModel(d *domain.StyleConfig) *StyleConfig {
	if d == nil {
		return nil
	}
	return &StyleConfig{
		ID:             d.ID,
		Style:          d.Style,
		Description:    d.Description,
		SampleImageURL: d.SampleImageURL,
		GroupID:        d.GroupID,
		UserID:         d.UserID,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

// ModelToStyleConfig 将 MySQL StyleConfig 模型转换为 domain.StyleConfig
func ModelToStyleConfig(m *StyleConfig) *domain.StyleConfig {
	if m == nil {
		return nil
	}
	return &domain.StyleConfig{
		ID:             m.ID,
		Style:          m.Style,
		Description:    m.Description,
		SampleImageURL: m.SampleImageURL,
		GroupID:        m.GroupID,
		UserID:         m.UserID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// ========== SearchHistory 转换 ==========

// SearchHistoryToModel 将 domain.SearchHistory 转换为 MySQL SearchHistory 模型
func SearchHistoryToModel(d *domain.SearchHistory) *SearchHistory {
	if d == nil {
		return nil
	}
	return &SearchHistory{
		ID:          d.ID,
		UserID:      d.UserID,
		Query:       d.Query,
		Type:        d.Type,
		ResultCount: d.ResultCount,
		CreatedAt:   unixToTime(d.CreatedAt),
	}
}

// ModelToSearchHistory 将 MySQL SearchHistory 模型转换为 domain.SearchHistory
func ModelToSearchHistory(m *SearchHistory) *domain.SearchHistory {
	if m == nil {
		return nil
	}
	return &domain.SearchHistory{
		ID:          m.ID,
		UserID:      m.UserID,
		Query:       m.Query,
		Type:        m.Type,
		ResultCount: m.ResultCount,
		CreatedAt:   timeToUnix(m.CreatedAt),
	}
}

// ========== ViewHistory 转换 ==========

// ViewHistoryToModel 将 domain.ViewHistory 转换为 MySQL ViewHistory 模型
func ViewHistoryToModel(d *domain.ViewHistory) *ViewHistory {
	if d == nil {
		return nil
	}
	return &ViewHistory{
		ID:         d.ID,
		UserID:     d.UserID,
		EntityType: d.EntityType,
		EntityID:   d.EntityID,
		Duration:   d.Duration,
		ViewedAt:   unixToTime(d.ViewedAt),
	}
}

// ModelToViewHistory 将 MySQL ViewHistory 模型转换为 domain.ViewHistory
func ModelToViewHistory(m *ViewHistory) *domain.ViewHistory {
	if m == nil {
		return nil
	}
	return &domain.ViewHistory{
		ID:         m.ID,
		UserID:     m.UserID,
		EntityType: m.EntityType,
		EntityID:   m.EntityID,
		Duration:   m.Duration,
		ViewedAt:   timeToUnix(m.ViewedAt),
	}
}

// ========== Report 转换 ==========

// ReportToModel 将 domain.Report 转换为 MySQL Report 模型
func ReportToModel(d *domain.Report) *Report {
	if d == nil {
		return nil
	}
	var reviewedAt *time.Time
	if d.ReviewedAt != nil && *d.ReviewedAt != 0 {
		t := unixToTime(*d.ReviewedAt)
		reviewedAt = &t
	}
	return &Report{
		ID:          d.ID,
		ReporterID:  d.ReporterID,
		EntityType:  d.EntityType,
		EntityID:    d.EntityID,
		Reason:      d.Reason,
		Description: d.Description,
		Status:      d.Status,
		ReviewerID:  d.ReviewerID,
		ReviewNote:  d.ReviewNote,
		CreatedAt:   unixToTime(d.CreatedAt),
		ReviewedAt:  reviewedAt,
	}
}

// ModelToReport 将 MySQL Report 模型转换为 domain.Report
func ModelToReport(m *Report) *domain.Report {
	if m == nil {
		return nil
	}
	var reviewedAt *int64
	if m.ReviewedAt != nil && !m.ReviewedAt.IsZero() {
		unix := timeToUnix(*m.ReviewedAt)
		reviewedAt = &unix
	}
	return &domain.Report{
		ID:          m.ID,
		ReporterID:  m.ReporterID,
		EntityType:  m.EntityType,
		EntityID:    m.EntityID,
		Reason:      m.Reason,
		Description: m.Description,
		Status:      m.Status,
		ReviewerID:  m.ReviewerID,
		ReviewNote:  m.ReviewNote,
		CreatedAt:   timeToUnix(m.CreatedAt),
		ReviewedAt:  reviewedAt,
	}
}

// ========== UserActivity 转换 ==========

// UserActivityToModel 将 domain.UserActivity 转换为 MySQL UserActivity 模型
func UserActivityToModel(d *domain.UserActivity) *UserActivity {
	if d == nil {
		return nil
	}
	return &UserActivity{
		ID:          d.ID,
		UserID:      d.UserID,
		Type:        d.Type,
		TargetID:    d.TargetID,
		TargetType:  d.TargetType,
		TargetTitle: d.TargetTitle,
		Message:     d.Message,
		CreatedAt:   unixToTime(d.CreatedAt),
	}
}

// ModelToUserActivity 将 MySQL UserActivity 模型转换为 domain.UserActivity
func ModelToUserActivity(m *UserActivity) *domain.UserActivity {
	if m == nil {
		return nil
	}
	return &domain.UserActivity{
		ID:          m.ID,
		UserID:      m.UserID,
		Type:        m.Type,
		TargetID:    m.TargetID,
		TargetType:  m.TargetType,
		TargetTitle: m.TargetTitle,
		Message:     m.Message,
		CreatedAt:   timeToUnix(m.CreatedAt),
	}
}

// ========== CharacterPoster 转换 ==========

// CharacterPosterToModel 将 domain.CharacterPoster 转换为 MySQL CharacterPoster 模型
func CharacterPosterToModel(d *domain.CharacterPoster) *CharacterPoster {
	if d == nil {
		return nil
	}
	// Serialize PosterConcept to JSON if present
	posterConceptJSON := d.PosterConceptJSON
	if posterConceptJSON == "" && d.PosterConcept != nil {
		posterConceptJSON = posterConceptDetailsToJSON(d.PosterConcept)
	}

	return &CharacterPoster{
		ID:                    d.ID,
		CharacterID:           d.CharacterID,
		AuthorID:              d.AuthorID,
		Type:                  d.Type,
		Title:                 d.Title,
		Image:                 d.Image,
		Video:                 d.Video,
		Thumbnail:             d.Thumbnail,
		Duration:              d.Duration,
		Prompt:                d.Prompt,
		Status:                string(d.Status),
		ReferenceStoryEnabled: d.ReferenceStoryEnabled,
		PosterConceptJSON:     posterConceptJSON,
		FinalImagePrompt:      d.FinalImagePrompt,
		ErrorMessage:          d.ErrorMessage,
		ConceptGenerationID:   d.ConceptGenerationID,
		ImageGenerationID:     d.ImageGenerationID,
		Likes:                 d.Likes,
		Shares:                d.Shares,
		CreatedAt:             unixToTime(d.CreatedAt),
	}
}

// ModelToCharacterPoster 将 MySQL CharacterPoster 模型转换为 domain.CharacterPoster
func ModelToCharacterPoster(m *CharacterPoster) *domain.CharacterPoster {
	if m == nil {
		return nil
	}
	// Parse PosterConcept from JSON if present
	var posterConcept *domain.PosterConceptDetails
	if m.PosterConceptJSON != "" {
		posterConcept = jsonToPosterConceptDetails(m.PosterConceptJSON)
	}

	return &domain.CharacterPoster{
		ID:                    m.ID,
		CharacterID:           m.CharacterID,
		AuthorID:              m.AuthorID,
		Type:                  m.Type,
		Title:                 m.Title,
		Image:                 m.Image,
		Video:                 m.Video,
		Thumbnail:             m.Thumbnail,
		Duration:              m.Duration,
		Prompt:                m.Prompt,
		Status:                domain.PosterStatus(m.Status),
		ReferenceStoryEnabled: m.ReferenceStoryEnabled,
		PosterConceptJSON:     m.PosterConceptJSON,
		PosterConcept:         posterConcept,
		FinalImagePrompt:      m.FinalImagePrompt,
		ErrorMessage:          m.ErrorMessage,
		ConceptGenerationID:   m.ConceptGenerationID,
		ImageGenerationID:     m.ImageGenerationID,
		Likes:                 m.Likes,
		Shares:                m.Shares,
		CreatedAt:             timeToUnix(m.CreatedAt),
	}
}

// ========== CharacterAnalytics 转换 ==========

// CharacterAnalyticsToModel 将 domain.CharacterAnalytics 转换为 MySQL CharacterAnalytics 模型
func CharacterAnalyticsToModel(d *domain.CharacterAnalytics) *CharacterAnalytics {
	if d == nil {
		return nil
	}
	return &CharacterAnalytics{
		ID:                   d.ID,
		CharacterID:          d.CharacterID,
		UsersWhoChattedCount: d.UsersWhoChattedCount,
		TotalMessagesSent:    d.TotalMessagesSent,
		TotalTokensConsumed:  d.TotalTokensConsumed,
		UpdatedAt:            unixToTime(d.UpdatedAt),
	}
}

// ModelToCharacterAnalytics 将 MySQL CharacterAnalytics 模型转换为 domain.CharacterAnalytics
func ModelToCharacterAnalytics(m *CharacterAnalytics) *domain.CharacterAnalytics {
	if m == nil {
		return nil
	}
	return &domain.CharacterAnalytics{
		ID:                   m.ID,
		CharacterID:          m.CharacterID,
		UsersWhoChattedCount: m.UsersWhoChattedCount,
		TotalMessagesSent:    m.TotalMessagesSent,
		TotalTokensConsumed:  m.TotalTokensConsumed,
		UpdatedAt:            timeToUnix(m.UpdatedAt),
	}
}

// ========== GroupActivity 转换 ==========

// GroupActivityToModel 将 domain.GroupActivity 转换为 MySQL GroupActivity 模型
func GroupActivityToModel(d *domain.GroupActivity) *GroupActivity {
	if d == nil {
		return nil
	}
	return &GroupActivity{
		ID:        d.ID,
		GroupID:   d.GroupID,
		Type:      d.Type,
		UserID:    d.UserID,
		StoryID:   stringToStringPtr(d.StoryID),
		Message:   d.Message,
		CreatedAt: unixToTime(d.Timestamp),
	}
}

// ModelToGroupActivity 将 MySQL GroupActivity 模型转换为 domain.GroupActivity
func ModelToGroupActivity(m *GroupActivity) *domain.GroupActivity {
	if m == nil {
		return nil
	}
	return &domain.GroupActivity{
		ID:         m.ID,
		GroupID:    m.GroupID,
		Type:       m.Type,
		UserID:     m.UserID,
		UserName:   m.User.DisplayName,
		UserAvatar: m.User.Avatar,
		StoryID:    stringPtrToString(m.StoryID),
		StoryTitle: getStoryTitle(m.Story),
		Message:    m.Message,
		Timestamp:  timeToUnix(m.CreatedAt),
	}
}

// getStoryTitle 安全获取 Story 标题
func getStoryTitle(story *Story) string {
	if story != nil {
		return story.Title
	}
	return ""
}

// ========== 批量转换函数 ==========

// ModelsToUsers 批量转换 User
func ModelsToUsers(models []User) []*domain.User {
	users := make([]*domain.User, len(models))
	for i := range models {
		users[i] = ModelToUser(&models[i])
	}
	return users
}

// ModelsToStories 批量转换 Story
func ModelsToStories(models []Story) []*domain.Story {
	stories := make([]*domain.Story, len(models))
	for i := range models {
		stories[i] = ModelToStory(&models[i])
	}
	return stories
}

// ModelsToCharacters 批量转换 Character
func ModelsToCharacters(models []Character) []*domain.Character {
	characters := make([]*domain.Character, len(models))
	for i := range models {
		characters[i] = ModelToCharacter(&models[i])
	}
	return characters
}

// ModelsToGroups 批量转换 Group
func ModelsToGroups(models []Group) []*domain.Group {
	groups := make([]*domain.Group, len(models))
	for i := range models {
		groups[i] = ModelToGroup(&models[i])
	}
	return groups
}

// ModelsToComments 批量转换 Comment
func ModelsToComments(models []Comment) []*domain.Comment {
	comments := make([]*domain.Comment, len(models))
	for i := range models {
		comments[i] = ModelToComment(&models[i])
	}
	return comments
}

// ModelsToStoryboards 批量转换 Storyboard
func ModelsToStoryboards(models []Storyboard) []*domain.Storyboard {
	storyboards := make([]*domain.Storyboard, len(models))
	for i := range models {
		storyboards[i] = ModelToStoryboard(&models[i])
	}
	return storyboards
}

// ModelsToChatThreads 批量转换 ChatThread
func ModelsToChatThreads(models []ChatThread) []*domain.ChatThread {
	threads := make([]*domain.ChatThread, len(models))
	for i := range models {
		threads[i] = ModelToChatThread(&models[i])
	}
	return threads
}

// ModelsToChatMessages 批量转换 ChatMessage
func ModelsToChatMessages(models []ChatMessage) []*domain.ChatMessage {
	messages := make([]*domain.ChatMessage, len(models))
	for i := range models {
		messages[i] = ModelToChatMessage(&models[i])
	}
	return messages
}

// ModelsToNotifications 批量转换 Notification
func ModelsToNotifications(models []Notification) []*domain.Notification {
	notifications := make([]*domain.Notification, len(models))
	for i := range models {
		notifications[i] = ModelToNotification(&models[i])
	}
	return notifications
}

// ModelsToAssets 批量转换 Asset
func ModelsToAssets(models []Asset) []*domain.Asset {
	assets := make([]*domain.Asset, len(models))
	for i := range models {
		assets[i] = ModelToAsset(&models[i])
	}
	return assets
}

// ModelsToTags 批量转换 Tag
func ModelsToTags(models []Tag) []*domain.Tag {
	tags := make([]*domain.Tag, len(models))
	for i := range models {
		tags[i] = ModelToTag(&models[i])
	}
	return tags
}

// ModelsToUserActivities 批量转换 UserActivity
func ModelsToUserActivities(models []UserActivity) []*domain.UserActivity {
	activities := make([]*domain.UserActivity, len(models))
	for i := range models {
		activities[i] = ModelToUserActivity(&models[i])
	}
	return activities
}

// ModelsToGroupActivities 批量转换 GroupActivity
func ModelsToGroupActivities(models []GroupActivity) []*domain.GroupActivity {
	activities := make([]*domain.GroupActivity, len(models))
	for i := range models {
		activities[i] = ModelToGroupActivity(&models[i])
	}
	return activities
}

// ========== Agent 转换 ==========

// AgentToModel 将 domain.Agent 转换为 MySQL Agent 模型
func AgentToModel(d *domain.Agent) *Agent {
	if d == nil {
		return nil
	}
	// Default to empty JSON object for MySQL JSON column
	configJSON := d.ConfigJSON
	if configJSON == "" {
		configJSON = "{}"
	}
	return &Agent{
		ID:               d.ID,
		CharacterID:      d.CharacterID,
		Name:             d.Name,
		Description:      d.Description,
		Status:           string(d.Status),
		SystemPrompt:     d.SystemPrompt,
		Temperature:      d.Temperature,
		Provider:         d.Provider,
		Model:            d.Model,
		MaxTokens:        d.MaxTokens,
		InteractionCount: d.InteractionCount,
		SkillCount:       d.SkillCount,
		Config:           configJSON,
		CreatedAt:        unixToTime(d.CreatedAt),
		UpdatedAt:        unixToTime(d.UpdatedAt),
	}
}

// ModelToAgent 将 MySQL Agent 模型转换为 domain.Agent
func ModelToAgent(m *Agent) *domain.Agent {
	if m == nil {
		return nil
	}
	agent := &domain.Agent{
		ID:               m.ID,
		CharacterID:      m.CharacterID,
		Name:             m.Name,
		Description:      m.Description,
		Status:           domain.AgentStatus(m.Status),
		SystemPrompt:     m.SystemPrompt,
		Temperature:      m.Temperature,
		Provider:         m.Provider,
		Model:            m.Model,
		MaxTokens:        m.MaxTokens,
		InteractionCount: m.InteractionCount,
		SkillCount:       m.SkillCount,
		ConfigJSON:       m.Config,
		CreatedAt:        timeToUnix(m.CreatedAt),
		UpdatedAt:        timeToUnix(m.UpdatedAt),
	}

	// Relations
	if m.Character.ID != "" {
		agent.Character = ModelToCharacter(&m.Character)
	}

	return agent
}

// ModelsToAgents 批量转换 Agent
func ModelsToAgents(models []Agent) []*domain.Agent {
	agents := make([]*domain.Agent, len(models))
	for i := range models {
		agents[i] = ModelToAgent(&models[i])
	}
	return agents
}

// ========== AgentSkill 转换 ==========

// AgentSkillToModel 将 domain.AgentSkill 转换为 MySQL AgentSkill 模型
func AgentSkillToModel(d *domain.AgentSkill) *AgentSkill {
	if d == nil {
		return nil
	}
	return &AgentSkill{
		ID:               d.ID,
		AgentID:          d.AgentID,
		Name:             d.Name,
		DisplayName:      d.DisplayName,
		Description:      d.Description,
		Type:             string(d.Type),
		Status:           string(d.Status),
		Instructions:     d.Instructions,
		Examples:         d.ExamplesJSON,
		Guidelines:       d.GuidelinesJSON,
		Metadata:         d.MetadataJSON,
		UsageCount:       d.UsageCount,
		SuccessCount:     d.SuccessCount,
		FailureCount:     d.FailureCount,
		AvgExecutionTime: d.AvgExecutionTime,
		Priority:         d.Priority,
		Enabled:          d.Enabled,
		CreatedAt:        unixToTime(d.CreatedAt),
		UpdatedAt:        unixToTime(d.UpdatedAt),
	}
}

// ModelToAgentSkill 将 MySQL AgentSkill 模型转换为 domain.AgentSkill
func ModelToAgentSkill(m *AgentSkill) *domain.AgentSkill {
	if m == nil {
		return nil
	}
	skill := &domain.AgentSkill{
		ID:               m.ID,
		AgentID:          m.AgentID,
		Name:             m.Name,
		DisplayName:      m.DisplayName,
		Description:      m.Description,
		Type:             domain.SkillType(m.Type),
		Status:           domain.SkillStatus(m.Status),
		Instructions:     m.Instructions,
		ExamplesJSON:     m.Examples,
		GuidelinesJSON:   m.Guidelines,
		MetadataJSON:     m.Metadata,
		UsageCount:       m.UsageCount,
		SuccessCount:     m.SuccessCount,
		FailureCount:     m.FailureCount,
		AvgExecutionTime: m.AvgExecutionTime,
		Priority:         m.Priority,
		Enabled:          m.Enabled,
		CreatedAt:        timeToUnix(m.CreatedAt),
		UpdatedAt:        timeToUnix(m.UpdatedAt),
	}

	// Relations
	if m.Agent.ID != "" {
		skill.Agent = ModelToAgent(&m.Agent)
	}

	return skill
}

// ModelsToAgentSkills 批量转换 AgentSkill
func ModelsToAgentSkills(models []AgentSkill) []*domain.AgentSkill {
	skills := make([]*domain.AgentSkill, len(models))
	for i := range models {
		skills[i] = ModelToAgentSkill(&models[i])
	}
	return skills
}

// ========== AgentSkillUsage 转换 ==========

// AgentSkillUsageToModel 将 domain.AgentSkillUsage 转换为 MySQL AgentSkillUsage 模型
func AgentSkillUsageToModel(d *domain.AgentSkillUsage) *AgentSkillUsage {
	if d == nil {
		return nil
	}
	return &AgentSkillUsage{
		ID:             d.ID,
		AgentID:        d.AgentID,
		SkillID:        d.SkillID,
		UserID:         d.UserID,
		ConversationID: d.ConversationID,
		Scenario:       d.Scenario,
		InputData:      d.InputData,
		OutputData:     d.OutputData,
		Success:        d.Success,
		ErrorMessage:   d.ErrorMessage,
		ExecutionTime:  d.ExecutionTime,
		TokensUsed:     d.TokensUsed,
		CreatedAt:      unixToTime(d.CreatedAt),
	}
}

// ModelToAgentSkillUsage 将 MySQL AgentSkillUsage 模型转换为 domain.AgentSkillUsage
func ModelToAgentSkillUsage(m *AgentSkillUsage) *domain.AgentSkillUsage {
	if m == nil {
		return nil
	}
	usage := &domain.AgentSkillUsage{
		ID:             m.ID,
		AgentID:        m.AgentID,
		SkillID:        m.SkillID,
		UserID:         m.UserID,
		ConversationID: m.ConversationID,
		Scenario:       m.Scenario,
		InputData:      m.InputData,
		OutputData:     m.OutputData,
		Success:        m.Success,
		ErrorMessage:   m.ErrorMessage,
		ExecutionTime:  m.ExecutionTime,
		TokensUsed:     m.TokensUsed,
		CreatedAt:      timeToUnix(m.CreatedAt),
	}

	// Relations
	if m.Agent.ID != "" {
		usage.Agent = ModelToAgent(&m.Agent)
	}
	if m.Skill.ID != "" {
		usage.Skill = ModelToAgentSkill(&m.Skill)
	}
	if m.User != nil && m.User.ID != "" {
		usage.User = ModelToUser(m.User)
	}

	return usage
}

// ModelsToAgentSkillUsages 批量转换 AgentSkillUsage
func ModelsToAgentSkillUsages(models []AgentSkillUsage) []*domain.AgentSkillUsage {
	usages := make([]*domain.AgentSkillUsage, len(models))
	for i := range models {
		usages[i] = ModelToAgentSkillUsage(&models[i])
	}
	return usages
}

// ========== AgentInteraction 转换 ==========

// AgentInteractionToModel 将 domain.AgentInteraction 转换为 MySQL AgentInteraction 模型
func AgentInteractionToModel(d *domain.AgentInteraction) *AgentInteraction {
	if d == nil {
		return nil
	}
	return &AgentInteraction{
		ID:          d.ID,
		AgentID:     d.AgentID,
		UserID:      d.UserID,
		CharacterID: d.CharacterID,
		MessageID:   d.MessageID,
		Type:        d.Type,
		InputText:   d.InputText,
		OutputText:  d.OutputText,
		TokensUsed:  d.TokensUsed,
		Duration:    d.Duration,
		SkillsUsed:  d.SkillsUsed,
		Success:     d.Success,
		CreatedAt:   unixToTime(d.CreatedAt),
	}
}

// ModelToAgentInteraction 将 MySQL AgentInteraction 模型转换为 domain.AgentInteraction
func ModelToAgentInteraction(m *AgentInteraction) *domain.AgentInteraction {
	if m == nil {
		return nil
	}
	interaction := &domain.AgentInteraction{
		ID:          m.ID,
		AgentID:     m.AgentID,
		UserID:      m.UserID,
		CharacterID: m.CharacterID,
		MessageID:   m.MessageID,
		Type:        m.Type,
		InputText:   m.InputText,
		OutputText:  m.OutputText,
		TokensUsed:  m.TokensUsed,
		Duration:    m.Duration,
		SkillsUsed:  m.SkillsUsed,
		Success:     m.Success,
		CreatedAt:   timeToUnix(m.CreatedAt),
	}

	// Relations
	if m.Agent.ID != "" {
		interaction.Agent = ModelToAgent(&m.Agent)
	}
	if m.User.ID != "" {
		interaction.User = ModelToUser(&m.User)
	}
	if m.Character.ID != "" {
		interaction.Character = ModelToCharacter(&m.Character)
	}

	return interaction
}

// ModelsToAgentInteractions 批量转换 AgentInteraction
func ModelsToAgentInteractions(models []AgentInteraction) []*domain.AgentInteraction {
	interactions := make([]*domain.AgentInteraction, len(models))
	for i := range models {
		interactions[i] = ModelToAgentInteraction(&models[i])
	}
	return interactions
}

// ========== AgentMemory 转换 ==========

// AgentMemoryToModel 将 domain.AgentMemory 转换为 MySQL AgentMemory 模型
func AgentMemoryToModel(d *domain.AgentMemory) *AgentMemory {
	if d == nil {
		return nil
	}
	memory := &AgentMemory{
		ID:          d.ID,
		AgentID:     d.AgentID,
		UserID:      d.UserID,
		MemoryType:  d.MemoryType,
		Key:         d.Key,
		Value:       d.Value,
		Importance:  d.Importance,
		AccessCount: d.AccessCount,
		CreatedAt:   unixToTime(d.CreatedAt),
		UpdatedAt:   unixToTime(d.UpdatedAt),
	}

	if d.LastAccessed != nil {
		t := unixToTime(*d.LastAccessed)
		memory.LastAccessed = t
	}
	if d.ExpiresAt != nil {
		t := unixToTime(*d.ExpiresAt)
		memory.ExpiresAt = &t
	}

	return memory
}

// ModelToAgentMemory 将 MySQL AgentMemory 模型转换为 domain.AgentMemory
func ModelToAgentMemory(m *AgentMemory) *domain.AgentMemory {
	if m == nil {
		return nil
	}
	memory := &domain.AgentMemory{
		ID:          m.ID,
		AgentID:     m.AgentID,
		UserID:      m.UserID,
		MemoryType:  m.MemoryType,
		Key:         m.Key,
		Value:       m.Value,
		Importance:  m.Importance,
		AccessCount: m.AccessCount,
		CreatedAt:   timeToUnix(m.CreatedAt),
		UpdatedAt:   timeToUnix(m.UpdatedAt),
	}

	if !m.LastAccessed.IsZero() {
		t := timeToUnix(m.LastAccessed)
		memory.LastAccessed = &t
	}
	if m.ExpiresAt != nil && !m.ExpiresAt.IsZero() {
		t := timeToUnix(*m.ExpiresAt)
		memory.ExpiresAt = &t
	}

	// Relations
	if m.Agent.ID != "" {
		memory.Agent = ModelToAgent(&m.Agent)
	}
	if m.User.ID != "" {
		memory.User = ModelToUser(&m.User)
	}

	return memory
}

// ModelsToAgentMemories 批量转换 AgentMemory
func ModelsToAgentMemories(models []AgentMemory) []*domain.AgentMemory {
	memories := make([]*domain.AgentMemory, len(models))
	for i := range models {
		memories[i] = ModelToAgentMemory(&models[i])
	}
	return memories
}

// ========== StoryPublication 转换 ==========

// StoryPublicationToModel 将 domain.StoryPublication 转换为 MySQL StoryPublication 模型
func StoryPublicationToModel(d *domain.StoryPublication) *StoryPublication {
	if d == nil {
		return nil
	}
	pub := &StoryPublication{
		ID:           d.ID,
		StoryID:      d.StoryID,
		Version:      d.Version,
		Status:       d.Status,
		RenderTaskID: d.RenderTaskID,
		PublishedAt:  unixToTime(d.PublishedAt),
		UpdatedAt:    unixToTime(d.UpdatedAt),
	}

	if d.UnpublishedAt != nil {
		t := unixToTime(*d.UnpublishedAt)
		pub.UnpublishedAt = &t
	}

	return pub
}

// ModelToStoryPublication 将 MySQL StoryPublication 模型转换为 domain.StoryPublication
func ModelToStoryPublication(m *StoryPublication) *domain.StoryPublication {
	if m == nil {
		return nil
	}
	pub := &domain.StoryPublication{
		ID:           m.ID,
		StoryID:      m.StoryID,
		Version:      m.Version,
		Status:       m.Status,
		RenderTaskID: m.RenderTaskID,
		PublishedAt:  timeToUnix(m.PublishedAt),
		UpdatedAt:    timeToUnix(m.UpdatedAt),
	}

	if m.UnpublishedAt != nil && !m.UnpublishedAt.IsZero() {
		t := timeToUnix(*m.UnpublishedAt)
		pub.UnpublishedAt = &t
	}

	// Relations
	if m.Story.ID != "" {
		pub.Story = ModelToStory(&m.Story)
	}

	return pub
}

// ModelsToStoryPublications 批量转换 StoryPublication
func ModelsToStoryPublications(models []StoryPublication) []*domain.StoryPublication {
	pubs := make([]*domain.StoryPublication, len(models))
	for i := range models {
		pubs[i] = ModelToStoryPublication(&models[i])
	}
	return pubs
}

// ========== SubscriptionPlan 转换 ==========

// SubscriptionPlanToModel 将 domain.SubscriptionPlan 转换为 MySQL SubscriptionPlan 模型
func SubscriptionPlanToModel(d *domain.SubscriptionPlan) *SubscriptionPlan {
	if d == nil {
		return nil
	}
	return &SubscriptionPlan{
		ID:            d.ID,
		Name:          d.Name,
		Price:         d.Price,
		Currency:      d.Currency,
		TokenQuota:    d.TokenQuota,
		StorageQuota:  d.StorageQuota,
		MaxStories:    d.MaxStories,
		MaxCharacters: d.MaxCharacters,
		Features:      d.Features,
		IsActive:      d.IsActive,
		SortOrder:     d.SortOrder,
		CreatedAt:     unixToTime(d.CreatedAt),
		UpdatedAt:     unixToTime(d.UpdatedAt),
	}
}

// ModelToSubscriptionPlan 将 MySQL SubscriptionPlan 模型转换为 domain.SubscriptionPlan
func ModelToSubscriptionPlan(m *SubscriptionPlan) *domain.SubscriptionPlan {
	if m == nil {
		return nil
	}
	return &domain.SubscriptionPlan{
		ID:            m.ID,
		Name:          m.Name,
		Price:         m.Price,
		Currency:      m.Currency,
		TokenQuota:    m.TokenQuota,
		StorageQuota:  m.StorageQuota,
		MaxStories:    m.MaxStories,
		MaxCharacters: m.MaxCharacters,
		Features:      m.Features,
		IsActive:      m.IsActive,
		SortOrder:     m.SortOrder,
		CreatedAt:     timeToUnix(m.CreatedAt),
		UpdatedAt:     timeToUnix(m.UpdatedAt),
	}
}

// ModelsToSubscriptionPlans 批量转换 SubscriptionPlan
func ModelsToSubscriptionPlans(models []SubscriptionPlan) []*domain.SubscriptionPlan {
	plans := make([]*domain.SubscriptionPlan, len(models))
	for i := range models {
		plans[i] = ModelToSubscriptionPlan(&models[i])
	}
	return plans
}

// ========== SubscriptionOrder 转换 ==========

// SubscriptionOrderToModel 将 domain.SubscriptionOrder 转换为 MySQL SubscriptionOrder 模型
func SubscriptionOrderToModel(d *domain.SubscriptionOrder) *SubscriptionOrder {
	if d == nil {
		return nil
	}
	order := &SubscriptionOrder{
		ID:            d.ID,
		UserID:        d.UserID,
		PlanID:        d.PlanID,
		Status:        d.Status,
		Amount:        d.Amount,
		Currency:      d.Currency,
		PaymentMethod: d.PaymentMethod,
		PaymentID:     d.PaymentID,
		StartDate:     unixToTime(d.StartDate),
		EndDate:       unixToTime(d.EndDate),
		InvoiceURL:    d.InvoiceURL,
		CreatedAt:     unixToTime(d.CreatedAt),
		UpdatedAt:     unixToTime(d.UpdatedAt),
	}

	if d.PaidAt != nil {
		t := unixToTime(*d.PaidAt)
		order.PaidAt = &t
	}

	return order
}

// ModelToSubscriptionOrder 将 MySQL SubscriptionOrder 模型转换为 domain.SubscriptionOrder
func ModelToSubscriptionOrder(m *SubscriptionOrder) *domain.SubscriptionOrder {
	if m == nil {
		return nil
	}
	order := &domain.SubscriptionOrder{
		ID:            m.ID,
		UserID:        m.UserID,
		PlanID:        m.PlanID,
		Status:        m.Status,
		Amount:        m.Amount,
		Currency:      m.Currency,
		PaymentMethod: m.PaymentMethod,
		PaymentID:     m.PaymentID,
		StartDate:     timeToUnix(m.StartDate),
		EndDate:       timeToUnix(m.EndDate),
		InvoiceURL:    m.InvoiceURL,
		CreatedAt:     timeToUnix(m.CreatedAt),
		UpdatedAt:     timeToUnix(m.UpdatedAt),
	}

	if m.PaidAt != nil && !m.PaidAt.IsZero() {
		t := timeToUnix(*m.PaidAt)
		order.PaidAt = &t
	}

	// Relations
	if m.User.ID != "" {
		order.User = ModelToUser(&m.User)
	}
	if m.Plan.ID != "" {
		order.Plan = ModelToSubscriptionPlan(&m.Plan)
	}

	return order
}

// ModelsToSubscriptionOrders 批量转换 SubscriptionOrder
func ModelsToSubscriptionOrders(models []SubscriptionOrder) []*domain.SubscriptionOrder {
	orders := make([]*domain.SubscriptionOrder, len(models))
	for i := range models {
		orders[i] = ModelToSubscriptionOrder(&models[i])
	}
	return orders
}

// ========== TokenTransaction 转换 ==========

// TokenTransactionToModel 将 domain.TokenTransaction 转换为 MySQL TokenTransaction 模型
func TokenTransactionToModel(d *domain.TokenTransaction) *TokenTransaction {
	if d == nil {
		return nil
	}
	return &TokenTransaction{
		ID:          d.ID,
		UserID:      d.UserID,
		Type:        d.Type,
		Amount:      d.Amount,
		Balance:     d.Balance,
		Source:      d.Source,
		RelatedID:   d.RelatedID,
		Description: d.Description,
		CreatedAt:   unixToTime(d.CreatedAt),
	}
}

// ModelToTokenTransaction 将 MySQL TokenTransaction 模型转换为 domain.TokenTransaction
func ModelToTokenTransaction(m *TokenTransaction) *domain.TokenTransaction {
	if m == nil {
		return nil
	}
	tx := &domain.TokenTransaction{
		ID:          m.ID,
		UserID:      m.UserID,
		Type:        m.Type,
		Amount:      m.Amount,
		Balance:     m.Balance,
		Source:      m.Source,
		RelatedID:   m.RelatedID,
		Description: m.Description,
		CreatedAt:   timeToUnix(m.CreatedAt),
	}

	// Relations
	if m.User.ID != "" {
		tx.User = ModelToUser(&m.User)
	}

	return tx
}

// ModelsToTokenTransactions 批量转换 TokenTransaction
func ModelsToTokenTransactions(models []TokenTransaction) []*domain.TokenTransaction {
	txs := make([]*domain.TokenTransaction, len(models))
	for i := range models {
		txs[i] = ModelToTokenTransaction(&models[i])
	}
	return txs
}

// ========== StoryComposition 转换 ==========

// StoryCompositionToModel 将 domain.StoryComposition 转换为 MySQL StoryComposition 模型
func StoryCompositionToModel(d *domain.StoryComposition) *StoryComposition {
	if d == nil {
		return nil
	}
	return &StoryComposition{
		ID:               d.ID,
		Title:            d.Title,
		CoverImage:       d.CoverImage,
		Background:       d.BackgroundDescription,
		Theme:            d.Theme,
		Genre:            d.Genre,
		RootStoryboardID: d.RootStoryboardID,
		TotalStoryboards: d.TotalStoryboards,
		TotalForks:       d.TotalForks,
		CreatedAt:        unixToTime(d.CreatedAt),
		UpdatedAt:        unixToTime(d.UpdatedAt),
	}
}

// ModelToStoryComposition 将 MySQL StoryComposition 模型转换为 domain.StoryComposition
func ModelToStoryComposition(m *StoryComposition) *domain.StoryComposition {
	if m == nil {
		return nil
	}
	return &domain.StoryComposition{
		ID:                    m.ID,
		Title:                 m.Title,
		CoverImage:            m.CoverImage,
		BackgroundDescription: m.Background,
		Theme:                 m.Theme,
		Genre:                 m.Genre,
		RootStoryboardID:      m.RootStoryboardID,
		TotalStoryboards:      m.TotalStoryboards,
		TotalForks:            m.TotalForks,
		CreatedAt:             timeToUnix(m.CreatedAt),
		UpdatedAt:             timeToUnix(m.UpdatedAt),
	}
}

// promptDetailsToJSON 将 ImagePromptDetails 转换为 JSON 字符串
// 返回 "null" 而不是空字符串，因为 MySQL JSON 列不接受空字符串
func promptDetailsToJSON(details *domain.ImagePromptDetails) string {
	if details == nil {
		return "null"
	}
	data, err := json.Marshal(details)
	if err != nil {
		return "null"
	}
	return string(data)
}

// jsonToPromptDetails 将 JSON 字符串转换为 ImagePromptDetails
func jsonToPromptDetails(jsonStr string) *domain.ImagePromptDetails {
	if jsonStr == "" || jsonStr == "null" {
		return nil
	}
	var details domain.ImagePromptDetails
	if err := json.Unmarshal([]byte(jsonStr), &details); err != nil {
		return nil
	}
	return &details
}

// videoPromptDetailsToJSON 将 VideoPromptDetails 转换为 JSON 字符串
// 返回 "null" 而不是空字符串，因为 MySQL JSON 列不接受空字符串
func videoPromptDetailsToJSON(details *domain.VideoPromptDetails) string {
	if details == nil {
		return "null"
	}
	data, err := json.Marshal(details)
	if err != nil {
		return "null"
	}
	return string(data)
}

// jsonToVideoPromptDetails 将 JSON 字符串转换为 VideoPromptDetails
func jsonToVideoPromptDetails(jsonStr string) *domain.VideoPromptDetails {
	if jsonStr == "" || jsonStr == "null" {
		return nil
	}
	var details domain.VideoPromptDetails
	if err := json.Unmarshal([]byte(jsonStr), &details); err != nil {
		return nil
	}
	return &details
}

// posterConceptDetailsToJSON 将 PosterConceptDetails 转换为 JSON 字符串
func posterConceptDetailsToJSON(details *domain.PosterConceptDetails) string {
	if details == nil {
		return ""
	}
	data, err := json.Marshal(details)
	if err != nil {
		return ""
	}
	return string(data)
}

// jsonToPosterConceptDetails 将 JSON 字符串转换为 PosterConceptDetails
func jsonToPosterConceptDetails(jsonStr string) *domain.PosterConceptDetails {
	if jsonStr == "" {
		return nil
	}
	var details domain.PosterConceptDetails
	if err := json.Unmarshal([]byte(jsonStr), &details); err != nil {
		return nil
	}
	return &details
}

// ========== Group Blacklist 转换 ==========

// BlacklistToModel 将 domain.GroupBlacklist 转换为 MySQL GroupBlacklist 模型
func BlacklistToModel(d *domain.GroupBlacklist) *GroupBlacklist {
	if d == nil {
		return nil
	}
	return &GroupBlacklist{
		ID:        d.ID,
		GroupID:   d.GroupID,
		UserID:    d.UserID,
		BlockedBy: d.BlockedBy,
		Reason:    d.Reason,
		CreatedAt: unixToTime(d.CreatedAt),
	}
}

// ModelToGroupBlacklist 将 MySQL GroupBlacklist 模型转换为 domain.GroupBlacklist
func ModelToGroupBlacklist(m *GroupBlacklist) *domain.GroupBlacklist {
	if m == nil {
		return nil
	}
	d := &domain.GroupBlacklist{
		ID:        m.ID,
		GroupID:   m.GroupID,
		UserID:    m.UserID,
		BlockedBy: m.BlockedBy,
		Reason:    m.Reason,
		CreatedAt: timeToUnix(m.CreatedAt),
	}
	if m.Group.ID != "" {
		d.Group = ModelToGroup(&m.Group)
	}
	if m.User.ID != "" {
		d.User = ModelToUser(&m.User)
	}
	if m.Admin.ID != "" {
		d.Admin = ModelToUser(&m.Admin)
	}
	return d
}

// ModelToGroupBlacklistInfo 将 MySQL GroupBlacklist 模型转换为 domain.GroupBlacklistInfo（扩展信息）
func ModelToGroupBlacklistInfo(m *GroupBlacklist) *domain.GroupBlacklistInfo {
	if m == nil {
		return nil
	}
	d := &domain.GroupBlacklistInfo{
		ID:        m.ID,
		GroupID:   m.GroupID,
		UserID:    m.UserID,
		BlockedBy: m.BlockedBy,
		Reason:    m.Reason,
		CreatedAt: timeToUnix(m.CreatedAt),
	}
	if m.User.ID != "" {
		d.User = ModelToUser(&m.User)
	}
	if m.Admin.ID != "" {
		d.Admin = ModelToUser(&m.Admin)
	}
	return d
}

// ========== Writers Room Converters ==========

// ModelToWritersRoom 将 MySQL WritersRoomDB 模型转换为 domain.WritersRoom
func ModelToWritersRoom(m *WritersRoomDB) *domain.WritersRoom {
	if m == nil {
		return nil
	}
	return &domain.WritersRoom{
		ID:               m.ID,
		StoryID:          m.StoryID,
		Title:            m.Title,
		LastMessage:      m.LastMessage,
		LastMessageTime:  m.LastMessageTime,
		MessageCount:     m.MessageCount,
		ParticipantCount: m.ParticipantCount,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// WritersRoomDB.ToDomain 将 MySQL WritersRoomDB 模型转换为 domain.WritersRoom
func (m *WritersRoomDB) ToDomain() *domain.WritersRoom {
	return ModelToWritersRoom(m)
}

// ModelToWritersRoomParticipant 将 MySQL WritersRoomParticipantDB 模型转换为 domain.WritersRoomParticipant
func ModelToWritersRoomParticipant(m *WritersRoomParticipantDB) *domain.WritersRoomParticipant {
	if m == nil {
		return nil
	}
	d := &domain.WritersRoomParticipant{
		ID:         m.ID,
		RoomID:     m.RoomID,
		UserID:     m.UserID,
		Role:       m.Role,
		JoinedAt:   m.JoinedAt,
		LastReadAt: m.LastReadAt,
	}
	if m.User.ID != "" {
		d.User = ModelToUser(&m.User)
	}
	return d
}

// WritersRoomParticipantDB.ToDomain 将 MySQL WritersRoomParticipantDB 模型转换为 domain.WritersRoomParticipant
func (m *WritersRoomParticipantDB) ToDomain() *domain.WritersRoomParticipant {
	return ModelToWritersRoomParticipant(m)
}

// ModelToWritersRoomMessage 将 MySQL WritersRoomMessageDB 模型转换为 domain.WritersRoomMessage
func ModelToWritersRoomMessage(m *WritersRoomMessageDB) *domain.WritersRoomMessage {
	if m == nil {
		return nil
	}
	d := &domain.WritersRoomMessage{
		ID:          m.ID,
		RoomID:      m.RoomID,
		SenderID:    m.SenderID,
		Content:     m.Content,
		MessageType: m.MessageType,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}

	// Parse attachments JSON
	if m.AttachmentsJSON != "" {
		var attachments []string
		if err := json.Unmarshal([]byte(m.AttachmentsJSON), &attachments); err == nil {
			d.Attachments = attachments
		}
	}

	// Parse mentions JSON
	if m.MentionsJSON != "" {
		var mentions []string
		if err := json.Unmarshal([]byte(m.MentionsJSON), &mentions); err == nil {
			d.Mentions = mentions
		}
	}

	// Handle reply to message
	if m.ReplyToMessageID != nil && *m.ReplyToMessageID != "" {
		d.ReplyToMessageID = m.ReplyToMessageID
	}

	// Load sender info
	if m.Sender != nil {
		d.SenderName = m.Sender.User.DisplayName
		d.SenderAvatar = m.Sender.User.Avatar
		d.Sender = ModelToWritersRoomParticipant(m.Sender)
	}

	return d
}

// WritersRoomMessageDB.ToDomain 将 MySQL WritersRoomMessageDB 模型转换为 domain.WritersRoomMessage
func (m *WritersRoomMessageDB) ToDomain() *domain.WritersRoomMessage {
	return ModelToWritersRoomMessage(m)
}

// ModelToWritersRoomMessageReaction 将 MySQL WritersRoomMessageReactionDB 模型转换为 domain.WritersRoomMessageReaction
func ModelToWritersRoomMessageReaction(m *WritersRoomMessageReactionDB) *domain.WritersRoomMessageReaction {
	if m == nil {
		return nil
	}
	d := &domain.WritersRoomMessageReaction{
		ID:           m.ID,
		MessageID:    m.MessageID,
		UserID:       m.UserID,
		ReactionType: m.ReactionType,
		EmojiCode:    m.EmojiCode,
		CreatedAt:    m.CreatedAt,
	}
	if m.User.ID != "" {
		d.UserName = m.User.DisplayName
		d.User = ModelToUser(&m.User)
	}
	return d
}

// WritersRoomMessageReactionDB.ToDomain 将 MySQL WritersRoomMessageReactionDB 模型转换为 domain.WritersRoomMessageReaction
func (m *WritersRoomMessageReactionDB) ToDomain() *domain.WritersRoomMessageReaction {
	return ModelToWritersRoomMessageReaction(m)
}

// ModelToMessageReadReceipt 将 MySQL MessageReadReceiptDB 模型转换为 domain.MessageReadReceipt
func ModelToMessageReadReceipt(m *MessageReadReceiptDB) *domain.MessageReadReceipt {
	if m == nil {
		return nil
	}
	d := &domain.MessageReadReceipt{
		ID:        m.ID,
		MessageID: m.MessageID,
		UserID:    m.UserID,
		ReadAt:    m.ReadAt,
	}
	if m.User.ID != "" {
		d.UserName = m.User.DisplayName
		d.User = ModelToUser(&m.User)
	}
	return d
}

// MessageReadReceiptDB.ToDomain 将 MySQL MessageReadReceiptDB 模型转换为 domain.MessageReadReceipt
func (m *MessageReadReceiptDB) ToDomain() *domain.MessageReadReceipt {
	return ModelToMessageReadReceipt(m)
}
