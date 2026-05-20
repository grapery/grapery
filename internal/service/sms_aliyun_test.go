package service

import (
	"strings"
	"testing"

	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	"github.com/alibabacloud-go/tea/tea"
)

func TestValidateSendSmsResponse_nilResponse(t *testing.T) {
	err := validateSendSmsResponse(nil)
	if err == nil || !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateSendSmsResponse_nilBody(t *testing.T) {
	err := validateSendSmsResponse(&dysmsapi.SendSmsResponse{})
	if err == nil || !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateSendSmsResponse_ok(t *testing.T) {
	err := validateSendSmsResponse(&dysmsapi.SendSmsResponse{
		Body: &dysmsapi.SendSmsResponseBody{
			Code:    tea.String("OK"),
			Message: tea.String("OK"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateSendSmsResponse_okEmptyCode(t *testing.T) {
	// Older behavior: treat missing Code as success
	err := validateSendSmsResponse(&dysmsapi.SendSmsResponse{
		Body: &dysmsapi.SendSmsResponseBody{},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateSendSmsResponse_apiError(t *testing.T) {
	err := validateSendSmsResponse(&dysmsapi.SendSmsResponse{
		Body: &dysmsapi.SendSmsResponseBody{
			Code:    tea.String("isv.BUSINESS_LIMIT_CONTROL"),
			Message: tea.String("frequency limited"),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "isv.BUSINESS_LIMIT_CONTROL") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "frequency limited") {
		t.Fatalf("got %v", err)
	}
}

func TestSendAliyunOTPCode_emptyPhone(t *testing.T) {
	if err := SendAliyunOTPCode("", "123456"); err == nil {
		t.Fatal("want error for empty phone")
	} else if !strings.Contains(err.Error(), "empty phone") {
		t.Fatalf("got %v", err)
	}
	if err := SendAliyunOTPCode("   ", "123456"); err == nil {
		t.Fatal("want error for whitespace-only phone")
	}
}

func TestAliyunSMSDefaults(t *testing.T) {
	t.Setenv("ALIYUN_SMS_SIGN_NAME", "")
	t.Setenv("ALIYUN_SMS_TEMPLATE_CODE", "")
	if got := aliyunSMSSignName(); got != defaultAliyunSMSSignName {
		t.Fatalf("sign default: got %q want %q", got, defaultAliyunSMSSignName)
	}
	if got := aliyunSMSTemplateCode(); got != defaultAliyunSMSTemplateCode {
		t.Fatalf("template default: got %q want %q", got, defaultAliyunSMSTemplateCode)
	}
	t.Setenv("ALIYUN_SMS_SIGN_NAME", "自定义签名")
	t.Setenv("ALIYUN_SMS_TEMPLATE_CODE", "SMS_override")
	if got := aliyunSMSSignName(); got != "自定义签名" {
		t.Fatalf("sign override: got %q", got)
	}
	if got := aliyunSMSTemplateCode(); got != "SMS_override" {
		t.Fatalf("template override: got %q", got)
	}
}

func TestSendAliyunOTPCode_notConfigured(t *testing.T) {
	envKeys := []string{
		"ALIYUN_SMS_ACCESS_KEY_ID",
		"ALIYUN_SMS_ACCESS_KEY_SECRET",
		"ALIYUN_OSS_ACCESS_KEY_ID",
		"ALIYUN_OSS_ACCESS_KEY_SECRET",
		"ALIYUN_ACCESS_KEY_ID",
		"ALIYUN_ACCESS_KEY_SECRET",
		"ALIYUN_SMS_REGION",
		"ALIYUN_SMS_ENDPOINT",
		"ALIYUN_SMS_USE_DEFAULT_CREDENTIAL",
	}
	for _, k := range envKeys {
		t.Setenv(k, "")
	}
	err := SendAliyunOTPCode("13800138000", "123456")
	if err == nil {
		t.Fatal("want error when SMS credentials not set")
	}
	if !strings.Contains(err.Error(), "aliyun SMS not configured") {
		t.Fatalf("got %v", err)
	}
}

func TestAliyunSMSCredentialFallback(t *testing.T) {
	t.Setenv("ALIYUN_SMS_ACCESS_KEY_ID", "")
	t.Setenv("ALIYUN_SMS_ACCESS_KEY_SECRET", "")
	t.Setenv("ALIYUN_SMS_ACCESS_ID", "sms-id")
	t.Setenv("ALIYUN_SMS_ACCESS_SECRET", "sms-secret")
	t.Setenv("ALIYUN_ACCESS_KEY_ID", "main-id")
	t.Setenv("ALIYUN_ACCESS_KEY_SECRET", "main-secret")
	if got := aliyunSMSAccessKeyID(); got != "sms-id" {
		t.Fatalf("id fallback: got %q want sms-id (ALIYUN_SMS_ACCESS_ID alias)", got)
	}
	if got := aliyunSMSAccessKeySecret(); got != "sms-secret" {
		t.Fatalf("secret fallback: got %q want sms-secret", got)
	}
}
