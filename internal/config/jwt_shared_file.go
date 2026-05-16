package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// fileJWTOnly matches the `jwt` block in Grapery API / VipPay YAML or JSON configs.
type fileJWTOnly struct {
	JWT JWTConfig `yaml:"jwt" json:"jwt"`
}

// SharedJWTConfigCandidatePaths returns ordered paths from env that may contain the same
// `jwt.secret` as the main Grapery HTTP API. Used by cmd/vippay to align Bearer validation.
func SharedJWTConfigCandidatePaths() []string {
	var out []string
	for _, k := range []string{
		"GRAPH_API_CONFIG_PATH",
		"GRAPERY_CONFIG_PATH",
		"JWT_FALLBACK_CONFIG_PATH",
	} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// ReadJWTSecretFromConfigFile reads non-empty jwt.secret from a Grapery-style config file
// (YAML or JSON; same top-level `jwt` key as cmd/server).
func ReadJWTSecretFromConfigFile(path string) (secret string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var f fileJWTOnly
	if err := yaml.Unmarshal(data, &f); err != nil {
		return "", false
	}
	s := strings.TrimSpace(f.JWT.Secret)
	if s == "" {
		return "", false
	}
	return s, true
}
