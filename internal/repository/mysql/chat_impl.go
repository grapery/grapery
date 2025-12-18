package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ========== ChatThread Methods ==========

func (r *Repository) ChatThreadByID(ctx context.Context, id string) (*domain.ChatThread, error) {
	var thread ChatThread
	if err := r.db.WithContext(ctx).
		Preload("Character").
		Preload("Character.Author").
		Preload("User").
		First(&thread, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat thread: %w", err)
	}
	dt := r.chatThreadToDomain(thread)
	return &dt, nil
}

func (r *Repository) ChatThreads(ctx context.Context, userID string) ([]*domain.ChatThread, error) {
	var threads []ChatThread
	if err := r.db.WithContext(ctx).
		Preload("Character").
		Preload("Character.Author").
		Where("user_id = ?", userID).
		Order("last_message_time DESC").
		Find(&threads).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat threads: %w", err)
	}

	result := make([]*domain.ChatThread, len(threads))
	for i, t := range threads {
		dt := r.chatThreadToDomain(t)
		result[i] = &dt
	}
	return result, nil
}

func (r *Repository) CreateChatThread(ctx context.Context, thread *domain.ChatThread) error {
	// Default to empty JSON array for MySQL JSON column
	contextStoryboardIDsJSON := "[]"
	if len(thread.ContextStoryboardIDs) > 0 {
		jsonBytes, err := json.Marshal(thread.ContextStoryboardIDs)
		if err != nil {
			return fmt.Errorf("failed to marshal context storyboard IDs: %w", err)
		}
		contextStoryboardIDsJSON = string(jsonBytes)
	}

	var selectedStoryboardID *string
	if thread.SelectedStoryboardID != "" {
		selectedStoryboardID = &thread.SelectedStoryboardID
	}

	// Use thread.UserID if provided, otherwise get from context
	userID := thread.UserID
	if userID == "" {
		userID = r.getCurrentUserID(ctx)
	}
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	dbThread := ChatThread{
		ID:                   uuid.New().String(),
		CharacterID:          thread.CharacterID,
		UserID:               userID,
		StoryTitle:           thread.StoryTitle,
		LastMessage:          thread.LastMessage,
		LastMessageTime:      time.Unix(thread.LastMessageTime, 0),
		UnreadCount:          thread.UnreadCount,
		MessageCount:         thread.MessageCount,
		InteractionFrequency: thread.InteractionFrequency,
		SelectedStoryboardID: selectedStoryboardID,
		ContextStoryboardIDs: contextStoryboardIDsJSON,
		TotalTokensUsed:      thread.TotalTokensUsed,
		IsArchived:           thread.IsArchived,
	}

	if err := r.db.WithContext(ctx).Create(&dbThread).Error; err != nil {
		return fmt.Errorf("failed to create chat thread: %w", err)
	}

	thread.ID = dbThread.ID
	thread.CreatedAt = dbThread.CreatedAt.Unix()
	return nil
}

func (r *Repository) UpdateChatThread(ctx context.Context, thread *domain.ChatThread) error {
	updates := map[string]interface{}{
		"last_message":          thread.LastMessage,
		"last_message_time":     time.Unix(thread.LastMessageTime, 0),
		"unread_count":          thread.UnreadCount,
		"message_count":         thread.MessageCount,
		"interaction_frequency": thread.InteractionFrequency,
		"total_tokens_used":     thread.TotalTokensUsed,
		"is_archived":           thread.IsArchived,
		"updated_at":            time.Now(),
	}

	if thread.SelectedStoryboardID != "" {
		updates["selected_storyboard_id"] = thread.SelectedStoryboardID
	}

	// Always set context_storyboard_ids to valid JSON
	if len(thread.ContextStoryboardIDs) > 0 {
		jsonBytes, err := json.Marshal(thread.ContextStoryboardIDs)
		if err != nil {
			return fmt.Errorf("failed to marshal context storyboard IDs: %w", err)
		}
		updates["context_storyboard_ids"] = string(jsonBytes)
	} else {
		updates["context_storyboard_ids"] = "[]"
	}

	// Update summary if provided
	if thread.Summary != "" {
		updates["summary"] = thread.Summary
	}

	if err := r.db.WithContext(ctx).Model(&ChatThread{}).
		Where("id = ?", thread.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update chat thread: %w", err)
	}
	return nil
}

func (r *Repository) DeleteChatThread(ctx context.Context, id string) error {
	// 先删除该thread下的所有消息
	if err := r.db.WithContext(ctx).Where("thread_id = ?", id).Delete(&ChatMessage{}).Error; err != nil {
		return fmt.Errorf("failed to delete chat messages: %w", err)
	}

	// 再删除thread本身
	if err := r.db.WithContext(ctx).Delete(&ChatThread{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete chat thread: %w", err)
	}
	return nil
}

// ========== ChatMessage Methods ==========

func (r *Repository) ChatMessageByID(ctx context.Context, id string) (*domain.ChatMessage, error) {
	var message ChatMessage
	if err := r.db.WithContext(ctx).First(&message, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat message: %w", err)
	}
	dm := r.chatMessageToDomain(message)
	
	// Load reactions
	reactions, _ := r.GetMessageReactions(ctx, id)
	dm.Reactions = reactions
	
	// Load token usage
	tokenUsage, _ := r.GetMessageTokenUsage(ctx, id)
	dm.TokenUsage = tokenUsage
	
	return &dm, nil
}

func (r *Repository) ChatMessages(ctx context.Context, threadID string, limit, offset int) ([]*domain.ChatMessage, error) {
	var messages []ChatMessage
	query := r.db.WithContext(ctx).
		Where("thread_id = ? AND is_archived = ?", threadID, false).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat messages: %w", err)
	}

	result := make([]*domain.ChatMessage, len(messages))
	for i, m := range messages {
		dm := r.chatMessageToDomain(m)
		
		// Load reactions (optional, can be lazy loaded)
		reactions, _ := r.GetMessageReactions(ctx, m.ID)
		dm.Reactions = reactions
		
		// Load token usage (optional, can be lazy loaded)
		tokenUsage, _ := r.GetMessageTokenUsage(ctx, m.ID)
		dm.TokenUsage = tokenUsage
		
		result[i] = &dm
	}
	return result, nil
}

func (r *Repository) CreateChatMessage(ctx context.Context, msg *domain.ChatMessage) error {
	dbMsg := ChatMessage{
		ID:           uuid.New().String(),
		ThreadID:     msg.ThreadID,
		SenderID:     msg.SenderID,
		SenderName:   msg.SenderName,
		SenderAvatar: msg.SenderAvatar,
		Content:      msg.Content,
		Image:        msg.Image,
		IsUser:       msg.IsUser,
		IsArchived:   msg.IsArchived,
		ReactionCount: 0,
	}

	if err := r.db.WithContext(ctx).Create(&dbMsg).Error; err != nil {
		return fmt.Errorf("failed to create chat message: %w", err)
	}

	// 更新thread的最后消息和消息计数
	// 使用 gorm.Expr 来实现自增操作，避免 SQL 注入
	updates := map[string]interface{}{
		"last_message":      msg.Content,
		"last_message_time": dbMsg.CreatedAt,
		"message_count":     gorm.Expr("message_count + 1"),
	}

	// 如果是角色发的消息（非用户），增加未读数
	if !msg.IsUser {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
	}

	if err := r.db.WithContext(ctx).Model(&ChatThread{}).
		Where("id = ?", msg.ThreadID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update chat thread: %w", err)
	}

	msg.ID = dbMsg.ID
	msg.Timestamp = dbMsg.CreatedAt.Unix()
	return nil
}

func (r *Repository) DeleteChatMessage(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&ChatMessage{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete chat message: %w", err)
	}
	return nil
}

// ========== Helper Methods ==========

func (r *Repository) chatThreadToDomain(t ChatThread) domain.ChatThread {
	var contextStoryboardIDs []string
	if t.ContextStoryboardIDs != "" {
		if err := json.Unmarshal([]byte(t.ContextStoryboardIDs), &contextStoryboardIDs); err != nil {
			// If unmarshal fails, use empty slice
			contextStoryboardIDs = []string{}
		}
	}

	var selectedStoryboardID string
	if t.SelectedStoryboardID != nil {
		selectedStoryboardID = *t.SelectedStoryboardID
	}

	return domain.ChatThread{
		ID:                   t.ID,
		UserID:               t.UserID,
		CharacterID:          t.CharacterID,
		CharacterName:        t.Character.Name,
		CharacterAvatar:      t.Character.Avatar,
		StoryTitle:           t.StoryTitle,
		LastMessage:          t.LastMessage,
		LastMessageTime:      t.LastMessageTime.Unix(),
		UnreadCount:          t.UnreadCount,
		MessageCount:         t.MessageCount,
		InteractionFrequency: t.InteractionFrequency,
		SelectedStoryboardID: selectedStoryboardID,
		ContextStoryboardIDs: contextStoryboardIDs,
		TotalTokensUsed:      t.TotalTokensUsed,
		IsArchived:           t.IsArchived,
		Summary:              t.Summary,
		CreatedAt:            t.CreatedAt.Unix(),
		UpdatedAt:            t.UpdatedAt.Unix(),
	}
}

func (r *Repository) chatMessageToDomain(m ChatMessage) domain.ChatMessage {
	return domain.ChatMessage{
		ID:           m.ID,
		ThreadID:     m.ThreadID,
		SenderID:     m.SenderID,
		SenderName:   m.SenderName,
		SenderAvatar: m.SenderAvatar,
		Content:      m.Content,
		Image:        m.Image,
		Timestamp:    m.CreatedAt.Unix(),
		IsUser:       m.IsUser,
		IsArchived:   m.IsArchived,
	}
}

// getCurrentUserID 从context获取当前用户ID
// 使用 auth 包提供的类型安全的 context key 来获取用户ID
// 用户ID由 AuthMiddleware 在认证成功后注入到 request context 中
func (r *Repository) getCurrentUserID(ctx context.Context) string {
	return auth.UserIDFromContext(ctx)
}

// ========== Chat Storyboard Branch Methods ==========

func (r *Repository) StoryboardLeafNodesByCharacter(ctx context.Context, characterID string) ([]*domain.Storyboard, error) {
	// Find all storyboards where the character participated and has no children
	var links []StoryboardCharacterLink
	if err := r.db.WithContext(ctx).
		Where("character_id = ?", characterID).
		Find(&links).Error; err != nil {
		return nil, fmt.Errorf("failed to get character storyboard links: %w", err)
	}

	if len(links) == 0 {
		return []*domain.Storyboard{}, nil
	}

	storyboardIDs := make([]string, 0, len(links))
	for _, link := range links {
		storyboardIDs = append(storyboardIDs, link.StoryboardID)
	}

	// Find storyboards that have no children (leaf nodes)
	var leafStoryboards []Storyboard
	subquery := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Select("parent_id").
		Where("parent_id IN ? AND deleted_at IS NULL", storyboardIDs)

	if err := r.db.WithContext(ctx).
		Where("id IN ? AND id NOT IN (?) AND deleted_at IS NULL", storyboardIDs, subquery).
		Preload("Creator").
		Order("created_at DESC").
		Find(&leafStoryboards).Error; err != nil {
		return nil, fmt.Errorf("failed to get leaf storyboards: %w", err)
	}

	result := make([]*domain.Storyboard, len(leafStoryboards))
	for i, sb := range leafStoryboards {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, err
		}
		result[i] = &domainSb
	}

	return result, nil
}

func (r *Repository) TraceStoryboardAncestors(ctx context.Context, leafNodeID, characterID string, limit int) ([]*domain.Storyboard, error) {
	var ancestors []*domain.Storyboard
	currentID := leafNodeID
	visited := make(map[string]bool)

	for i := 0; i < limit; i++ {
		if visited[currentID] {
			break // Prevent cycles
		}
		visited[currentID] = true

		// Get current storyboard
		sb, err := r.StoryboardByID(ctx, currentID)
		if err != nil {
			break
		}

		// Verify character participated in this storyboard
		var link StoryboardCharacterLink
		if err := r.db.WithContext(ctx).
			Where("storyboard_id = ? AND character_id = ?", currentID, characterID).
			First(&link).Error; err != nil {
			// Character didn't participate, skip
			if sb.ParentID == "" || sb.ParentID == domain.StoryboardRootMarker {
				break
			}
			currentID = sb.ParentID
			continue
		}

		ancestors = append(ancestors, sb)

		// Move to parent
		if sb.ParentID == "" || sb.ParentID == domain.StoryboardRootMarker {
			break
		}
		currentID = sb.ParentID
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
		ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
	}

	return ancestors, nil
}

func (r *Repository) CreateStoryboardBranch(ctx context.Context, branch *domain.StoryboardBranch) error {
	dbBranch := ChatThreadStoryboardBranch{
		ID:           uuid.New().String(),
		ThreadID:     branch.ThreadID,
		StoryboardID: branch.StoryboardID,
		CharacterID:  branch.CharacterID,
		SelectedAt:   time.Unix(branch.SelectedAt, 0),
	}

	if err := r.db.WithContext(ctx).Create(&dbBranch).Error; err != nil {
		return fmt.Errorf("failed to create storyboard branch: %w", err)
	}

	branch.ID = dbBranch.ID
	branch.CreatedAt = dbBranch.CreatedAt.Unix()
	return nil
}

func (r *Repository) GetStoryboardBranchByThread(ctx context.Context, threadID string) (*domain.StoryboardBranch, error) {
	var branch ChatThreadStoryboardBranch
	if err := r.db.WithContext(ctx).
		Where("thread_id = ?", threadID).
		Order("selected_at DESC").
		First(&branch).Error; err != nil {
		return nil, fmt.Errorf("failed to get storyboard branch: %w", err)
	}

	return &domain.StoryboardBranch{
		ID:           branch.ID,
		ThreadID:     branch.ThreadID,
		StoryboardID: branch.StoryboardID,
		CharacterID:  branch.CharacterID,
		SelectedAt:   branch.SelectedAt.Unix(),
		CreatedAt:    branch.CreatedAt.Unix(),
	}, nil
}

// ========== Chat Message Reaction Methods ==========

func (r *Repository) CreateMessageReaction(ctx context.Context, reaction *domain.MessageReaction) error {
	dbReaction := ChatMessageReaction{
		ID:           uuid.New().String(),
		MessageID:    reaction.MessageID,
		UserID:       reaction.UserID,
		ReactionType: reaction.ReactionType,
		EmojiCode:    reaction.EmojiCode,
	}

	if err := r.db.WithContext(ctx).Create(&dbReaction).Error; err != nil {
		return fmt.Errorf("failed to create message reaction: %w", err)
	}

	// Update message reaction count
	if err := r.db.WithContext(ctx).Model(&ChatMessage{}).
		Where("id = ?", reaction.MessageID).
		Update("reaction_count", gorm.Expr("reaction_count + 1")).Error; err != nil {
		return fmt.Errorf("failed to update reaction count: %w", err)
	}

	reaction.ID = dbReaction.ID
	reaction.CreatedAt = dbReaction.CreatedAt.Unix()
	return nil
}

func (r *Repository) GetMessageReactions(ctx context.Context, messageID string) ([]*domain.MessageReaction, error) {
	var reactions []ChatMessageReaction
	if err := r.db.WithContext(ctx).
		Where("message_id = ? AND deleted_at IS NULL", messageID).
		Order("created_at ASC").
		Find(&reactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get message reactions: %w", err)
	}

	result := make([]*domain.MessageReaction, len(reactions))
	for i, r := range reactions {
		result[i] = &domain.MessageReaction{
			ID:           r.ID,
			MessageID:    r.MessageID,
			UserID:       r.UserID,
			ReactionType: r.ReactionType,
			EmojiCode:    r.EmojiCode,
			CreatedAt:    r.CreatedAt.Unix(),
		}
	}

	return result, nil
}

func (r *Repository) DeleteMessageReaction(ctx context.Context, messageID, userID, reactionType, emojiCode string) error {
	query := r.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ? AND reaction_type = ?", messageID, userID, reactionType)

	if emojiCode != "" {
		query = query.Where("emoji_code = ?", emojiCode)
	}

	if err := query.Delete(&ChatMessageReaction{}).Error; err != nil {
		return fmt.Errorf("failed to delete message reaction: %w", err)
	}

	// Update message reaction count
	if err := r.db.WithContext(ctx).Model(&ChatMessage{}).
		Where("id = ?", messageID).
		Update("reaction_count", gorm.Expr("GREATEST(reaction_count - 1, 0)")).Error; err != nil {
		return fmt.Errorf("failed to update reaction count: %w", err)
	}

	return nil
}

func (r *Repository) GetUserMessageReaction(ctx context.Context, messageID, userID string) (*domain.MessageReaction, error) {
	var reaction ChatMessageReaction
	if err := r.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ? AND deleted_at IS NULL", messageID, userID).
		First(&reaction).Error; err != nil {
		return nil, fmt.Errorf("failed to get user message reaction: %w", err)
	}

	return &domain.MessageReaction{
		ID:           reaction.ID,
		MessageID:    reaction.MessageID,
		UserID:       reaction.UserID,
		ReactionType: reaction.ReactionType,
		EmojiCode:    reaction.EmojiCode,
		CreatedAt:    reaction.CreatedAt.Unix(),
	}, nil
}

// ========== Chat Message Token Methods ==========

func (r *Repository) CreateMessageTokenUsage(ctx context.Context, tokenUsage *domain.TokenUsage) error {
	dbToken := ChatMessageToken{
		ID:          uuid.New().String(),
		MessageID:   tokenUsage.MessageID,
		InputTokens: tokenUsage.InputTokens,
		OutputTokens: tokenUsage.OutputTokens,
		TotalTokens: tokenUsage.TotalTokens,
		Model:       tokenUsage.Model,
	}

	if err := r.db.WithContext(ctx).Create(&dbToken).Error; err != nil {
		return fmt.Errorf("failed to create message token usage: %w", err)
	}

	tokenUsage.ID = dbToken.ID
	tokenUsage.CreatedAt = dbToken.CreatedAt.Unix()
	return nil
}

func (r *Repository) GetMessageTokenUsage(ctx context.Context, messageID string) (*domain.TokenUsage, error) {
	var token ChatMessageToken
	if err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		First(&token).Error; err != nil {
		return nil, fmt.Errorf("failed to get message token usage: %w", err)
	}

	return &domain.TokenUsage{
		ID:          token.ID,
		MessageID:   token.MessageID,
		InputTokens: token.InputTokens,
		OutputTokens: token.OutputTokens,
		TotalTokens: token.TotalTokens,
		Model:       token.Model,
		CreatedAt:   token.CreatedAt.Unix(),
	}, nil
}

func (r *Repository) UpdateThreadTokenUsage(ctx context.Context, threadID string, tokens int64) error {
	if err := r.db.WithContext(ctx).Model(&ChatThread{}).
		Where("id = ?", threadID).
		Update("total_tokens_used", gorm.Expr("total_tokens_used + ?", tokens)).Error; err != nil {
		return fmt.Errorf("failed to update thread token usage: %w", err)
	}
	return nil
}

// ========== Chat Message Archive Methods ==========

func (r *Repository) ArchiveMessage(ctx context.Context, messageID, userID string) error {
	// Verify message belongs to user's thread
	var message ChatMessage
	if err := r.db.WithContext(ctx).
		Preload("Thread").
		Where("id = ?", messageID).
		First(&message).Error; err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if message.Thread.UserID != userID {
		return fmt.Errorf("unauthorized: message does not belong to user")
	}

	if err := r.db.WithContext(ctx).Model(&ChatMessage{}).
		Where("id = ?", messageID).
		Update("is_archived", true).Error; err != nil {
		return fmt.Errorf("failed to archive message: %w", err)
	}

	return nil
}

func (r *Repository) UnarchiveMessage(ctx context.Context, messageID, userID string) error {
	// Verify message belongs to user's thread
	var message ChatMessage
	if err := r.db.WithContext(ctx).
		Preload("Thread").
		Where("id = ?", messageID).
		First(&message).Error; err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if message.Thread.UserID != userID {
		return fmt.Errorf("unauthorized: message does not belong to user")
	}

	if err := r.db.WithContext(ctx).Model(&ChatMessage{}).
		Where("id = ?", messageID).
		Update("is_archived", false).Error; err != nil {
		return fmt.Errorf("failed to unarchive message: %w", err)
	}

	return nil
}

func (r *Repository) ArchiveThread(ctx context.Context, threadID, userID string) error {
	if err := r.db.WithContext(ctx).Model(&ChatThread{}).
		Where("id = ? AND user_id = ?", threadID, userID).
		Update("is_archived", true).Error; err != nil {
		return fmt.Errorf("failed to archive thread: %w", err)
	}
	return nil
}

func (r *Repository) UnarchiveThread(ctx context.Context, threadID, userID string) error {
	if err := r.db.WithContext(ctx).Model(&ChatThread{}).
		Where("id = ? AND user_id = ?", threadID, userID).
		Update("is_archived", false).Error; err != nil {
		return fmt.Errorf("failed to unarchive thread: %w", err)
	}
	return nil
}

func (r *Repository) ChatMessagesBefore(ctx context.Context, threadID, beforeMessageID string, limit int) ([]*domain.ChatMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var beforeMessage ChatMessage
	if err := r.db.WithContext(ctx).
		Where("id = ? AND thread_id = ?", beforeMessageID, threadID).
		First(&beforeMessage).Error; err != nil {
		return nil, fmt.Errorf("before message not found: %w", err)
	}

	var messages []ChatMessage
	if err := r.db.WithContext(ctx).
		Where("thread_id = ? AND created_at < ? AND is_archived = ?", threadID, beforeMessage.CreatedAt, false).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get messages before: %w", err)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	result := make([]*domain.ChatMessage, len(messages))
	for i, m := range messages {
		dm := r.chatMessageToDomain(m)
		result[i] = &dm
	}

	return result, nil
}

func (r *Repository) ChatMessagesArchived(ctx context.Context, threadID string, limit, offset int) ([]*domain.ChatMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var messages []ChatMessage
	if err := r.db.WithContext(ctx).
		Where("thread_id = ? AND is_archived = ?", threadID, true).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get archived messages: %w", err)
	}

	result := make([]*domain.ChatMessage, len(messages))
	for i, m := range messages {
		dm := r.chatMessageToDomain(m)
		result[i] = &dm
	}

	return result, nil
}
