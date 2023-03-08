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

import warnings

from tencentcloud.common.abstract_model import AbstractModel


class ImageToImageRequest(AbstractModel):
    """ImageToImage请求参数结构体

    """

    def __init__(self):
        r"""
        :param InputImage: 输入图base64
        :type InputImage: str
        :param InputUrl: 输入图url，url和base64二选一必须传一个
        :type InputUrl: str
        :param Prompt: 提示词，可用于微调生成图效果，推荐使用中文。最多支持512个utf-8字符
        :type Prompt: str
        :param NegativePrompt: 反向提示词，可用于拒绝生成图形成某种效果，推荐使用中文。最多支持512个utf-8字符
        :type NegativePrompt: str
        :param Styles: 绘画风格，详情可参见列表里的所有风格，不传默认使用201
        :type Styles: list of str
        :param ResultConfig: 生成图结果配置
        :type ResultConfig: :class:`tencentcloud.aiart.v20221229.models.ResultConfig`
        :param LogoAdd: 为生成结果图添加标识的开关，默认为1。
1：添加标识。
0：不添加标识。
其他数值：默认按1处理。
建议您使用显著标识来提示结果图使用了AI绘画技术，是AI生成的图片。
        :type LogoAdd: int
        :param LogoParam: 标识内容设置。
默认在生成结果图右下角添加“图片由AI生成”字样，您可根据自身需要替换为其他的Logo图片。
        :type LogoParam: :class:`tencentcloud.aiart.v20221229.models.LogoParam`
        :param Strength: 生成图和原图相似程度，值越小和原图越接近，取值范围0~1。不传默认为0.6
        :type Strength: float
        """
        self.InputImage = None
        self.InputUrl = None
        self.Prompt = None
        self.NegativePrompt = None
        self.Styles = None
        self.ResultConfig = None
        self.LogoAdd = None
        self.LogoParam = None
        self.Strength = None


    def _deserialize(self, params):
        self.InputImage = params.get("InputImage")
        self.InputUrl = params.get("InputUrl")
        self.Prompt = params.get("Prompt")
        self.NegativePrompt = params.get("NegativePrompt")
        self.Styles = params.get("Styles")
        if params.get("ResultConfig") is not None:
            self.ResultConfig = ResultConfig()
            self.ResultConfig._deserialize(params.get("ResultConfig"))
        self.LogoAdd = params.get("LogoAdd")
        if params.get("LogoParam") is not None:
            self.LogoParam = LogoParam()
            self.LogoParam._deserialize(params.get("LogoParam"))
        self.Strength = params.get("Strength")
        memeber_set = set(params.keys())
        for name, value in vars(self).items():
            if name in memeber_set:
                memeber_set.remove(name)
        if len(memeber_set) > 0:
            warnings.warn("%s fileds are useless." % ",".join(memeber_set))
        


class ImageToImageResponse(AbstractModel):
    """ImageToImage返回参数结构体

    """

    def __init__(self):
        r"""
        :param ResultImage: 返回结果
        :type ResultImage: str
        :param RequestId: 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
        :type RequestId: str
        """
        self.ResultImage = None
        self.RequestId = None


    def _deserialize(self, params):
        self.ResultImage = params.get("ResultImage")
        self.RequestId = params.get("RequestId")


class LogoParam(AbstractModel):
    """logo参数

    """

    def __init__(self):
        r"""
        :param LogoUrl: 水印url
注意：此字段可能返回 null，表示取不到有效值。
        :type LogoUrl: str
        :param LogoImage: 水印base64，url和base64二选一传入
注意：此字段可能返回 null，表示取不到有效值。
        :type LogoImage: str
        :param LogoRect: 水印图片位于融合结果图中的坐标，将按照坐标对标识图片进行位置和大小的拉伸匹配
注意：此字段可能返回 null，表示取不到有效值。
        :type LogoRect: :class:`tencentcloud.aiart.v20221229.models.LogoRect`
        """
        self.LogoUrl = None
        self.LogoImage = None
        self.LogoRect = None


    def _deserialize(self, params):
        self.LogoUrl = params.get("LogoUrl")
        self.LogoImage = params.get("LogoImage")
        if params.get("LogoRect") is not None:
            self.LogoRect = LogoRect()
            self.LogoRect._deserialize(params.get("LogoRect"))
        memeber_set = set(params.keys())
        for name, value in vars(self).items():
            if name in memeber_set:
                memeber_set.remove(name)
        if len(memeber_set) > 0:
            warnings.warn("%s fileds are useless." % ",".join(memeber_set))
        


class LogoRect(AbstractModel):
    """输入框

    """

    def __init__(self):
        r"""
        :param X: 左上角X坐标
注意：此字段可能返回 null，表示取不到有效值。
        :type X: int
        :param Y: 左上角Y坐标
注意：此字段可能返回 null，表示取不到有效值。
        :type Y: int
        :param Width: 方框宽度
注意：此字段可能返回 null，表示取不到有效值。
        :type Width: int
        :param Height: 方框高度
注意：此字段可能返回 null，表示取不到有效值。
        :type Height: int
        """
        self.X = None
        self.Y = None
        self.Width = None
        self.Height = None


    def _deserialize(self, params):
        self.X = params.get("X")
        self.Y = params.get("Y")
        self.Width = params.get("Width")
        self.Height = params.get("Height")
        memeber_set = set(params.keys())
        for name, value in vars(self).items():
            if name in memeber_set:
                memeber_set.remove(name)
        if len(memeber_set) > 0:
            warnings.warn("%s fileds are useless." % ",".join(memeber_set))
        


class ResultConfig(AbstractModel):
    """返回结果配置

    """

    def __init__(self):
        r"""
        :param Resolution: 生成图支持分辨率，不传默认为"512*512"
1:1尺寸支持"512:512"，"1024:1024"
3:4尺寸支持"512:704"，"768:1024"
9:16尺寸支持"448:704"
4:3尺寸支持"704:512"，"1024:768"
16:9尺寸支持"704:448"
1:2尺寸支持”512:1024“

注意：此字段可能返回 null，表示取不到有效值。
        :type Resolution: str
        """
        self.Resolution = None


    def _deserialize(self, params):
        self.Resolution = params.get("Resolution")
        memeber_set = set(params.keys())
        for name, value in vars(self).items():
            if name in memeber_set:
                memeber_set.remove(name)
        if len(memeber_set) > 0:
            warnings.warn("%s fileds are useless." % ",".join(memeber_set))
        


class TextToImageRequest(AbstractModel):
    """TextToImage请求参数结构体

    """

    def __init__(self):
        r"""
        :param Prompt: 输入描述文本，算法会根据文本生成对应的图片。
不能为空，推荐使用中文。最多可传512个utf-8字符
        :type Prompt: str
        :param NegativePrompt: 反向提示词，阻止算法生成对应类型的图片
推荐使用中文。最多可传512个utf-8字符
        :type NegativePrompt: str
        :param Styles: 所选择的风格编号，支持风格叠加，推荐只使用一种风格。不传默认使用101
        :type Styles: list of str
        :param ResultConfig: 生成结果的配置，包括输出分辨率、尺寸、张数等
        :type ResultConfig: :class:`tencentcloud.aiart.v20221229.models.ResultConfig`
        :param LogoAdd: 为生成结果图添加标识的开关，默认为1。
1：添加标识。
0：不添加标识。
其他数值：默认按1处理。
建议您使用显著标识来提示结果图使用了AI绘画技术，是AI生成的图片。
        :type LogoAdd: int
        :param LogoParam: 标识内容设置。
默认在生成结果图右下角添加“图片由AI生成”字样，您可根据自身需要替换为其他的Logo图片。
        :type LogoParam: :class:`tencentcloud.aiart.v20221229.models.LogoParam`
        """
        self.Prompt = None
        self.NegativePrompt = None
        self.Styles = None
        self.ResultConfig = None
        self.LogoAdd = None
        self.LogoParam = None


    def _deserialize(self, params):
        self.Prompt = params.get("Prompt")
        self.NegativePrompt = params.get("NegativePrompt")
        self.Styles = params.get("Styles")
        if params.get("ResultConfig") is not None:
            self.ResultConfig = ResultConfig()
            self.ResultConfig._deserialize(params.get("ResultConfig"))
        self.LogoAdd = params.get("LogoAdd")
        if params.get("LogoParam") is not None:
            self.LogoParam = LogoParam()
            self.LogoParam._deserialize(params.get("LogoParam"))
        memeber_set = set(params.keys())
        for name, value in vars(self).items():
            if name in memeber_set:
                memeber_set.remove(name)
        if len(memeber_set) > 0:
            warnings.warn("%s fileds are useless." % ",".join(memeber_set))
        


class TextToImageResponse(AbstractModel):
    """TextToImage返回参数结构体

    """

    def __init__(self):
        r"""
        :param ResultImage: 返回的结果数组，当ResultConfig
        :type ResultImage: str
        :param RequestId: 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
        :type RequestId: str
        """
        self.ResultImage = None
        self.RequestId = None


    def _deserialize(self, params):
        self.ResultImage = params.get("ResultImage")
        self.RequestId = params.get("RequestId")