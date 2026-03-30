package domain

// ImageSize 图片尺寸级别
type ImageSize string

const (
	ImageSizeThumbnail ImageSize = "thumbnail" // 缩略图: 150x150
	ImageSizeSmall    ImageSize = "small"    // 小图: 300x300
	ImageSizeMedium   ImageSize = "medium"    // 中图: 600x600
	ImageSizeLarge    ImageSize = "large"     // 大图: 1024x1024
	ImageSizeOriginal ImageSize = "original"  // 原图: 保持原始尺寸
)

// ImageDimensions 图片尺寸信息
type ImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Size   int `json:"size"` // 字节大小
}

// ImageInfo 图片信息
type ImageInfo struct {
	URL          string          `json:"url"`
	ThumbnailURL *string         `json:"thumbnailUrl,omitempty"`
	SquareURL    *string         `json:"smallUrl,omitempty"`
	MediumURL   *string         `json:"mediumUrl,omitempty"`
	LargeURL    *string         `json:"largeUrl,omitempty"`
	OriginalURL string       `json:"originalUrl"`
	Width         int          `json:"width"`
	Height       int          `json:"height"`
	Size         int64         `json:"size"`
	MimeType     string    `json:"mimeType"`
	CreatedAt   int64         `json:"createdAt"`
}

// ImageVariants 图片多尺寸变体
type ImageVariants struct {
	Thumbnail *ImageInfo `json:"thumbnail,omitempty"` // 缩略图 150x150
	Small     *ImageInfo `json:"small,omitempty"`    // 小图 300x300
	Medium    *ImageInfo `json:"medium,omitempty"`     // 中图: 600x600
	Large     *ImageInfo `json:"large,omitempty"`       // 大图: 1024x1024
	Original *ImageInfo `json:"original,omitempty"` // 废图: 保持原始尺寸
}
