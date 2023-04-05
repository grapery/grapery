package s3

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/tencentyun/cos-go-sdk-v5"
)

/*avator,jpg,png,content*/

const (
	DefaultBucket       = "grapery-1301865260"
	DefaultBucketUrl    = "https://grapery-1301865260.cos.ap-shanghai.myqcloud.com"
	DefaultServiceQuery = "service.cos.myqcloud.com"
	SecreID             = ""
	SecretKey           = ""
)

type ResourceType int

const (
	ResourceTypeAvator  ResourceType = 1
	ResourceTypeContent ResourceType = 2
	ResourceTypeMusic   ResourceType = 3
	ResourceTypeImage   ResourceType = 4
	ResourceTypeVideo   ResourceType = 5
)

func (r ResourceType) String() string {
	switch r {
	case ResourceTypeAvator:
		return "avator"
	case ResourceTypeContent:
		return "content"
	case ResourceTypeMusic:
		return "music"
	case ResourceTypeImage:
		return "image"
	case ResourceTypeVideo:
		return "video"
	}
	return ""
}

type S3Client struct {
	TencentS3Client *cos.Client
}

func NewS3Client() *S3Client {
	u, _ := url.Parse(DefaultBucketUrl)
	su, _ := url.Parse(DefaultServiceQuery)
	b := &cos.BaseURL{BucketURL: u, ServiceURL: su}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("SECRETID"),
			SecretKey: os.Getenv("SECRETKEY"),
		},
	})
	return &S3Client{
		TencentS3Client: client,
	}
}

func (s3c *S3Client) CreateBucket(ctx context.Context, name string) (bool, error) {
	opt := cos.BucketPutOptions{}
	resp, err := s3c.TencentS3Client.Bucket.Put(context.Background(), &opt)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, fmt.Errorf("create bucket %s failed", name)
}

func (s3c *S3Client) ListBuckets(ctx context.Context) ([]string, error) {
	s, _, err := s3c.TencentS3Client.Service.Get(context.Background())
	if err != nil {
		panic(err)
	}
	var result = make([]string, 0)
	for _, b := range s.Buckets {
		result = append(result, b.Name)
	}
	return result, nil
}

func (s3c *S3Client) UploadContent(ctx context.Context, data []byte, name string, res_type int) (bool, error) {
	reader := bytes.NewReader(data)
	resp, err := s3c.TencentS3Client.Object.Put(context.Background(), name, reader, nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, fmt.Errorf("upload content %s failed", name)
}

func (s3c *S3Client) QueryContent(ctx context.Context, name string, res_type int, prefix string) ([]string, error) {
	opt := &cos.BucketGetOptions{
		Prefix:  prefix,
		MaxKeys: 10,
	}
	v, _, err := s3c.TencentS3Client.Bucket.Get(context.Background(), opt)
	if err != nil {
		return nil, err
	}
	var result = make([]string, 0)
	for _, c := range v.Contents {
		result = append(result, c.Key)
	}
	return result, nil
}

func (s3c *S3Client) GetContent(ctx context.Context, name string, res_type int) ([]byte, error) {
	resp, err := s3c.TencentS3Client.Object.Get(context.Background(), name, nil)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}

func (s3c *S3Client) DeleteContent(ctx context.Context, name string, res_type int) (bool, error) {
	resp, err := s3c.TencentS3Client.Object.Delete(context.Background(), name)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, fmt.Errorf("delete content %s failed", name)
}
