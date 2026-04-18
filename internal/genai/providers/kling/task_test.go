package kling

import "testing"

func TestFormatParseTaskID(t *testing.T) {
	raw := "550e8400-e29b-41d4-a716-446655440000"
	c := FormatTaskID(TaskText2Video, raw)
	k, id, err := ParseTaskID(c)
	if err != nil || k != TaskText2Video || id != raw {
		t.Fatalf("got %q %q err=%v", k, id, err)
	}
	path, err := QueryPath(k, id)
	if err != nil || path != "/v1/videos/text2video/"+raw {
		t.Fatalf("path=%q err=%v", path, err)
	}
}

func TestParseTaskIDInvalid(t *testing.T) {
	if _, _, err := ParseTaskID("not-kling"); err == nil {
		t.Fatal("expected error")
	}
}
