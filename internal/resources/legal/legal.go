package legal

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

const (
	LangEN     = "en"
	LangJA     = "ja"
	LangZHHans = "zh-Hans"
)

const (
	KeyTermsOfService = "terms_of_service"
	KeyPrivacyPolicy  = "privacy_policy"
)

//go:embed */*.md
var legalFS embed.FS

// ExtractTitle returns the first Markdown H1 title ("# ...") if present.
func ExtractTitle(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
		// If the doc starts with other content, stop early.
		return ""
	}
	return ""
}

// ExtractLastUpdated returns the "Last updated" line if present in the first few lines.
// Supports en/zh-Hans/ja variants used by our markdown assets:
// - **Last updated: ...**
// - **最后更新：...**
// - **最終更新：...**
func ExtractLastUpdated(markdown string) string {
	lines := strings.Split(markdown, "\n")
	max := 12
	if len(lines) < max {
		max = len(lines)
	}
	for i := 0; i < max; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// Strip surrounding **...**
		if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") && len(line) >= 4 {
			line = strings.TrimSuffix(strings.TrimPrefix(line, "**"), "**")
			line = strings.TrimSpace(line)
		}
		if strings.HasPrefix(line, "Last updated:") ||
			strings.HasPrefix(line, "最后更新：") ||
			strings.HasPrefix(line, "最終更新：") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func ParsePreferredLang(acceptLanguageHeader, queryLang string) string {
	if lang := parseAcceptLanguage(acceptLanguageHeader); lang != "" {
		return lang
	}
	if lang := normalizeLang(queryLang); lang != "" {
		return lang
	}
	return LangEN
}

// PreferredLangForAPI prefers explicit ?lang= (mobile clients), then Accept-Language, then English.
func PreferredLangForAPI(queryLang, acceptLanguageHeader string) string {
	if lang := normalizeLang(queryLang); lang != "" {
		return lang
	}
	if lang := parseAcceptLanguage(acceptLanguageHeader); lang != "" {
		return lang
	}
	return LangEN
}

func Get(key, lang string) (content string, chosenLang string, err error) {
	normalized := normalizeLang(lang)
	if normalized == "" {
		normalized = LangEN
	}

	content, err = readDoc(key, normalized)
	if err == nil {
		return content, normalized, nil
	}

	if normalized != LangEN {
		content, err2 := readDoc(key, LangEN)
		if err2 == nil {
			return content, LangEN, nil
		}
	}

	return "", "", err
}

func readDoc(key, lang string) (string, error) {
	filename, err := keyToFilename(key)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("%s/%s", lang, filename)
	b, err := fs.ReadFile(legalFS, path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func keyToFilename(key string) (string, error) {
	switch key {
	case KeyTermsOfService:
		return "terms-of-service.md", nil
	case KeyPrivacyPolicy:
		return "privacy-policy.md", nil
	default:
		return "", fmt.Errorf("unknown legal doc key: %s", key)
	}
}

func normalizeLang(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)

	if s == "zh-hans" || strings.HasPrefix(s, "zh") {
		return LangZHHans
	}
	if strings.HasPrefix(s, "ja") {
		return LangJA
	}
	if strings.HasPrefix(s, "en") {
		return LangEN
	}
	return ""
}

func parseAcceptLanguage(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	type cand struct {
		lang string
		q    float64
		i    int
	}
	var cands []cand

	parts := strings.Split(header, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		langRange := part
		q := 1.0
		if semi := strings.Index(part, ";"); semi >= 0 {
			langRange = strings.TrimSpace(part[:semi])
			params := strings.Split(part[semi+1:], ";")
			for _, p := range params {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "q=") {
					if v, err := strconv.ParseFloat(strings.TrimPrefix(p, "q="), 64); err == nil {
						q = v
					}
				}
			}
		}

		lang := normalizeLang(langRange)
		if lang == "" {
			continue
		}
		cands = append(cands, cand{lang: lang, q: q, i: i})
	}

	if len(cands) == 0 {
		return ""
	}

	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].q == cands[j].q {
			return cands[i].i < cands[j].i
		}
		return cands[i].q > cands[j].q
	})

	// Return best supported language.
	for _, c := range cands {
		if c.lang != "" {
			return c.lang
		}
	}
	return ""
}

var ErrNotFound = errors.New("legal doc not found")
