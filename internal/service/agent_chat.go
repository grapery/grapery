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

	// 4. 构建系统提示词和配置
	systemPrompt := s.buildSystemPrompt(agent, character)
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

	// 5. 构建完整的对话内容（包含历史）
	contents := append(conversationHistory, &genai.Content{
		Role:  genai.RoleUser,
		Parts: []*genai.Part{genai.NewPartFromText(userMessage.Content)},
	})

	// 6. 调用AI生成回复
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

	// 7. 计算 token 使用量
	tokensUsed := 0
	if geminiResp.UsageMetadata != nil {
		tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
	}

	// 8. 创建Agent回复消息
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

	// 9. 记录交互
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

	// 10. 更新Agent统计
	if err := s.repo.IncrementAgentInteractionCount(ctx, agent.ID); err != nil {
		s.logger.Warn("failed to increment agent interaction count", zap.Error(err))
	}

	// 11. 更新角色分析数据
	if err := s.repo.IncrementCharacterMessages(ctx, character.ID, 1); err != nil {
		s.logger.Warn("failed to increment character messages", zap.Error(err))
	}
	if err := s.repo.IncrementCharacterTokens(ctx, character.ID, int64(tokensUsed)); err != nil {
		s.logger.Warn("failed to increment character tokens", zap.Error(err))
	}

	s.logger.Info("agent reply generated",
		zap.String("messageId", agentReply.ID),
		zap.String("agentId", agent.ID),
		zap.Int("tokensUsed", tokensUsed),
		zap.Int64("duration", duration))

	return agentReply, nil
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
func (s *AgentChatService) buildSystemPrompt(agent *domain.Agent, character *domain.Character) string {
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

	// 添加技能和特质
	if len(character.Traits) > 0 {
		traitsJSON, _ := json.Marshal(character.Traits)
		prompt.WriteString(fmt.Sprintf("特质：%s\n", string(traitsJSON)))
	}
	if len(character.Skills) > 0 {
		skillsJSON, _ := json.Marshal(character.Skills)
		prompt.WriteString(fmt.Sprintf("技能：%s\n", string(skillsJSON)))
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
