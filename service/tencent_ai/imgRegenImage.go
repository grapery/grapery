package tencentai

const BaseUrl = "aiart.tencentcloudapi.com"

type TextToImageReq struct {
	Action  string
	Version string
	Region  string
	Prompt  string
}

type TencentAI struct {
	Url       string
	AccountID string
	AppID     string
	Appkey    string
}

func NewTencentAI(url, appID, appKey string, accountID string) *TencentAI {
	return &TencentAI{
		Url:       url,
		AppID:     appID,
		Appkey:    appKey,
		AccountID: accountID,
	}
}
