package utils

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// FileExtensionForRenderType 根据渲染类型获取文件扩展名
func FileExtensionForRenderType(renderType domain.RenderTaskType) string {
	switch renderType {
	case domain.RenderTaskTypeVideo:
		return "mp4"
	case domain.RenderTaskTypeImageSet:
		return "zip"
	case domain.RenderTaskTypeAnimation:
		return "gif"
	default:
		return "mp4"
	}
}

// StringPtr 字符串指针辅助函数
func StringPtr(s string) *string {
	return &s
}

// JSONMarshal JSON序列化辅助函数
func JSONMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// GenerateID 生成ID辅助函数
func GenerateID() string {
	return fmt.Sprintf("id_%d", time.Now().UnixNano())
}
