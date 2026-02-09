package domain

// CollaborationStatus 故事协作状态
type CollaborationStatus string

const (
	CollaborationStatusOpen       CollaborationStatus = "open"       // 开放协作：任何人可以参与
	CollaborationStatusRestricted CollaborationStatus = "restricted" // 受限协作：仅小组成员可以参与
	CollaborationStatusClosed     CollaborationStatus = "closed"     // 封闭创作：仅创作者可以参与
)

// GetCollaborationStatus 从 Story 导出协作状态
func (s *Story) GetCollaborationStatus() CollaborationStatus {
	if s.IsCollaborationOpen {
		return CollaborationStatusOpen
	}
	// 如果 GroupID 不为空，认为是受限协作
	if s.GroupID != "" {
		return CollaborationStatusRestricted
	}
	return CollaborationStatusClosed
}

// CanUseAI 检查故事是否允许使用 AI
func (s *Story) CanUseAI() bool {
	return s.UseAI
}

// CanUseAIGenerateMetadata 检查是否可以使用 AI 生成元数据
func (s *Story) CanUseAIGenerateMetadata() bool {
	if !s.CanUseAI() {
		return false
	}
	if s.AIAssistanceOptions == nil {
		return true // 默认允许
	}
	return s.AIAssistanceOptions.GenerateMetadata
}

// CanUseAIGenerateVisuals 检查是否可以使用 AI 生成视觉素材
func (s *Story) CanUseAIGenerateVisuals() bool {
	if !s.CanUseAI() {
		return false
	}
	if s.AIAssistanceOptions == nil {
		return true // 默认允许
	}
	return s.AIAssistanceOptions.GenerateVisuals
}

// CanUseAIAssistStoryboard 检查是否可以使用 AI 辅助创建故事板
func (s *Story) CanUseAIAssistStoryboard() bool {
	if !s.CanUseAI() {
		return false
	}
	if s.AIAssistanceOptions == nil {
		return true // 默认允许
	}
	return s.AIAssistanceOptions.AssistStoryboard
}

// CanUseAIGenerateVideo 检查是否可以使用 AI 生成视频
func (s *Story) CanUseAIGenerateVideo() bool {
	if !s.CanUseAI() {
		return false
	}
	if s.AIAssistanceOptions == nil {
		return false // 默认禁用
	}
	return s.AIAssistanceOptions.GenerateVideo
}

// CanCreateCharacter 检查用户是否可以创建故事角色
// 核心规则：只有故事创作者可以创建角色
func (s *Story) CanCreateCharacter(userID string) bool {
	return s.AuthorID == userID
}

// CanCreateCharacterPoster 检查用户是否可以为角色创建海报
// 根据故事协作状态不同而不同
func (s *Story) CanCreateCharacterPoster(userID string, isGroupMember bool) bool {
	status := s.GetCollaborationStatus()

	switch status {
	case CollaborationStatusOpen:
		// 开放协作：任何人
		return true

	case CollaborationStatusRestricted:
		// 受限协作：仅小组成员或创作者
		return isGroupMember || s.AuthorID == userID

	case CollaborationStatusClosed:
		// 封闭创作：仅创作者
		return s.AuthorID == userID
	}

	return false
}

// CanCreateScene 检查用户是否可以创建故事场景
// 核心规则：只有故事创作者可以创建场景
func (s *Story) CanCreateScene(userID string) bool {
	return s.AuthorID == userID
}

// CanCreateSceneVariant 检查用户是否可以创建场景变体
// 权限与角色海报一致
func (s *Story) CanCreateSceneVariant(userID string, isGroupMember bool) bool {
	return s.CanCreateCharacterPoster(userID, isGroupMember)
}

// CanCreateStoryboard 检查用户是否可以创建故事板
func (s *Story) CanCreateStoryboard(userID string, isGroupMember bool) bool {
	status := s.GetCollaborationStatus()

	switch status {
	case CollaborationStatusOpen:
		// 开放协作：任何人
		return true

	case CollaborationStatusRestricted:
		// 受限协作：小组成员或创作者
		return isGroupMember || s.AuthorID == userID

	case CollaborationStatusClosed:
		// 封闭创作：仅创作者
		return s.AuthorID == userID
	}

	return false
}

// AIOperation AI 操作类型
type AIOperation string

const (
	AIOperationGenerateTitle       AIOperation = "generate_title"
	AIOperationGenerateDescription AIOperation = "generate_description"
	AIOperationGenerateCover       AIOperation = "generate_cover"
	AIOperationContinuePlot        AIOperation = "continue_plot"
	AIOperationGenerateSceneText   AIOperation = "generate_scene_text"
	AIOperationGenerateSceneImage  AIOperation = "generate_scene_image"
	AIOperationGenerateVideo       AIOperation = "generate_video"
)

// DisplayName 返回 AI 操作的显示名称
func (op AIOperation) DisplayName() string {
	switch op {
	case AIOperationGenerateTitle:
		return "生成标题"
	case AIOperationGenerateDescription:
		return "生成描述"
	case AIOperationGenerateCover:
		return "生成封面"
	case AIOperationContinuePlot:
		return "续写剧情"
	case AIOperationGenerateSceneText:
		return "生成场景文字"
	case AIOperationGenerateSceneImage:
		return "生成场景图片"
	case AIOperationGenerateVideo:
		return "生成视频"
	default:
		return string(op)
	}
}

// EnforceCanUseAI 验证是否可以执行 AI 操作
// 如果故事是非 AI 故事，返回错误
func (s *Story) EnforceCanUseAI(operation AIOperation) error {
	if !s.CanUseAI() {
		return &AIUsageError{
			StoryID:   s.ID,
			Operation: operation,
		}
	}
	return nil
}

// AIUsageError AI 使用错误
type AIUsageError struct {
	StoryID   string
	Operation AIOperation
}

func (e *AIUsageError) Error() string {
	return "故事 " + e.StoryID + " 是非 AI 故事，不能使用 AI " + e.Operation.DisplayName()
}
