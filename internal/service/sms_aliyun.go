package service

// SendAliyunOTPCode sends a mainland-China OTP via Alibaba Cloud SMS OpenAPI SendSms (2017-05-25).
//
// Matches official sample flow: dysmsapi client + SendSmsWithOptions + util.RuntimeOptions;
// credential is either explicit access keys or the default credential chain (RAM / env file / etc.).
//
// Required env:
//   - ALIYUN_SMS_SIGN_NAME（短信签名，如 「上海秩量科技」）
//   - ALIYUN_SMS_TEMPLATE_CODE（模板 CODE；模板变量需包含 JSON 字段 code，与本服务 marshal 一致）
//
// Credential (either):
//   - ALIYUN_SMS_ACCESS_KEY_ID + ALIYUN_SMS_ACCESS_KEY_SECRET
//   - Or ALIYUN_SMS_USE_DEFAULT_CREDENTIAL=1 / true → credential.NewCredential(nil)（推荐使用 RAM/无 AK 挂载方式）
//
// Optional:
//   - ALIYUN_SMS_REGION（默认 cn-hangzhou）
//   - ALIYUN_SMS_ENDPOINT（默认 dysmsapi.aliyuncs.com）

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

// SendAliyunOTPCode sends a 6-digit verification SMS via Alibaba Cloud SMS (China).
func SendAliyunOTPCode(domesticPhone, code string) error {
	domesticPhone = strings.TrimSpace(domesticPhone)
	if domesticPhone == "" {
		return fmt.Errorf("empty phone")
	}

	signName := strings.TrimSpace(os.Getenv("ALIYUN_SMS_SIGN_NAME"))
	templateCode := strings.TrimSpace(os.Getenv("ALIYUN_SMS_TEMPLATE_CODE"))
	if signName == "" || templateCode == "" {
		return fmt.Errorf("aliyun SMS not configured (set ALIYUN_SMS_SIGN_NAME and ALIYUN_SMS_TEMPLATE_CODE)")
	}

	accessKeyID := strings.TrimSpace(os.Getenv("ALIYUN_SMS_ACCESS_KEY_ID"))
	accessKeySecret := strings.TrimSpace(os.Getenv("ALIYUN_SMS_ACCESS_KEY_SECRET"))
	useDefaultChain := smsEnvTruthy("ALIYUN_SMS_USE_DEFAULT_CREDENTIAL")

	var cred credential.Credential
	var err error
	switch {
	case accessKeyID != "" && accessKeySecret != "":
		cred, err = credential.NewCredential(&credential.Config{
			Type:            tea.String("access_key"),
			AccessKeyId:     tea.String(accessKeyID),
			AccessKeySecret: tea.String(accessKeySecret),
		})
	case useDefaultChain:
		cred, err = credential.NewCredential(nil)
	default:
		return fmt.Errorf(
			"aliyun SMS not configured (set ALIYUN_SMS_ACCESS_KEY_ID & ALIYUN_SMS_ACCESS_KEY_SECRET, or ALIYUN_SMS_USE_DEFAULT_CREDENTIAL=1 with a cloud credential chain)",
		)
	}
	if err != nil {
		return fmt.Errorf("aliyun sms credential: %w", err)
	}

	region := strings.TrimSpace(os.Getenv("ALIYUN_SMS_REGION"))
	if region == "" {
		region = "cn-hangzhou"
	}
	endpoint := strings.TrimSpace(os.Getenv("ALIYUN_SMS_ENDPOINT"))
	if endpoint == "" {
		endpoint = "dysmsapi.aliyuncs.com"
	}

	cfg := &openapi.Config{
		Credential: cred,
		RegionId:   tea.String(region),
		Endpoint:   tea.String(endpoint),
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
	runtime := &util.RuntimeOptions{}

	resp, err := client.SendSmsWithOptions(req, runtime)
	if err != nil {
		return wrapAliyunSendSmsError(err)
	}
	return validateSendSmsResponse(resp)
}

func smsEnvTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func wrapAliyunSendSmsError(err error) error {
	var sdkErr *tea.SDKError
	if t, ok := err.(*tea.SDKError); ok {
		sdkErr = t
	} else {
		return fmt.Errorf("aliyun SendSms: %w", err)
	}

	msg := strings.TrimSpace(tea.StringValue(sdkErr.Message))
	if msg == "" {
		msg = err.Error()
	}
	dataRaw := strings.TrimSpace(tea.StringValue(sdkErr.Data))
	if dataRaw != "" {
		var data map[string]any
		if json.Unmarshal([]byte(dataRaw), &data) == nil {
			if rec, ok := data["Recommend"]; ok && rec != nil {
				msg = fmt.Sprintf("%s (diagnostic: %v)", msg, rec)
			}
		}
	}
	return fmt.Errorf("aliyun SendSms: %s", msg)
}

// validateSendSmsResponse checks SendSms RPC body; OK when Code empty or OK.
func validateSendSmsResponse(resp *dysmsapi.SendSmsResponse) error {
	if resp == nil || resp.Body == nil {
		return fmt.Errorf("aliyun sms: empty response")
	}
	code := smsStrPtr(resp.Body.Code)
	if code != "" && code != "OK" {
		return fmt.Errorf("aliyun sms failed: %s %s", code, smsStrPtr(resp.Body.Message))
	}
	return nil
}

func smsStrPtr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
