import json
from tencentcloud.common import credential
from tencentcloud.common.profile.client_profile import ClientProfile
from tencentcloud.common.profile.http_profile import HttpProfile
from tencentcloud.common.exception.tencent_cloud_sdk_exception import TencentCloudSDKException
from tencentcloud.aiart.v20221229 import aiart_client, models

try:
    # 实例化一个认证对象，入参需要传入腾讯云账户secretId，secretKey,此处还需注意密钥对的保密
    # 密钥可前往https://console.cloud.tencent.com/cam/capi网站进行获取
    cred = credential.Credential("SecretId", "SecretKey")
    # 实例化一个http选项，可选的，没有特殊需求可以跳过
    httpProfile = HttpProfile()
    httpProfile.endpoint = "aiart.tencentcloudapi.com"

    # 实例化一个client选项，可选的，没有特殊需求可以跳过
    clientProfile = ClientProfile()
    clientProfile.httpProfile = httpProfile
    # 实例化要请求产品的client对象,clientProfile是可选的
    client = aiart_client.AiartClient(cred, "ap-guangzhou", clientProfile)

    # 实例化一个请求对象,每个接口都会对应一个request对象
    req = models.TextToImageRequest()
    params = {
        "Prompt": "新年、日漫、漂亮的女孩子",
        "Styles": ["101"],
        "ReaultConfig": "1024*1024"
    }
    req.from_json_string(json.dumps(params))
    # 返回的resp是一个ImageToImageGeneralResponse的实例，与请求对象对应
    resp = client.TextToImage(req)
    # 输出json格式的字符串回包
    print(resp.to_json_string())


except TencentCloudSDKException as err:
    print(err)
