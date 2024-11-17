package zhipu

import "github.com/grapery/grapery/pkg/cloud/zhipu/zhipu"

const (
	ZhipuAPIURL = "https://open.zhipu.com/api/v1"
	ZhipuToken  = "f6d850f112e9d3c8840ffebe687f63d5.q5c3jxnZlxizkAsP"
)

type ZhipuAPI struct {
	*zhipu.Client
}

func NewZhipuAPI() *ZhipuAPI {
	client, err := zhipu.NewClient(zhipu.WithAPIKey(ZhipuToken))
	if err != nil {
		return nil
	}
	return &ZhipuAPI{
		Client: client,
	}
}
