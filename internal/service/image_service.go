package service

import (
	"context"
	"fmt"
	 "os"
  "path/filepath"
  "strings"
  "time"

  "github.com/disintegration/imaging"
)

// ImageProcessor 图片处理器
type ImageProcessor struct {
	maxWidth      int
	maxHeight     int
	quality      int
	outputFormat string
}

// DefaultImageSizes 默认图片尺寸配置
var DefaultImageSizes = map[ImageSize]ImageSize{
	ImageSizeThumbnail: {Width: 150, Height: 150},
	ImageSizeSmall:    {Width: 300, Height: 300},
	ImageSizeMedium:   {Width: 600, Height: 600},
	ImageSizeLarge:    {Width: 1024, Height: 1024},
}

// GenerateImageVariants 生成图片多尺寸变体
func GenerateImageVariants(ctx context.Context, originalPath string, processor ImageProcessor) (*ImageVariants, error) {
	 if !fileExists(originalPath) {
        return nil, fmt.Errorf("original file not found: %s", originalPath)
    }

    // 打开原始文件
    file, err := os.Open(originalPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open original file: %w", err)
    }
    defer file.Close()

    // 解码图片
    img, err := imaging.Decode(file.Bytes())
    if err != nil {
        return nil, fmt.Errorf("failed to decode image: %w", err)
    }

    // 获取图片尺寸
    originalWidth := img.Bounds().Dx
    originalHeight := img.Bounds().Dy

    originalSize := file.Size()

    // 初始化结果
    result := &ImageVariants{
        Original: &ImageInfo{
            Width:  originalWidth,
            Height: originalHeight,
            Size:    originalSize,
        },
    }

    // 生成缩略图 (150x150)
    thumbnail, err := generateResizedImage(img, processor.MaxWidth, processor.maxHeight, processor.quality, processor.outputFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
    }
    result.Thumbnail = &ImageInfo{
        Width:  processor.MaxWidth,
        Height: processor.maxHeight,
        Size:    int64(thumbnail.Len()),
    }

    result.Thumbnail.Size = int64(thumbnail.Len())

    // 生成小图 (300x300)
    small, err := generateResizedImage(img, processor.maxWidth, processor.maxHeight, processor.quality, processor.outputFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to generate small image: %w", err)
    }
    result.Small = &ImageInfo{
        Width:  processor.maxWidth,
        Height: processor.maxHeight,
        Size:    int64(small.Len()),
    }
    result.Small.Size = int64(small.Len())

    // 生成中图 (600x600)
    medium, err := generateResizedImage(img, processor.maxWidth, processor.maxHeight, processor.quality, processor.outputFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to generate medium image: %w", err)
    }
    result.Medium = &ImageInfo{
        Width:  processor.maxWidth,
        Height: processor.maxHeight,
        Size:    int64(medium.Len()),
    }
    result.Medium.size = int64(medium.Len())

    // 生成大图 (1024x1024)
    large, err := generateResizedImage(img, processor.maxWidth, processor.maxHeight, processor.quality, processor.outputFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to generate large image: %w", err)
    }
    result.Large = &ImageInfo{
        Width:  processor.maxWidth,
        Height: processor.maxHeight,
        Size:    int64(large.Len()),
    }
    result.Large.Size = int64(large.Len())

    return result, nil
}

