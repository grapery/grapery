package aliyun

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"os"
	"strings"

	"encoding/base64"

	stssdk "github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
	"github.com/grapery/grapery/utils/log"
)

var (
	APIKey       = os.Getenv("ALIYUN_API_KEY")
	SecretKey    = os.Getenv("ALIYUN_SECRET_KEY")
	Endpoint     = "oss-cn-shanghai.aliyuncs.com" // 比如: oss-cn-hangzhou.aliyuncs.com
	Bucket       = "grapery-dev"
	GlobalClient *AliyunClient
)

func init() {
	GlobalClient, _ = NewAliyunClient()
}

func GetGlobalClient() *AliyunClient {
	return GlobalClient
}

type AliyunClient struct {
	client *oss.Client
	bucket *oss.Bucket
}

func NewAliyunClient() (*AliyunClient, error) {
	if APIKey == "" || SecretKey == "" || Bucket == "" {
		return nil, errors.New("ALIYUN_API_KEY, ALIYUN_SECRET_KEY, ALIYUN_BUCKET is not set")
	}
	client, err := oss.New(Endpoint, APIKey, SecretKey)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(Bucket)
	if err != nil {
		return nil, err
	}

	aliyunClient := &AliyunClient{
		client: client,
		bucket: bucket,
	}
	return aliyunClient, nil
}

// UploadFile 上传文件到阿里云 OSS
func (c *AliyunClient) UploadFile(objectKey string, filePath string) (string, error) {
	err := c.bucket.PutObjectFromFile(objectKey, filePath)
	if err != nil {
		return "", err
	}
	return c.GetFileURL(objectKey, 3600)
}

// UploadBytes 上传字节数据到阿里云 OSS
func (c *AliyunClient) UploadBytes(objectKey string, data []byte) (string, error) {
	err := c.bucket.PutObject(objectKey, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	return c.GetFileURL(objectKey, 3600)
}

// UploadFileFromURL 从URL上传文件到阿里云 OSS
func (c *AliyunClient) UploadFileFromURL(objectKey string, url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if objectKey == "" {
		objectKey = fmt.Sprintf("images/%s.jpg", uuid.New().String())
	}
	fmt.Println("UploadFileFromURL: ", objectKey, url)
	err = c.bucket.PutObject(objectKey, response.Body)
	if err != nil {
		return "", err
	}
	// http://grapery-dev.oss-cn-shanghai.aliyuncs.com/images%2F5e407747-9553-425f-aa0f-73c926847ca4.jpg?Expires=1744559510&OSSAccessKeyId=LTAI5t9opRTB3NKb3nBiikx5&Signature=MUK6qt5H0cVvZNy4rQNlp9GTUgg%3D
	newUrl, err := c.GetFileURL(objectKey, 3600)
	if err != nil {
		return "", err
	}
	imageLevels, err := c.PersistMultiLevelImages(objectKey)
	if err != nil {
		return "", err
	}
	log.Log().Sugar().Infof("imageLevels: %v", imageLevels)
	// 去除掉url中的Expires和Signature
	newUrl = strings.Split(newUrl, "?")[0]
	newUrl = strings.ReplaceAll(newUrl, "http://", "https://")
	return newUrl, nil
}

// DownloadFile 从阿里云 OSS 下载文件
func (c *AliyunClient) DownloadFile(objectKey string, filePath string) error {
	return c.bucket.GetObjectToFile(objectKey, filePath)
}

// GetFileURL 获取文件的访问URL
func (c *AliyunClient) GetFileURL(objectKey string, expiredInSec int64) (string, error) {
	signedURL, err := c.bucket.SignURL(objectKey, oss.HTTPGet, expiredInSec)
	if err != nil {
		return "", err
	}
	return signedURL, nil
}

// DeleteFile 删除阿里云 OSS 上的文件
func (c *AliyunClient) DeleteFile(objectKey string) error {
	return c.bucket.DeleteObject(objectKey)
}

// 处理阿里云oss bucket的图片，生成对应的缩略图
func (c *AliyunClient) GenerateThumbnail(objectKey string) (string, error) {
	// 获取图片
	pathSlice := strings.Split(objectKey, "/")
	id := strings.Split(pathSlice[len(pathSlice)-1], ".")[0]
	imgReader, err := c.bucket.GetObject(objectKey)
	if err != nil {
		return "", err
	}
	defer imgReader.Close()

	// 解码图片
	img, _, err := image.Decode(imgReader)
	if err != nil {
		return "", err
	}

	// 获取图片的宽高
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 生成缩略图
	thumbnail := image.NewRGBA(image.Rect(0, 0, width/2, height/2))

	// 使用基本的图像绘制功能进行缩小
	// 这是一个简单的像素复制方法，在实际应用中可能需要更高级的缩放算法
	for y := 0; y < height/2; y++ {
		for x := 0; x < width/2; x++ {
			thumbnail.Set(x, y, img.At(x*2, y*2))
		}
	}

	// 将缩略图编码为JPEG格式
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return "", err
	}

	// 上传缩略图
	thumbnailKey := fmt.Sprintf("thumbnail/%s.jpg", id)
	_, err = c.UploadBytes(thumbnailKey, buf.Bytes())
	if err != nil {
		return "", err
	}
	return c.GetFileURL(thumbnailKey, 3600)
}

// GenerateImageLevels 根据原始 OSS 图片 URL 生成不同等级的图片 URL
func GenerateImageLevels(originalURL string) ImageLevels {
	// 阿里云 OSS 图片处理参数
	// 参考：https://help.aliyun.com/zh/oss/user-guide/resize-images-4
	return ImageLevels{
		Original:  originalURL,
		Content:   originalURL + "?x-oss-process=image/resize,m_lfit,w_1280,h_1280",
		Preview:   originalURL + "?x-oss-process=image/resize,m_lfit,w_512,h_512",
		Thumbnail: originalURL + "?x-oss-process=image/resize,m_lfit,w_200,h_200",
		Small:     originalURL + "?x-oss-process=image/resize,m_lfit,w_64,h_64",
	}
}

// ImageLevels 表示不同等级的图片 OSS 直链
// 用于多级图片持久化和访问
// original: 原图
// content: 内容展示
// preview: 预览
// thumbnail: 头像
// small: 小图
type ImageLevels struct {
	Original  string // 原图
	Content   string // 内容展示
	Preview   string // 预览
	Thumbnail string // 头像
	Small     string // 小图
}

// PersistMultiLevelImages 持久化多级图片到 OSS，并返回各级图片的直链
// objectKey: 原图在 OSS 的 object key（如 images/xxx.jpg）
// 返回 ImageLevels 结构体，包含所有等级图片的直链
func (c *AliyunClient) PersistMultiLevelImages(objectKey string) (ImageLevels, error) {
	// 生成目标图片名（去除.jpg后缀，拼接不同等级后缀）
	baseName := strings.TrimSuffix(objectKey, ".jpg")
	contentObj := fmt.Sprintf("%s_content.jpg", baseName)
	previewObj := fmt.Sprintf("%s_preview.jpg", baseName)
	thumbnailObj := fmt.Sprintf("%s_thumbnail.jpg", baseName)
	smallObj := fmt.Sprintf("%s_small.jpg", baseName)

	// 定义各级图片的处理参数和目标 object key
	levels := []struct {
		Style     string
		TargetObj string
	}{
		{"image/resize,m_lfit,w_1280,h_1280", contentObj},
		{"image/resize,m_lfit,w_512,h_512", previewObj},
		{"image/resize,m_lfit,w_200,h_200", thumbnailObj},
		{"image/resize,m_lfit,w_64,h_64", smallObj},
	}

	// 获取原图直链
	originalUrl, _ := c.GetFileURL(objectKey, 3600)
	originalUrl = strings.Split(originalUrl, "?")[0]
	originalUrl = strings.ReplaceAll(originalUrl, "http://", "https://")
	result := ImageLevels{Original: originalUrl}

	// 持久化每一级图片
	for _, lv := range levels {
		process := fmt.Sprintf("%s|sys/saveas,o_%s,b_%s",
			lv.Style,
			base64.URLEncoding.EncodeToString([]byte(lv.TargetObj)),
			base64.URLEncoding.EncodeToString([]byte(Bucket)),
		)
		_, err := c.bucket.ProcessObject(objectKey, process)
		if err != nil {
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

// ListAllObjects 遍历 bucket 下所有目录和文件，返回所有文件的 object key 列表
// prefix: 指定前缀（如 "original/"），为空则遍历整个 bucket
// 返回所有文件（非目录）的 object key 列表
func (c *AliyunClient) ListAllObjects(prefix string) ([]string, error) {
	var allKeys []string
	marker := ""
	for {
		lsRes, err := c.bucket.ListObjects(oss.Prefix(prefix), oss.Marker(marker), oss.MaxKeys(1000))
		if err != nil {
			return nil, err
		}
		for _, obj := range lsRes.Objects {
			// 过滤掉"目录"，只处理文件
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

// STSCredentials 表示 STS 临时凭证
type STSCredentials struct {
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SecurityToken   string `json:"securityToken"`
	Expiration      string `json:"expiration"`
}

// GetSTSToken 获取阿里云 STS 临时凭证
func (c *AliyunClient) GetSTSToken() (*STSCredentials, error) {
	roleArn := os.Getenv("ALIYUN_ROLE_ARN")
	if roleArn == "" {
		return nil, errors.New("ALIYUN_ROLE_ARN is not set")
	}

	// 创建 STS 客户端
	client, err := stssdk.NewClientWithAccessKey("cn-shanghai", APIKey, SecretKey)
	if err != nil {
		return nil, err
	}

	// 构造 AssumeRole 请求
	req := stssdk.CreateAssumeRoleRequest()
	req.Scheme = "https"
	req.RoleArn = roleArn
	req.RoleSessionName = "grapery-dev"
	req.DurationSeconds = "1200" // 2分钟

	// 调用 AssumeRole 获取临时凭证
	resp, err := client.AssumeRole(req)
	if err != nil {
		return nil, err
	}

	cred := resp.Credentials
	return &STSCredentials{
		AccessKeyId:     cred.AccessKeyId,
		AccessKeySecret: cred.AccessKeySecret,
		SecurityToken:   cred.SecurityToken,
		Expiration:      cred.Expiration,
	}, nil
}
