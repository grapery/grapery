package domain

// AgentStatus Agent 状态
type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"   // 活跃
	AgentStatusInactive AgentStatus = "inactive" // 非活跃
	AgentStatusTraining AgentStatus = "training" // 训练中
)

// Agent represents the AI mapping of a Character
// When users interact with a Character, they're actually interacting with its Agent
type Agent struct {
	ID               string      `json:"id"`
	CharacterID      string      `json:"characterId"` // 一对一关联
	Name             string      `json:"name"`
	Description      string      `json:"description,omitempty"`
	Status           AgentStatus `json:"status"`
	SystemPrompt     string      `json:"systemPrompt"` // Agent 的系统提示词
	Temperature      float64     `json:"temperature"`  // 0-1
	Provider         string      `json:"provider"`     // huoshan, gemini
	Model            string      `json:"model,omitempty"`
	MaxTokens        int         `json:"maxTokens"`
	InteractionCount int         `json:"interactionCount"` // 交互次数
	SkillCount       int         `json:"skillCount"`       // 技能数量
	ConfigJSON       string      `json:"-"`                // 扩展配置
	CreatedAt        int64       `json:"createdAt"`
	UpdatedAt        int64       `json:"updatedAt"`

	// Business fields
	Config    map[string]interface{} `json:"config,omitempty"` // 扩展配置（如记忆窗口、回复风格等）
	Character *Character             `json:"character,omitempty"`
	Skills    []AgentSkill           `json:"skills,omitempty"`
}

// SkillType 技能类型
type SkillType string

const (
	SkillTypeCreative      SkillType = "creative"      // 创意类（生成故事、角色等）
	SkillTypeTechnical     SkillType = "technical"     // 技术类（代码、分析等）
	SkillTypeConversation  SkillType = "conversation"  // 对话类（聊天、咨询等）
	SkillTypeAnalysis      SkillType = "analysis"      // 分析类（情感分析、总结等）
	SkillTypeAction        SkillType = "action"        // 动作类（执行任务、调用API等）
	SkillTypeKnowledge     SkillType = "knowledge"     // 知识类（问答、教学等）
	SkillTypeEntertainment SkillType = "entertainment" // 娱乐类（游戏、互动等）
)

// SkillStatus 技能状态
type SkillStatus string

const (
	SkillStatusActive   SkillStatus = "active"   // 启用
	SkillStatusInactive SkillStatus = "inactive" // 禁用
	SkillStatusDraft    SkillStatus = "draft"    // 草稿
)

// AgentSkill represents a skill that an Agent can use
// Based on Anthropic's Agent Skills concept: https://github.com/anthropics/skills
type AgentSkill struct {
	ID               string      `json:"id"`
	AgentID          string      `json:"agentId"`
	Name             string      `json:"name"`
	DisplayName      string      `json:"displayName"`
	Description      string      `json:"description"` // 技能描述，说明何时使用
	Type             SkillType   `json:"type"`
	Status           SkillStatus `json:"status"`
	Instructions     string      `json:"instructions"`     // 技能的详细指令（Markdown 格式）
	ExamplesJSON     string      `json:"-"`                // 使用示例
	GuidelinesJSON   string      `json:"-"`                // 使用指南
	MetadataJSON     string      `json:"-"`                // 元数据（如版本、作者等）
	UsageCount       int         `json:"usageCount"`       // 使用次数
	SuccessCount     int         `json:"successCount"`     // 成功次数
	FailureCount     int         `json:"failureCount"`     // 失败次数
	AvgExecutionTime int         `json:"avgExecutionTime"` // 平均执行时间（毫秒）
	Priority         int         `json:"priority"`         // 优先级（0-100，越高越优先）
	Enabled          bool        `json:"enabled"`
	CreatedAt        int64       `json:"createdAt"`
	UpdatedAt        int64       `json:"updatedAt"`

	// Business fields
	Examples   []string               `json:"examples,omitempty"`
	Guidelines []string               `json:"guidelines,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Agent      *Agent                 `json:"agent,omitempty"`
}

// AgentSkillUsage records each time a skill is used
type AgentSkillUsage struct {
	ID             string `json:"id"`
	AgentID        string `json:"agentId"`
	SkillID        string `json:"skillId"`
	UserID         string `json:"userId,omitempty"`         // 触发使用的用户
	ConversationID string `json:"conversationId,omitempty"` // 关联的对话
	Scenario       string `json:"scenario,omitempty"`       // 使用场景描述
	InputData      string `json:"inputData,omitempty"`      // 输入数据
	OutputData     string `json:"outputData,omitempty"`     // 输出数据
	Success        bool   `json:"success"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	ExecutionTime  int    `json:"executionTime"` // 执行时间（毫秒）
	TokensUsed     int    `json:"tokensUsed"`
	CreatedAt      int64  `json:"createdAt"`

	// Relations
	Agent *Agent      `json:"agent,omitempty"`
	Skill *AgentSkill `json:"skill,omitempty"`
	User  *User       `json:"user,omitempty"`
}

// AgentInteraction records interactions between users and agents
type AgentInteraction struct {
	ID          string `json:"id"`
	AgentID     string `json:"agentId"`
	UserID      string `json:"userId"`
	CharacterID string `json:"characterId"`
	MessageID   string `json:"messageId,omitempty"` // 关联的消息ID
	Type        string `json:"type"`                // chat, story_generation, etc.
	InputText   string `json:"inputText,omitempty"`
	OutputText  string `json:"outputText,omitempty"`
	TokensUsed  int    `json:"tokensUsed"`
	Duration    int    `json:"duration"` // 毫秒
	SkillsUsed  string `json:"-"`        // 使用的技能ID列表
	Success     bool   `json:"success"`
	CreatedAt   int64  `json:"createdAt"`

	// Business fields
	SkillsUsedList []string   `json:"skillsUsed,omitempty"`
	Agent          *Agent     `json:"agent,omitempty"`
	User           *User      `json:"user,omitempty"`
	Character      *Character `json:"character,omitempty"`
}

// AgentMemory stores conversation context and learning for agents
type AgentMemory struct {
	ID           string `json:"id"`
	AgentID      string `json:"agentId"`
	UserID       string `json:"userId"`
	MemoryType   string `json:"memoryType"`             // short_term, long_term, episodic
	Key          string `json:"key"`                    // 记忆键（如话题、偏好等）
	Value        string `json:"value"`                  // 记忆内容
	Importance   int    `json:"importance"`             // 重要性（0-100）
	AccessCount  int    `json:"accessCount"`            // 访问次数
	LastAccessed *int64 `json:"lastAccessed,omitempty"` // 最后访问时间
	ExpiresAt    *int64 `json:"expiresAt,omitempty"`    // 过期时间
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`

	// Relations
	Agent *Agent `json:"agent,omitempty"`
	User  *User  `json:"user,omitempty"`
}

// ==================== Filters and Stats ====================

// SkillFilter for listing skills
type SkillFilter struct {
	Type    SkillType
	Status  SkillStatus
	Enabled *bool
	SortBy  string // usage_count, priority, success_rate, created_at
	Order   string // asc, desc
	Limit   int
	Offset  int
}

// SkillUsageFilter for listing skill usages
type SkillUsageFilter struct {
	AgentID  string
	SkillID  string
	UserID   string
	Success  *bool
	FromTime int64
	ToTime   int64
	Limit    int
	Offset   int
}

// SkillUsageStats statistics for a skill
type SkillUsageStats struct {
	TotalUsages      int64   `json:"totalUsages"`
	SuccessCount     int64   `json:"successCount"`
	FailureCount     int64   `json:"failureCount"`
	SuccessRate      float64 `json:"successRate"`      // percentage
	AvgExecutionTime int     `json:"avgExecutionTime"` // milliseconds
	TotalTokens      int64   `json:"totalTokens"`
}

// InteractionFilter for listing interactions
type InteractionFilter struct {
	AgentID     string
	UserID      string
	CharacterID string
	Type        string
	Limit       int
	Offset      int
}

// MemoryFilter for listing memories
type MemoryFilter struct {
	AgentID    string
	UserID     string
	MemoryType string
	Key        string
	Limit      int
	Offset     int
}
