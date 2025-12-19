package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

// AgentChatService 提供基于Agent的智能聊天服务
type AgentChatService struct {
	repo         domain.Repository
	geminiClient *gemini.Client
	logger       *zap.Logger
}

// NewAgentChatService 创建Agent聊天服务
func NewAgentChatService(repo domain.Repository, geminiClient *gemini.Client, logger *zap.Logger) *AgentChatService {
	return &AgentChatService{
		repo:         repo,
		geminiClient: geminiClient,
		logger:       logger,
	}
}

// SendMessageToAgent 用户向Agent发送消息并获取AI回复
func (s *AgentChatService) SendMessageToAgent(ctx context.Context, userID, characterID, content string, threadID *string) (*domain.ChatMessage, *domain.ChatMessage, error) {
	// 1. 获取或创建聊天线程
	var thread *domain.ChatThread
	var err error

	if threadID != nil && *threadID != "" {
		thread, err = s.repo.ChatThreadByID(ctx, *threadID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get chat thread: %w", err)
		}
	} else {
		// 创建新的聊天线程
		thread, err = s.createChatThreadForCharacter(ctx, userID, characterID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create chat thread: %w", err)
		}
	}

	// 2. 获取用户信息
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	// 3. 创建用户消息
	userMessage := &domain.ChatMessage{
		ThreadID:     thread.ID,
		SenderID:     userID,
		SenderName:   user.DisplayName,
		SenderAvatar: user.Avatar,
		Content:      content,
		Timestamp:    time.Now().Unix(),
		IsUser:       true,
	}

	if err := s.repo.CreateChatMessage(ctx, userMessage); err != nil {
		return nil, nil, fmt.Errorf("failed to save user message: %w", err)
	}

	s.logger.Info("user message saved",
		zap.String("messageId", userMessage.ID),
		zap.String("threadId", thread.ID),
		zap.String("userId", userID))

	// 4. 生成Agent回复
	agentReply, err := s.generateAgentReply(ctx, thread, userMessage)
	if err != nil {
		s.logger.Error("failed to generate agent reply", zap.Error(err))
		// 返回用户消息，但Agent回复失败
		return userMessage, nil, fmt.Errorf("failed to generate agent reply: %w", err)
	}

	return userMessage, agentReply, nil
}

// generateAgentReply 生成Agent的AI回复
func (s *AgentChatService) generateAgentReply(ctx context.Context, thread *domain.ChatThread, userMessage *domain.ChatMessage) (*domain.ChatMessage, error) {
	startTime := time.Now()

	// 1. 获取角色信息
	character, err := s.repo.CharacterByID(ctx, thread.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("character not found: %w", err)
	}

	// 2. 获取或创建Agent
	agent, err := s.getOrCreateAgent(ctx, character)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// 3. 构建对话上下文（获取最近的对话历史）
	conversationHistory, err := s.buildConversationHistory(ctx, thread.ID, 10)
	if err != nil {
		s.logger.Warn("failed to build conversation context", zap.Error(err))
		conversationHistory = []*genai.Content{} // 使用空上下文
	}

	// 4. 构建故事板上下文
	var storyboards []*domain.Storyboard
	if thread.SelectedStoryboardID != "" {
		storyboards, _ = s.TraceStoryboardHistory(ctx, thread.SelectedStoryboardID, character.ID, 5)
		// Update thread context if needed
		if len(thread.ContextStoryboardIDs) == 0 && len(storyboards) > 0 {
			storyboardIDs := make([]string, len(storyboards))
			for i, sb := range storyboards {
				storyboardIDs[i] = sb.ID
			}
			thread.ContextStoryboardIDs = storyboardIDs
			s.repo.UpdateChatThread(ctx, thread)
		}
	}

	// 5. 构建系统提示词和配置
	systemPrompt := s.buildSystemPrompt(agent, character, storyboards)
	temperature := float32(agent.Temperature)
	maxTokens := int32(agent.MaxTokens)

	genConfig := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: maxTokens,
		SystemInstruction: &genai.Content{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{genai.NewPartFromText(systemPrompt)},
		},
	}

	// 6. 构建完整的对话内容（包含历史）
	contents := append(conversationHistory, &genai.Content{
		Role:  genai.RoleUser,
		Parts: []*genai.Part{genai.NewPartFromText(userMessage.Content)},
	})

	// 7. 调用AI生成回复
	s.logger.Info("calling AI service",
		zap.String("agentId", agent.ID),
		zap.String("characterId", character.ID))

	// 使用指定的模型或默认模型
	model := agent.Model
	geminiResp, err := s.geminiClient.GenerateContent(ctx, model, contents, genConfig)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// 提取响应文本
	responseText := geminiResp.Text()
	if responseText == "" && len(geminiResp.Candidates) > 0 && geminiResp.Candidates[0].Content != nil {
		for _, part := range geminiResp.Candidates[0].Content.Parts {
			if part != nil && part.Text != "" {
				responseText = part.Text
				break
			}
		}
	}

	duration := time.Since(startTime).Milliseconds()

	// 8. 计算 token 使用量
	tokensUsed := 0
	inputTokens := 0
	outputTokens := 0
	if geminiResp.UsageMetadata != nil {
		tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
		inputTokens = int(geminiResp.UsageMetadata.PromptTokenCount)
		outputTokens = int(geminiResp.UsageMetadata.CandidatesTokenCount)
	}

	// 9. 创建Agent回复消息
	agentReply := &domain.ChatMessage{
		ThreadID:     thread.ID,
		SenderID:     character.ID,
		SenderName:   character.Name,
		SenderAvatar: character.Avatar,
		Content:      responseText,
		Timestamp:    time.Now().Unix(),
		IsUser:       false,
	}

	if err := s.repo.CreateChatMessage(ctx, agentReply); err != nil {
		return nil, fmt.Errorf("failed to save agent reply: %w", err)
	}

	// 10. 保存token使用统计
	if tokensUsed > 0 {
		tokenUsage := &domain.TokenUsage{
			MessageID:   agentReply.ID,
			InputTokens: inputTokens,
			OutputTokens: outputTokens,
			TotalTokens: tokensUsed,
			Model:       model,
			CreatedAt:   time.Now().Unix(),
		}
		if err := s.repo.CreateMessageTokenUsage(ctx, tokenUsage); err != nil {
			s.logger.Warn("failed to save token usage", zap.Error(err))
		}

		// 更新线程token统计
		if err := s.repo.UpdateThreadTokenUsage(ctx, thread.ID, int64(tokensUsed)); err != nil {
			s.logger.Warn("failed to update thread token usage", zap.Error(err))
		}
	}

	// 11. 记录交互
	interaction := &domain.AgentInteraction{
		AgentID:     agent.ID,
		UserID:      userMessage.SenderID,
		CharacterID: character.ID,
		MessageID:   agentReply.ID,
		Type:        "chat",
		InputText:   userMessage.Content,
		OutputText:  responseText,
		TokensUsed:  tokensUsed,
		Duration:    int(duration),
		Success:     true,
		CreatedAt:   time.Now().Unix(),
	}

	if err := s.repo.CreateAgentInteraction(ctx, interaction); err != nil {
		s.logger.Warn("failed to record interaction", zap.Error(err))
	}

	// 12. 更新Agent统计
	if err := s.repo.IncrementAgentInteractionCount(ctx, agent.ID); err != nil {
		s.logger.Warn("failed to increment agent interaction count", zap.Error(err))
	}

	// 13. 更新角色分析数据
	if err := s.repo.IncrementCharacterMessages(ctx, character.ID, 1); err != nil {
		s.logger.Warn("failed to increment character messages", zap.Error(err))
	}
	if err := s.repo.IncrementCharacterTokens(ctx, character.ID, int64(tokensUsed)); err != nil {
		s.logger.Warn("failed to increment character tokens", zap.Error(err))
	}
	
	// Note: Metrics recording for AgentChatService would need to be added via Service layer
	// For now, metrics are recorded in the main Service's chat.go

	s.logger.Info("agent reply generated",
		zap.String("messageId", agentReply.ID),
		zap.String("agentId", agent.ID),
		zap.Int("tokensUsed", tokensUsed),
		zap.Int64("duration", duration))

	// Load token usage for response
	if tokensUsed > 0 {
		tokenUsage, _ := s.repo.GetMessageTokenUsage(ctx, agentReply.ID)
		agentReply.TokenUsage = tokenUsage
	}

	return agentReply, nil
}

// ========== New Service Methods ==========

// GetStoryboardLeafNodes 获取角色参与的所有故事板叶子节点
func (s *AgentChatService) GetStoryboardLeafNodes(ctx context.Context, characterID string) ([]*domain.Storyboard, error) {
	return s.repo.StoryboardLeafNodesByCharacter(ctx, characterID)
}

// TraceStoryboardHistory 回溯最近N个故事板
func (s *AgentChatService) TraceStoryboardHistory(ctx context.Context, leafNodeID, characterID string, limit int) ([]*domain.Storyboard, error) {
	return s.repo.TraceStoryboardAncestors(ctx, leafNodeID, characterID, limit)
}

// BuildCharacterContext 构建角色上下文（性格、穿着、目标 + 故事板记忆）
func (s *AgentChatService) BuildCharacterContext(ctx context.Context, character *domain.Character, storyboards []*domain.Storyboard) string {
	return s.buildSystemPrompt(nil, character, storyboards)
}

// SelectStoryboardBranch 为聊天线程选择故事板分支
func (s *AgentChatService) SelectStoryboardBranch(ctx context.Context, threadID, leafNodeID, characterID string) error {
	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		return fmt.Errorf("thread not found: %w", err)
	}

	// Trace storyboard history
	storyboards, err := s.TraceStoryboardHistory(ctx, leafNodeID, characterID, 5)
	if err != nil {
		return fmt.Errorf("failed to trace storyboard history: %w", err)
	}

	// Update thread with selected branch and context
	storyboardIDs := make([]string, len(storyboards))
	for i, sb := range storyboards {
		storyboardIDs[i] = sb.ID
	}

	thread.SelectedStoryboardID = leafNodeID
	thread.ContextStoryboardIDs = storyboardIDs

	if err := s.repo.UpdateChatThread(ctx, thread); err != nil {
		return fmt.Errorf("failed to update thread: %w", err)
	}

	// Create branch record
	branch := &domain.StoryboardBranch{
		ThreadID:     threadID,
		StoryboardID: leafNodeID,
		CharacterID:  characterID,
		SelectedAt:   time.Now().Unix(),
	}

	if err := s.repo.CreateStoryboardBranch(ctx, branch); err != nil {
		s.logger.Warn("failed to create branch record", zap.Error(err))
	}

	return nil
}

// ReactToMessage 消息互动（支持toggle：如果已有相同类型的reaction则删除，否则添加）
func (s *AgentChatService) ReactToMessage(ctx context.Context, messageID, userID, reactionType, emojiCode string) error {
	// Check if user already has a reaction of the same type
	existing, err := s.repo.GetUserMessageReaction(ctx, messageID, userID)
	if err == nil && existing != nil {
		// Check if it's the same reaction type and emoji code
		if existing.ReactionType == reactionType {
			// For emoji reactions, also check emojiCode
			if reactionType == "emoji" {
				if existing.EmojiCode == emojiCode {
					// Same emoji reaction exists, toggle off (remove)
					return s.repo.DeleteMessageReaction(ctx, messageID, userID, existing.ReactionType, existing.EmojiCode)
				}
			} else {
				// Same reaction type (like/dislike), toggle off (remove)
				return s.repo.DeleteMessageReaction(ctx, messageID, userID, existing.ReactionType, existing.EmojiCode)
			}
		}
		// Different reaction type, remove old one first
		if err := s.repo.DeleteMessageReaction(ctx, messageID, userID, existing.ReactionType, existing.EmojiCode); err != nil {
			return fmt.Errorf("failed to remove existing reaction: %w", err)
		}
	}

	// Create new reaction (toggle on)
	reaction := &domain.MessageReaction{
		MessageID:    messageID,
		UserID:       userID,
		ReactionType: reactionType,
		EmojiCode:    emojiCode,
		CreatedAt:    time.Now().Unix(),
	}

	return s.repo.CreateMessageReaction(ctx, reaction)
}

// ArchiveMessage 归档消息
func (s *AgentChatService) ArchiveMessage(ctx context.Context, messageID, userID string) error {
	return s.repo.ArchiveMessage(ctx, messageID, userID)
}

// UnarchiveMessage 取消归档消息
func (s *AgentChatService) UnarchiveMessage(ctx context.Context, messageID, userID string) error {
	return s.repo.UnarchiveMessage(ctx, messageID, userID)
}

// ArchiveThread 归档线程并生成摘要
func (s *AgentChatService) ArchiveThread(ctx context.Context, threadID, userID string) error {
	// 获取线程并验证所有权
	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		return fmt.Errorf("thread not found: %w", err)
	}

	if thread.UserID != userID {
		return fmt.Errorf("access denied: thread does not belong to user")
	}

	// 生成会话摘要
	summary, err := s.generateThreadSummary(ctx, thread)
	if err != nil {
		s.logger.Warn("failed to generate thread summary", zap.Error(err))
		// 不阻止归档，只是没有摘要
	}

	// 更新线程状态
	thread.IsArchived = true
	thread.Summary = summary
	thread.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateChatThread(ctx, thread); err != nil {
		return fmt.Errorf("failed to archive thread: %w", err)
	}

	s.logger.Info("chat thread archived successfully",
		zap.String("threadId", threadID),
		zap.Bool("hasSummary", summary != ""))

	return nil
}

// UnarchiveThread 取消归档线程
func (s *AgentChatService) UnarchiveThread(ctx context.Context, threadID, userID string) error {
	return s.repo.UnarchiveThread(ctx, threadID, userID)
}

// LoadMoreMessages 加载更多消息（下拉刷新）
func (s *AgentChatService) LoadMoreMessages(ctx context.Context, threadID, beforeMessageID string, limit int) ([]*domain.ChatMessage, error) {
	return s.repo.ChatMessagesBefore(ctx, threadID, beforeMessageID, limit)
}

// GetThreadStats 获取会话统计
func (s *AgentChatService) GetThreadStats(ctx context.Context, threadID string) (map[string]interface{}, error) {
	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("thread not found: %w", err)
	}

	stats := map[string]interface{}{
		"messageCount":    thread.MessageCount,
		"totalTokensUsed": thread.TotalTokensUsed,
		"unreadCount":     thread.UnreadCount,
		"isArchived":      thread.IsArchived,
	}

	return stats, nil
}

// SendMessageWithStream 支持HTTP/2流式传输的消息发送
// 返回一个channel用于发送流式数据块
func (s *AgentChatService) SendMessageWithStream(ctx context.Context, userID, characterID, content, image string, threadID, storyboardBranchID *string) (<-chan string, <-chan *domain.ChatMessage, <-chan error) {
	chunkChan := make(chan string, 10)
	messageChan := make(chan *domain.ChatMessage, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(messageChan)
		defer close(errChan)

		// 1. 获取或创建聊天线程
		var thread *domain.ChatThread
		var err error

		if threadID != nil && *threadID != "" {
			thread, err = s.repo.ChatThreadByID(ctx, *threadID)
			if err != nil {
				errChan <- fmt.Errorf("failed to get chat thread: %w", err)
				return
			}
		} else {
			thread, err = s.createChatThreadForCharacter(ctx, userID, characterID)
			if err != nil {
				errChan <- fmt.Errorf("failed to create chat thread: %w", err)
				return
			}
		}

		// 2. 如果提供了故事板分支，选择它
		if storyboardBranchID != nil && *storyboardBranchID != "" {
			if err := s.SelectStoryboardBranch(ctx, thread.ID, *storyboardBranchID, characterID); err != nil {
				s.logger.Warn("failed to select storyboard branch", zap.Error(err))
			}
		}

		// 3. 获取用户信息
		user, err := s.repo.UserByID(ctx, userID)
		if err != nil || user == nil {
			errChan <- fmt.Errorf("user not found")
			return
		}

		// 4. 创建用户消息
		userMessage := &domain.ChatMessage{
			ThreadID:     thread.ID,
			SenderID:     userID,
			SenderName:   user.DisplayName,
			SenderAvatar: user.Avatar,
			Content:      content,
			Image:        image,
			Timestamp:    time.Now().Unix(),
			IsUser:       true,
		}

		if err := s.repo.CreateChatMessage(ctx, userMessage); err != nil {
			errChan <- fmt.Errorf("failed to save user message: %w", err)
			return
		}

		// 5. 获取角色和Agent
		character, err := s.repo.CharacterByID(ctx, characterID)
		if err != nil {
			errChan <- fmt.Errorf("character not found: %w", err)
			return
		}

		agent, err := s.getOrCreateAgent(ctx, character)
		if err != nil {
			errChan <- fmt.Errorf("failed to get agent: %w", err)
			return
		}

		// 6. 构建故事板上下文
		var storyboards []*domain.Storyboard
		if thread.SelectedStoryboardID != "" {
			storyboards, _ = s.TraceStoryboardHistory(ctx, thread.SelectedStoryboardID, character.ID, 5)
		}

		// 7. 构建系统提示词
		systemPrompt := s.buildSystemPrompt(agent, character, storyboards)
		temperature := float32(agent.Temperature)
		maxTokens := int32(agent.MaxTokens)

		genConfig := &genai.GenerateContentConfig{
			Temperature:     &temperature,
			MaxOutputTokens: maxTokens,
			SystemInstruction: &genai.Content{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{genai.NewPartFromText(systemPrompt)},
			},
		}

		// 8. 构建对话历史
		conversationHistory, _ := s.buildConversationHistory(ctx, thread.ID, 10)

		// 9. 构建完整的对话内容
		contents := append(conversationHistory, &genai.Content{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{genai.NewPartFromText(content)},
		})

		// 10. 调用AI生成回复（流式）
		model := agent.Model
		// Note: For now, we use non-streaming API and simulate streaming by sending chunks
		// In production, use the streaming API from genai SDK
		geminiResp, err := s.geminiClient.GenerateContent(ctx, model, contents, genConfig)
		if err != nil {
			errChan <- fmt.Errorf("AI generation failed: %w", err)
			return
		}

		// 11. 提取响应文本并模拟流式发送
		responseText := geminiResp.Text()
		if responseText == "" && len(geminiResp.Candidates) > 0 && geminiResp.Candidates[0].Content != nil {
			for _, part := range geminiResp.Candidates[0].Content.Parts {
				if part != nil && part.Text != "" {
					responseText = part.Text
					break
				}
			}
		}

		// Simulate streaming by sending chunks (in production, use actual streaming API)
		chunkSize := 10
		for i := 0; i < len(responseText); i += chunkSize {
			end := i + chunkSize
			if end > len(responseText) {
				end = len(responseText)
			}
			chunk := responseText[i:end]
			select {
			case chunkChan <- chunk:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}

		// 12. 计算token使用量
		tokensUsed := 0
		inputTokens := 0
		outputTokens := 0
		if geminiResp.UsageMetadata != nil {
			tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
			inputTokens = int(geminiResp.UsageMetadata.PromptTokenCount)
			outputTokens = int(geminiResp.UsageMetadata.CandidatesTokenCount)
		}

		// 13. 创建Agent回复消息
		agentReply := &domain.ChatMessage{
			ThreadID:     thread.ID,
			SenderID:     character.ID,
			SenderName:   character.Name,
			SenderAvatar: character.Avatar,
			Content:      responseText,
			Timestamp:    time.Now().Unix(),
			IsUser:       false,
		}

		if err := s.repo.CreateChatMessage(ctx, agentReply); err != nil {
			errChan <- fmt.Errorf("failed to save agent reply: %w", err)
			return
		}

		// 14. 保存token使用统计
		if tokensUsed > 0 {
			tokenUsage := &domain.TokenUsage{
				MessageID:   agentReply.ID,
				InputTokens: inputTokens,
				OutputTokens: outputTokens,
				TotalTokens: tokensUsed,
				Model:       model,
				CreatedAt:   time.Now().Unix(),
			}
			s.repo.CreateMessageTokenUsage(ctx, tokenUsage)
			s.repo.UpdateThreadTokenUsage(ctx, thread.ID, int64(tokensUsed))
			agentReply.TokenUsage = tokenUsage
		}

		messageChan <- agentReply
	}()

	return chunkChan, messageChan, errChan
}

// buildConversationHistory 构建对话历史（转换为 Gemini Content 格式）
func (s *AgentChatService) buildConversationHistory(ctx context.Context, threadID string, limit int) ([]*genai.Content, error) {
	messages, err := s.repo.ChatMessages(ctx, threadID, limit, 0)
	if err != nil {
		return nil, err
	}

	contents := make([]*genai.Content, 0, len(messages))
	for _, msg := range messages {
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

// buildSystemPrompt 构建系统提示词（定义Agent的角色和行为）
func (s *AgentChatService) buildSystemPrompt(agent *domain.Agent, character *domain.Character, storyboards []*domain.Storyboard) string {
	var prompt strings.Builder

	// 使用自定义系统提示词
	if agent.SystemPrompt != "" {
		prompt.WriteString(agent.SystemPrompt)
		prompt.WriteString("\n\n")
	} else {
		// 默认提示词
		prompt.WriteString(fmt.Sprintf("你是 %s，一个虚拟角色。", character.Name))
		prompt.WriteString("\n\n")
	}

	// 添加角色描述
	if character.Description != "" {
		prompt.WriteString(fmt.Sprintf("角色描述：%s\n\n", character.Description))
	}

	// 添加性格特征
	if character.Personality != "" {
		prompt.WriteString(fmt.Sprintf("性格特征：%s\n\n", character.Personality))
	}

	// 添加背景故事
	if character.Background != "" {
		prompt.WriteString(fmt.Sprintf("背景故事：%s\n\n", character.Background))
	}

	// 添加角色属性（性格、穿着、习惯、目标等）
	if character.Appearance != "" {
		prompt.WriteString(fmt.Sprintf("外貌特征：%s\n\n", character.Appearance))
	}
	if character.DressPreference != "" {
		prompt.WriteString(fmt.Sprintf("穿着偏好：%s\n\n", character.DressPreference))
	}
	if character.ShortTermGoal != "" {
		prompt.WriteString(fmt.Sprintf("短期目标：%s\n\n", character.ShortTermGoal))
	}
	if character.LongTermGoal != "" {
		prompt.WriteString(fmt.Sprintf("长期目标：%s\n\n", character.LongTermGoal))
	}
	if character.HandlingStyle != "" {
		prompt.WriteString(fmt.Sprintf("处事风格：%s\n\n", character.HandlingStyle))
	}

	// 添加技能和特质
	if len(character.Traits) > 0 {
		traitsJSON, _ := json.Marshal(character.Traits)
		prompt.WriteString(fmt.Sprintf("特质：%s\n", string(traitsJSON)))
	}
	if len(character.Skills) > 0 {
		skillsJSON, _ := json.Marshal(character.Skills)
		prompt.WriteString(fmt.Sprintf("技能：%s\n", string(skillsJSON)))
	}

	// 添加故事板上下文记忆（最近5个参与的故事板）
	if len(storyboards) > 0 {
		prompt.WriteString("\n## 最近的剧情记忆（按时间顺序）：\n\n")
		for i, sb := range storyboards {
			prompt.WriteString(fmt.Sprintf("### 剧情片段 %d: %s\n", i+1, sb.Title))
			if sb.Content != "" {
				prompt.WriteString(fmt.Sprintf("内容：%s\n", sb.Content))
			} else if sb.RawInput != "" {
				prompt.WriteString(fmt.Sprintf("内容：%s\n", sb.RawInput))
			}
			prompt.WriteString("\n")
		}
		prompt.WriteString("以上是你最近经历的故事剧情，请结合这些记忆来理解当前对话的上下文。\n\n")
	}

	prompt.WriteString("\n请以这个角色的身份和语气进行对话，保持角色一致性。")

	return prompt.String()
}

// getOrCreateAgent 获取或创建角色的Agent
func (s *AgentChatService) getOrCreateAgent(ctx context.Context, character *domain.Character) (*domain.Agent, error) {
	// 尝试获取现有Agent
	agent, err := s.repo.GetAgentByCharacterID(ctx, character.ID)
	if err == nil {
		return agent, nil
	}

	// Agent不存在，创建新的
	if err == domain.ErrNotFound {
		agent = &domain.Agent{
			CharacterID:  character.ID,
			Name:         character.Name,
			Description:  character.Description,
			Status:       domain.AgentStatusActive,
			SystemPrompt: s.generateDefaultSystemPrompt(character),
			Temperature:  0.7,
			Provider:     "", // 使用默认提供商
			Model:        "", // 使用默认模型
			MaxTokens:    2048,
		}

		if err := s.repo.CreateAgent(ctx, agent); err != nil {
			return nil, fmt.Errorf("failed to create agent: %w", err)
		}

		s.logger.Info("created new agent for character",
			zap.String("agentId", agent.ID),
			zap.String("characterId", character.ID))

		return agent, nil
	}

	return nil, err
}

// generateDefaultSystemPrompt 生成默认的系统提示词
func (s *AgentChatService) generateDefaultSystemPrompt(character *domain.Character) string {
	return fmt.Sprintf("你是 %s。%s 请以这个角色的身份进行对话。",
		character.Name,
		character.Description)
}

// createChatThreadForCharacter 为角色创建聊天线程
func (s *AgentChatService) createChatThreadForCharacter(ctx context.Context, userID, characterID string) (*domain.ChatThread, error) {
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("character not found: %w", err)
	}

	thread := &domain.ChatThread{
		UserID:               userID,
		CharacterID:          characterID,
		CharacterName:        character.Name,
		CharacterAvatar:      character.Avatar,
		StoryTitle:           "",
		LastMessage:          "",
		LastMessageTime:      time.Now().Unix(),
		UnreadCount:          0,
		MessageCount:         0,
		InteractionFrequency: 0,
		CreatedAt:            time.Now().Unix(),
	}

	if err := s.repo.CreateChatThread(ctx, thread); err != nil {
		return nil, err
	}

	s.logger.Info("created new chat thread",
		zap.String("threadId", thread.ID),
		zap.String("userId", userID),
		zap.String("characterId", characterID))

	return thread, nil
}

// GetChatHistory 获取聊天历史
func (s *AgentChatService) GetChatHistory(ctx context.Context, userID, threadID string, limit, offset int) ([]*domain.ChatMessage, error) {
	s.logger.Info("getting chat history",
		zap.String("userId", userID),
		zap.String("threadId", threadID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// 参数验证
	if threadID == "" {
		s.logger.Error("thread ID is empty")
		return nil, fmt.Errorf("thread ID is required")
	}
	if userID == "" {
		s.logger.Error("user ID is empty")
		return nil, fmt.Errorf("user ID is required")
	}
	if limit <= 0 {
		limit = 20 // 默认值
	}
	if limit > 100 {
		limit = 100 // 最大限制
	}
	if offset < 0 {
		offset = 0
	}

	// 验证用户权限：确认该聊天线程属于该用户
	thread, err := s.repo.ChatThreadByID(ctx, threadID)
	if err != nil {
		s.logger.Error("failed to get chat thread",
			zap.String("threadId", threadID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get chat thread: %w", err)
	}

	if thread.UserID != userID {
		s.logger.Warn("user attempting to access unauthorized thread",
			zap.String("userId", userID),
			zap.String("threadId", threadID),
			zap.String("threadOwnerId", thread.UserID))
		return nil, fmt.Errorf("access denied: thread does not belong to user")
	}

	// 获取聊天消息
	messages, err := s.repo.ChatMessages(ctx, threadID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get chat messages",
			zap.String("threadId", threadID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get chat messages: %w", err)
	}

	s.logger.Info("chat history retrieved successfully",
		zap.String("threadId", threadID),
		zap.Int("messageCount", len(messages)))

	return messages, nil
}

// ListUserChatThreads 列出用户的聊天线程
func (s *AgentChatService) ListUserChatThreads(ctx context.Context, userID string) ([]*domain.ChatThread, error) {
	s.logger.Info("listing user chat threads",
		zap.String("userId", userID))

	// 参数验证
	if userID == "" {
		s.logger.Error("user ID is empty")
		return nil, fmt.Errorf("user ID is required")
	}

	// 验证用户是否存在
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil || user == nil {
		s.logger.Error("user not found",
			zap.String("userId", userID),
			zap.Error(err))
		return nil, fmt.Errorf("user not found")
	}

	// 获取聊天线程列表
	threads, err := s.repo.ChatThreads(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get chat threads",
			zap.String("userId", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get chat threads: %w", err)
	}

	s.logger.Info("chat threads retrieved successfully",
		zap.String("userId", userID),
		zap.Int("threadCount", len(threads)))

	return threads, nil
}

// UpdateAgentConfig 更新Agent配置
func (s *AgentChatService) UpdateAgentConfig(ctx context.Context, agentID string, config *domain.Agent) error {
	s.logger.Info("updating agent config",
		zap.String("agentId", agentID),
		zap.String("configId", config.ID))

	// 参数验证
	if agentID == "" {
		s.logger.Error("agent ID is empty")
		return fmt.Errorf("agent ID is required")
	}
	if config == nil {
		s.logger.Error("config is nil")
		return fmt.Errorf("config is required")
	}

	// 确保 agentID 与 config.ID 一致
	if config.ID == "" {
		config.ID = agentID
	} else if config.ID != agentID {
		s.logger.Error("agent ID mismatch",
			zap.String("agentId", agentID),
			zap.String("configId", config.ID))
		return fmt.Errorf("agent ID mismatch: expected %s, got %s", agentID, config.ID)
	}

	// 验证配置参数
	if config.Temperature < 0 || config.Temperature > 2 {
		s.logger.Error("invalid temperature value",
			zap.Float64("temperature", config.Temperature))
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if config.MaxTokens <= 0 || config.MaxTokens > 8192 {
		s.logger.Error("invalid max tokens value",
			zap.Int("maxTokens", config.MaxTokens))
		return fmt.Errorf("max tokens must be between 1 and 8192")
	}

	// 更新配置
	if err := s.repo.UpdateAgent(ctx, config); err != nil {
		s.logger.Error("failed to update agent",
			zap.String("agentId", agentID),
			zap.Error(err))
		return fmt.Errorf("failed to update agent: %w", err)
	}

	s.logger.Info("agent config updated successfully",
		zap.String("agentId", agentID),
		zap.Float64("temperature", config.Temperature),
		zap.Int("maxTokens", config.MaxTokens))

	return nil
}

// GetAgentByCharacter 获取角色的Agent
func (s *AgentChatService) GetAgentByCharacter(ctx context.Context, characterID string) (*domain.Agent, error) {
	s.logger.Info("getting agent by character",
		zap.String("characterId", characterID))

	// 参数验证
	if characterID == "" {
		s.logger.Error("character ID is empty")
		return nil, fmt.Errorf("character ID is required")
	}

	// 验证角色是否存在
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		s.logger.Error("failed to get character",
			zap.String("characterId", characterID),
			zap.Error(err))
		return nil, fmt.Errorf("character not found: %w", err)
	}

	// 获取Agent
	agent, err := s.repo.GetAgentByCharacterID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Info("agent not found for character, will need to create",
				zap.String("characterId", characterID),
				zap.String("characterName", character.Name))
			return nil, fmt.Errorf("agent not found for character %s: %w", characterID, err)
		}
		s.logger.Error("failed to get agent by character",
			zap.String("characterId", characterID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	s.logger.Info("agent retrieved successfully",
		zap.String("agentId", agent.ID),
		zap.String("characterId", characterID),
		zap.String("characterName", character.Name))

	return agent, nil
}

// CreateChatThread 创建新的聊天线程
func (s *AgentChatService) CreateChatThread(ctx context.Context, userID, characterID string) (*domain.ChatThread, error) {
	s.logger.Info("creating chat thread",
		zap.String("userId", userID),
		zap.String("characterId", characterID))

	// 参数验证
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if characterID == "" {
		return nil, fmt.Errorf("character ID is required")
	}

	// 验证用户是否存在
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil || user == nil {
		s.logger.Error("user not found", zap.String("userId", userID))
		return nil, fmt.Errorf("user not found")
	}

	// 验证角色是否存在
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		s.logger.Error("character not found", zap.String("characterId", characterID))
		return nil, fmt.Errorf("character not found: %w", err)
	}

	// 创建新的聊天线程
	thread, err := s.createChatThreadForCharacter(ctx, userID, characterID)
	if err != nil {
		s.logger.Error("failed to create chat thread", zap.Error(err))
		return nil, fmt.Errorf("failed to create chat thread: %w", err)
	}

	// 填充角色信息
	thread.CharacterName = character.Name
	thread.CharacterAvatar = character.Avatar

	s.logger.Info("chat thread created successfully",
		zap.String("threadId", thread.ID),
		zap.String("characterId", characterID))

	return thread, nil
}

// generateThreadSummary 使用AI生成会话摘要
func (s *AgentChatService) generateThreadSummary(ctx context.Context, thread *domain.ChatThread) (string, error) {
	// 获取会话中的消息
	messages, err := s.repo.ChatMessages(ctx, thread.ID, 50, 0)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}

	if len(messages) == 0 {
		return "Empty conversation", nil
	}

	// 构建对话内容
	var conversationParts []string
	for _, msg := range messages {
		role := "User"
		if !msg.IsUser {
			role = thread.CharacterName
		}
		conversationParts = append(conversationParts, fmt.Sprintf("%s: %s", role, msg.Content))
	}

	// 使用AI生成摘要
	if s.geminiClient == nil {
		// 如果没有AI客户端，返回简单摘要
		if len(messages) > 0 {
			return fmt.Sprintf("Conversation with %d messages about: %s...", len(messages), truncateString(messages[len(messages)-1].Content, 50)), nil
		}
		return "Conversation ended", nil
	}

	prompt := fmt.Sprintf(`Please summarize the following conversation between a user and %s in 1-2 sentences. Focus on the main topic or theme discussed.

Conversation:
%s

Summary:`, thread.CharacterName, strings.Join(conversationParts, "\n"))

	// 生成摘要
	summaryText, _, err := s.geminiClient.GenerateText(ctx, "gemini-2.0-flash", prompt, nil)
	if err != nil {
		s.logger.Warn("failed to generate summary with AI", zap.Error(err))
		return fmt.Sprintf("Conversation with %d messages", len(messages)), nil
	}

	result := strings.TrimSpace(summaryText)
	if result == "" {
		return fmt.Sprintf("Conversation with %d messages", len(messages)), nil
	}

	return result, nil
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
