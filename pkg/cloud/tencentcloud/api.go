package tencentcloud

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/google/uuid"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func UploadObject(imageData []byte, fileType string) (string, error) {
	bucket := "https://grapery-1301865260.cos.ap-shanghai.myqcloud.com"
	u, _ := url.Parse("https://grapery-1301865260.cos.ap-shanghai.myqcloud.com")
	b := &cos.BaseURL{BucketURL: u}
	c := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("TSECRETID"),
			SecretKey: os.Getenv("TSECRETKEY"),
		},
	})
	var name string
	if fileType == "jpg" {
		name = "avator/" + uuid.New().String() + ".jpg"
	} else if fileType == "png" {
		name = "avator/" + uuid.New().String() + ".png"
	} else if fileType == "gif" {
		name = "pic/" + uuid.New().String() + ".gif"
	} else if fileType == "jpeg" {
		name = "avator/" + uuid.New().String() + ".jpeg"
	} else if fileType == "txt" {
		name = "content/" + uuid.New().String() + ".txt"
	} else if fileType == "mp4" {
		name = "video/" + uuid.New().String() + ".mp4"
	} else {
		return "", fmt.Errorf("unsupported file type")
	}
	f := bytes.NewReader(imageData)

	_, err := c.Object.Put(context.Background(), name, f, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", bucket, name), nil
}

func DownloadObject() {
	// 将 examplebucket-1250000000 和 COS_REGION 修改为真实的信息
	// 存储桶名称，由 bucketname-appid 组成，appid 必须填入，可以在 COS 控制台查看存储桶名称。https://console.cloud.tencent.com/cos5/bucket
	// COS_REGION 可以在控制台查看，https://console.cloud.tencent.com/cos5/bucket, 关于地域的详情见 https://cloud.tencent.com/document/product/436/6224
	u, _ := url.Parse("https://examplebucket-1250000000.cos.COS_REGION.myqcloud.com")
	b := &cos.BaseURL{BucketURL: u}
	c := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("SECRETID"),  // 用户的 SecretId，建议使用子账号密钥，授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考 https://cloud.tencent.com/document/product/598/37140
			SecretKey: os.Getenv("SECRETKEY"), // 用户的 SecretKey，建议使用子账号密钥，授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考 https://cloud.tencent.com/document/product/598/37140
		},
	})
	// 1.通过响应体获取对象
	name := "test/objectPut.go"
	resp, err := c.Object.Get(context.Background(), name, nil)
	if err != nil {
		panic(err)
	}
	bs, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("%s\n", string(bs))
	// 2.获取对象到本地文件
	_, err = c.Object.GetToFile(context.Background(), name, "exampleobject", nil)
	if err != nil {
		panic(err)
	}
}
