package utils

import (
	"os"
	"strings"
	"testing"
)

func TestLoadDotEnvSkipEmpty_doesNotSetBlank(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	if err := os.WriteFile(path, []byte("EMPTY_KEY=\nFILLED_KEY=filled\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EMPTY_KEY", "")
	t.Setenv("FILLED_KEY", "")

	if err := loadDotEnvSkipEmpty(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("EMPTY_KEY") != "" {
		t.Fatalf("empty placeholder should be skipped, got %q", os.Getenv("EMPTY_KEY"))
	}
	if got := os.Getenv("FILLED_KEY"); got != "filled" {
		t.Fatalf("FILLED_KEY: got %q want filled", got)
	}
}

func TestLoadDotEnvSkipEmpty_doesNotOverrideExisting(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	if err := os.WriteFile(path, []byte("MY_KEY=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MY_KEY", "from-shell")

	if err := loadDotEnvSkipEmpty(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("MY_KEY"); got != "from-shell" {
		t.Fatalf("existing env should win, got %q", got)
	}
}

func TestLoadDotEnvSkipEmpty_trimsValue(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	content := "TRIM_KEY=" + strings.Repeat("x", 3) + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_KEY", "")

	if err := loadDotEnvSkipEmpty(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("TRIM_KEY"); got != "xxx" {
		t.Fatalf("got %q", got)
	}
}
