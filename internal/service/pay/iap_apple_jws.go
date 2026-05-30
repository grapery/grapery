package pay

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseAppleTransactionPayloadMap 从 signedTransactionInfo JWS 的 payload 段解析字段。
func ParseAppleTransactionPayloadMap(txPayload map[string]interface{}) (*AppleSignedTransactionFields, bool, error) {
	if txPayload == nil {
		return nil, false, fmt.Errorf("empty transaction payload")
	}
	out := &AppleSignedTransactionFields{}
	if v, ok := txPayload["productId"].(string); ok {
		out.ProductID = v
	}
	if v, ok := txPayload["transactionId"].(string); ok {
		out.TransactionID = v
	}
	if v, ok := txPayload["originalTransactionId"].(string); ok {
		out.OriginalTransactionID = v
	}
	if out.OriginalTransactionID == "" {
		out.OriginalTransactionID = out.TransactionID
	}
	if ms, ok := appleMillisFromJSONValue(txPayload["expiresDate"]); ok {
		t := time.UnixMilli(ms)
		out.ExpiresDate = &t
	}
	if ms, ok := appleMillisFromJSONValue(txPayload["purchaseDate"]); ok {
		out.PurchaseDate = time.UnixMilli(ms)
	}
	if out.ProductID == "" || out.TransactionID == "" {
		return nil, false, fmt.Errorf("incomplete transaction payload from app store")
	}
	sandbox := appleEnvironmentIsSandbox(txPayload["environment"])
	return out, sandbox, nil
}

// ParseAppleSignedTransactionInfo 解码 JWS 并校验 bundleId；返回字段与是否沙盒环境。
func ParseAppleSignedTransactionInfo(signedTransactionInfo, expectedBundleID string) (*AppleSignedTransactionFields, bool, error) {
	expectedBundleID = strings.TrimSpace(expectedBundleID)
	txPayload, err := DecodeAppleJWSPayload(signedTransactionInfo)
	if err != nil {
		return nil, false, err
	}
	if expectedBundleID != "" {
		bid, _ := txPayload["bundleId"].(string)
		if strings.TrimSpace(bid) != "" && bid != expectedBundleID {
			return nil, false, fmt.Errorf("bundle id mismatch: got %s want %s", bid, expectedBundleID)
		}
	}
	return ParseAppleTransactionPayloadMap(txPayload)
}

func appleMillisFromJSONValue(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case float64:
		if x > 0 {
			return int64(x), true
		}
	case json.Number:
		n, err := x.Int64()
		if err == nil && n > 0 {
			return n, true
		}
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err == nil && n > 0 {
			return n, true
		}
	}
	return 0, false
}

func appleEnvironmentIsSandbox(v interface{}) bool {
	s, _ := v.(string)
	return strings.EqualFold(strings.TrimSpace(s), "Sandbox")
}

// ExtractAppleTransactionFromNotificationPayload 解析 V2 通知 signedPayload 内的 signedTransactionInfo。
func ExtractAppleTransactionFromNotificationPayload(signedPayload string) (*AppleSignedTransactionFields, error) {
	root, err := DecodeAppleJWSPayload(signedPayload)
	if err != nil {
		return nil, err
	}
	data, _ := root["data"].(map[string]interface{})
	if data == nil {
		return nil, fmt.Errorf("missing data in notification payload")
	}
	signedTx, _ := data["signedTransactionInfo"].(string)
	if strings.TrimSpace(signedTx) == "" {
		return nil, fmt.Errorf("missing signedTransactionInfo")
	}
	fields, _, err := ParseAppleSignedTransactionInfo(signedTx, "")
	if err != nil {
		return nil, err
	}
	return fields, nil
}
