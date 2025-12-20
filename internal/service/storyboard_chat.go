package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// StoryboardChatService handles storyboard creation chat workflow
type StoryboardChatService struct {
	repo           domain.Repository
	storyService   *Service
	logger         *zap.Logger
}

// NewStoryboardChatService creates a new storyboard chat service
func NewStoryboardChatService(repo domain.Repository, storyService *Service, logger *zap.Logger) *StoryboardChatService {
	return &StoryboardChatService{
		repo:         repo,
		storyService: storyService,
		logger:       logger,
	}
}

// StartSession starts a new storyboard creation chat session
func (s *StoryboardChatService) StartSession(ctx context.Context, userID, storyID string) (*domain.StoryboardChatSession, *domain.StoryboardChatMessage, error) {
	s.logger.Info("starting storyboard chat session",
		zap.String("userID", userID),
		zap.String("storyID", storyID))

	// Check if there's an active session for this user and story
	existingSession, err := s.repo.GetActiveStoryboardChatSession(ctx, userID, storyID)
	if err != nil {
		s.logger.Error("failed to check existing session", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to check existing session: %w", err)
	}

	if existingSession != nil {
		s.logger.Info("returning existing active session",
			zap.String("sessionID", existingSession.ID))
		// Get the last message
		lastMsg, err := s.repo.GetLastStoryboardChatMessage(ctx, existingSession.ID)
		if err != nil {
			s.logger.Warn("failed to get last message", zap.Error(err))
		}
		return existingSession, lastMsg, nil
	}

	// Verify story exists
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		s.logger.Error("story not found", zap.String("storyID", storyID), zap.Error(err))
		return nil, nil, fmt.Errorf("story not found: %w", err)
	}

	// Create new session
	session := &domain.StoryboardChatSession{
		UserID:      userID,
		StoryID:     storyID,
		CurrentStep: domain.StepCharacterSelection,
		Status:      domain.SessionStatusActive,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	if err := s.repo.CreateStoryboardChatSession(ctx, session); err != nil {
		s.logger.Error("failed to create session", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	session.Story = story

	// Create initial character selection message
	msg, err := s.createCharacterSelectionMessage(ctx, session)
	if err != nil {
		s.logger.Error("failed to create character selection message", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to create initial message: %w", err)
	}

	s.logger.Info("storyboard chat session started",
		zap.String("sessionID", session.ID),
		zap.String("storyID", storyID))

	return session, msg, nil
}

// GetSession retrieves a session by ID
func (s *StoryboardChatService) GetSession(ctx context.Context, sessionID string) (*domain.StoryboardChatSession, error) {
	return s.repo.GetStoryboardChatSession(ctx, sessionID)
}

// GetMessages retrieves messages for a session
func (s *StoryboardChatService) GetMessages(ctx context.Context, sessionID string, limit, offset int) ([]*domain.StoryboardChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListStoryboardChatMessages(ctx, sessionID, limit, offset)
}

// ListSessions lists user's storyboard chat sessions
func (s *StoryboardChatService) ListSessions(ctx context.Context, userID string, limit, offset int) ([]*domain.StoryboardChatSession, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListStoryboardChatSessions(ctx, userID, limit, offset)
}

// SendMessage handles user input and returns the next message
func (s *StoryboardChatService) SendMessage(ctx context.Context, sessionID string, req *domain.SendMessageRequest) (*domain.StoryboardChatMessage, error) {
	s.logger.Info("processing storyboard chat message",
		zap.String("sessionID", sessionID),
		zap.String("actionID", req.ActionID))

	// Get current session
	session, err := s.repo.GetStoryboardChatSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if session.Status != domain.SessionStatusActive {
		return nil, fmt.Errorf("session is not active")
	}

	// Save user message if there's content
	if req.Content != "" {
		userMsg := &domain.StoryboardChatMessage{
			SessionID:   sessionID,
			MessageType: domain.MsgTypeUserInput,
			Status:      "sent",
			Step:        session.CurrentStep,
			Content:     req.Content,
			IsUser:      true,
			Timestamp:   time.Now().Unix(),
		}
		if err := s.repo.CreateStoryboardChatMessage(ctx, userMsg); err != nil {
			s.logger.Warn("failed to save user message", zap.Error(err))
		}
	}

	// Route to appropriate handler based on current step and action
	var responseMsg *domain.StoryboardChatMessage

	switch session.CurrentStep {
	case domain.StepCharacterSelection:
		responseMsg, err = s.handleCharacterSelection(ctx, session, req)
	case domain.StepContentGeneration:
		responseMsg, err = s.handleContentConfirmation(ctx, session, req)
	case domain.StepImagePrompts:
		responseMsg, err = s.handleImagePromptConfirmation(ctx, session, req)
	case domain.StepImageGeneration:
		responseMsg, err = s.handleImageGenerationComplete(ctx, session, req)
	case domain.StepVideoChoice:
		responseMsg, err = s.handleVideoChoice(ctx, session, req)
	case domain.StepVideoGeneration:
		responseMsg, err = s.handleVideoGenerationStatus(ctx, session, req)
	case domain.StepCompletion:
		responseMsg, err = s.handlePublishChoice(ctx, session, req)
	default:
		return nil, fmt.Errorf("unknown step: %d", session.CurrentStep)
	}

	if err != nil {
		s.logger.Error("failed to handle message",
			zap.Int("step", session.CurrentStep),
			zap.Error(err))
		return s.createErrorMessage(ctx, session, err)
	}

	return responseMsg, nil
}

// GetGenerationStatus returns the current generation status for polling
func (s *StoryboardChatService) GetGenerationStatus(ctx context.Context, sessionID string) (*domain.StoryboardChatMessage, error) {
	session, err := s.repo.GetStoryboardChatSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if session.StoryboardID == "" {
		return nil, fmt.Errorf("no storyboard in session")
	}

	// Get generation progress from storyboard service
	progress, err := s.storyService.GetGenerationProgress(ctx, session.StoryboardID)
	if err != nil {
		s.logger.Warn("failed to get generation progress", zap.Error(err))
	}

	// Create status message based on current step
	switch session.CurrentStep {
	case domain.StepImageGeneration:
		return s.createImageGenerationStatusMessage(ctx, session, progress)
	case domain.StepVideoGeneration:
		return s.createVideoGenerationStatusMessage(ctx, session, progress)
	default:
		// Return last message
		return s.repo.GetLastStoryboardChatMessage(ctx, sessionID)
	}
}

// ========== Step Handlers ==========

func (s *StoryboardChatService) handleCharacterSelection(ctx context.Context, session *domain.StoryboardChatSession, req *domain.SendMessageRequest) (*domain.StoryboardChatMessage, error) {
	if req.ActionID != "confirm" {
		return nil, fmt.Errorf("invalid action for character selection: %s", req.ActionID)
	}

	// Parse character selection input
	var input domain.CharacterSelectionInput
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &input); err != nil {
			return nil, fmt.Errorf("invalid character selection data: %w", err)
		}
	}

	if len(input.CharacterIDs) == 0 {
		return nil, fmt.Errorf("at least one character must be selected")
	}

	// Update session with selected characters
	session.SelectedCharacters = input.CharacterIDs
	session.SelectedStyle = input.StyleID
	session.CurrentStep = domain.StepContentGeneration
	session.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Create storyboard with initial data
	storyboard := &domain.Storyboard{
		ID:             uuid.New().String(),
		StoryID:        session.StoryID,
		ParentID:       domain.StoryboardRootMarker,
		CreatorID:      session.UserID,
		Title:          "New Storyboard",
		RawInput:       req.Content,
		WorkflowStatus: domain.WorkflowStatusDraft,
		CurrentStep:    1,
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}

	// Add character references
	for i, charID := range input.CharacterIDs {
		storyboard.CharacterRefs = append(storyboard.CharacterRefs, domain.StoryboardCharacterRef{
			CharacterID: charID,
			Order:       i,
		})
	}

	if err := s.storyService.CreateStoryboard(ctx, storyboard); err != nil {
		return nil, fmt.Errorf("failed to create storyboard: %w", err)
	}

	// Update session with storyboard ID
	session.StoryboardID = storyboard.ID
	if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
		s.logger.Warn("failed to update session with storyboard ID", zap.Error(err))
	}

	// Create processing message
	processingMsg := s.createProcessingMessage(ctx, session, "正在根据您的选择生成故事内容...")

	// Start content generation asynchronously
	go s.generateStoryContent(context.Background(), session, storyboard, req.Content)

	return processingMsg, nil
}

func (s *StoryboardChatService) handleContentConfirmation(ctx context.Context, session *domain.StoryboardChatSession, req *domain.SendMessageRequest) (*domain.StoryboardChatMessage, error) {
	switch req.ActionID {
	case "confirm":
		// User confirmed the content, move to image prompts
		session.CurrentStep = domain.StepImagePrompts
		session.UpdatedAt = time.Now().Unix()

		if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}

		// Generate image prompts
		return s.createImagePromptsMessage(ctx, session)

	case "edit":
		// Parse edited content
		var input domain.ContentConfirmationInput
		if len(req.Data) > 0 {
			if err := json.Unmarshal(req.Data, &input); err != nil {
				return nil, fmt.Errorf("invalid content edit data: %w", err)
			}
		}

		// Update storyboard with edited content
		if session.StoryboardID != "" {
			storyboard, err := s.repo.StoryboardByID(ctx, session.StoryboardID)
			if err != nil {
				return nil, fmt.Errorf("storyboard not found: %w", err)
			}

			if input.Title != "" {
				storyboard.Title = input.Title
			}
			if input.Content != "" {
				storyboard.Content = input.Content
			}

			// Update scenes if provided
			if len(input.Scenes) > 0 {
				for i, scene := range input.Scenes {
					if i < len(storyboard.StoryboardScenes) {
						storyboard.StoryboardScenes[i].Title = scene.Title
						storyboard.StoryboardScenes[i].Description = scene.Description
					}
				}
			}

			if err := s.repo.UpdateStoryboard(ctx, storyboard); err != nil {
				return nil, fmt.Errorf("failed to update storyboard: %w", err)
			}
		}

		// Return updated content message
		return s.createStoryContentMessage(ctx, session)

	default:
		return nil, fmt.Errorf("invalid action: %s", req.ActionID)
	}
}

func (s *StoryboardChatService) handleImagePromptConfirmation(ctx context.Context, session *domain.StoryboardChatSession, req *domain.SendMessageRequest) (*domain.StoryboardChatMessage, error) {
	switch req.ActionID {
	case "confirm":
		// User confirmed prompts, start image generation
		session.CurrentStep = domain.StepImageGeneration
		session.UpdatedAt = time.Now().Unix()

		if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}

		// Create processing message
		processingMsg := s.createProcessingMessage(ctx, session, "正在生成场景图片，请稍候...")

		// Start image generation asynchronously
		go s.generateSceneImages(context.Background(), session)

		return processingMsg, nil

	case "edit":
		// Parse edited prompts
		var input domain.ImagePromptConfirmationInput
		if len(req.Data) > 0 {
			if err := json.Unmarshal(req.Data, &input); err != nil {
				return nil, fmt.Errorf("invalid prompt edit data: %w", err)
			}
		}

		// Update scene prompts (stored in generation records)
		// For now, just return the updated prompts message
		return s.createImagePromptsMessage(ctx, session)

	default:
		return nil, fmt.Errorf("invalid action: %s", req.ActionID)
	}
}

func (s *StoryboardChatService) handleImageGenerationComplete(ctx context.Context, session *domain.StoryboardChatSession, req *domain.SendMessageRequest) (*domain.StoryboardChatMessage, error) {
	// Check if images are ready
	if session.StoryboardID == "" {
		return nil, fmt.Errorf("no storyboard in session")
	}

	progress, err := s.storyService.GetGenerationProgress(ctx, session.StoryboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	// Check if all images are complete
	allComplete := true
	if progress.ImageGenerations != nil {
		for _, img := range progress.ImageGenerations {
			if img.Status != domain.GenerationStatusCompleted {
				allComplete = false
				break
			}
		}
	}

	if !allComplete {
		// Still generating, return status message
		return s.createImageGenerationStatusMessage(ctx, session, progress)
	}

	// Images complete, move to video choice
	session.CurrentStep = domain.StepVideoChoice
	session.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return s.createImagesResultMessage(ctx, session)
}

func (s *StoryboardChatService) handleVideoChoice(ctx context.Context, session *domain.StoryboardChatSession, req *domain.SendMessageRequest) (*domain.StoryboardChatMessage, error) {
	switch req.ActionID {
	case "generate_video":
		// User wants to generate video
		session.CurrentStep = domain.StepVideoGeneration
		session.UpdatedAt = time.Now().Unix()

		if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}

		// Create processing message
		processingMsg := s.createProcessingMessage(ctx, session, "正在生成视频，这可能需要几分钟时间...")

		// Start video generation asynchronously
		go s.generateSceneVideos(context.Background(), session)

		return processingMsg, nil

	case "skip_video":
		// User skipped video, go to completion
		session.CurrentStep = domain.StepCompletion
		session.UpdatedAt = time.Now().Unix()

		if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}

		return s.createCompletionMessage(ctx, session, false)

	default:
		return nil, fmt.Errorf("invalid action: %s", req.ActionID)
	}
}

func (s *StoryboardChatService) handleVideoGenerationStatus(ctx context.Context, session *domain.StoryboardChatSession, req *domain.SendMessageRequest) (*domain.StoryboardChatMessage, error) {
	if session.StoryboardID == "" {
		return nil, fmt.Errorf("no storyboard in session")
	}

	progress, err := s.storyService.GetGenerationProgress(ctx, session.StoryboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	// Check if all videos are complete
	allComplete := true
	if progress.VideoGenerations != nil {
		for _, vid := range progress.VideoGenerations {
			if vid.Status != domain.GenerationStatusCompleted {
				allComplete = false
				break
			}
		}
	}

	if !allComplete {
		// Still generating, return status message
		return s.createVideoGenerationStatusMessage(ctx, session, progress)
	}

	// Videos complete, move to completion
	session.CurrentStep = domain.StepCompletion
	session.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return s.createCompletionMessage(ctx, session, true)
}

func (s *StoryboardChatService) handlePublishChoice(ctx context.Context, session *domain.StoryboardChatSession, req *domain.SendMessageRequest) (*domain.StoryboardChatMessage, error) {
	switch req.ActionID {
	case "publish":
		// Publish the storyboard
		if session.StoryboardID != "" {
			if err := s.storyService.PublishStoryboard(ctx, session.StoryboardID); err != nil {
				return nil, fmt.Errorf("failed to publish: %w", err)
			}
		}

		// Mark session as completed
		session.Status = domain.SessionStatusCompleted
		session.UpdatedAt = time.Now().Unix()

		if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
			s.logger.Warn("failed to update session status", zap.Error(err))
		}

		// Create success message
		return s.createPublishSuccessMessage(ctx, session, true)

	case "draft":
		// Save as draft
		session.Status = domain.SessionStatusCompleted
		session.UpdatedAt = time.Now().Unix()

		if err := s.repo.UpdateStoryboardChatSession(ctx, session); err != nil {
			s.logger.Warn("failed to update session status", zap.Error(err))
		}

		return s.createPublishSuccessMessage(ctx, session, false)

	default:
		return nil, fmt.Errorf("invalid action: %s", req.ActionID)
	}
}

// ========== Message Creators ==========

func (s *StoryboardChatService) createCharacterSelectionMessage(ctx context.Context, session *domain.StoryboardChatSession) (*domain.StoryboardChatMessage, error) {
	// Get characters for this story
	characters, err := s.repo.CharactersByStory(ctx, session.StoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get characters: %w", err)
	}

	// Convert to selection items
	charItems := make([]domain.CharacterSelectionItem, len(characters))
	for i, c := range characters {
		charItems[i] = domain.CharacterSelectionItem{
			ID:          c.ID,
			Name:        c.Name,
			Avatar:      c.Avatar,
			Description: c.Description,
		}
	}

	// Get available styles
	styles := []domain.StyleOption{
		{ID: "summer_sunshine", Name: "夏日阳光", Description: "温暖明亮的夏日风格"},
		{ID: "dark_mystery", Name: "黑暗神秘", Description: "神秘深邃的暗色调"},
		{ID: "watercolor", Name: "水彩画风", Description: "柔和的水彩画效果"},
		{ID: "anime", Name: "动漫风格", Description: "日系动漫画风"},
		{ID: "realistic", Name: "写实风格", Description: "接近真实的画面效果"},
	}

	data := domain.CharacterSelectionData{
		Prompt:       "请选择参与这个故事的角色",
		Characters:   charItems,
		MinSelection: 1,
		MaxSelection: 5,
		Styles:       styles,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeCharacterSelection,
		Status:      "pending",
		Step:        domain.StepCharacterSelection,
		Data:        dataBytes,
		Actions: []domain.ChatAction{
			{ID: "confirm", Label: "确认选择", Type: "primary"},
		},
		Timestamp: time.Now().Unix(),
		IsUser:    false,
	}

	if err := s.repo.CreateStoryboardChatMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return msg, nil
}

func (s *StoryboardChatService) createProcessingMessage(ctx context.Context, session *domain.StoryboardChatSession, message string) *domain.StoryboardChatMessage {
	data := domain.ProcessingData{
		Message:  message,
		Progress: 0,
	}

	dataBytes, _ := json.Marshal(data)

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeProcessing,
		Status:      "processing",
		Step:        session.CurrentStep,
		Data:        dataBytes,
		Timestamp:   time.Now().Unix(),
		IsUser:      false,
	}

	if err := s.repo.CreateStoryboardChatMessage(ctx, msg); err != nil {
		s.logger.Warn("failed to create processing message", zap.Error(err))
	}

	return msg
}

func (s *StoryboardChatService) createStoryContentMessage(ctx context.Context, session *domain.StoryboardChatSession) (*domain.StoryboardChatMessage, error) {
	if session.StoryboardID == "" {
		return nil, fmt.Errorf("no storyboard in session")
	}

	storyboard, err := s.repo.StoryboardByID(ctx, session.StoryboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storyboard: %w", err)
	}

	// Get scenes
	scenes, _ := s.repo.StoryboardScenes(ctx, session.StoryboardID)

	sceneItems := make([]domain.SceneContentItem, len(scenes))
	for i, scene := range scenes {
		sceneItems[i] = domain.SceneContentItem{
			ID:          scene.ID,
			Sequence:    scene.Sequence,
			Title:       scene.Title,
			Description: scene.Description,
			Location:    scene.Location,
			TimeOfDay:   scene.TimeOfDay,
			Characters:  scene.Characters,
			Mood:        scene.Mood,
		}
	}

	data := domain.StoryContentData{
		Title:    storyboard.Title,
		Content:  storyboard.Content,
		Scenes:   sceneItems,
		Editable: true,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeStoryContent,
		Status:      "completed",
		Step:        domain.StepContentGeneration,
		Data:        dataBytes,
		Actions: []domain.ChatAction{
			{ID: "edit", Label: "编辑", Type: "secondary"},
			{ID: "confirm", Label: "确认", Type: "primary"},
		},
		Timestamp: time.Now().Unix(),
		IsUser:    false,
	}

	if err := s.repo.CreateStoryboardChatMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return msg, nil
}

func (s *StoryboardChatService) createImagePromptsMessage(ctx context.Context, session *domain.StoryboardChatSession) (*domain.StoryboardChatMessage, error) {
	if session.StoryboardID == "" {
		return nil, fmt.Errorf("no storyboard in session")
	}

	// Get scenes
	scenes, err := s.repo.StoryboardScenes(ctx, session.StoryboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenes: %w", err)
	}

	// Generate prompts for each scene (or get existing ones)
	scenePrompts := make([]domain.ScenePromptItem, len(scenes))
	for i, scene := range scenes {
		// Generate a prompt from the scene description
		prompt := s.generateImagePrompt(scene)
		scenePrompts[i] = domain.ScenePromptItem{
			ID:          scene.ID,
			Sequence:    scene.Sequence,
			Title:       scene.Title,
			Description: scene.Description,
			Prompt:      prompt,
			Editable:    true,
		}
	}

	data := domain.ImagePromptsData{
		Scenes:   scenePrompts,
		Editable: true,
		Style:    session.SelectedStyle,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeImagePrompts,
		Status:      "completed",
		Step:        domain.StepImagePrompts,
		Data:        dataBytes,
		Actions: []domain.ChatAction{
			{ID: "edit", Label: "编辑提示词", Type: "secondary"},
			{ID: "confirm", Label: "开始生成图片", Type: "primary"},
		},
		Timestamp: time.Now().Unix(),
		IsUser:    false,
	}

	if err := s.repo.CreateStoryboardChatMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return msg, nil
}

func (s *StoryboardChatService) createImageGenerationStatusMessage(ctx context.Context, session *domain.StoryboardChatSession, progress *domain.StoryboardGenerationProgress) (*domain.StoryboardChatMessage, error) {
	completedCount := 0
	totalCount := 0

	if progress != nil && progress.ImageGenerations != nil {
		totalCount = len(progress.ImageGenerations)
		for _, img := range progress.ImageGenerations {
			if img.Status == domain.GenerationStatusCompleted {
				completedCount++
			}
		}
	}

	progressPercent := 0
	if totalCount > 0 {
		progressPercent = (completedCount * 100) / totalCount
	}

	data := domain.VideoProcessingData{
		Message:       fmt.Sprintf("正在生成图片 (%d/%d)", completedCount, totalCount),
		Progress:      progressPercent,
		CurrentScene:  completedCount + 1,
		TotalScenes:   totalCount,
		EstimatedTime: fmt.Sprintf("约 %d 秒", (totalCount-completedCount)*10),
	}

	dataBytes, _ := json.Marshal(data)

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeProcessing,
		Status:      "processing",
		Step:        domain.StepImageGeneration,
		Data:        dataBytes,
		Timestamp:   time.Now().Unix(),
		IsUser:      false,
	}

	return msg, nil
}

func (s *StoryboardChatService) createImagesResultMessage(ctx context.Context, session *domain.StoryboardChatSession) (*domain.StoryboardChatMessage, error) {
	if session.StoryboardID == "" {
		return nil, fmt.Errorf("no storyboard in session")
	}

	storyboard, err := s.repo.StoryboardByID(ctx, session.StoryboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storyboard: %w", err)
	}

	// Get scenes with images
	scenes, _ := s.repo.StoryboardScenes(ctx, session.StoryboardID)

	sceneResults := make([]domain.SceneImageResultItem, len(scenes))
	for i, scene := range scenes {
		sceneResults[i] = domain.SceneImageResultItem{
			ID:          scene.ID,
			Sequence:    scene.Sequence,
			Title:       scene.Title,
			Description: scene.Description,
			ImageURL:    scene.Image,
			Prompt:      "", // Could retrieve from generation record
		}
	}

	// Get characters
	charItems := make([]domain.CharacterSelectionItem, 0)
	for _, charRef := range storyboard.CharacterRefs {
		if char, err := s.repo.CharacterByID(ctx, charRef.CharacterID); err == nil {
			charItems = append(charItems, domain.CharacterSelectionItem{
				ID:     char.ID,
				Name:   char.Name,
				Avatar: char.Avatar,
			})
		}
	}

	data := domain.ImagesResultData{
		StoryboardID: storyboard.ID,
		Title:        storyboard.Title,
		Style:        session.SelectedStyle,
		Scenes:       sceneResults,
		Characters:   charItems,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeImagesResult,
		Status:      "completed",
		Step:        domain.StepVideoChoice,
		Data:        dataBytes,
		Actions: []domain.ChatAction{
			{ID: "generate_video", Label: "生成视频", Type: "primary"},
			{ID: "skip_video", Label: "跳过视频", Type: "secondary"},
		},
		Timestamp: time.Now().Unix(),
		IsUser:    false,
	}

	if err := s.repo.CreateStoryboardChatMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return msg, nil
}

func (s *StoryboardChatService) createVideoGenerationStatusMessage(ctx context.Context, session *domain.StoryboardChatSession, progress *domain.StoryboardGenerationProgress) (*domain.StoryboardChatMessage, error) {
	completedCount := 0
	totalCount := 0
	var sceneStatuses []domain.SceneVideoStatus

	if progress != nil && progress.VideoGenerations != nil {
		totalCount = len(progress.VideoGenerations)
		for _, vid := range progress.VideoGenerations {
			if vid.Status == domain.GenerationStatusCompleted {
				completedCount++
			}
			sceneStatuses = append(sceneStatuses, domain.SceneVideoStatus{
				ID:       vid.SceneID,
				Title:    vid.SceneTitle,
				Status:   vid.Status,
				VideoURL: vid.GeneratedVideoURL,
			})
		}
	}

	progressPercent := 0
	if totalCount > 0 {
		progressPercent = (completedCount * 100) / totalCount
	}

	data := domain.VideoProcessingData{
		Message:       fmt.Sprintf("正在生成视频 (%d/%d)", completedCount, totalCount),
		Progress:      progressPercent,
		CurrentScene:  completedCount + 1,
		TotalScenes:   totalCount,
		EstimatedTime: fmt.Sprintf("约 %d 分钟", (totalCount-completedCount)*2),
		SceneStatuses: sceneStatuses,
	}

	dataBytes, _ := json.Marshal(data)

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeVideoProcessing,
		Status:      "processing",
		Step:        domain.StepVideoGeneration,
		Data:        dataBytes,
		Timestamp:   time.Now().Unix(),
		IsUser:      false,
	}

	return msg, nil
}

func (s *StoryboardChatService) createCompletionMessage(ctx context.Context, session *domain.StoryboardChatSession, hasVideo bool) (*domain.StoryboardChatMessage, error) {
	if session.StoryboardID == "" {
		return nil, fmt.Errorf("no storyboard in session")
	}

	storyboard, err := s.repo.StoryboardByID(ctx, session.StoryboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storyboard: %w", err)
	}

	// Get scenes
	scenes, _ := s.repo.StoryboardScenes(ctx, session.StoryboardID)

	sceneItems := make([]domain.SceneCompletionItem, len(scenes))
	for i, scene := range scenes {
		sceneItems[i] = domain.SceneCompletionItem{
			ID:          scene.ID,
			Sequence:    scene.Sequence,
			Title:       scene.Title,
			Description: scene.Description,
			ImageURL:    scene.Image,
			VideoURL:    scene.VideoUrl,
		}
	}

	// Get characters
	charItems := make([]domain.CharacterSelectionItem, 0)
	for _, charRef := range storyboard.CharacterRefs {
		if char, err := s.repo.CharacterByID(ctx, charRef.CharacterID); err == nil {
			charItems = append(charItems, domain.CharacterSelectionItem{
				ID:     char.ID,
				Name:   char.Name,
				Avatar: char.Avatar,
			})
		}
	}

	// Get cover image (first scene image)
	coverImage := ""
	if len(scenes) > 0 && scenes[0].Image != "" {
		coverImage = scenes[0].Image
	}

	data := domain.CompletionData{
		StoryboardID:   storyboard.ID,
		Title:          storyboard.Title,
		Content:        storyboard.Content,
		CoverImage:     coverImage,
		Scenes:         sceneItems,
		Characters:     charItems,
		WorkflowStatus: storyboard.WorkflowStatus,
		HasVideo:       hasVideo,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeCompletion,
		Status:      "completed",
		Step:        domain.StepCompletion,
		Data:        dataBytes,
		Actions: []domain.ChatAction{
			{ID: "publish", Label: "发布", Type: "primary"},
			{ID: "draft", Label: "保存草稿", Type: "secondary"},
		},
		Timestamp: time.Now().Unix(),
		IsUser:    false,
	}

	if err := s.repo.CreateStoryboardChatMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return msg, nil
}

func (s *StoryboardChatService) createPublishSuccessMessage(ctx context.Context, session *domain.StoryboardChatSession, published bool) (*domain.StoryboardChatMessage, error) {
	message := "故事板已保存为草稿"
	if published {
		message = "故事板已成功发布！"
	}

	data := domain.CompletionData{
		StoryboardID: session.StoryboardID,
	}

	dataBytes, _ := json.Marshal(data)

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeCompletion,
		Status:      "completed",
		Step:        domain.StepCompletion,
		Data:        dataBytes,
		Content:     message,
		Timestamp:   time.Now().Unix(),
		IsUser:      false,
	}

	if err := s.repo.CreateStoryboardChatMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return msg, nil
}

func (s *StoryboardChatService) createErrorMessage(ctx context.Context, session *domain.StoryboardChatSession, err error) (*domain.StoryboardChatMessage, error) {
	data := domain.ErrorData{
		Code:      "error",
		Message:   err.Error(),
		Retryable: true,
	}

	dataBytes, _ := json.Marshal(data)

	msg := &domain.StoryboardChatMessage{
		SessionID:   session.ID,
		MessageType: domain.MsgTypeError,
		Status:      "error",
		Step:        session.CurrentStep,
		Data:        dataBytes,
		Actions: []domain.ChatAction{
			{ID: "retry", Label: "重试", Type: "primary"},
		},
		Timestamp: time.Now().Unix(),
		IsUser:    false,
	}

	if err := s.repo.CreateStoryboardChatMessage(ctx, msg); err != nil {
		s.logger.Warn("failed to create error message", zap.Error(err))
	}

	return msg, nil
}

// ========== Async Generation Functions ==========

func (s *StoryboardChatService) generateStoryContent(ctx context.Context, session *domain.StoryboardChatSession, storyboard *domain.Storyboard, userInput string) {
	s.logger.Info("starting async content generation",
		zap.String("sessionID", session.ID),
		zap.String("storyboardID", storyboard.ID))

	// Use existing storyboard service to generate content
	if err := s.storyService.GenerateStoryboardWithAI(ctx, storyboard); err != nil {
		s.logger.Error("content generation failed", zap.Error(err))
		s.createErrorMessage(ctx, session, err)
		return
	}

	// Create story content message
	if _, err := s.createStoryContentMessage(ctx, session); err != nil {
		s.logger.Error("failed to create story content message", zap.Error(err))
	}
}

func (s *StoryboardChatService) generateSceneImages(ctx context.Context, session *domain.StoryboardChatSession) {
	s.logger.Info("starting async image generation",
		zap.String("sessionID", session.ID),
		zap.String("storyboardID", session.StoryboardID))

	// Get scenes
	scenes, err := s.repo.StoryboardScenes(ctx, session.StoryboardID)
	if err != nil {
		s.logger.Error("failed to get scenes", zap.Error(err))
		return
	}

	// Generate images for each scene
	for _, scene := range scenes {
		req := &ImageGenerationRequest{
			StoryboardID:     session.StoryboardID,
			SceneID:          scene.ID,
			SceneTitle:       scene.Title,
			SceneDescription: scene.Description,
		}

		if _, err := s.storyService.GenerateSceneImage(ctx, req); err != nil {
			s.logger.Error("failed to start image generation",
				zap.String("sceneID", scene.ID),
				zap.Error(err))
		}
	}
}

func (s *StoryboardChatService) generateSceneVideos(ctx context.Context, session *domain.StoryboardChatSession) {
	s.logger.Info("starting async video generation",
		zap.String("sessionID", session.ID),
		zap.String("storyboardID", session.StoryboardID))

	// Get scenes with images
	scenes, err := s.repo.StoryboardScenes(ctx, session.StoryboardID)
	if err != nil {
		s.logger.Error("failed to get scenes", zap.Error(err))
		return
	}

	// Generate videos for each scene that has an image
	for _, scene := range scenes {
		if scene.Image == "" {
			continue
		}

		req := &VideoGenerationRequest{
			StoryboardID:      session.StoryboardID,
			SceneID:           scene.ID,
			SceneTitle:        scene.Title,
			InputDescription:  scene.Description,
			ReferenceImageURL: scene.Image,
		}

		if _, err := s.storyService.GenerateSceneVideo(ctx, req); err != nil {
			s.logger.Error("failed to start video generation",
				zap.String("sceneID", scene.ID),
				zap.Error(err))
		}
	}
}

// ========== Helper Functions ==========

func (s *StoryboardChatService) generateImagePrompt(scene *domain.StoryboardScene) string {
	// Generate a basic image prompt from scene data
	prompt := scene.Description
	if scene.Location != "" {
		prompt += ", 场景: " + scene.Location
	}
	if scene.TimeOfDay != "" {
		prompt += ", 时间: " + scene.TimeOfDay
	}
	if scene.Mood != "" {
		prompt += ", 氛围: " + scene.Mood
	}
	return prompt
}

