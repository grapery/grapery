package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// canonicalLanguage maps legacy / BCP-47 aliases to stored API values (zh-Hans, en, ja, system).
func (s *userSettingsService) canonicalLanguage(lang string) string {
	lang = strings.TrimSpace(strings.ReplaceAll(lang, "_", "-"))
	low := strings.ToLower(lang)
	switch low {
	case "", "system":
		return string(domain.LanguageSystem)
	case "zh", "zh-cn", "zh-hans", "zh_cn", "zh_hans", "cmn", "zh-sg":
		return string(domain.LanguageChineseCN)
	case "en", "en-us", "en-gb", "en_us", "en_gb":
		return string(domain.LanguageEnglish)
	case "ja", "ja-jp", "ja_jp":
		return string(domain.LanguageJapanese)
	default:
		return lang
	}
}

func (s *userSettingsService) canonicalAllowFrom(v string) string {
	v = strings.TrimSpace(v)
	if v == "followers" {
		return string(domain.AllowFromFollowersOnly)
	}
	return v
}

func (s *userSettingsService) canonicalRegion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "auto"
	}
	return v
}

// canonicalizeStoredUserSettings normalizes DB / legacy values. Returns true if any field changed (caller may persist).
func (s *userSettingsService) canonicalizeStoredUserSettings(st *domain.UserSettings) bool {
	dirty := false
	if nl := s.canonicalLanguage(st.Language); nl != st.Language {
		st.Language = nl
		dirty = true
	}
	if nr := s.canonicalRegion(st.Region); nr != st.Region {
		st.Region = nr
		dirty = true
	}
	for _, pair := range []struct {
		dst *string
		val string
	}{
		{&st.AllowFollowFrom, s.canonicalAllowFrom(st.AllowFollowFrom)},
		{&st.AllowCommentsFrom, s.canonicalAllowFrom(st.AllowCommentsFrom)},
		{&st.AllowMessagesFrom, s.canonicalAllowFrom(st.AllowMessagesFrom)},
	} {
		if *pair.dst != pair.val {
			*pair.dst = pair.val
			dirty = true
		}
	}
	ns, persistNotification := s.reconcileNotificationJSON(st.NotificationSettings)
	if ns != st.NotificationSettings {
		st.NotificationSettings = ns
	}
	if persistNotification {
		dirty = true
	}
	return dirty
}

// reconcileNotificationJSON returns API-ready nested JSON. persistDB is true only for empty/invalid/legacy flat rows
// (avoid rewriting on every GET when only default-merge would change map key ordering).
func (s *userSettingsService) reconcileNotificationJSON(raw string) (out string, persistDB bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return s.getDefaultNotificationSettings(), true
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil || m == nil {
		return s.getDefaultNotificationSettings(), true
	}

	if pv, ok := m["push"]; ok {
		if _, isBool := pv.(bool); isBool {
			ns, _ := s.legacyFlatNotificationToJSON(m)
			return ns, true
		}
	}

	base := s.getDefaultNotificationSettingsMap()
	mergeNotificationMaps(base, m)
	b, err := json.Marshal(base)
	if err != nil {
		return raw, false
	}
	return string(b), false
}

func (s *userSettingsService) legacyFlatNotificationToJSON(flat map[string]interface{}) (string, bool) {
	base := s.getDefaultNotificationSettingsMap()
	push := base["push"].(map[string]interface{})
	if v, ok := flat["push"].(bool); ok {
		push["enabled"] = v
	}
	if v, ok := flat["likes"].(bool); ok {
		push["newLike"] = v
	}
	if v, ok := flat["comments"].(bool); ok {
		push["newComment"] = v
	}
	if v, ok := flat["follows"].(bool); ok {
		push["newFollower"] = v
	}
	email := base["email"].(map[string]interface{})
	if v, ok := flat["email"].(bool); ok {
		email["enabled"] = v
	}
	b, _ := json.Marshal(base)
	return string(b), true
}

// parseStringSliceFlexible accepts []string or JSON []interface{} (e.g. from encoding/json unmarshalling).
func parseStringSliceFlexible(v interface{}) ([]string, bool) {
	switch x := v.(type) {
	case []string:
		return x, true
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, it := range x {
			switch t := it.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					out = append(out, s)
				}
			case float64:
				out = append(out, fmt.Sprintf("%.0f", t))
			case json.Number:
				out = append(out, strings.TrimSpace(t.String()))
			}
		}
		return out, true
	default:
		return nil, false
	}
}
