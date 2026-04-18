package domain

// FragmentPanelGenerationRequest 多图面板生成请求（持久化在任务表 request_json）
// 分镜规划/文本 LLM 仅使用 huoshan 或 gemini（由服务端按 UserRegion 决定）；图片可走 kling 等，由 AI_IMAGE_PROVIDER 等配置。
type FragmentPanelGenerationRequest struct {
	UserInput         string `json:"userInput"`
	ReferenceImageURL string `json:"referenceImageUrl"`
	Style             string `json:"style"`
	PanelCount        int    `json:"panelCount"`
	Visibility        string `json:"visibility"`
	// Topic 话题标签（展示用，不含 #）；与碎片表 topic 对齐。
	Topic string `json:"topic,omitempty"`
	// UserRegion 服务端根据用户资料写入（如 CN / US），用于选择 Huoshan vs Gemini；客户端勿伪造。
	UserRegion string `json:"userRegion,omitempty"`
	// AspectRatio 配图长宽比：1:1、16:9、9:16、3:4、4:3；空则默认 16:9（与碎片 AI 生成一致）。
	AspectRatio string `json:"aspectRatio,omitempty"`
}

// FragmentPanelPlanItem Step1 输出的单格规划
type FragmentPanelPlanItem struct {
	Index         int      `json:"index"`
	ImagePrompt   string   `json:"image_prompt"`
	Caption       string   `json:"caption"`
	ReferenceKeys []string `json:"reference_keys,omitempty"` // 引用 visualBible 锚点 key（与普通碎片 expandScenes 对齐）
}

// FragmentPanelResultItem 最终每格结果
type FragmentPanelResultItem struct {
	Index    int    `json:"index"`
	ImageURL string `json:"imageUrl"`
	Caption  string `json:"caption"`
}

// FragmentPanelResultData 任务结果（result_json）
type FragmentPanelResultData struct {
	Panels            []FragmentPanelResultItem  `json:"panels"`
	CombinedContent   string                     `json:"combinedContent,omitempty"`
	VisualBible       *FragmentVisualBible       `json:"visualBible,omitempty"`
	AnchorImages      []FragmentAnchorImage      `json:"anchorImages,omitempty"`
	ConsistencyIssues []FragmentConsistencyIssue `json:"consistencyIssues,omitempty"`
}

// FragmentPanelStepMetric 单步 AI 指标
type FragmentPanelStepMetric struct {
	Name       string `json:"name"`
	Tokens     int    `json:"tokens"`
	DurationMs int64  `json:"duration_ms"`
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model,omitempty"`
}

// FragmentPanelMetricsData 全流程指标（metrics_json）
type FragmentPanelMetricsData struct {
	Steps           []FragmentPanelStepMetric `json:"steps"`
	TotalTokens     int                       `json:"total_tokens"`
	TotalDurationMs int64                     `json:"total_duration_ms"`
}

// FragmentPanelGenerationTask 多图面板生成任务
type FragmentPanelGenerationTask struct {
	ID          string                         `json:"id"`
	UserID      string                         `json:"userId"`
	Status      string                         `json:"status"` // pending, processing, completed, failed
	Progress    int                            `json:"progress"`
	CurrentStep string                         `json:"currentStep"`
	Request     FragmentPanelGenerationRequest `json:"request"`
	Plan        []FragmentPanelPlanItem        `json:"plan,omitempty"`
	// VisualBible / AnchorImages 由 Step1+锚点阶段写入；与 Plan 一并持久化在 plan_json（见 repository包装格式）。
	VisualBible     *FragmentVisualBible      `json:"visualBible,omitempty"`
	AnchorImages    []FragmentAnchorImage     `json:"anchorImages,omitempty"`
	Result          *FragmentPanelResultData  `json:"result,omitempty"`
	Metrics         *FragmentPanelMetricsData `json:"metrics,omitempty"`
	DraftFragmentID string                    `json:"draftFragmentId,omitempty"`
	ErrorMessage    string                    `json:"errorMessage,omitempty"`
	CreatedAt       int64                     `json:"createdAt"`
	StartedAt       *int64                    `json:"startedAt,omitempty"`
	CompletedAt     *int64                    `json:"completedAt,omitempty"`
	UpdatedAt       int64                     `json:"updatedAt"`
}
