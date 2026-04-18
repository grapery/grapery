package domain

// StyleConfig 图片/视频风格配置
type StyleConfig struct {
	ID             string `json:"id"`
	Style          string `json:"style"`
	Description    string `json:"description"`
	SampleImageURL string `json:"sampleImageUrl,omitempty"` // 示例图片URL
	UserID         string `json:"userId,omitempty"`         // 所属用户ID（私有风格）
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`
}
