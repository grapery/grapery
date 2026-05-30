package pay

import (
	"hash/fnv"
	"os"
	"strings"
)

// AllowStoreKitLocalVerify 是否允许 storekit_local 免苹果验单（仅开发环境显式开启）。
func AllowStoreKitLocalVerify() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("IAP_ALLOW_STOREKIT_LOCAL")), "true")
}

// UserIDUint64FromAuthString 与 vippay transport 的 FNV-1a 哈希一致，用于从 JWT 用户 ID 查库。
func UserIDUint64FromAuthString(userID string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(userID))
	return h.Sum64()
}

// IsAppleStoreKitTransactionID 判断是否为 StoreKit 2 数字 transaction id。
func IsAppleStoreKitTransactionID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 8 || len(s) > 32 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
