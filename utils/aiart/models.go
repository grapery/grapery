// Copyright (c) 2017-2018 THL A29 Limited, a Tencent company. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aiart

import (
	"encoding/json"

	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
)

// Predefined struct for user
type ImageToImageRequestParams struct {
	// 输入图base64
	InputImage *string `json:"InputImage,omitempty" name:"InputImage"`

	// 输入图url，url和base64二选一必须传一个
	InputUrl *string `json:"InputUrl,omitempty" name:"InputUrl"`

	// 提示词，可用于微调生成图效果，推荐使用中文。最多支持512个utf-8字符
	Prompt *string `json:"Prompt,omitempty" name:"Prompt"`

	// 反向提示词，可用于拒绝生成图形成某种效果，推荐使用中文。最多支持512个utf-8字符
	NegativePrompt *string `json:"NegativePrompt,omitempty" name:"NegativePrompt"`

	// 绘画风格，详情可参见列表里的所有风格，不传默认使用201
	Styles []*string `json:"Styles,omitempty" name:"Styles"`

	// 生成图结果配置
	ResultConfig *ResultConfig `json:"ResultConfig,omitempty" name:"ResultConfig"`

	// 为生成结果图添加标识的开关，默认为1。
	// 1：添加标识。
	// 0：不添加标识。
	// 其他数值：默认按1处理。
	// 建议您使用显著标识来提示结果图使用了AI绘画技术，是AI生成的图片。
	LogoAdd *int64 `json:"LogoAdd,omitempty" name:"LogoAdd"`

	// 标识内容设置。
	// 默认在生成结果图右下角添加“图片由AI生成”字样，您可根据自身需要替换为其他的Logo图片。
	LogoParam *LogoParam `json:"LogoParam,omitempty" name:"LogoParam"`

	// 生成图和原图相似程度，值越小和原图越接近，取值范围0~1。不传默认为0.6
	Strength *float64 `json:"Strength,omitempty" name:"Strength"`
}

type ImageToImageRequest struct {
	*tchttp.BaseRequest

	// 输入图base64
	InputImage *string `json:"InputImage,omitempty" name:"InputImage"`

	// 输入图url，url和base64二选一必须传一个
	InputUrl *string `json:"InputUrl,omitempty" name:"InputUrl"`

	// 提示词，可用于微调生成图效果，推荐使用中文。最多支持512个utf-8字符
	Prompt *string `json:"Prompt,omitempty" name:"Prompt"`

	// 反向提示词，可用于拒绝生成图形成某种效果，推荐使用中文。最多支持512个utf-8字符
	NegativePrompt *string `json:"NegativePrompt,omitempty" name:"NegativePrompt"`

	// 绘画风格，详情可参见列表里的所有风格，不传默认使用201
	Styles []*string `json:"Styles,omitempty" name:"Styles"`

	// 生成图结果配置
	ResultConfig *ResultConfig `json:"ResultConfig,omitempty" name:"ResultConfig"`

	// 为生成结果图添加标识的开关，默认为1。
	// 1：添加标识。
	// 0：不添加标识。
	// 其他数值：默认按1处理。
	// 建议您使用显著标识来提示结果图使用了AI绘画技术，是AI生成的图片。
	LogoAdd *int64 `json:"LogoAdd,omitempty" name:"LogoAdd"`

	// 标识内容设置。
	// 默认在生成结果图右下角添加“图片由AI生成”字样，您可根据自身需要替换为其他的Logo图片。
	LogoParam *LogoParam `json:"LogoParam,omitempty" name:"LogoParam"`

	// 生成图和原图相似程度，值越小和原图越接近，取值范围0~1。不传默认为0.6
	Strength *float64 `json:"Strength,omitempty" name:"Strength"`
}

func (r *ImageToImageRequest) ToJsonString() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// FromJsonString It is highly **NOT** recommended to use this function
// because it has no param check, nor strict type check
func (r *ImageToImageRequest) FromJsonString(s string) error {
	f := make(map[string]interface{})
	if err := json.Unmarshal([]byte(s), &f); err != nil {
		return err
	}
	delete(f, "InputImage")
	delete(f, "InputUrl")
	delete(f, "Prompt")
	delete(f, "NegativePrompt")
	delete(f, "Styles")
	delete(f, "ResultConfig")
	delete(f, "LogoAdd")
	delete(f, "LogoParam")
	delete(f, "Strength")
	if len(f) > 0 {
		return tcerr.NewTencentCloudSDKError("ClientError.BuildRequestError", "ImageToImageRequest has unknown keys!", "")
	}
	return json.Unmarshal([]byte(s), &r)
}

// Predefined struct for user
type ImageToImageResponseParams struct {
	// 返回结果
	ResultImage *string `json:"ResultImage,omitempty" name:"ResultImage"`

	// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
	RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
}

type ImageToImageResponse struct {
	*tchttp.BaseResponse
	Response *ImageToImageResponseParams `json:"Response"`
}

func (r *ImageToImageResponse) ToJsonString() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// FromJsonString It is highly **NOT** recommended to use this function
// because it has no param check, nor strict type check
func (r *ImageToImageResponse) FromJsonString(s string) error {
	return json.Unmarshal([]byte(s), &r)
}

type LogoParam struct {
	// 水印url
	// 注意：此字段可能返回 null，表示取不到有效值。
	LogoUrl *string `json:"LogoUrl,omitempty" name:"LogoUrl"`

	// 水印base64，url和base64二选一传入
	// 注意：此字段可能返回 null，表示取不到有效值。
	LogoImage *string `json:"LogoImage,omitempty" name:"LogoImage"`

	// 水印图片位于融合结果图中的坐标，将按照坐标对标识图片进行位置和大小的拉伸匹配
	// 注意：此字段可能返回 null，表示取不到有效值。
	LogoRect *LogoRect `json:"LogoRect,omitempty" name:"LogoRect"`
}

type LogoRect struct {
	// 左上角X坐标
	// 注意：此字段可能返回 null，表示取不到有效值。
	X *int64 `json:"X,omitempty" name:"X"`

	// 左上角Y坐标
	// 注意：此字段可能返回 null，表示取不到有效值。
	Y *int64 `json:"Y,omitempty" name:"Y"`

	// 方框宽度
	// 注意：此字段可能返回 null，表示取不到有效值。
	Width *int64 `json:"Width,omitempty" name:"Width"`

	// 方框高度
	// 注意：此字段可能返回 null，表示取不到有效值。
	Height *int64 `json:"Height,omitempty" name:"Height"`
}

type ResultConfig struct {
	// 生成图支持分辨率，不传默认为"512*512"
	// 1:1尺寸支持"512:512"，"1024:1024"
	// 3:4尺寸支持"512:704"，"768:1024"
	// 9:16尺寸支持"448:704"
	// 4:3尺寸支持"704:512"，"1024:768"
	// 16:9尺寸支持"704:448"
	// 1:2尺寸支持”512:1024“
	//
	// 注意：此字段可能返回 null，表示取不到有效值。
	Resolution *string `json:"Resolution,omitempty" name:"Resolution"`
}

// Predefined struct for user
type TextToImageRequestParams struct {
	// 输入描述文本，算法会根据文本生成对应的图片。
	// 不能为空，推荐使用中文。最多可传512个utf-8字符
	Prompt *string `json:"Prompt,omitempty" name:"Prompt"`

	// 反向提示词，阻止算法生成对应类型的图片
	// 推荐使用中文。最多可传512个utf-8字符
	NegativePrompt *string `json:"NegativePrompt,omitempty" name:"NegativePrompt"`

	// 所选择的风格编号，支持风格叠加，推荐只使用一种风格。不传默认使用101
	Styles []*string `json:"Styles,omitempty" name:"Styles"`

	// 生成结果的配置，包括输出分辨率、尺寸、张数等
	ResultConfig *ResultConfig `json:"ResultConfig,omitempty" name:"ResultConfig"`

	// 为生成结果图添加标识的开关，默认为1。
	// 1：添加标识。
	// 0：不添加标识。
	// 其他数值：默认按1处理。
	// 建议您使用显著标识来提示结果图使用了AI绘画技术，是AI生成的图片。
	LogoAdd *int64 `json:"LogoAdd,omitempty" name:"LogoAdd"`

	// 标识内容设置。
	// 默认在生成结果图右下角添加“图片由AI生成”字样，您可根据自身需要替换为其他的Logo图片。
	LogoParam *LogoParam `json:"LogoParam,omitempty" name:"LogoParam"`
}

type TextToImageRequest struct {
	*tchttp.BaseRequest

	// 输入描述文本，算法会根据文本生成对应的图片。
	// 不能为空，推荐使用中文。最多可传512个utf-8字符
	Prompt *string `json:"Prompt,omitempty" name:"Prompt"`

	// 反向提示词，阻止算法生成对应类型的图片
	// 推荐使用中文。最多可传512个utf-8字符
	NegativePrompt *string `json:"NegativePrompt,omitempty" name:"NegativePrompt"`

	// 所选择的风格编号，支持风格叠加，推荐只使用一种风格。不传默认使用101
	Styles []*string `json:"Styles,omitempty" name:"Styles"`

	// 生成结果的配置，包括输出分辨率、尺寸、张数等
	ResultConfig *ResultConfig `json:"ResultConfig,omitempty" name:"ResultConfig"`

	// 为生成结果图添加标识的开关，默认为1。
	// 1：添加标识。
	// 0：不添加标识。
	// 其他数值：默认按1处理。
	// 建议您使用显著标识来提示结果图使用了AI绘画技术，是AI生成的图片。
	LogoAdd *int64 `json:"LogoAdd,omitempty" name:"LogoAdd"`

	// 标识内容设置。
	// 默认在生成结果图右下角添加“图片由AI生成”字样，您可根据自身需要替换为其他的Logo图片。
	LogoParam *LogoParam `json:"LogoParam,omitempty" name:"LogoParam"`
}

func (r *TextToImageRequest) ToJsonString() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// FromJsonString It is highly **NOT** recommended to use this function
// because it has no param check, nor strict type check
func (r *TextToImageRequest) FromJsonString(s string) error {
	f := make(map[string]interface{})
	if err := json.Unmarshal([]byte(s), &f); err != nil {
		return err
	}
	delete(f, "Prompt")
	delete(f, "NegativePrompt")
	delete(f, "Styles")
	delete(f, "ResultConfig")
	delete(f, "LogoAdd")
	delete(f, "LogoParam")
	if len(f) > 0 {
		return tcerr.NewTencentCloudSDKError("ClientError.BuildRequestError", "TextToImageRequest has unknown keys!", "")
	}
	return json.Unmarshal([]byte(s), &r)
}

// Predefined struct for user
type TextToImageResponseParams struct {
	// 返回的结果数组，当ResultConfig
	ResultImage *string `json:"ResultImage,omitempty" name:"ResultImage"`

	// 唯一请求 ID，每次请求都会返回。定位问题时需要提供该次请求的 RequestId。
	RequestId *string `json:"RequestId,omitempty" name:"RequestId"`
}

type TextToImageResponse struct {
	*tchttp.BaseResponse
	Response *TextToImageResponseParams `json:"Response"`
}

func (r *TextToImageResponse) ToJsonString() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// FromJsonString It is highly **NOT** recommended to use this function
// because it has no param check, nor strict type check
func (r *TextToImageResponse) FromJsonString(s string) error {
	return json.Unmarshal([]byte(s), &r)
}
