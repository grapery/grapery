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
		Desc: "日系动漫",
		Kind: "浮光",
		Code: "101",
	},

	"102": {
		Desc: "日系动漫",
		Kind: "飞羽",
		Code: "102",
	},

	"103": {
		Desc: "日系动漫",
		Kind: "云海",
		Code: "103",
	},

	"104": {
		Desc: "日系动漫",
		Kind: "圣诞",
		Code: "104",
	},

	"105": {
		Desc: "日系动漫",
		Kind: "新年",
		Code: "105",
	},

	"201": {
		Desc: "动漫类",
		Kind: "日系动漫",
		Code: "201",
	},

	"202": {
		Desc: "动漫类",
		Kind: "可爱动漫",
		Code: "202",
	},

	"203": {
		Desc: "动漫类",
		Kind: "唯美古风",
		Code: "203",
	},

	"204": {
		Desc: "动漫类",
		Kind: "魔幻风格",
		Code: "204",
	},

	"205": {
		Desc: "动漫类",
		Kind: "美系动漫",
		Code: "205",
	},

	"206": {
		Desc: "动漫类",
		Kind: "间谍日漫",
		Code: "206",
	},

	"207": {
		Desc: "动漫类",
		Kind: "花式艺术动漫",
		Code: "207",
	},

	"208": {
		Desc: "动漫类",
		Kind: "病娇风",
		Code: "208",
	},

	"209": {
		Desc: "动漫类",
		Kind: "性感动漫",
		Code: "209",
	},
	"210": {
		Desc: "动漫类",
		Kind: "唯美日漫",
		Code: "210",
	},
	"211": {
		Desc: "动漫类",
		Kind: "纯真动漫",
		Code: "211",
	},
	"212": {
		Desc: "动漫类",
		Kind: "漫画男孩",
		Code: "212",
	},
	"213": {
		Desc: "动漫类",
		Kind: "丑萌风",
		Code: "213",
	},
	"301": {
		Desc: "游戏类",
		Kind: "性感动漫",
		Code: "301",
	},
	"302": {
		Desc: "游戏类",
		Kind: "Q版卡通风格",
		Code: "302",
	},
	"303": {
		Desc: "游戏类",
		Kind: "杀马特风",
		Code: "303",
	},
	"304": {
		Desc: "游戏类",
		Kind: "厚涂画风",
		Code: "304",
	},
	"305": {
		Desc: "游戏类",
		Kind: "欧洲中古世纪风",
		Code: "305",
	},
	"306": {
		Desc: "游戏类",
		Kind: "电子游戏",
		Code: "306",
	},
	"401": {
		Desc: "传统绘画类",
		Kind: "中国艺术",
		Code: "401",
	},
	"402": {
		Desc: "传统绘画类",
		Kind: "水彩画",
		Code: "402",
	},
	"403": {
		Desc: "传统绘画类",
		Kind: "日系艺术",
		Code: "403",
	},
	"404": {
		Desc: "传统绘画类",
		Kind: "数码绘画",
		Code: "404",
	},
	"405": {
		Desc: "传统绘画类",
		Kind: "中世纪",
		Code: "405",
	},
	"501": {
		Desc: "视觉风格类",
		Kind: "Q版",
		Code: "501",
	},
	"502": {
		Desc: "视觉风格类",
		Kind: "装甲概念",
		Code: "502",
	},
	"503": {
		Desc: "视觉风格类",
		Kind: "英雄主义幻想",
		Code: "503",
	},
	"504": {
		Desc: "视觉风格类",
		Kind: "梦幻女孩",
		Code: "504",
	},
	"505": {
		Desc: "视觉风格类",
		Kind: "科幻艺术",
		Code: "505",
	},
	"506": {
		Desc: "视觉风格类",
		Kind: "机械黑暗风",
		Code: "506",
	},
	"507": {
		Desc: "视觉风格类",
		Kind: "黑暗幻想艺术",
		Code: "507",
	},
	"508": {
		Desc: "视觉风格类",
		Kind: "哥特艺术",
		Code: "508",
	},
	"509": {
		Desc: "视觉风格类",
		Kind: "真人卡通风格",
		Code: "509",
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
