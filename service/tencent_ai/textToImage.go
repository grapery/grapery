package tencentai

import (
	aiart "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/aiart/v"
	common "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	profile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func TextToImage() {
	cred := common.Credential("AKIDjJfnY4b2HKAmCcy72iAOPccN6i7HCwTm", "85oJZeOuBF8oGTVJtENUN2TikY7WkofN")
	//"aiart.tencentcloudapi.com"
	clientProfile := profile.ClientProfile{}
	aiClient, err := aiart.NewClient(cred, "ap-shanghai", clientProfile)
	if err != nil {
		println("new client error: ", err.Error())
		return
	}
	_ = aiClient
	// req = models.TextToImageRequest()
	// // params = {
	// //     "Prompt": "new year,girl",
	// //     "Styles": ["101"],
	// //     "ReaultConfig": "1024*1024"
	// // }
	// req.from_json_string(json.dumps(params))
	// resp = client.TextToImage(req)
	// print(resp["RequestId"])
	// imgdata = base64.b64decode(resp["ResultImage"])
	// f = open("/go/src/github.com/grapery/grapery/service/tencent_ai/test.jpg", mode="wb")
	// f.write(imgdata)
}
