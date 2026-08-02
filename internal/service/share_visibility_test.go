package service

import (
	"context"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// Only the paths that resolve without touching the repository are covered here;
// the parent-story fallback is exercised through the handler-level suites.
func TestCanViewerSeeCharacter(t *testing.T) {
	s := &Service{}
	ctx := context.Background()

	cases := []struct {
		name       string
		viewer     string
		character  *domain.Character
		shareGrant bool
		want       bool
	}{
		{
			name:      "nil character is never visible",
			character: nil,
		},
		{
			name:       "valid share grant unlocks a private character",
			character:  &domain.Character{UserID: "author", IsPublic: false, StoryID: "story-1"},
			shareGrant: true,
			want:       true,
		},
		{
			name:      "author sees their own private character",
			viewer:    "author",
			character: &domain.Character{UserID: "author", IsPublic: false, StoryID: "story-1"},
			want:      true,
		},
		{
			name:      "public character is visible to anonymous viewers",
			character: &domain.Character{UserID: "author", IsPublic: true},
			want:      true,
		},
		{
			name:      "private standalone character is hidden from others",
			viewer:    "someone-else",
			character: &domain.Character{UserID: "author", IsPublic: false},
		},
		{
			name:      "private standalone character is hidden from anonymous viewers",
			character: &domain.Character{UserID: "author", IsPublic: false},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.CanViewerSeeCharacter(ctx, tc.viewer, tc.character, tc.shareGrant); got != tc.want {
				t.Errorf("CanViewerSeeCharacter = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCanViewerSeeStoryboardOrphan(t *testing.T) {
	s := &Service{}
	ctx := context.Background()

	draft := &domain.Storyboard{WorkflowStatus: domain.WorkflowStatusDraft}
	draft.UserID = "author"
	published := &domain.Storyboard{WorkflowStatus: domain.WorkflowStatusPublished}
	published.UserID = "author"

	if s.CanViewerSeeStoryboard(ctx, "someone-else", draft, false) {
		t.Error("unpublished orphan storyboard must not be readable by other users")
	}
	if s.CanViewerSeeStoryboard(ctx, "", draft, false) {
		t.Error("unpublished orphan storyboard must not be readable anonymously")
	}
	if !s.CanViewerSeeStoryboard(ctx, "author", draft, false) {
		t.Error("author must still read their own unpublished orphan storyboard")
	}
	if !s.CanViewerSeeStoryboard(ctx, "", draft, true) {
		t.Error("a valid share grant must unlock an orphan storyboard")
	}
	if !s.CanViewerSeeStoryboard(ctx, "", published, false) {
		t.Error("published orphan storyboard should stay publicly readable")
	}
}

func TestNormalizeShareDimensionsRejectArbitraryLabels(t *testing.T) {
	if got := NormalizeSharePlatform("", SharePlatformApp); got != SharePlatformApp {
		t.Errorf("empty platform = %q, want fallback %q", got, SharePlatformApp)
	}
	if got := NormalizeSharePlatform("  WeChat ", SharePlatformApp); got != "wechat" {
		t.Errorf("platform normalization = %q, want %q", got, "wechat")
	}
	if got := NormalizeSharePlatform("attacker-controlled-8f3a", SharePlatformApp); got != shareDimensionOther {
		t.Errorf("unknown platform = %q, want %q (unbounded Prometheus labels)", got, shareDimensionOther)
	}
	if got := NormalizeShareSource("nope", ShareSourceUniversalLink); got != shareDimensionOther {
		t.Errorf("unknown source = %q, want %q", got, shareDimensionOther)
	}
	if got := NormalizeShareSource("", ShareSourceUniversalLink); got != ShareSourceUniversalLink {
		t.Errorf("empty source = %q, want fallback %q", got, ShareSourceUniversalLink)
	}
}
