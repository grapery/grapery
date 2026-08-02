package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	shareLinkVersion       = "v1"
	defaultShareLinkTTL    = 30 * 24 * time.Hour
	maxShareLinkTTL        = 90 * 24 * time.Hour
	shareLinkCanonicalHost = "www.rankquantity.xyz"
)

// ShareKind identifies shareable content types (matches iOS Universal Link paths).
type ShareKind string

const (
	ShareKindFragment   ShareKind = "fragment"
	ShareKindStoryboard ShareKind = "storyboard"
	ShareKindStory      ShareKind = "story"
	ShareKindCharacter  ShareKind = "character"
)

// Share funnel dimensions. Both end up as Prometheus labels, so they must come from a
// closed set — anything a client sends that is not listed collapses to "other".
const (
	SharePlatformApp = "app"
	SharePlatformWeb = "web"

	ShareSourceAPIIssue      = "api_issue"
	ShareSourceContentGet    = "content_get"
	ShareSourceUniversalLink = "universal_link"

	shareDimensionOther = "other"
)

var (
	allowedSharePlatforms = map[string]struct{}{
		SharePlatformApp: {}, SharePlatformWeb: {},
		"ios": {}, "android": {}, "wechat": {}, "moments": {}, "copy": {}, "system": {},
	}
	allowedShareSources = map[string]struct{}{
		ShareSourceAPIIssue: {}, ShareSourceContentGet: {}, ShareSourceUniversalLink: {},
		"deeplink": {}, "qrcode": {},
	}
)

// NormalizeSharePlatform maps a client-supplied platform onto the allowed label set.
func NormalizeSharePlatform(raw, fallback string) string {
	return normalizeShareDimension(raw, fallback, allowedSharePlatforms)
}

// NormalizeShareSource maps a client-supplied source onto the allowed label set.
func NormalizeShareSource(raw, fallback string) string {
	return normalizeShareDimension(raw, fallback, allowedShareSources)
}

func normalizeShareDimension(raw, fallback string, allowed map[string]struct{}) string {
	v := strings.TrimSpace(strings.ToLower(raw))
	if v == "" {
		return fallback
	}
	if _, ok := allowed[v]; ok {
		return v
	}
	return shareDimensionOther
}

// ShareLinkIssue holds a signed public share URL.
type ShareLinkIssue struct {
	ShareURL string `json:"shareUrl"`
	Token    string `json:"token"`
	Exp      int64  `json:"exp"`
}

// ShareLinkSigner mints and verifies HMAC share tokens.
type ShareLinkSigner struct {
	secret []byte
}

func NewShareLinkSigner(secret string) *ShareLinkSigner {
	return &ShareLinkSigner{secret: []byte(strings.TrimSpace(secret))}
}

func (s *ShareLinkSigner) IsConfigured() bool {
	return s != nil && len(s.secret) > 0
}

// Issue creates a signed share URL for the given kind and content id.
func (s *ShareLinkSigner) Issue(kind ShareKind, id string, ttl time.Duration) (*ShareLinkIssue, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("share id is required")
	}
	if !validShareKind(kind) {
		return nil, fmt.Errorf("invalid share kind")
	}
	if !s.IsConfigured() {
		return nil, fmt.Errorf("share signing is not configured")
	}
	if ttl <= 0 {
		ttl = defaultShareLinkTTL
	}
	if ttl > maxShareLinkTTL {
		ttl = maxShareLinkTTL
	}
	exp := time.Now().Add(ttl).Unix()
	token := s.sign(kind, id, exp)
	// Land on the public website with the grant so H5 can pass t/exp to content APIs.
	// Open tracking is recorded when Grapery serves the content GET with a valid grant.
	shareURL := WebLandingURL(kind, id, token, exp)
	return &ShareLinkIssue{
		ShareURL: shareURL,
		Token:    token,
		Exp:      exp,
	}, nil
}

// WebLandingURL builds the public website URL for a signed share (after open redirect).
func WebLandingURL(kind ShareKind, id, token string, exp int64) string {
	id = strings.TrimSpace(id)
	token = strings.TrimSpace(token)
	if id == "" {
		return fmt.Sprintf("https://%s/", shareLinkCanonicalHost)
	}
	if token != "" && exp > 0 {
		return fmt.Sprintf("https://%s/%s/%s?t=%s&exp=%d", shareLinkCanonicalHost, kind, id, token, exp)
	}
	return fmt.Sprintf("https://%s/%s/%s", shareLinkCanonicalHost, kind, id)
}

// Verify checks token and expiry for a share link grant.
func (s *ShareLinkSigner) Verify(kind ShareKind, id, token string, exp int64) bool {
	if !s.IsConfigured() {
		return false
	}
	id = strings.TrimSpace(id)
	token = strings.TrimSpace(token)
	if id == "" || token == "" || exp <= 0 {
		return false
	}
	if !validShareKind(kind) {
		return false
	}
	if time.Now().Unix() > exp {
		return false
	}
	expected := s.sign(kind, id, exp)
	return hmac.Equal([]byte(expected), []byte(token))
}

func (s *ShareLinkSigner) sign(kind ShareKind, id string, exp int64) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(fmt.Sprintf("share:%s:%s:%s:%d", shareLinkVersion, kind, id, exp)))
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum)[:32]
}

func validShareKind(kind ShareKind) bool {
	switch kind {
	case ShareKindFragment, ShareKindStoryboard, ShareKindStory, ShareKindCharacter:
		return true
	default:
		return false
	}
}

// ParseShareGrantFromQuery reads ?t=&exp= from a request.
func ParseShareGrantFromQuery(token string, expRaw string) (tokenOut string, exp int64, ok bool) {
	tokenOut = strings.TrimSpace(token)
	if tokenOut == "" {
		return "", 0, false
	}
	exp, err := strconv.ParseInt(strings.TrimSpace(expRaw), 10, 64)
	if err != nil || exp <= 0 {
		return "", 0, false
	}
	return tokenOut, exp, true
}
