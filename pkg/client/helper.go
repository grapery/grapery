package client

const (
	RateLimiter = 10
)

const (
	ApiCodeOK    = 0
	ApiErrorCode = 1
)

type StoryInfoParams struct {
	RoleDesc string `json:"role_desc"`
	Content  string `json:"content"`
}

type StoryInfoResult struct {
	Content string `json:"content"`
}

type GenStoryPosterParams struct {
	Title        string  `json:"title"`          // 海报标题
	SubTitle     string  `json:"sub_title"`      // 海报副标题
	BodyText     string  `json:"body_text"`      // 海报正文内容
	PromptTextZh string  `json:"prompt_text_zh"` // 提示词
	WhRatios     string  `json:"wh_ratios"`      // 图像宽高比
	LoraName     string  `json:"lora_name"`      // Lora名称
	LoraWeight   float64 `json:"lora_weight"`    // Lora权重
	CtrlRatio    float64 `json:"ctrl_ratio"`     // 控制比率
	CtrlStep     float64 `json:"ctrl_step"`      // 控制步长
	GenerateMode string  `json:"generate_mode"`  // 生成模式
	GenerateNum  int     `json:"generate_num"`   // 生成数量
	UserId       string  `json:"user_id"`        // 用户ID
	RequestId    string  `json:"request_id"`     // 请求ID
}

type GenStoryImagesParams struct {
	Content        string `json:"content"`
	RefImage       string `json:"ref_image"`
	Size           int    `json:"size"`
	UserId         string `json:"user_id"`
	RequestId      string `json:"request_id"`
	Prompt         string `json:"prompt"`          // 额外的提示词
	NegativePrompt string `json:"negative_prompt"` // 负面提示词
	MaskImageUrl   string `json:"mask_image_url"`  // 掩码图像
	Seed           int64
}

type GenStoryImagesResult struct {
	ImageUrls []string `json:"image_urls"`
}

type ScaleStoryImagesParams struct {
	ImageUrls []string `json:"image_urls"`
	Size      int      `json:"size"`
}

type ScaleStoryImagesResult struct {
	ImageUrls []string `json:"image_urls"`
	TimeCost  int
}

type GenStoryCharactorParams struct {
	Content string `json:"content"`
}

type GenStoryCharactorResult struct {
	Content string `json:"content"`
}

type ChatWithRoleParams struct {
	Role           string `json:"role"`
	MessageContent string `json:"message_content"`
	Background     string `json:"background"`
	SenseDesc      string `json:"sense_desc"`
	RolePositive   string `json:"role_positive"`
	RoleNegative   string `json:"role_negative"`
	RequestId      string `json:"request_id"`
	UserId         string `json:"user_id"`
}

type ChatWithRoleResult struct {
	Content string `json:"content"`
}

type GenStoryRoleInfoParams struct {
	Content string `json:"content"`
}

type GenStoryRoleInfoResult struct {
	Content string `json:"content"`
}
