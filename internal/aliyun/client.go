package aliyun

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"strings"
	"sync"

	stssdk "github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	globalClient *Client
	once         sync.Once
)

// Config holds Aliyun OSS configuration
type Config struct {
	APIKey    string
	SecretKey string
	Endpoint  string
	Bucket    string
	RoleARN   string // for STS token
}

// Client wraps Aliyun OSS client
type Client struct {
	config *Config
	client *oss.Client
	bucket *oss.Bucket
	logger *zap.Logger
}

// NewClient creates a new Aliyun OSS client
func NewClient(cfg *Config, logger *zap.Logger) (*Client, error) {
	if cfg.APIKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("ALIYUN_API_KEY and ALIYUN_SECRET_KEY are required")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("ALIYUN_BUCKET is required")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "oss-cn-shanghai.aliyuncs.com"
	}

	client, err := oss.New(cfg.Endpoint, cfg.APIKey, cfg.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %w", err)
	}

	bucket, err := client.Bucket(cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &Client{
		config: cfg,
		client: client,
		bucket: bucket,
		logger: logger,
	}, nil
}

// InitGlobalClient initializes the global Aliyun client (singleton)
func InitGlobalClient(cfg *Config, logger *zap.Logger) error {
	var initErr error
	once.Do(func() {
		globalClient, initErr = NewClient(cfg, logger)
	})
	return initErr
}

// GetGlobalClient returns the global Aliyun client
func GetGlobalClient() *Client {
	return globalClient
}

// UploadFile uploads a file from local path to OSS
func (c *Client) UploadFile(objectKey string, filePath string) (string, error) {
	err := c.bucket.PutObjectFromFile(objectKey, filePath)
	if err != nil {
		c.logger.Error("failed to upload file", zap.Error(err), zap.String("objectKey", objectKey))
		return "", err
	}
	return c.GetFileURL(objectKey, 3600)
}

// UploadBytes uploads byte data to OSS
func (c *Client) UploadBytes(objectKey string, data []byte) (string, error) {
	err := c.bucket.PutObject(objectKey, bytes.NewReader(data))
	if err != nil {
		c.logger.Error("failed to upload bytes", zap.Error(err), zap.String("objectKey", objectKey))
		return "", err
	}
	return c.GetFileURL(objectKey, 3600)
}

// UploadVideoBytes uploads raw video bytes to OSS and returns a cleaned public URL
func (c *Client) UploadVideoBytes(objectKey string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("video content is empty")
	}
	if objectKey == "" {
		objectKey = fmt.Sprintf("videos/%s.mp4", uuid.New().String())
	}

	url, err := c.UploadBytes(objectKey, data)
	if err != nil {
		return "", err
	}

	cleaned := strings.Split(url, "?")[0]
	cleaned = strings.ReplaceAll(cleaned, "http://", "https://")
	return cleaned, nil
}

// UploadFileFromURL downloads a file from URL and uploads it to OSS
func (c *Client) UploadFileFromURL(objectKey string, url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download from URL: %w", err)
	}
	defer response.Body.Close()

	if objectKey == "" {
		objectKey = fmt.Sprintf("%s.jpg", uuid.New().String())
	}

	c.logger.Debug("uploading file from URL", zap.String("objectKey", objectKey), zap.String("url", url))

	err = c.bucket.PutObject(objectKey, response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to upload to OSS: %w", err)
	}

	newURL, err := c.GetFileURL(objectKey, 3600)
	if err != nil {
		return "", err
	}

	// Persist multi-level images
	imageLevels, err := c.PersistMultiLevelImages(objectKey)
	if err != nil {
		c.logger.Warn("failed to persist multi-level images", zap.Error(err), zap.String("objectKey", objectKey))
		// Continue even if multi-level persistence fails
	} else {
		levelData, _ := json.Marshal(imageLevels)
		c.logger.Debug("multi-level images persisted", zap.String("levels", string(levelData)))
	}

	// Clean URL: remove query params and ensure HTTPS
	newURL = strings.Split(newURL, "?")[0]
	newURL = strings.ReplaceAll(newURL, "http://", "https://")

	return newURL, nil
}

// DownloadFile downloads a file from OSS to local path
func (c *Client) DownloadFile(objectKey string, filePath string) error {
	return c.bucket.GetObjectToFile(objectKey, filePath)
}

// GetFileURL returns a signed URL for the file
func (c *Client) GetFileURL(objectKey string, expiredInSec int64) (string, error) {
	signedURL, err := c.bucket.SignURL(objectKey, oss.HTTPGet, expiredInSec)
	if err != nil {
		return "", err
	}
	return signedURL, nil
}

// DeleteFile deletes a file from OSS
func (c *Client) DeleteFile(objectKey string) error {
	return c.bucket.DeleteObject(objectKey)
}

// GenerateThumbnail generates a thumbnail for the image
func (c *Client) GenerateThumbnail(objectKey string) (string, error) {
	// Extract ID from object key
	pathSlice := strings.Split(objectKey, "/")
	id := strings.Split(pathSlice[len(pathSlice)-1], ".")[0]

	imgReader, err := c.bucket.GetObject(objectKey)
	if err != nil {
		return "", fmt.Errorf("failed to get object: %w", err)
	}
	defer imgReader.Close()

	// Decode image
	img, _, err := image.Decode(imgReader)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Generate thumbnail (half size)
	thumbnail := image.NewRGBA(image.Rect(0, 0, width/2, height/2))
	for y := 0; y < height/2; y++ {
		for x := 0; x < width/2; x++ {
			thumbnail.Set(x, y, img.At(x*2, y*2))
		}
	}

	// Encode thumbnail as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	// Upload thumbnail
	thumbnailKey := fmt.Sprintf("thumbnail/%s.jpg", id)
	_, err = c.UploadBytes(thumbnailKey, buf.Bytes())
	if err != nil {
		return "", err
	}

	return c.GetFileURL(thumbnailKey, 3600)
}

// ImageLevels represents different resolution levels of an image
type ImageLevels struct {
	Original  string `json:"original"`  // Original image
	Content   string `json:"content"`   // For content display (max 1280x1280)
	Preview   string `json:"preview"`   // For preview (max 512x512)
	Thumbnail string `json:"thumbnail"` // For avatar/thumbnail (max 200x200)
	Small     string `json:"small"`     // Small icon (max 64x64)
}

// GenerateImageLevels generates OSS image processing URLs for different levels
func GenerateImageLevels(originalURL string) ImageLevels {
	// Aliyun OSS Image Processing Parameters
	// Reference: https://help.aliyun.com/zh/oss/user-guide/resize-images-4
	return ImageLevels{
		Original:  originalURL,
		Content:   originalURL + "?x-oss-process=image/resize,m_lfit,w_1280,h_1280",
		Preview:   originalURL + "?x-oss-process=image/resize,m_lfit,w_512,h_512",
		Thumbnail: originalURL + "?x-oss-process=image/resize,m_lfit,w_200,h_200",
		Small:     originalURL + "?x-oss-process=image/resize,m_lfit,w_64,h_64",
	}
}

// PersistMultiLevelImages persists multi-level images to OSS
// Returns ImageLevels with direct URLs for all resolution levels
func (c *Client) PersistMultiLevelImages(objectKey string) (ImageLevels, error) {
	// Generate target object names
	baseName := strings.TrimSuffix(objectKey, ".jpg")
	contentObj := fmt.Sprintf("%s_content.jpg", baseName)
	previewObj := fmt.Sprintf("%s_preview.jpg", baseName)
	thumbnailObj := fmt.Sprintf("%s_thumbnail.jpg", baseName)
	smallObj := fmt.Sprintf("%s_small.jpg", baseName)

	c.logger.Debug("persisting multi-level images",
		zap.String("objectKey", objectKey),
		zap.String("contentObj", contentObj),
		zap.String("previewObj", previewObj),
		zap.String("thumbnailObj", thumbnailObj),
		zap.String("smallObj", smallObj))

	// Define processing parameters for each level
	levels := []struct {
		Style     string
		TargetObj string
	}{
		{"image/resize,m_lfit,w_1280,h_1280", contentObj},
		{"image/resize,m_lfit,w_512,h_512", previewObj},
		{"image/resize,m_lfit,w_200,h_200", thumbnailObj},
		{"image/resize,m_lfit,w_64,h_64", smallObj},
	}

	// Get original image URL
	originalURL, _ := c.GetFileURL(objectKey, 3600)
	originalURL = strings.Split(originalURL, "?")[0]
	originalURL = strings.ReplaceAll(originalURL, "http://", "https://")

	result := ImageLevels{Original: originalURL}

	// Persist each level
	for _, lv := range levels {
		process := fmt.Sprintf("%s|sys/saveas,o_%s,b_%s",
			lv.Style,
			base64.URLEncoding.EncodeToString([]byte(lv.TargetObj)),
			base64.URLEncoding.EncodeToString([]byte(c.config.Bucket)),
		)
		_, err := c.bucket.ProcessObject(objectKey, process)
		if err != nil {
			c.logger.Warn("failed to process object", zap.Error(err), zap.String("objectKey", objectKey), zap.String("targetObj", lv.TargetObj))
			return result, err
		}

		url, _ := c.GetFileURL(lv.TargetObj, 3600)
		url = strings.Split(url, "?")[0]
		url = strings.ReplaceAll(url, "http://", "https://")

		switch lv.TargetObj {
		case contentObj:
			result.Content = url
		case previewObj:
			result.Preview = url
		case thumbnailObj:
			result.Thumbnail = url
		case smallObj:
			result.Small = url
		}
	}

	return result, nil
}

// ListAllObjects lists all objects in the bucket with the given prefix
func (c *Client) ListAllObjects(prefix string) ([]string, error) {
	var allKeys []string
	marker := ""

	for {
		lsRes, err := c.bucket.ListObjects(oss.Prefix(prefix), oss.Marker(marker), oss.MaxKeys(1000))
		if err != nil {
			return nil, err
		}

		for _, obj := range lsRes.Objects {
			// Filter out directories (keys ending with /)
			if !strings.HasSuffix(obj.Key, "/") {
				allKeys = append(allKeys, obj.Key)
			}
		}

		if lsRes.IsTruncated {
			marker = lsRes.NextMarker
		} else {
			break
		}
	}

	return allKeys, nil
}

// STSCredentials represents STS temporary credentials
type STSCredentials struct {
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SecurityToken   string `json:"securityToken"`
	Expiration      string `json:"expiration"`
}

// GetSTSToken returns Aliyun STS temporary credentials
func (c *Client) GetSTSToken() (*STSCredentials, error) {
	if c.config.RoleARN == "" {
		return nil, errors.New("ALIYUN_ROLE_ARN is not set")
	}

	// Create STS client
	client, err := stssdk.NewClientWithAccessKey("cn-shanghai", c.config.APIKey, c.config.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create STS client: %w", err)
	}

	// Build AssumeRole request
	req := stssdk.CreateAssumeRoleRequest()
	req.Scheme = "https"
	req.RoleArn = c.config.RoleARN
	req.RoleSessionName = "grapery-session"
	req.DurationSeconds = "1200" // 20 minutes

	// Call AssumeRole to get temporary credentials
	resp, err := client.AssumeRole(req)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	cred := resp.Credentials
	return &STSCredentials{
		AccessKeyId:     cred.AccessKeyId,
		AccessKeySecret: cred.AccessKeySecret,
		SecurityToken:   cred.SecurityToken,
		Expiration:      cred.Expiration,
	}, nil
}
