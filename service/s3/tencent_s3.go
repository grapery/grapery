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
	DefaultBucket = "grapery-1301865260"
	SECRETID      = ""
	SecretKey     = ""
)

type S3Client struct {
	TencentS3Client *cos.Client
}

func NewS3Client() *S3Client {
	// 将 examplebucket-1250000000 和 COS_REGION 修改为用户真实的信息
	// 存储桶名称，由 bucketname-appid 组成，appid 必须填入，可以在 COS 控制台查看存储桶名称。https://console.cloud.tencent.com/cos5/bucket
	// COS_REGION 可以在控制台查看，https://console.cloud.tencent.com/cos5/bucket, 关于地域的详情见 https://cloud.tencent.com/document/product/436/6224
	u, _ := url.Parse("https://examplebucket-1250000000.cos.COS_REGION.myqcloud.com")
	// 用于 Get Service 查询，默认全地域 service.cos.myqcloud.com
	su, _ := url.Parse("https://cos.COS_REGION.myqcloud.com")
	b := &cos.BaseURL{BucketURL: u, ServiceURL: su}
	// 1.永久密钥
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("SECRETID"),  // 用户的 SecretId，建议使用子账号密钥，授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考 https://cloud.tencent.com/document/product/598/37140
			SecretKey: os.Getenv("SECRETKEY"), // 用户的 SecretKey，建议使用子账号密钥，授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考 https://cloud.tencent.com/document/product/598/37140
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
