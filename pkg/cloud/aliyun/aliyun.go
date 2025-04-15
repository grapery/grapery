package aliyun

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"strings"

	"encoding/base64"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
)

var (
	APIKey       = "LTAI5t9opRTB3NKb3nBiikx5"
	SecretKey    = "YxeCMpnWeY82KLnElGVNaNZ4RdMJuI"
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
	client, err := oss.New(Endpoint, APIKey, SecretKey)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(Bucket)
	if err != nil {
		return nil, err
	}

	return &AliyunClient{
		client: client,
		bucket: bucket,
	}, nil
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

// 处理阿里云oss bucket的图片，生成对应的缩略图
func (c *AliyunClient) GenerateThumbnailV2(objectKey string, size int) (string, error) {
	// 生成目标图片的key
	pathSlice := strings.Split(objectKey, "/")
	id := strings.Split(pathSlice[len(pathSlice)-1], ".")[0]
	targetKey := fmt.Sprintf("thumbnail/%s.jpg", id)

	// 构建图片处理参数
	// 将图片缩放为固定宽高200px
	style := fmt.Sprintf("image/resize,m_fixed,w_%d,h_%d", size, size)
	// 使用base64编码目标文件名和bucket名
	process := fmt.Sprintf("%s|sys/saveas,o_%v,b_%v",
		style,
		base64.URLEncoding.EncodeToString([]byte(targetKey)),
		base64.URLEncoding.EncodeToString([]byte(Bucket)))

	// 执行图片处理
	_, err := c.bucket.ProcessObject(objectKey, process)
	if err != nil {
		return "", err
	}

	// 返回处理后的图片URL
	return c.GetFileURL(targetKey, 3600)
}
