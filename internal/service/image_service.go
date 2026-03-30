package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ImageProcessor 图片处理器
type ImageProcessor struct {
	MaxWidth      int
	MaxHeight     int
	Quality       int
	OutputFormat  string
}

// ImageSizeConfig 图片尺寸配置
type ImageSizeConfig struct {
	Width  int
	Height int
}

// 预定义图片尺寸
var (
	ThumbnailSize = ImageSizeConfig{Width: 150, Height: 150}
	SmallSize     = ImageSizeConfig{Width: 300, Height: 300}
	MediumSize    = ImageSizeConfig{Width: 600, Height: 600}
	LargeSize     = ImageSizeConfig{Width: 1024, Height: 1024}
)

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GenerateImageVariants 生成图片多尺寸变体
func GenerateImageVariants(ctx context.Context, originalPath string, quality int) (*domain.ImageVariants, error) {
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
	img, err := imaging.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// 获取图片尺寸
	originalWidth := img.Bounds().Dx()
	originalHeight := img.Bounds().Dy()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 初始化结果
	result := &domain.ImageVariants{
		Original: &domain.ImageInfo{
			Width:  originalWidth,
			Height: originalHeight,
			Size:   fileInfo.Size(),
		},
	}

	// 确定输出格式
	outputFormat := strings.ToLower(filepath.Ext(originalPath))
	if outputFormat == ".jpg" {
		outputFormat = ".jpeg"
	}

	// 生成缩略图 (150x150)
	thumbnailImg := generateResizedImage(img, ThumbnailSize.Width, ThumbnailSize.Height)
	thumbnailSize, err := getImageSize(thumbnailImg, outputFormat, quality)
	if err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}
	result.Thumbnail = &domain.ImageInfo{
		Width:  ThumbnailSize.Width,
		Height: ThumbnailSize.Height,
		Size:   thumbnailSize,
	}

	// 生成小图 (300x300)
	smallImg := generateResizedImage(img, SmallSize.Width, SmallSize.Height)
	smallSize, err := getImageSize(smallImg, outputFormat, quality)
	if err != nil {
		return nil, fmt.Errorf("failed to generate small image: %w", err)
	}
	result.Small = &domain.ImageInfo{
		Width:  SmallSize.Width,
		Height: SmallSize.Height,
		Size:   smallSize,
	}

	// 生成中图 (600x600)
	mediumImg := generateResizedImage(img, MediumSize.Width, MediumSize.Height)
	mediumSize, err := getImageSize(mediumImg, outputFormat, quality)
	if err != nil {
		return nil, fmt.Errorf("failed to generate medium image: %w", err)
	}
	result.Medium = &domain.ImageInfo{
		Width:  MediumSize.Width,
		Height: MediumSize.Height,
		Size:   mediumSize,
	}

	// 生成大图 (1024x1024)
	largeImg := generateResizedImage(img, LargeSize.Width, LargeSize.Height)
	largeSize, err := getImageSize(largeImg, outputFormat, quality)
	if err != nil {
		return nil, fmt.Errorf("failed to generate large image: %w", err)
	}
	result.Large = &domain.ImageInfo{
		Width:  LargeSize.Width,
		Height: LargeSize.Height,
		Size:   largeSize,
	}

	return result, nil
}

// generateResizedImage 生成指定尺寸的图片
func generateResizedImage(img image.Image, maxWidth, maxHeight int) image.Image {
	// 使用 imaging 库的缩放功能
	return imaging.Fill(img, maxWidth, maxHeight, imaging.Center, imaging.Lanczos)
}

// getImageSize 获取图片的字节大小
func getImageSize(img image.Image, format string, quality int) (int64, error) {
	var buf bytes.Buffer

	switch format {
	case ".png":
		if err := png.Encode(&buf, img); err != nil {
			return 0, err
		}
	default: // 默认使用 jpeg
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return 0, err
		}
	}

	return int64(buf.Len()), nil
}

// SaveImageVariants 保存图片变体到文件系统
func SaveImageVariants(ctx context.Context, originalPath string, outputDir string, quality int) (*domain.ImageVariants, error) {
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
	img, err := imaging.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// 获取图片尺寸
	originalWidth := img.Bounds().Dx()
	originalHeight := img.Bounds().Dy()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// 确定输出格式
	ext := strings.ToLower(filepath.Ext(originalPath))
	if ext == ".jpg" {
		ext = ".jpeg"
	}

	// 基础文件名（不含扩展名）
	baseName := strings.TrimSuffix(filepath.Base(originalPath), filepath.Ext(originalPath))

	// 初始化结果
	result := &domain.ImageVariants{
		Original: &domain.ImageInfo{
			URL:    originalPath,
			Width:  originalWidth,
			Height: originalHeight,
			Size:   fileInfo.Size(),
		},
	}

	// 生成并保存缩略图
	thumbnailPath := filepath.Join(outputDir, baseName+"_thumbnail"+ext)
	if err := saveResizedImage(img, thumbnailPath, ThumbnailSize.Width, ThumbnailSize.Height, quality); err != nil {
		return nil, fmt.Errorf("failed to save thumbnail: %w", err)
	}
	thumbnailInfo, _ := os.Stat(thumbnailPath)
	result.Thumbnail = &domain.ImageInfo{
		URL:    thumbnailPath,
		Width:  ThumbnailSize.Width,
		Height: ThumbnailSize.Height,
		Size:   thumbnailInfo.Size(),
	}

	// 生成并保存小图
	smallPath := filepath.Join(outputDir, baseName+"_small"+ext)
	if err := saveResizedImage(img, smallPath, SmallSize.Width, SmallSize.Height, quality); err != nil {
		return nil, fmt.Errorf("failed to save small image: %w", err)
	}
	smallInfo, _ := os.Stat(smallPath)
	result.Small = &domain.ImageInfo{
		URL:    smallPath,
		Width:  SmallSize.Width,
		Height: SmallSize.Height,
		Size:   smallInfo.Size(),
	}

	// 生成并保存中图
	mediumPath := filepath.Join(outputDir, baseName+"_medium"+ext)
	if err := saveResizedImage(img, mediumPath, MediumSize.Width, MediumSize.Height, quality); err != nil {
		return nil, fmt.Errorf("failed to save medium image: %w", err)
	}
	mediumInfo, _ := os.Stat(mediumPath)
	result.Medium = &domain.ImageInfo{
		URL:    mediumPath,
		Width:  MediumSize.Width,
		Height: MediumSize.Height,
		Size:   mediumInfo.Size(),
	}

	// 生成并保存大图
	largePath := filepath.Join(outputDir, baseName+"_large"+ext)
	if err := saveResizedImage(img, largePath, LargeSize.Width, LargeSize.Height, quality); err != nil {
		return nil, fmt.Errorf("failed to save large image: %w", err)
	}
	largeInfo, _ := os.Stat(largePath)
	result.Large = &domain.ImageInfo{
		URL:    largePath,
		Width:  LargeSize.Width,
		Height: LargeSize.Height,
		Size:   largeInfo.Size(),
	}

	return result, nil
}

// saveResizedImage 保存指定尺寸的图片到文件
func saveResizedImage(img image.Image, outputPath string, maxWidth, maxHeight, quality int) error {
	resized := generateResizedImage(img, maxWidth, maxHeight)

	ext := strings.ToLower(filepath.Ext(outputPath))

	switch ext {
	case ".png":
		return imaging.Save(resized, outputPath)
	default: // 默认使用 jpeg
		outFile, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer outFile.Close()
		return jpeg.Encode(outFile, resized, &jpeg.Options{Quality: quality})
	}
}
