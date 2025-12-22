package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

// ========== Chat Thread Operations ==========

// ListChatThreads 获取用户的聊天列表
func (s *Service) ListChatThreads(ctx context.Context, userID string) ([]*domain.ChatThread, error) {
	threads, err := s.repo.ChatThreads(ctx, userID)
	if err != nil {
		s.logger.Error("failed to list chat threads", zap.Error(err), zap.String("userId", userID))
		return nil, fmt.Errorf("failed to list chat threads: %w", err)
	}
	return threads, nil
}

// GetChatThread 获取聊天线程详情
func (s *Service) GetChatThread(ctx context.Context, threadID, userID string) (*domain.ChatThread, error) {
	s.logger.Debug("getting chat thread",
		zap.String("threadID", threadID),
		zap.String("userID", userID))

	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		s.logger.Error("failed to get chat thread", zap.Error(err), zap.String("threadId", threadID))
		return nil, fmt.Errorf("failed to get chat thread: %w", err)
	}

	// 验证用户权限（确保是该thread的所有者）
	if thread.UserID != userID {
		s.logger.Warn("unauthorized chat thread access attempt",
			zap.String("threadID", threadID),
			zap.String("requestUserID", userID),
			zap.String("threadOwnerID", thread.UserID))
		return nil, fmt.Errorf("unauthorized: you don't have access to this chat thread")
	}

	s.logger.Debug("chat thread access authorized",
		zap.String("threadID", threadID),
		zap.String("userID", userID))

	return thread, nil
}

// CreateChatThread 创建聊天线程（与角色开始聊天）
func (s *Service) CreateChatThread(ctx context.Context, userID, characterID string) (*domain.ChatThread, error) {
	// 获取角色信息
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		s.logger.Error("character not found", zap.Error(err), zap.String("characterId", characterID))
		return nil, fmt.Errorf("character not found: %w", err)
	}

	// 创建新的聊天线程
	thread := &domain.ChatThread{
		CharacterID:          characterID,
		CharacterName:        character.Name,
		CharacterAvatar:      character.Avatar,
		StoryTitle:           "", // 可以关联到某个故事，目前为空
		LastMessage:          "",
		LastMessageTime:      time.Now().Unix(),
		UnreadCount:          0,
		MessageCount:         0,
		InteractionFrequency: 0,
	}

	if err := s.repo.CreateChatThread(ctx, thread); err != nil {
		s.logger.Error("failed to create chat thread", zap.Error(err))
		return nil, fmt.Errorf("failed to create chat thread: %w", err)
	}

	s.logger.Info("chat thread created",
		zap.String("threadId", thread.ID),
		zap.String("userId", userID),
		zap.String("characterId", characterID))

	return thread, nil
}

// DeleteChatThread 删除聊天线程
func (s *Service) DeleteChatThread(ctx context.Context, threadID, userID string) error {
	s.logger.Info("deleting chat thread",
		zap.String("threadID", threadID),
		zap.String("userID", userID))

	// 验证用户权限
	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		s.logger.Error("failed to get chat thread for deletion",
			zap.Error(err),
			zap.String("threadId", threadID))
		return fmt.Errorf("failed to get chat thread: %w", err)
	}

	// 确保只有thread的所有者可以删除
	if thread.UserID != userID {
		s.logger.Warn("unauthorized chat thread deletion attempt",
			zap.String("threadID", threadID),
			zap.String("requestUserID", userID),
			zap.String("threadOwnerID", thread.UserID))
		return fmt.Errorf("unauthorized: you can only delete your own chat threads")
	}

	s.logger.Debug("chat thread deletion authorized",
		zap.String("threadID", threadID),
		zap.String("userID", userID))

	if err := s.repo.DeleteChatThread(ctx, threadID); err != nil {
		s.logger.Error("failed to delete chat thread", zap.Error(err), zap.String("threadId", threadID))
		return fmt.Errorf("failed to delete chat thread: %w", err)
	}

	s.logger.Info("chat thread deleted successfully",
		zap.String("threadId", threadID),
		zap.String("userId", userID))
	return nil
}

// MarkChatThreadAsRead 标记聊天为已读
func (s *Service) MarkChatThreadAsRead(ctx context.Context, threadID, userID string) error {
	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		return fmt.Errorf("failed to get chat thread: %w", err)
	}

	// 清空未读计数
	thread.UnreadCount = 0
	if err := s.repo.UpdateChatThread(ctx, thread); err != nil {
		s.logger.Error("failed to mark thread as read", zap.Error(err), zap.String("threadId", threadID))
		return fmt.Errorf("failed to mark thread as read: %w", err)
	}

	return nil
}

// ========== Chat Message Operations ==========

// ListChatMessages 获取聊天消息列表
func (s *Service) ListChatMessages(ctx context.Context, threadID, userID string, limit, offset int) ([]*domain.ChatMessage, error) {
	s.logger.Debug("listing chat messages",
		zap.String("threadID", threadID),
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// 验证用户是否有权限访问该thread
	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		s.logger.Error("failed to get chat thread for message listing",
			zap.Error(err),
			zap.String("threadId", threadID))
		return nil, fmt.Errorf("failed to get chat thread: %w", err)
	}

	// 确保只有thread的所有者可以查看消息
	if thread.UserID != userID {
		s.logger.Warn("unauthorized chat messages access attempt",
			zap.String("threadID", threadID),
			zap.String("requestUserID", userID),
			zap.String("threadOwnerID", thread.UserID))
		return nil, fmt.Errorf("unauthorized: you don't have access to these messages")
	}

	s.logger.Debug("chat messages access authorized",
		zap.String("threadID", threadID),
		zap.String("userID", userID))

	if limit <= 0 {
		limit = 50 // 默认获取50条消息
	}
	if limit > 200 {
		limit = 200 // 最多200条
	}

	messages, err := s.repo.ChatMessages(ctx, threadID, limit, offset)
	if err != nil {
		s.logger.Error("failed to list chat messages", zap.Error(err), zap.String("threadId", threadID))
		return nil, fmt.Errorf("failed to list chat messages: %w", err)
	}

	s.logger.Debug("chat messages listed successfully",
		zap.String("threadID", threadID),
		zap.Int("count", len(messages)))

	return messages, nil
}

// SendChatMessage 发送聊天消息（用户发送给角色）
func (s *Service) SendChatMessage(ctx context.Context, threadID, userID, content, image string) (*domain.ChatMessage, error) {
	// 获取用户信息
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 获取thread信息
	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("thread not found: %w", err)
	}

	// 创建用户消息
	message := &domain.ChatMessage{
		ThreadID:     threadID,
		SenderID:     userID,
		SenderName:   user.DisplayName,
		SenderAvatar: user.Avatar,
		Content:      content,
		Image:        image,
		Timestamp:    time.Now().Unix(),
		IsUser:       true,
	}

	if err := s.repo.CreateChatMessage(ctx, message); err != nil {
		s.logger.Error("failed to create chat message", zap.Error(err))
		return nil, fmt.Errorf("failed to create chat message: %w", err)
	}

	// 更新聊天线程信息
	thread.LastMessage = content
	thread.LastMessageTime = time.Now().Unix()
	thread.MessageCount++
	if err := s.repo.UpdateChatThread(ctx, thread); err != nil {
		s.logger.Warn("failed to update chat thread after user message", zap.Error(err))
	}

	s.logger.Info("user message sent",
		zap.String("messageId", message.ID),
		zap.String("threadId", threadID),
		zap.String("userId", userID))

	// 异步调用 AI 生成角色回复
	go s.generateCharacterReply(context.Background(), thread, message)

	return message, nil
}

// generateCharacterReply 生成角色回复（AI生成）
func (s *Service) generateCharacterReply(ctx context.Context, thread *domain.ChatThread, userMessage *domain.ChatMessage) {
	startTime := time.Now()

	// 获取角色信息
	character, err := s.repo.CharacterByID(ctx, thread.CharacterID)
	if err != nil {
		s.logger.Error("failed to get character for reply", zap.Error(err))
		return
	}

	// 生成回复内容
	var replyContent string
	var tokensUsed int

	// 检查是否有 Gemini 客户端可用
	if s.geminiClient != nil {
		// 使用 AI 生成回复
		replyContent, tokensUsed, err = s.generateAIReply(ctx, thread, character, userMessage)
		if err != nil {
			s.logger.Error("failed to generate AI reply, falling back to simple reply",
				zap.Error(err),
				zap.String("threadId", thread.ID),
				zap.String("characterId", character.ID))
			// 降级到简单回复
			replyContent = s.generateFallbackReply(character, userMessage)
		}
	} else {
		// 没有 AI 客户端，使用简单回复
		s.logger.Warn("gemini client not available, using fallback reply")
		replyContent = s.generateFallbackReply(character, userMessage)
	}

	duration := time.Since(startTime).Milliseconds()

	// 创建角色回复消息
	reply := &domain.ChatMessage{
		ThreadID:     thread.ID,
		SenderID:     character.ID,
		SenderName:   character.Name,
		SenderAvatar: character.Avatar,
		Content:      replyContent,
		Timestamp:    time.Now().Unix(),
		IsUser:       false,
	}

	if err := s.repo.CreateChatMessage(ctx, reply); err != nil {
		s.logger.Error("failed to create character reply", zap.Error(err))
		return
	}

	// 更新聊天线程信息
	thread.LastMessage = replyContent
	thread.LastMessageTime = time.Now().Unix()
	thread.MessageCount++
	thread.UnreadCount++
	if err := s.repo.UpdateChatThread(ctx, thread); err != nil {
		s.logger.Warn("failed to update chat thread", zap.Error(err))
	}

	// 更新角色统计数据（如果使用了 AI）
	if tokensUsed > 0 {
		if err := s.repo.IncrementCharacterMessages(ctx, character.ID, 1); err != nil {
			s.logger.Warn("failed to increment character messages", zap.Error(err))
		}
		if err := s.repo.IncrementCharacterTokens(ctx, character.ID, int64(tokensUsed)); err != nil {
			s.logger.Warn("failed to increment character tokens", zap.Error(err))
		}

		// Record metrics
		if s.metrics != nil {
			s.metrics.RecordCharacterMessage(character.ID)
			s.metrics.RecordCharacterTokenConsumed(character.ID, float64(tokensUsed))
		}
	}

	s.logger.Info("character reply generated",
		zap.String("messageId", reply.ID),
		zap.String("threadId", thread.ID),
		zap.String("characterId", character.ID),
		zap.Int("tokensUsed", tokensUsed),
		zap.Int64("durationMs", duration))
}

// generateAIReply 使用 AI 生成角色回复
func (s *Service) generateAIReply(ctx context.Context, thread *domain.ChatThread, character *domain.Character, userMessage *domain.ChatMessage) (string, int, error) {
	// 1. 构建对话历史（获取最近10条消息）
	conversationHistory, err := s.buildChatConversationHistory(ctx, thread.ID, 10)
	if err != nil {
		s.logger.Warn("failed to build conversation history", zap.Error(err))
		conversationHistory = []*genai.Content{} // 使用空上下文
	}

	// 2. 构建系统提示词
	systemPrompt := s.buildCharacterSystemPrompt(character)

	// 3. 配置生成参数
	temperature := float32(0.8) // 角色对话使用较高创意度
	maxTokens := int32(1024)    // 限制回复长度

	genConfig := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: maxTokens,
		SystemInstruction: &genai.Content{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{genai.NewPartFromText(systemPrompt)},
		},
	}

	// 4. 构建完整的对话内容（历史 + 当前消息）
	contents := append(conversationHistory, &genai.Content{
		Role:  genai.RoleUser,
		Parts: []*genai.Part{genai.NewPartFromText(userMessage.Content)},
	})

	// 5. 调用 AI 生成回复
	s.logger.Debug("calling AI service for character reply",
		zap.String("characterId", character.ID),
		zap.String("characterName", character.Name),
		zap.Int("historyLength", len(conversationHistory)))

	geminiResp, err := s.geminiClient.GenerateContent(ctx, "", contents, genConfig)
	if err != nil {
		return "", 0, fmt.Errorf("AI generation failed: %w", err)
	}

	// 6. 提取响应文本
	responseText := geminiResp.Text()
	if responseText == "" && len(geminiResp.Candidates) > 0 && geminiResp.Candidates[0].Content != nil {
		for _, part := range geminiResp.Candidates[0].Content.Parts {
			if part != nil && part.Text != "" {
				responseText = part.Text
				break
			}
		}
	}

	// 7. 计算 token 使用量
	tokensUsed := 0
	if geminiResp.UsageMetadata != nil {
		tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
	}

	return responseText, tokensUsed, nil
}

// buildChatConversationHistory 构建对话历史（转换为 Gemini Content 格式）
func (s *Service) buildChatConversationHistory(ctx context.Context, threadID string, limit int) ([]*genai.Content, error) {
	messages, err := s.repo.ChatMessages(ctx, threadID, limit, 0)
	if err != nil {
		return nil, err
	}

	// 消息是按时间倒序返回的，需要反转以保持对话顺序
	contents := make([]*genai.Content, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		role := genai.RoleUser
		if !msg.IsUser {
			role = genai.RoleModel
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []*genai.Part{genai.NewPartFromText(msg.Content)},
		})
	}

	return contents, nil
}

// buildCharacterSystemPrompt 构建角色系统提示词
func (s *Service) buildCharacterSystemPrompt(character *domain.Character) string {
	var prompt strings.Builder

	// 角色基本设定
	prompt.WriteString(fmt.Sprintf("你是 %s，一个虚拟角色。请始终以这个角色的身份进行对话。\n\n", character.Name))

	// 添加角色描述
	if character.Description != "" {
		prompt.WriteString(fmt.Sprintf("## 角色描述\n%s\n\n", character.Description))
	}

	// 添加性格特征
	if character.Personality != "" {
		prompt.WriteString(fmt.Sprintf("## 性格特征\n%s\n\n", character.Personality))
	}

	// 添加背景故事
	if character.Background != "" {
		prompt.WriteString(fmt.Sprintf("## 背景故事\n%s\n\n", character.Background))
	}

	// 添加特质
	if len(character.Traits) > 0 {
		traitsJSON, _ := json.Marshal(character.Traits)
		prompt.WriteString(fmt.Sprintf("## 特质\n%s\n\n", string(traitsJSON)))
	}

	// 添加技能
	if len(character.Skills) > 0 {
		skillsJSON, _ := json.Marshal(character.Skills)
		prompt.WriteString(fmt.Sprintf("## 技能\n%s\n\n", string(skillsJSON)))
	}

	// 添加对话指导
	prompt.WriteString("## 对话指导\n")
	prompt.WriteString("- 保持角色一致性，始终以角色的语气和风格回应\n")
	prompt.WriteString("- 回复要自然流畅，像真人对话一样\n")
	prompt.WriteString("- 适当表达角色的情感和态度\n")
	prompt.WriteString("- 回复长度适中，避免过长或过短\n")
	prompt.WriteString("- 可以使用适当的语气词和表情来增加亲和力\n")

	return prompt.String()
}

// generateFallbackReply 生成降级回复（当 AI 不可用时）
func (s *Service) generateFallbackReply(character *domain.Character, userMessage *domain.ChatMessage) string {
	// 根据角色性格生成简单的回复
	name := character.Name
	if name == "" {
		name = "我"
	}

	// 简单的模板回复
	templates := []string{
		fmt.Sprintf("你好！%s收到了你的消息。", name),
		fmt.Sprintf("嗯，我明白了。让%s想想该怎么回复你。", name),
		fmt.Sprintf("谢谢你的消息！%s很高兴和你聊天。", name),
	}

	// 根据消息长度选择模板
	idx := len(userMessage.Content) % len(templates)
	return templates[idx]
}

// DeleteChatMessage 删除聊天消息
func (s *Service) DeleteChatMessage(ctx context.Context, messageID, userID string) error {
	s.logger.Info("deleting chat message",
		zap.String("messageID", messageID),
		zap.String("userID", userID))

	// 验证用户权限（只能删除自己的消息）
	message, err := s.repo.ChatMessageByID(ctx, messageID)
	if err != nil {
		s.logger.Error("failed to get chat message for deletion",
			zap.Error(err),
			zap.String("messageId", messageID))
		return fmt.Errorf("failed to get chat message: %w", err)
	}

	// 只能删除用户自己发送的消息（IsUser=true 且 SenderID 匹配）
	if !message.IsUser || message.SenderID != userID {
		s.logger.Warn("unauthorized chat message deletion attempt",
			zap.String("messageID", messageID),
			zap.String("requestUserID", userID),
			zap.String("messageSenderID", message.SenderID),
			zap.Bool("isUser", message.IsUser))
		return fmt.Errorf("unauthorized: you can only delete your own messages")
	}

	s.logger.Debug("chat message deletion authorized",
		zap.String("messageID", messageID),
		zap.String("userID", userID))

	if err := s.repo.DeleteChatMessage(ctx, messageID); err != nil {
		s.logger.Error("failed to delete chat message", zap.Error(err), zap.String("messageId", messageID))
		return fmt.Errorf("failed to delete chat message: %w", err)
	}

	s.logger.Info("chat message deleted successfully",
		zap.String("messageId", messageID),
		zap.String("userId", userID))
	return nil
}

// GetUnreadChatCount 获取未读聊天数
func (s *Service) GetUnreadChatCount(ctx context.Context, userID string) (int, error) {
	threads, err := s.repo.ChatThreads(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get chat threads: %w", err)
	}

	unreadCount := 0
	for _, thread := range threads {
		unreadCount += thread.UnreadCount
	}

	return unreadCount, nil
}
