package service

// SendAliyunOTPCode uses Dysmsapi SendMessageWithTemplate (2018-05-01).
//
// Required env:
//   - ALIYUN_SMS_ACCESS_KEY_ID, ALIYUN_SMS_ACCESS_KEY_SECRET
//   - ALIYUN_SMS_SIGN_NAME (maps to API field From — 短信签名)
//   - ALIYUN_SMS_TEMPLATE_CODE (模板 CODE，模板变量需包含与 TemplateParam 一致的 code 字段)
// Optional: ALIYUN_SMS_REGION (default cn-hangzhou)

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

// SendAliyunOTPCode sends a 6-digit verification SMS via Alibaba Cloud SMS (China).
func SendAliyunOTPCode(domesticPhone, code string) error {
	domesticPhone = strings.TrimSpace(domesticPhone)
	if domesticPhone == "" {
		return fmt.Errorf("empty phone")
	}

	region := os.Getenv("ALIYUN_SMS_REGION")
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

	client, err := dysmsapi.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		return fmt.Errorf("aliyun sms client: %w", err)
	}

	// Alibaba Cloud SMS v20180501 — SendMessageWithTemplate (replaces legacy SendSms in current SDK)
	req := dysmsapi.CreateSendMessageWithTemplateRequest()
	req.Scheme = "https"
	req.To = domesticPhone
	req.From = signName
	req.TemplateCode = templateCode
	paramJSON, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return fmt.Errorf("sms template param: %w", err)
	}
	req.TemplateParam = string(paramJSON)

	resp, err := client.SendMessageWithTemplate(req)
	if err != nil {
		return fmt.Errorf("aliyun SendMessageWithTemplate: %w", err)
	}
	if resp.ResponseCode != "" && resp.ResponseCode != "OK" {
		return fmt.Errorf("aliyun sms failed: %s %s", resp.ResponseCode, resp.ResponseDescription)
	}
	return nil
}
