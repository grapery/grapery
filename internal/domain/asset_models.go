package domain

// Asset 用户资产（图片、音频等）
type Asset struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	Type       string `json:"type"` // image, audio, video
	Name       string `json:"name"`
	URL        string `json:"url"`
	Thumbnail  string `json:"thumbnail,omitempty"`
	Size       int64  `json:"size"` // 字节
	MimeType   string `json:"mimeType"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Duration   int    `json:"duration,omitempty"` // 秒（音视频）
	TagsJSON   string `json:"-"`
	UsageCount int    `json:"usageCount"`
	CreatedAt  int64  `json:"createdAt"`

	// Business fields
	Tags []string `json:"tags,omitempty"`
	User *User    `json:"user,omitempty"`
}
