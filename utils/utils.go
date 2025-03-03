package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

const (
	GrpcGateWayCookie = "grpcgateway-cookie"
	SecretKey         = "grapery"
	ExpirationHours   = 24 * 7
	UserIdKey         = "user_id"
)

var (
	CookieName = "grapery"
	Domain     = ""
	// for 7 day
	CookieMaxAge = 60 * 60 * 24 * 7
	CookiePath   = "grapery.xyz"
)

func GetUserInfoFromMetadata(ctx context.Context) int64 {
	uid := ctx.Value(UserIdKey)
	return uid.(int64)
}

// HasPrefixes returns true if the string s has any of the given prefixes.
func HasPrefixes(src string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(src, prefix) {
			return true
		}
	}
	return false
}

// ValidateEmail validates the email.
func ValidateEmail(email string) bool {
	if _, err := mail.ParseAddress(email); err != nil {
		return false
	}
	return true
}

func GenUUID() string {
	return uuid.New().String()
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandomString returns a random string with length n.
func RandomString(n int) (string, error) {
	var sb strings.Builder
	sb.Grow(n)
	for i := 0; i < n; i++ {
		// The reason for using crypto/rand instead of math/rand is that
		// the former relies on hardware to generate random numbers and
		// thus has a stronger source of random numbers.
		randNum, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		if _, err := sb.WriteRune(letters[randNum.Uint64()]); err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

type EmbedType string

const (
	EmbedTypeRich    EmbedType = "rich"
	EmbedTypeImage   EmbedType = "image"
	EmbedTypeVideo   EmbedType = "video"
	EmbedTypeGifv    EmbedType = "gif"
	EmbedTypeArticle EmbedType = "article"
	EmbedTypeLink    EmbedType = "link"
)

func CleanLLmJsonResult(jsonStr string) string {
	jsonStr = strings.Trim(
		strings.Trim(
			strings.Trim(
				strings.Trim(
					strings.Trim(jsonStr, "\r"), "\n"), "\\"), "‘"), "```json")
	return jsonStr
}

// GetUserIDFromContext 从 context 中获取 user_id
func GetUserIDFromContext(ctx context.Context) (int64, error) {
	userID, ok := ctx.Value(UserIdKey).(int64)
	if !ok {
		return 0, fmt.Errorf("user id not found in context")
	}
	return userID, nil
}
