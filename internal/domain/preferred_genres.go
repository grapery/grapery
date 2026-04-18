package domain

// AllowedPreferredGenreSlugs 与客户端 onboarding / 设置页一致；用于校验 user_settings.preferred_genres_json。
var AllowedPreferredGenreSlugs = []string{
	"scifi",
	"romance",
	"mystery",
	"fantasy",
	"youth",
	"history",
	"urban",
	"comedy",
	"healing",
	"horror",
	"adventure",
	"slice",
}

// AllowedPreferredGenreSet 快速成员检查。
func AllowedPreferredGenreSet() map[string]struct{} {
	m := make(map[string]struct{}, len(AllowedPreferredGenreSlugs))
	for _, g := range AllowedPreferredGenreSlugs {
		m[g] = struct{}{}
	}
	return m
}
