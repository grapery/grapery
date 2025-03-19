package aliyun

import (
	"bytes"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var (
	APIKey    = ""
	SecretKey = ""
	Endpoint  = "oss-cn-shanghai.aliyuncs.com" // 比如: oss-cn-hangzhou.aliyuncs.com
	Bucket    = "grapery-dev"                  // 你的 Bucket 名称
)

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
func (c *AliyunClient) UploadFile(objectKey string, filePath string) error {
	return c.bucket.PutObjectFromFile(objectKey, filePath)
}

// UploadBytes 上传字节数据到阿里云 OSS
func (c *AliyunClient) UploadBytes(objectKey string, data []byte) error {
	return c.bucket.PutObject(objectKey, bytes.NewReader(data))
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
