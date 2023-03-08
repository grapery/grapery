package tencentai

import (
	"encoding/json"
	"base64"

	"github.com/grapery/grapery/utils/tencentcloud-sdk-go/common"
	"github.com/grapery/grapery/utils/tencentcloud-sdk-go/aiart/v20221229"
)
func TextToImage(){
	cred = credential.Credential("AKIDjJfnY4b2HKAmCcy72iAOPccN6i7HCwTm", "85oJZeOuBF8oGTVJtENUN2TikY7WkofN")
    httpProfile = HttpProfile()
    httpProfile.endpoint = "aiart.tencentcloudapi.com"
    clientProfile = ClientProfile()
    clientProfile.httpProfile = httpProfile
    client = aiart_client.AiartClient(cred, "ap-shanghai", clientProfile)
    req = models.TextToImageRequest()
    // params = {
    //     "Prompt": "new year,girl",
    //     "Styles": ["101"],
    //     "ReaultConfig": "1024*1024"
    // }
    req.from_json_string(json.dumps(params))
    resp = client.TextToImage(req)
    print(resp["RequestId"])
    imgdata = base64.b64decode(resp["ResultImage"])
    f = open("/go/src/github.com/grapery/grapery/service/tencent_ai/test.jpg", mode="wb")
    f.write(imgdata)
}
    
