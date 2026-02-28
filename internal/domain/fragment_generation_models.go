package domain

const (
	// Fragment AI 任务类型
	AITaskGenerateFragment        AITaskType = "generate_fragment"
	AITaskGenerateFragmentContent AITaskType = "generate_fragment_content"
	AITaskGenerateFragmentImages  AITaskType = "generate_fragment_images"
)

// FragmentGenerationRequest 碎片故事生成请求
type FragmentGenerationRequest struct {
	UserInput  string   `json:"userInput"`  // 用户输入的描述
	ImageUrls  []string `json:"imageUrls"`  // 用户提供的参考图片（可选）
	ImageCount int      `json:"imageCount"` // 需要生成的图片数量 (1-10)
	Style      string   `json:"style"`      // 风格：fantasy, realistic, anime, etc.
	Mood       string   `json:"mood"`       // 情绪：happy, sad, mysterious, etc.
	Length     string   `json:"length"`     // 内容长度：short, medium, long
	Language   string   `json:"language"`   // 语言：zh-Hans, en, ja
	Visibility string   `json:"visibility"` // 可见性：public, followers, private
}

// FragmentGenerationResult 碎片故事生成结果
type FragmentGenerationResult struct {
	Content    string   `json:"content"`    // 生成的文字内容
	ImageUrls  []string `json:"imageUrls"`  // 生成的图片URL列表
	TokensUsed int      `json:"tokensUsed"` // 使用的token数量
}

// FragmentContentGenerationRequest 碎片故事内容生成请求
type FragmentContentGenerationRequest struct {
	UserInput string `json:"userInput"` // 用户输入
	Style     string `json:"style"`     // 风格
	Mood      string `json:"mood"`      // 情绪
	Length    string `json:"length"`    // 长度
	Language  string `json:"language"`  // 语言
	Context   string `json:"context"`   // 额外上下文
}

// FragmentContentGenerationResult 碎片故事内容生成结果
type FragmentContentGenerationResult struct {
	Content    string `json:"content"`    // 生成的文字内容
	TokensUsed int    `json:"tokensUsed"` // 使用的token数量
}

// FragmentImageGenerationRequest 碎片故事图片生成请求
type FragmentImageGenerationRequest struct {
	Content    string `json:"content"`    // 文字内容（用于生成图片）
	ImageCount int    `json:"imageCount"` // 生成图片数量
	Style      string `json:"style"`      // 图片风格
	Size       string `json:"size"`       // 图片尺寸
	Quality    string `json:"quality"`    // 图片质量
}

// FragmentImageGenerationResult 碎片故事图片生成结果
type FragmentImageGenerationResult struct {
	ImageUrls  []string `json:"imageUrls"`  // 生成的图片URL列表
	TokensUsed int      `json:"tokensUsed"` // 使用的token数量
}

// FragmentGenerationTask 碎片故事生成任务
type FragmentGenerationTask struct {
	ID           string                    `json:"id"`
	UserID       string                    `json:"userId"`
	Status       string                    `json:"status"` // pending, processing, completed, failed
	Request      FragmentGenerationRequest `json:"request"`
	Result       *FragmentGenerationResult `json:"result,omitempty"`
	Progress     int                       `json:"progress"`    // 0-100
	CurrentStep  string                    `json:"currentStep"` // 当前步骤
	ErrorMessage string                    `json:"errorMessage,omitempty"`
	TokensUsed   int                       `json:"tokensUsed"`
	CreatedAt    int64                     `json:"createdAt"`
	StartedAt    *int64                    `json:"startedAt,omitempty"`
	CompletedAt  *int64                    `json:"completedAt,omitempty"`
	UpdatedAt    int64                     `json:"updatedAt"`

	// Relations
	User *User `json:"user,omitempty"`
}

// TableName specifies the table name for FragmentGenerationTask
func (FragmentGenerationTask) TableName() string {
	return "fragment_generation_tasks"
}
