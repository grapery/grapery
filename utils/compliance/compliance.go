package compliance

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	imageaudit20191230 "github.com/alibabacloud-go/imageaudit-20191230/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

var (
	APIKey    = os.Getenv("ALIYUN_API_KEY")
	SecretKey = os.Getenv("ALIYUN_SECRET_KEY")
)

// ComplianceContext 表示检测到的敏感内容上下文
type ComplianceContext struct {
	Context string `json:"Context"`
}

// ComplianceDetail 表示检测详情
type ComplianceDetail struct {
	Label    string              `json:"Label"`
	Contexts []ComplianceContext `json:"Contexts"`
}

// ComplianceResult 表示单个检测结果
type ComplianceResult struct {
	Suggestion string             `json:"Suggestion"` // "pass" | "block" | "review"
	Rate       float64            `json:"Rate"`       // 置信度
	Label      string             `json:"Label"`      // "normal" | "porn" | "politics" 等
	Details    []ComplianceDetail `json:"Details"`    // 详细信息（仅在违规时有值）
}

// ComplianceElement 表示单个任务的检测结果
type ComplianceElement struct {
	TaskId  string             `json:"TaskId"`
	Results []ComplianceResult `json:"Results"`
}

// AliCloudData 表示阿里云API返回的Data结构
type AliCloudData struct {
	Elements []ComplianceElement `json:"Elements"`
}

// AliCloudResponseBody 表示阿里云API返回的body结构
type AliCloudResponseBody struct {
	Data      AliCloudData `json:"Data"`
	RequestId string       `json:"RequestId"`
}

// AliCloudSDKResponse 表示阿里云SDK返回的完整响应结构
type AliCloudSDKResponse struct {
	Headers    map[string]interface{} `json:"headers"`
	StatusCode int                    `json:"statusCode"`
	Body       AliCloudResponseBody   `json:"body"`
}

// 保留原有的ComplianceResponse结构用于向后兼容
type ComplianceResponse struct {
	Success   bool                `json:"success"`
	Data      ComplianceInnerData `json:"data"`
	RequestId string              `json:"requestId"`
}

// ComplianceInnerData 表示阿里云API返回的内层data结构
type ComplianceInnerData struct {
	RequestId string `json:"RequestId"`
	Data      struct {
		Elements []ComplianceElement `json:"Elements"`
	} `json:"Data"`
}

// TextComplianceResult 表示文本合规检测的统一返回结果
type TextComplianceResult struct {
	IsCompliant   bool     `json:"isCompliant"`   // true: 合规, false: 违规
	Suggestion    string   `json:"suggestion"`    // "pass" | "block" | "review"
	Label         string   `json:"label"`         // 检测标签
	Rate          float64  `json:"rate"`          // 置信度
	ViolationInfo []string `json:"violationInfo"` // 违规信息详情
	RequestId     string   `json:"requestId"`     // 请求ID
}

// TextCompliance 检测文本内容是否合规
// 参数: content - 待检测的文本内容
// 返回: isCompliant - true表示合规，false表示违规; error - 错误信息
func TextCompliance(content string) (bool, error) {
	log.Printf("[TextCompliance] 开始检测文本内容，长度: %d 字符", len(content))
	if len(content) == 0 {
		log.Printf("[TextCompliance] 文本内容为空，默认为合规")
		return true, nil
	}
	// 调用阿里云文本审核API
	ret, err := DetectText(content)
	if err != nil {
		log.Printf("[TextCompliance] 调用DetectText失败: %v", err)
		return false, fmt.Errorf("文本合规检测失败: %w", err)
	}

	log.Printf("[TextCompliance] DetectText调用成功，开始解析响应")

	// 将响应转换为JSON字节数组进行解析
	jsonData, err := json.Marshal(ret)
	if err != nil {
		log.Printf("[TextCompliance] 序列化响应失败: %v", err)
		return false, fmt.Errorf("解析响应失败: %w", err)
	}
	log.Printf("[TextCompliance] 响应JSON数据: %s", string(jsonData))

	// 解析阿里云响应数据
	responseBody, err := parseAliCloudResponse(jsonData)
	if err != nil {
		log.Printf("[TextCompliance] 解析响应失败: %v", err)
		return false, fmt.Errorf("解析响应数据失败: %w", err)
	}

	log.Printf("[TextCompliance] 响应解析成功，RequestId: %s", responseBody.RequestId)
	responseData, _ := json.Marshal(responseBody)
	log.Printf("[TextCompliance] 响应数据: %s", string(responseData))

	// 检查是否有检测结果
	if len(responseBody.Data.Elements) == 0 {
		log.Printf("[TextCompliance] 无检测结果，默认为合规")
		return true, nil
	}

	// 遍历所有检测结果
	for i, element := range responseBody.Data.Elements {
		log.Printf("[TextCompliance] 处理第%d个任务结果，TaskId: %s", i+1, element.TaskId)

		if len(element.Results) == 0 {
			log.Printf("[TextCompliance] 任务%s无结果", element.TaskId)
			continue
		}

		for j, result := range element.Results {
			log.Printf("[TextCompliance] 处理第%d个检测结果 - Suggestion: %s, Label: %s, Rate: %.2f",
				j+1, result.Suggestion, result.Label, result.Rate)

			// 根据建议判断合规性
			switch result.Suggestion {
			case "block":
				// 违规内容，记录详细信息
				var violations []string
				for _, detail := range result.Details {
					for _, ctx := range detail.Contexts {
						violations = append(violations, fmt.Sprintf("标签:%s, 内容:%s", detail.Label, ctx.Context))
					}
				}
				log.Printf("[TextCompliance] 检测到违规内容 - 标签: %s, 置信度: %.2f, 违规详情: %v",
					result.Label, result.Rate, violations)
				return false, nil

			case "review":
				// 需要人工审核，按违规处理
				log.Printf("[TextCompliance] 内容需要人工审核 - 标签: %s, 置信度: %.2f",
					result.Label, result.Rate)
				continue

			case "pass":
				// 合规内容
				log.Printf("[TextCompliance] 内容合规 - 标签: %s, 置信度: %.2f",
					result.Label, result.Rate)
				continue

			default:
				// 未知建议，按违规处理
				log.Printf("[TextCompliance] 未知的建议类型: %s，按违规处理", result.Suggestion)
				continue
			}
		}
	}

	log.Printf("[TextCompliance] 所有检测结果均显示内容合规")
	return true, nil
}

// TextComplianceDetail 检测文本内容是否合规，并返回详细结果
// 参数: content - 待检测的文本内容
// 返回: TextComplianceResult - 详细的检测结果结构体; error - 错误信息
func TextComplianceDetail(content string) (*TextComplianceResult, error) {
	log.Printf("[TextComplianceDetail] 开始详细检测文本内容，长度: %d 字符", len(content))

	// 调用阿里云文本审核API
	ret, err := DetectText(content)
	if err != nil {
		log.Printf("[TextComplianceDetail] 调用DetectText失败: %v", err)
		return &TextComplianceResult{
			IsCompliant:   false,
			Suggestion:    "error",
			Label:         "error",
			Rate:          0,
			ViolationInfo: []string{fmt.Sprintf("检测失败: %s", err.Error())},
			RequestId:     "",
		}, fmt.Errorf("文本合规检测失败: %w", err)
	}

	log.Printf("[TextComplianceDetail] DetectText调用成功，开始解析响应")

	// 将响应转换为JSON字节数组进行解析
	jsonData, err := json.Marshal(ret)
	if err != nil {
		log.Printf("[TextComplianceDetail] 序列化响应失败: %v", err)
		return &TextComplianceResult{
			IsCompliant:   false,
			Suggestion:    "error",
			Label:         "error",
			Rate:          0,
			ViolationInfo: []string{fmt.Sprintf("解析响应失败: %s", err.Error())},
			RequestId:     "",
		}, fmt.Errorf("解析响应失败: %w", err)
	}

	// 解析阿里云响应数据
	responseBody, err := parseAliCloudResponse(jsonData)
	if err != nil {
		log.Printf("[TextComplianceDetail] 解析响应失败: %v", err)
		return &TextComplianceResult{
			IsCompliant:   false,
			Suggestion:    "error",
			Label:         "error",
			Rate:          0,
			ViolationInfo: []string{fmt.Sprintf("解析响应失败: %s", err.Error())},
			RequestId:     "",
		}, fmt.Errorf("解析响应数据失败: %w", err)
	}

	log.Printf("[TextComplianceDetail] 响应解析成功，RequestId: %s", responseBody.RequestId)

	// 初始化返回结果
	result := &TextComplianceResult{
		IsCompliant:   true,
		Suggestion:    "pass",
		Label:         "normal",
		Rate:          0,
		ViolationInfo: []string{},
		RequestId:     responseBody.RequestId,
	}

	// 检查是否有检测结果
	if len(responseBody.Data.Elements) == 0 {
		log.Printf("[TextComplianceDetail] 无检测结果，默认为合规")
		return result, nil
	}

	// 遍历所有检测结果，找到最严重的违规
	for i, element := range responseBody.Data.Elements {
		log.Printf("[TextComplianceDetail] 处理第%d个任务结果，TaskId: %s", i+1, element.TaskId)

		if len(element.Results) == 0 {
			log.Printf("[TextComplianceDetail] 任务%s无结果", element.TaskId)
			continue
		}

		for j, complianceResult := range element.Results {
			log.Printf("[TextComplianceDetail] 处理第%d个检测结果 - Suggestion: %s, Label: %s, Rate: %.2f",
				j+1, complianceResult.Suggestion, complianceResult.Label, complianceResult.Rate)

			// 更新结果信息（选择置信度最高的）
			if complianceResult.Rate > result.Rate {
				result.Rate = complianceResult.Rate
				result.Label = complianceResult.Label
				result.Suggestion = complianceResult.Suggestion
			}

			// 根据建议判断合规性
			switch complianceResult.Suggestion {
			case "block":
				// 违规内容，记录详细信息
				result.IsCompliant = false
				for _, detail := range complianceResult.Details {
					for _, ctx := range detail.Contexts {
						violationMsg := fmt.Sprintf("违规标签: %s, 内容: %s", detail.Label, ctx.Context)
						result.ViolationInfo = append(result.ViolationInfo, violationMsg)
					}
				}
				log.Printf("[TextComplianceDetail] 检测到违规内容 - 标签: %s, 置信度: %.2f",
					complianceResult.Label, complianceResult.Rate)

			case "review":
				// 需要人工审核，按违规处理
				result.IsCompliant = false
				result.ViolationInfo = append(result.ViolationInfo,
					fmt.Sprintf("需要人工审核 - 标签: %s, 置信度: %.2f", complianceResult.Label, complianceResult.Rate))
				log.Printf("[TextComplianceDetail] 内容需要人工审核 - 标签: %s, 置信度: %.2f",
					complianceResult.Label, complianceResult.Rate)

			case "pass":
				// 合规内容
				log.Printf("[TextComplianceDetail] 内容合规 - 标签: %s, 置信度: %.2f",
					complianceResult.Label, complianceResult.Rate)

			default:
				// 未知建议，按违规处理
				result.IsCompliant = false
				result.ViolationInfo = append(result.ViolationInfo,
					fmt.Sprintf("未知检测结果 - 建议: %s, 标签: %s", complianceResult.Suggestion, complianceResult.Label))
				log.Printf("[TextComplianceDetail] 未知的建议类型: %s，按违规处理", complianceResult.Suggestion)
			}
		}
	}

	if result.IsCompliant {
		log.Printf("[TextComplianceDetail] 最终结果: 内容合规")
	} else {
		log.Printf("[TextComplianceDetail] 最终结果: 内容违规，违规项数量: %d", len(result.ViolationInfo))
	}

	return result, nil
}

// parseAliCloudResponse 解析阿里云响应，支持不同的响应格式
func parseAliCloudResponse(jsonData []byte) (*AliCloudResponseBody, error) {
	// 首先尝试解析SDK格式的响应
	var sdkResponse AliCloudSDKResponse
	if err := json.Unmarshal(jsonData, &sdkResponse); err == nil && sdkResponse.StatusCode > 0 {
		// 检查HTTP状态码
		if sdkResponse.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP状态码异常: %d", sdkResponse.StatusCode)
		}
		return &sdkResponse.Body, nil
	}

	// 如果SDK格式解析失败，尝试直接解析响应体格式
	var bodyResponse AliCloudResponseBody
	if err := json.Unmarshal(jsonData, &bodyResponse); err == nil && bodyResponse.RequestId != "" {
		return &bodyResponse, nil
	}

	// 如果都解析失败，返回错误
	return nil, fmt.Errorf("无法解析响应格式")
}

// CreateClient 使用凭据初始化账号Client
func CreateClient() (*imageaudit20191230.Client, error) {
	cfg := &credential.Config{
		AccessKeyId:     tea.String(APIKey),
		AccessKeySecret: tea.String(SecretKey),
		Type:            tea.String("access_key"),
	}
	cred, err := credential.NewCredential(cfg)
	if err != nil {
		return nil, err
	}
	config := &openapi.Config{
		Credential: cred,
	}
	config.Endpoint = tea.String("imageaudit.cn-shanghai.aliyuncs.com")
	return imageaudit20191230.NewClient(config)
}

// DetectText 检测文本内容是否合规，返回检测结果和错误信息
func DetectText(content string) (interface{}, error) {
	client, err := CreateClient()
	if err != nil {
		return nil, err
	}

	labels := []*imageaudit20191230.ScanTextRequestLabels{
		{Label: tea.String("spam")},
		{Label: tea.String("politics")},
		{Label: tea.String("abuse")},
		{Label: tea.String("terrorism")},
		{Label: tea.String("porn")},
		{Label: tea.String("flood")},
		{Label: tea.String("contraband")},
		{Label: tea.String("ad")},
	}
	tasks := []*imageaudit20191230.ScanTextRequestTasks{
		{Content: tea.String(content)},
	}
	scanTextRequest := &imageaudit20191230.ScanTextRequest{
		Tasks:  tasks,
		Labels: labels,
	}
	runtime := &util.RuntimeOptions{}

	resp, err := client.ScanTextWithOptions(scanTextRequest, runtime)
	if err != nil {
		var sdkErr *tea.SDKError
		if t, ok := err.(*tea.SDKError); ok {
			sdkErr = t
		} else {
			sdkErr = &tea.SDKError{Message: tea.String(err.Error())}
		}
		var data interface{}
		d := json.NewDecoder(strings.NewReader(tea.StringValue(sdkErr.Data)))
		d.Decode(&data)
		return nil, fmt.Errorf("检测失败: %v, 推荐: %v", tea.StringValue(sdkErr.Message), data)
	}
	return resp, nil
}
