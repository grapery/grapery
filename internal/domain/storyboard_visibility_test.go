package domain

import "testing"

func TestRedactStoryboardViewsUnlessCreator(t *testing.T) {
	sb := &Storyboard{
		UserID: "creator-1",
	}
	sb.Views = 42

	RedactStoryboardViewsUnlessCreator(sb, "creator-1")
	if sb.Views != 42 {
		t.Fatalf("creator should keep views, got %d", sb.Views)
	}

	RedactStoryboardViewsUnlessCreator(sb, "other")
	if sb.Views != 0 {
		t.Fatalf("non-creator should get 0 views, got %d", sb.Views)
	}

	sb.Views = 10
	RedactStoryboardViewsUnlessCreator(sb, "")
	if sb.Views != 0 {
		t.Fatalf("empty viewer should redact, got %d", sb.Views)
	}
}

func TestRedactStoryboardViewsUnlessCreatorMany(t *testing.T) {
	a := &Storyboard{UserID: "u1"}
	a.Views = 5
	b := &Storyboard{UserID: "u2"}
	b.Views = 7
	RedactStoryboardViewsUnlessCreatorMany([]*Storyboard{a, b}, "u1")
	if a.Views != 5 || b.Views != 0 {
		t.Fatalf("got a=%d b=%d", a.Views, b.Views)
	}
}
