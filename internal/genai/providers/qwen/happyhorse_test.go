package qwen

import "testing"

func TestNormalizeHappyHorseResolution(t *testing.T) {
	if got := NormalizeHappyHorseResolution(""); got != "1080P" {
		t.Fatalf("empty: want 1080P, got %q", got)
	}
	if got := NormalizeHappyHorseResolution("720p"); got != "720P" {
		t.Fatalf("720p: want 720P, got %q", got)
	}
}

func TestClampHappyHorseDuration(t *testing.T) {
	if got := ClampHappyHorseDuration(0, 5); got != 5 {
		t.Fatalf("default: got %d", got)
	}
	if got := ClampHappyHorseDuration(1, 5); got != 3 {
		t.Fatalf("min clamp: got %d", got)
	}
	if got := ClampHappyHorseDuration(99, 5); got != 15 {
		t.Fatalf("max clamp: got %d", got)
	}
}

func TestResolveHappyHorseModelID(t *testing.T) {
	if got := ResolveHappyHorseModelID("", "t2v"); got != ModelHappyHorseT2V {
		t.Fatalf("t2v: got %q", got)
	}
	if got := ResolveHappyHorseModelID("happyhorse", "i2v"); got != ModelHappyHorseI2V {
		t.Fatalf("i2v placeholder: got %q", got)
	}
	if got := ResolveHappyHorseModelID(ModelHappyHorseT2V, "i2v"); got != ModelHappyHorseT2V {
		t.Fatalf("explicit ID must be preserved: got %q", got)
	}
}
