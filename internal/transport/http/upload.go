package http

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// UploadImage 上传图片
// POST /api/upload/image
// Form: file (multipart/form-data)
func (h *Handler) UploadImage(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		InvalidParams(c, "file is required")
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if !isImageType(contentType) {
		InvalidParams(c, "only image files are allowed")
		return
	}

	// 验证文件大小 (最大 10MB)
	maxSize := int64(10 * 1024 * 1024)
	if file.Size > maxSize {
		InvalidParams(c, "file size exceeds 10MB limit")
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = getExtFromContentType(contentType)
	}
	newFilename := fmt.Sprintf("%s_%s%s", userID, uuid.New().String(), ext)

	// 尝试使用阿里云 OSS 上传
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil {
		// 读取文件内容
		src, err := file.Open()
		if err != nil {
			InternalError(c, "failed to open file")
			return
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			InternalError(c, "failed to read file")
			return
		}

		// 上传到 OSS
		objectKey := fmt.Sprintf("images/%s", newFilename)
		fileURL, err := ossClient.UploadBytes(objectKey, data)
		if err != nil {
			InternalError(c, "failed to upload to OSS")
			return
		}

		// 清理 URL (移除签名参数，使用 HTTPS)
		fileURL = strings.Split(fileURL, "?")[0]
		fileURL = strings.ReplaceAll(fileURL, "http://", "https://")

		// 生成多级图片 URL
		imageLevels := aliyun.GenerateImageLevels(fileURL)

		Success(c, gin.H{
			"url":      fileURL,
			"filename": newFilename,
			"size":     file.Size,
			"type":     contentType,
			"levels":   imageLevels,
			"storage":  "oss",
		})
		return
	}

	// 回退到本地存储
	uploadDir := "uploads/images"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		InternalError(c, "failed to create upload directory")
		return
	}

	// 保存文件
	dst := filepath.Join(uploadDir, newFilename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		InternalError(c, "failed to save file")
		return
	}

	// 返回文件URL
	fileURL := fmt.Sprintf("/uploads/images/%s", newFilename)

	Success(c, gin.H{
		"url":      fileURL,
		"filename": newFilename,
		"size":     file.Size,
		"type":     contentType,
		"storage":  "local",
	})
}

// UploadAvatar 上传头像
// POST /api/upload/avatar
func (h *Handler) UploadAvatar(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		InvalidParams(c, "file is required")
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if !isImageType(contentType) {
		InvalidParams(c, "only image files are allowed")
		return
	}

	// 验证文件大小 (最大 5MB for avatars)
	maxSize := int64(5 * 1024 * 1024)
	if file.Size > maxSize {
		InvalidParams(c, "file size exceeds 5MB limit")
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = getExtFromContentType(contentType)
	}
	newFilename := fmt.Sprintf("avatar_%s_%d%s", userID, time.Now().Unix(), ext)

	// 尝试使用阿里云 OSS 上传
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil {
		// 读取文件内容
		src, err := file.Open()
		if err != nil {
			InternalError(c, "failed to open file")
			return
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			InternalError(c, "failed to read file")
			return
		}

		// 上传到 OSS
		objectKey := fmt.Sprintf("avatars/%s", newFilename)
		avatarURL, err := ossClient.UploadBytes(objectKey, data)
		if err != nil {
			InternalError(c, "failed to upload to OSS")
			return
		}

		// 清理 URL
		avatarURL = strings.Split(avatarURL, "?")[0]
		avatarURL = strings.ReplaceAll(avatarURL, "http://", "https://")

		// 更新用户头像
		if err := h.svc.UpdateUserAvatar(c.Request.Context(), userID, avatarURL); err != nil {
			InternalError(c, "failed to update user avatar")
			return
		}

		// 生成多级图片 URL
		imageLevels := aliyun.GenerateImageLevels(avatarURL)

		Success(c, gin.H{
			"url":      avatarURL,
			"filename": newFilename,
			"levels":   imageLevels,
			"storage":  "oss",
		})
		return
	}

	// 回退到本地存储
	uploadDir := "uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		InternalError(c, "failed to create upload directory")
		return
	}

	// 保存文件
	dst := filepath.Join(uploadDir, newFilename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		InternalError(c, "failed to save file")
		return
	}

	// 返回文件URL
	avatarURL := fmt.Sprintf("/uploads/avatars/%s", newFilename)

	// 更新用户头像
	if err := h.svc.UpdateUserAvatar(c.Request.Context(), userID, avatarURL); err != nil {
		// 文件已保存，但更新数据库失败
		InternalError(c, "failed to update user avatar")
		return
	}

	Success(c, gin.H{
		"url":      avatarURL,
		"filename": newFilename,
		"storage":  "local",
	})
}

// UploadCover 上传封面图
// POST /api/upload/cover
func (h *Handler) UploadCover(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		InvalidParams(c, "file is required")
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if !isImageType(contentType) {
		InvalidParams(c, "only image files are allowed")
		return
	}

	// 验证文件大小 (最大 10MB)
	maxSize := int64(10 * 1024 * 1024)
	if file.Size > maxSize {
		InvalidParams(c, "file size exceeds 10MB limit")
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = getExtFromContentType(contentType)
	}
	newFilename := fmt.Sprintf("cover_%s_%s%s", userID, uuid.New().String(), ext)

	// 尝试使用阿里云 OSS 上传
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil {
		// 读取文件内容
		src, err := file.Open()
		if err != nil {
			InternalError(c, "failed to open file")
			return
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			InternalError(c, "failed to read file")
			return
		}

		// 上传到 OSS
		objectKey := fmt.Sprintf("covers/%s", newFilename)
		coverURL, err := ossClient.UploadBytes(objectKey, data)
		if err != nil {
			InternalError(c, "failed to upload to OSS")
			return
		}

		// 清理 URL
		coverURL = strings.Split(coverURL, "?")[0]
		coverURL = strings.ReplaceAll(coverURL, "http://", "https://")

		// 生成多级图片 URL
		imageLevels := aliyun.GenerateImageLevels(coverURL)

		Success(c, gin.H{
			"url":      coverURL,
			"filename": newFilename,
			"size":     file.Size,
			"levels":   imageLevels,
			"storage":  "oss",
		})
		return
	}

	// 回退到本地存储
	uploadDir := "uploads/covers"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		InternalError(c, "failed to create upload directory")
		return
	}

	// 保存文件
	dst := filepath.Join(uploadDir, newFilename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		InternalError(c, "failed to save file")
		return
	}

	// 返回文件URL
	coverURL := fmt.Sprintf("/uploads/covers/%s", newFilename)

	Success(c, gin.H{
		"url":      coverURL,
		"filename": newFilename,
		"size":     file.Size,
		"storage":  "local",
	})
}

// UploadMultiple 批量上传图片
// POST /api/upload/multiple
func (h *Handler) UploadMultiple(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		InvalidParams(c, "invalid form data")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		InvalidParams(c, "no files uploaded")
		return
	}

	// 限制最多上传10个文件
	if len(files) > 10 {
		InvalidParams(c, "maximum 10 files allowed")
		return
	}

	uploadDir := "uploads/images"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		InternalError(c, "failed to create upload directory")
		return
	}

	var uploadedFiles []gin.H

	for _, file := range files {
		// 验证文件类型
		contentType := file.Header.Get("Content-Type")
		if !isImageType(contentType) {
			continue // 跳过非图片文件
		}

		// 验证文件大小
		if file.Size > 10*1024*1024 {
			continue // 跳过超过10MB的文件
		}

		// 生成文件名
		ext := filepath.Ext(file.Filename)
		if ext == "" {
			ext = getExtFromContentType(contentType)
		}
		newFilename := fmt.Sprintf("%s_%s%s", userID, uuid.New().String(), ext)

		// 保存文件
		dst := filepath.Join(uploadDir, newFilename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			continue // 保存失败，跳过
		}

		uploadedFiles = append(uploadedFiles, gin.H{
			"url":      fmt.Sprintf("/uploads/images/%s", newFilename),
			"filename": newFilename,
			"size":     file.Size,
		})
	}

	Success(c, gin.H{
		"files": uploadedFiles,
		"count": len(uploadedFiles),
	})
}

// DeleteUpload 删除上传的文件
// DELETE /api/upload
func (h *Handler) DeleteUpload(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// 验证URL格式
	if !strings.HasPrefix(req.URL, "/uploads/") {
		InvalidParams(c, "invalid file URL")
		return
	}

	// 删除文件
	filePath := strings.TrimPrefix(req.URL, "/")
	if err := os.Remove(filePath); err != nil {
		InternalError(c, "failed to delete file")
		return
	}

	Success(c, gin.H{"message": "file deleted successfully"})
}

// ========== Helper Functions ==========

func isImageType(contentType string) bool {
	allowedTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/svg+xml",
	}

	for _, t := range allowedTypes {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}
	return false
}

func getExtFromContentType(contentType string) string {
	switch {
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		return ".jpg"
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "svg"):
		return ".svg"
	default:
		return ".jpg"
	}
}

// ServeUploadedFile 提供上传文件的静态服务
func ServeUploadedFile(c *gin.Context) {
	filepath := c.Param("filepath")

	// 安全检查：防止目录遍历攻击
	if strings.Contains(filepath, "..") {
		c.Status(404)
		return
	}

	fullPath := "uploads/" + filepath

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.Status(404)
		return
	}

	// 打开文件
	file, err := os.Open(fullPath)
	if err != nil {
		c.Status(500)
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		c.Status(500)
		return
	}

	// 设置响应头
	c.Header("Content-Type", getContentType(fullPath))
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Cache-Control", "public, max-age=31536000")

	// 发送文件
	io.Copy(c.Writer, file)
}

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

// UploadImageFromURL 从URL上传图片到OSS
// POST /api/upload/from-url
func (h *Handler) UploadImageFromURL(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		URL       string `json:"url" binding:"required"`
		ObjectKey string `json:"objectKey"` // optional, will be generated if not provided
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	ossClient := aliyun.GetGlobalClient()
	if ossClient == nil {
		InternalError(c, "OSS client not configured")
		return
	}

	// 生成 object key
	objectKey := req.ObjectKey
	if objectKey == "" {
		objectKey = fmt.Sprintf("images/%s_%s.jpg", userID, uuid.New().String())
	}

	// 从 URL 上传到 OSS
	fileURL, err := ossClient.UploadFileFromURL(objectKey, req.URL)
	if err != nil {
		InternalError(c, "failed to upload from URL: "+err.Error())
		return
	}

	// 生成多级图片 URL
	imageLevels := aliyun.GenerateImageLevels(fileURL)

	Success(c, gin.H{
		"url":       fileURL,
		"objectKey": objectKey,
		"levels":    imageLevels,
	})
}

// UploadVideo 上传视频
// POST /api/upload/video
func (h *Handler) UploadVideo(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		InvalidParams(c, "file is required")
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if !isVideoType(contentType) {
		InvalidParams(c, "only video files are allowed")
		return
	}

	// 验证文件大小 (最大 100MB)
	maxSize := int64(100 * 1024 * 1024)
	if file.Size > maxSize {
		InvalidParams(c, "file size exceeds 100MB limit")
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".mp4"
	}
	newFilename := fmt.Sprintf("%s_%s%s", userID, uuid.New().String(), ext)

	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil {
		// 读取文件内容
		src, err := file.Open()
		if err != nil {
			InternalError(c, "failed to open file")
			return
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			InternalError(c, "failed to read file")
			return
		}

		// 上传到 OSS
		objectKey := fmt.Sprintf("videos/%s", newFilename)
		videoURL, err := ossClient.UploadVideoBytes(objectKey, data)
		if err != nil {
			InternalError(c, "failed to upload to OSS")
			return
		}

		Success(c, gin.H{
			"url":      videoURL,
			"filename": newFilename,
			"size":     file.Size,
			"type":     contentType,
			"storage":  "oss",
		})
		return
	}

	// 回退到本地存储
	uploadDir := "uploads/videos"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		InternalError(c, "failed to create upload directory")
		return
	}

	dst := filepath.Join(uploadDir, newFilename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		InternalError(c, "failed to save file")
		return
	}

	videoURL := fmt.Sprintf("/uploads/videos/%s", newFilename)

	Success(c, gin.H{
		"url":      videoURL,
		"filename": newFilename,
		"size":     file.Size,
		"type":     contentType,
		"storage":  "local",
	})
}

// GetSTSToken 获取阿里云 STS 临时凭证
// GET /api/upload/sts-token
func (h *Handler) GetSTSToken(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	ossClient := aliyun.GetGlobalClient()
	if ossClient == nil {
		InternalError(c, "OSS client not configured")
		return
	}

	credentials, err := ossClient.GetSTSToken()
	if err != nil {
		InternalError(c, "failed to get STS token: "+err.Error())
		return
	}

	Success(c, gin.H{
		"accessKeyId":     credentials.AccessKeyId,
		"accessKeySecret": credentials.AccessKeySecret,
		"securityToken":   credentials.SecurityToken,
		"expiration":      credentials.Expiration,
	})
}

// GetImageLevels 获取图片的多级 URL
// GET /api/upload/image-levels
func (h *Handler) GetImageLevels(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		InvalidParams(c, "url is required")
		return
	}

	levels := aliyun.GenerateImageLevels(url)

	Success(c, gin.H{
		"levels": levels,
	})
}

func isVideoType(contentType string) bool {
	allowedTypes := []string{
		"video/mp4",
		"video/mpeg",
		"video/quicktime",
		"video/x-msvideo",
		"video/webm",
	}

	for _, t := range allowedTypes {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}
	return false
}
