package tencentai

import (
	common "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	profile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	aiart "github.com/grapery/grapery/utils/aiart"
)

const (
	BaseUrl    = "aiart.tencentcloudapi.com"
	BaseRegion = "ap-shanghai"
)

const (
	ResolutionLevelDefault    = "512*512"
	ResolutionLevel_1024_1024 = "1024:1024"
	ResolutionLevel_512_704   = "512:704"
	ResolutionLevel_768_1024  = "768:1024"
	ResolutionLevel_448_704   = "448:704"
	ResolutionLevel_704_512   = "704:512"
	ResolutionLevel_1024_768  = "1024:768"
	ResolutionLevel_704_448   = "704:448"
	ResolutionLevel_512_1024  = "512:1024"
)

type StyleInfo struct {
	Desc string
	Kind string
	Code string
}

var StyleTypeMap = map[string]StyleInfo{
	"101": {
		Desc: "艺术风格类",
		Kind: "水墨画",
		Code: "101",
	},
	"102": {
		Desc: "艺术风格类",
		Kind: "概念艺术",
		Code: "102",
	},
	"103": {
		Desc: "艺术风格类",
		Kind: "油画",
		Code: "103",
	},
	"104": {
		Desc: "艺术风格类",
		Kind: "水彩画",
		Code: "104",
	},
	"106": {
		Desc: "艺术风格类",
		Kind: "厚涂风格",
		Code: "106",
	},
	"107": {
		Desc: "艺术风格类",
		Kind: "插图",
		Code: "107",
	},
	"108": {
		Desc: "艺术风格类",
		Kind: "剪纸风格",
		Code: "108",
	},
	"109": {
		Desc: "艺术风格类",
		Kind: "印象派",
		Code: "109",
	},
	"110": {
		Desc: "艺术风格类",
		Kind: "2.5d人像",
		Code: "110",
	},
	"111": {
		Desc: "艺术风格类",
		Kind: "肖像画",
		Code: "111",
	},
	"112": {
		Desc: "艺术风格类",
		Kind: "黑白素描",
		Code: "112",
	},
	"113": {
		Desc: "艺术风格类",
		Kind: "赛博朋克",
		Code: "113",
	},
	"115": {
		Desc: "艺术风格类",
		Kind: "黑暗艺术",
		Code: "115",
	},
	"201": {
		Desc: "艺术风格类",
		Kind: "日系动漫",
		Code: "201",
	},
	"202": {
		Desc: "艺术风格类",
		Kind: "怪兽风格",
		Code: "202",
	},
	"301": {
		Desc: "艺术风格类",
		Kind: "游戏卡通手绘",
		Code: "301",
	},
}

type TextToImageReq struct {
	Action  string
	Version string
	Region  string
	Prompt  string
}

type TencentAI struct {
	AppID       string
	Appkey      string
	AIArtClient *aiart.Client
}

func NewTencentAI(appID, appKey string) (*TencentAI, error) {
	cred := common.NewCredential(appID, appKey)
	clientProfile := profile.NewClientProfile()
	clientProfile.HttpProfile.RootDomain = BaseUrl
	clientProfile.HttpProfile.Endpoint = BaseUrl
	aiClient, err := aiart.NewClient(cred, BaseRegion, clientProfile)
	if err != nil {
		println("new client error: ", err.Error())
		return nil, err
	}
	_ = aiClient
	return &TencentAI{
		AppID:       appID,
		Appkey:      appKey,
		AIArtClient: aiClient,
	}, nil
}
