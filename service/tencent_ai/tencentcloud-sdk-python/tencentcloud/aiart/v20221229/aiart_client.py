# -*- coding: utf8 -*-
# Copyright (c) 2017-2021 THL A29 Limited, a Tencent company. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json

from tencentcloud.common.exception.tencent_cloud_sdk_exception import TencentCloudSDKException
from tencentcloud.common.abstract_client import AbstractClient
from tencentcloud.aiart.v20221229 import models


class AiartClient(AbstractClient):
    _apiVersion = '2022-12-29'
    _endpoint = 'aiart.tencentcloudapi.com'
    _service = 'aiart'


    def ImageToImage(self, request):
        """根据一段文本和输入图片AI绘画生成结果图片的接口

        输入图限制：单边分辨率小于2000，转成base64字符串后小于10MB
        输入图推荐：宽高比接近所选尺寸最佳，否则可能裁剪重要主体

        输出图：对应尺寸的AI生成图

        Style参数支持的风格类目表

        |风格大类|风格细项|风格取值|
        |-|-|-|
        |日系动漫|浮光|101|
        |日系动漫|飞羽|102|
        |日系动漫|云海|103|
        |日系动漫|圣诞|104|
        |日系动漫|新年|105|
        |动漫类|日系动漫|201|
        |动漫类|可爱动漫|202|
        |动漫类|唯美古风|203|
        |动漫类|魔幻风格|204|
        |动漫类|美系动漫|205|
        |动漫类|间谍日漫|206|
        |动漫类|花式艺术动漫|207|
        |动漫类|病娇风|208|
        |动漫类|性感动漫|209|
        |动漫类|唯美日漫|210|
        |动漫类|纯真动漫|211|
        |动漫类|漫画男孩|212|
        |动漫类|丑萌风|213|
        |游戏类|美式卡通风|301|
        |游戏类|Q版卡通风格|302|
        |游戏类|杀马特风|303|
        |游戏类|厚涂画风|304|
        |游戏类|欧洲中古世纪风|305|
        |游戏类|电子游戏|306|
        |传统绘画类|中国艺术|401|
        |传统绘画类|水彩画|402|
        |传统绘画类|日系艺术|403|
        |传统绘画类|数码绘画|404|
        |传统绘画类|中世纪|405|
        |视觉风格类|Q版|501|
        |视觉风格类|装甲概念|502|
        |视觉风格类|英雄主义幻想|503|
        |视觉风格类|梦幻女孩|504|
        |视觉风格类|科幻艺术|505|
        |视觉风格类|机械黑暗风|506|
        |视觉风格类|黑暗幻想艺术|507|
        |视觉风格类|哥特艺术|508|
        |视觉风格类|真人卡通风格|509|

        :param request: Request instance for ImageToImage.
        :type request: :class:`tencentcloud.aiart.v20221229.models.ImageToImageRequest`
        :rtype: :class:`tencentcloud.aiart.v20221229.models.ImageToImageResponse`

        """
        try:
            params = request._serialize()
            headers = request.headers
            body = self.call("ImageToImage", params, headers=headers)
            response = json.loads(body)
            if "Error" not in response["Response"]:
                model = models.ImageToImageResponse()
                model._deserialize(response["Response"])
                return model
            else:
                code = response["Response"]["Error"]["Code"]
                message = response["Response"]["Error"]["Message"]
                reqid = response["Response"]["RequestId"]
                raise TencentCloudSDKException(code, message, reqid)
        except Exception as e:
            if isinstance(e, TencentCloudSDKException):
                raise
            else:
                raise TencentCloudSDKException(e.message, e.message)


    def TextToImage(self, request):
        """根据一段输入的描述文本生成特定场景的结果图

        输入：512个字符以内，包括中英文字符和符号
        输出：对应尺寸的AI生成图

        可支持风格如下，若需选择对应风格需将风格编号传入Styles数组

        |风格大类|风格细项|风格编号|
        |-|-|-|
        |传统绘画类|水墨画|101|
        |传统绘画类|马赛克|102|
        |传统绘画类|油画|103|
        |传统绘画类|水彩画|104|
        |传统绘画类|中国画|105|
        |传统绘画类|卡通画|106|
        |传统绘画类|绘画|107|
        |传统绘画类|剪纸主义|108|
        |传统绘画类|印象主义|109|
        |漫画类|插画日漫|201|
        |漫画类|美式漫画|202|
        |漫画类|中国风漫画|203|
        |漫画类|唯美日漫|204|
        |漫画类|可爱日漫|205|
        |游戏类|元气漫游|301|
        |游戏类|塔防建模|302|
        |游戏类|重锤建模|303|
        |游戏类|悠仙美地|304|
        |游戏类|信仰未来|305|
        |游戏类|菲利普科幻|306|
        |游戏类|机械建模|307|
        |游戏类|硬核生物|308|
        |游戏类|伊藤手绘|309|
        |游戏类|光晕手绘|310|
        |游戏类|渲染手绘|311|
        |游戏类|异类手绘|312|
        |游戏类|刺客手绘|313|
        |视觉风格类|梦幻风格|401|
        |视觉风格类|哥特艺术|402|
        |视觉风格类|黑暗艺术|403|
        |视觉风格类|人偶风|404|
        |视觉风格类|3D|405|
        |视觉风格类|Q版|406|

        :param request: Request instance for TextToImage.
        :type request: :class:`tencentcloud.aiart.v20221229.models.TextToImageRequest`
        :rtype: :class:`tencentcloud.aiart.v20221229.models.TextToImageResponse`

        """
        try:
            params = request._serialize()
            headers = request.headers
            body = self.call("TextToImage", params, headers=headers)
            response = json.loads(body)
            if "Error" not in response["Response"]:
                model = models.TextToImageResponse()
                model._deserialize(response["Response"])
                return model
            else:
                code = response["Response"]["Error"]["Code"]
                message = response["Response"]["Error"]["Message"]
                reqid = response["Response"]["RequestId"]
                raise TencentCloudSDKException(code, message, reqid)
        except Exception as e:
            if isinstance(e, TencentCloudSDKException):
                raise
            else:
                raise TencentCloudSDKException(e.message, e.message)