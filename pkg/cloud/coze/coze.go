package coze

import (
	"errors"
	"os"
)

var (
	APIKey     = os.Getenv("COZE_API_KEY")
	Endpoint   = "https://api.coze.cn"
	AppName    = "grapery"
	APPID      = "7521236942802206759"
	CozeClient *HuoShanCozeClient
)

func init() {
	CozeClient, _ = NewCozeClient()
}

func GetCozeClient() *HuoShanCozeClient {
	return CozeClient
}

type HuoShanCozeClient struct {
}

func NewCozeClient() (*HuoShanCozeClient, error) {
	if APIKey == "" {
		return nil, errors.New("COZE_API_KEY environment variable is not set")
	}
	client := &HuoShanCozeClient{}
	return client, nil
}

func (c *HuoShanCozeClient) GetAPIKey() string {
	return os.Getenv("COZE_API_KEY")
}

func (c *HuoShanCozeClient) RefreshToken() string {
	return ""
}
