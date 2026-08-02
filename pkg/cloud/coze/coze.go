package coze

import (
	"os"
)

var (
	APIKey        = os.Getenv("COZE_API_KEY")
	Endpoint      = "https://api.coze.cn"
	AppName       = "grapery"
	APPID         = "7521236942802206759"
	SPACEID       = "7357736388571971647"
	SERVICE_TOKEN = "sat_H0nclBVwArEr40xNizXxCCfB6aCyIdTVKRTrpX90zV0SloZsSAH6xNGDwVEKWjRJ"
	CozeClient    *HuoShanCozeClient
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
	APIKey = SERVICE_TOKEN
	client := &HuoShanCozeClient{}
	return client, nil
}

func (c *HuoShanCozeClient) GetAPIKey() string {
	return SERVICE_TOKEN
}

func (c *HuoShanCozeClient) RefreshToken() string {
	return ""
}
