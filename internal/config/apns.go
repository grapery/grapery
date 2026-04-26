package config

import (
	"fmt"
	"os"
	"strings"
)

// DefaultAPNsConfig Voyager / graperyproductionapplekey defaults (Team ID & Key ID are public identifiers).
func DefaultAPNsConfig() APNsConfig {
	return APNsConfig{
		BundleID:       "com.rankquantity.voyager",
		TeamID:         "UZLNTVX73Y",
		KeyID:          "NLX828W4A6",
		PrivateKeyPath: "certs/AuthKey_NLX828W4A6.p8",
		UseSandbox:     true,
	}
}

// mergeAPNsEmptyFields fills missing fields so YAML unmarshaling (which zeroes absent structs) still gets sensible defaults.
func mergeAPNsEmptyFields(cfg APNsConfig) APNsConfig {
	def := DefaultAPNsConfig()
	if cfg.BundleID == "" {
		cfg.BundleID = def.BundleID
	}
	if cfg.TeamID == "" {
		cfg.TeamID = def.TeamID
	}
	if cfg.KeyID == "" {
		cfg.KeyID = def.KeyID
	}
	if cfg.PrivateKeyPath == "" && strings.TrimSpace(cfg.PrivateKey) == "" {
		cfg.PrivateKeyPath = def.PrivateKeyPath
	}
	return cfg
}

// APNsKeyPEM returns PEM text for the APNs .p8 key from inline config or from PrivateKeyPath.
func APNsKeyPEM(c APNsConfig) (string, error) {
	if strings.TrimSpace(c.PrivateKey) != "" {
		return c.PrivateKey, nil
	}
	path := strings.TrimSpace(c.PrivateKeyPath)
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read APNs private key file %q: %w", path, err)
	}
	return string(data), nil
}
