package service

// SendAliyunOTPCode uses Alibaba Cloud SMS OpenAPI SendSms (2017-05-25).
//
// Required env:
//   - ALIYUN_SMS_ACCESS_KEY_ID, ALIYUN_SMS_ACCESS_KEY_SECRET
//   - ALIYUN_SMS_SIGN_NAME（短信签名）
//   - ALIYUN_SMS_TEMPLATE_CODE（模板 CODE，模板变量需包含 JSON 字段 code）
//
// Optional:
//   - ALIYUN_SMS_REGION（默认 cn-hangzhou，用于 Endpoint 解析）

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

// SendAliyunOTPCode sends a 6-digit verification SMS via Alibaba Cloud SMS (China).
func SendAliyunOTPCode(domesticPhone, code string) error {
	domesticPhone = strings.TrimSpace(domesticPhone)
	if domesticPhone == "" {
		return fmt.Errorf("empty phone")
	}

	region := strings.TrimSpace(os.Getenv("ALIYUN_SMS_REGION"))
	if region == "" {
		region = "cn-hangzhou"
	}
	accessKeyID := strings.TrimSpace(os.Getenv("ALIYUN_SMS_ACCESS_KEY_ID"))
	accessKeySecret := strings.TrimSpace(os.Getenv("ALIYUN_SMS_ACCESS_KEY_SECRET"))
	signName := strings.TrimSpace(os.Getenv("ALIYUN_SMS_SIGN_NAME"))
	templateCode := strings.TrimSpace(os.Getenv("ALIYUN_SMS_TEMPLATE_CODE"))

	if accessKeyID == "" || accessKeySecret == "" || signName == "" || templateCode == "" {
		return fmt.Errorf("aliyun SMS not configured (set ALIYUN_SMS_ACCESS_KEY_ID, ALIYUN_SMS_ACCESS_KEY_SECRET, ALIYUN_SMS_SIGN_NAME, ALIYUN_SMS_TEMPLATE_CODE)")
	}

	cred, err := credential.NewCredential(&credential.Config{
		Type:            tea.String("access_key"),
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
	})
	if err != nil {
		return fmt.Errorf("aliyun sms credential: %w", err)
	}

	cfg := &openapi.Config{
		Credential: cred,
		RegionId:   tea.String(region),
		Endpoint:   tea.String("dysmsapi.aliyuncs.com"),
	}
	client, err := dysmsapi.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("aliyun sms client: %w", err)
	}

	paramJSON, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return fmt.Errorf("sms template param: %w", err)
	}

	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(domesticPhone),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(string(paramJSON)),
	}
	runtime := &dara.RuntimeOptions{}

	resp, err := client.SendSmsWithOptions(req, runtime)
	if err != nil {
		return fmt.Errorf("aliyun SendSms: %w", err)
	}
	return validateSendSmsResponse(resp)
}

// validateSendSmsResponse checks SendSms RPC body; nil err means the request was accepted (Code empty or OK).
func validateSendSmsResponse(resp *dysmsapi.SendSmsResponse) error {
	if resp == nil || resp.Body == nil {
		return fmt.Errorf("aliyun sms: empty response")
	}
	if c := dara.StringValue(resp.Body.Code); c != "" && c != "OK" {
		return fmt.Errorf("aliyun sms failed: %s %s", c, dara.StringValue(resp.Body.Message))
	}
	return nil
}
