package service

import (
	"context"
	"errors"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

type forkPermissionRepository struct {
	domain.Repository
	story        *domain.Story
	storyboard   *domain.Storyboard
	contributors []*domain.StoryContributor
	forked       *domain.Storyboard
	includeDraft bool
}

func TestCreateStoryboardRejectsStandaloneChild(t *testing.T) {
	repo := &forkPermissionRepository{story: &domain.Story{
		BaseModel:  common.BaseModel{ID: "story-1"},
		UserID:     "owner",
		Status:     "published",
		Visibility: string(domain.StoryVisibilityPublic),
	}}
	svc := &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
	err := svc.CreateStoryboard(context.Background(), &domain.Storyboard{
		StoryID:      "story-1",
		ParentID:     "board-1",
		UserID:       "owner",
		IsStandalone: true,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("CreateStoryboard error = %v, want ErrInvalidInput", err)
	}
}

func TestCreateStoryboardRejectsNonOwnerRoot(t *testing.T) {
	repo := &forkPermissionRepository{story: &domain.Story{
		BaseModel:           common.BaseModel{ID: "story-1"},
		UserID:              "owner",
		Status:              "published",
		Visibility:          string(domain.StoryVisibilityPublic),
		IsCollaborationOpen: true,
	}}
	svc := &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
	err := svc.CreateStoryboard(context.Background(), &domain.Storyboard{
		StoryID:      "story-1",
		UserID:       "reader",
		IsStandalone: true,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("CreateStoryboard error = %v, want ErrForbidden", err)
	}
}

func TestInviteStoryContributorRequiresStoryOwner(t *testing.T) {
	repo := &forkPermissionRepository{
		story:        &domain.Story{BaseModel: common.BaseModel{ID: "story-1"}, UserID: "owner"},
		contributors: []*domain.StoryContributor{{UserID: "contributor"}},
	}
	svc := &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
	_, err := svc.InviteStoryContributor(context.Background(), "contributor", "story-1", InviteStoryContributorRequest{
		UserID: "reader",
		Role:   string(domain.StoryRoleContributor),
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("InviteStoryContributor error = %v, want ErrForbidden", err)
	}
}

func TestCanViewerSeePrivateStoryAllowsInvitedContributor(t *testing.T) {
	story := &domain.Story{
		BaseModel:  common.BaseModel{ID: "story-1"},
		UserID:     "owner",
		Status:     "draft",
		Visibility: string(domain.StoryVisibilityPrivate),
	}
	repo := &forkPermissionRepository{
		story:        story,
		contributors: []*domain.StoryContributor{{UserID: "contributor"}},
	}
	svc := &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
	if !svc.CanViewerSeeStory(context.Background(), "contributor", story) {
		t.Fatal("invited contributor should be able to read private collaborative Story")
	}
	if svc.CanViewerSeeStory(context.Background(), "reader", story) {
		t.Fatal("ordinary reader should not be able to read private collaborative Story")
	}
}

func TestDraftVisibilityDoesNotLeakToOrdinaryReader(t *testing.T) {
	story := &domain.Story{
		BaseModel:  common.BaseModel{ID: "story-1"},
		UserID:     "owner",
		Status:     "draft",
		Visibility: string(domain.StoryVisibilityPublic),
	}
	repo := &forkPermissionRepository{story: story}
	svc := &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
	if svc.CanViewerSeeStory(context.Background(), "reader", story) {
		t.Fatal("ordinary reader should not see a draft Story even when its audience visibility is public")
	}

	draftNode := &domain.Storyboard{
		BaseModel:      common.BaseModel{ID: "board-1"},
		StoryID:        story.ID,
		UserID:         "owner",
		WorkflowStatus: domain.WorkflowStatusDraft,
	}
	if svc.CanViewerSeeStoryboard(context.Background(), "reader", draftNode, false) {
		t.Fatal("ordinary reader should not directly read an unpublished storyboard node")
	}

	repo.contributors = []*domain.StoryContributor{{UserID: "contributor"}}
	if !svc.CanViewerSeeStoryboard(context.Background(), "contributor", draftNode, false) {
		t.Fatal("invited contributor should directly read an unpublished storyboard node")
	}
}

func (r *forkPermissionRepository) StoryByID(context.Context, string) (*domain.Story, error) {
	if r.story == nil {
		return nil, domain.ErrNotFound
	}
	return r.story, nil
}

func (r *forkPermissionRepository) StoryboardByID(context.Context, string) (*domain.Storyboard, error) {
	if r.storyboard == nil {
		return nil, domain.ErrNotFound
	}
	return r.storyboard, nil
}

func (r *forkPermissionRepository) GetStoryContributors(context.Context, string, int, int) ([]*domain.StoryContributor, error) {
	return r.contributors, nil
}

func (r *forkPermissionRepository) CharactersByStory(context.Context, string) ([]*domain.Character, error) {
	return nil, nil
}

func (r *forkPermissionRepository) StoryScenes(context.Context, string, int, int) ([]*domain.StoryScene, error) {
	return nil, nil
}

func (r *forkPermissionRepository) IsStoryContributor(_ context.Context, _ string, userID string) (bool, error) {
	for _, contributor := range r.contributors {
		if contributor != nil && contributor.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (r *forkPermissionRepository) ForkStoryboard(_ context.Context, _ string, _ string, storyboard *domain.Storyboard) error {
	r.forked = storyboard
	return nil
}

func (r *forkPermissionRepository) RootStoryboardsByStory(_ context.Context, _ string, _, _ int, includeUnpublished bool) ([]*domain.Storyboard, error) {
	r.includeDraft = includeUnpublished
	return nil, nil
}

func TestForkStoryboardForcesInheritedParentContext(t *testing.T) {
	repo := &forkPermissionRepository{
		story: &domain.Story{BaseModel: common.BaseModel{ID: "story-1"}, UserID: "owner"},
		storyboard: &domain.Storyboard{
			BaseModel:      common.BaseModel{ID: "board-1"},
			StoryID:        "story-1",
			UserID:         "owner",
			WorkflowStatus: domain.WorkflowStatusDraft,
		},
	}
	svc := &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
	child := &domain.Storyboard{Title: "child", IsStandalone: true}
	if err := svc.ForkStoryboard(context.Background(), "board-1", "owner", child); err != nil {
		t.Fatalf("ForkStoryboard returned error: %v", err)
	}
	if repo.forked == nil || repo.forked.IsStandalone {
		t.Fatalf("forked storyboard = %+v, want inherited non-standalone child", repo.forked)
	}
}

func TestCanForkStoryboard(t *testing.T) {
	serviceFor := func(story *domain.Story, storyboard *domain.Storyboard, contributors ...*domain.StoryContributor) *Service {
		repo := &forkPermissionRepository{story: story, storyboard: storyboard, contributors: contributors}
		return &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
	}

	publicStory := func(open bool) *domain.Story {
		return &domain.Story{
			BaseModel:           common.BaseModel{ID: "story-1"},
			UserID:              "owner",
			Status:              "published",
			Visibility:          string(domain.StoryVisibilityPublic),
			IsCollaborationOpen: open,
		}
	}
	published := &domain.Storyboard{
		BaseModel:      common.BaseModel{ID: "board-1"},
		StoryID:        "story-1",
		UserID:         "owner",
		WorkflowStatus: domain.WorkflowStatusPublished,
	}

	tests := []struct {
		name         string
		story        *domain.Story
		storyboard   *domain.Storyboard
		userID       string
		contributors []*domain.StoryContributor
		wantAllowed  bool
		wantReason   string
	}{
		{name: "owner can fork when collaboration is closed", story: publicStory(false), storyboard: published, userID: "owner", wantAllowed: true, wantReason: ForkPermissionAllowed},
		{name: "visible user can fork published node in open collaboration", story: publicStory(true), storyboard: published, userID: "reader", wantAllowed: true, wantReason: ForkPermissionAllowed},
		{name: "closed collaboration denies ordinary reader", story: publicStory(false), storyboard: published, userID: "reader", wantReason: ForkPermissionCollaborationClosed},
		{name: "listed contributor can fork in closed collaboration", story: publicStory(false), storyboard: published, userID: "contributor", contributors: []*domain.StoryContributor{{UserID: "contributor"}}, wantAllowed: true, wantReason: ForkPermissionAllowed},
		{name: "non-owner cannot fork unpublished source", story: publicStory(true), storyboard: &domain.Storyboard{BaseModel: common.BaseModel{ID: "board-1"}, StoryID: "story-1", WorkflowStatus: domain.WorkflowStatusDraft}, userID: "reader", wantReason: ForkPermissionSourceNotPublished},
		{name: "non-owner cannot fork inside an unpublished story", story: &domain.Story{BaseModel: common.BaseModel{ID: "story-1"}, UserID: "owner", Status: "draft", Visibility: string(domain.StoryVisibilityPublic), IsCollaborationOpen: true}, storyboard: published, userID: "reader", wantReason: ForkPermissionSourceNotPublished},
		{name: "private story stays unavailable even when collaboration flag is open", story: &domain.Story{BaseModel: common.BaseModel{ID: "story-1"}, UserID: "owner", Status: "published", Visibility: string(domain.StoryVisibilityPrivate), IsCollaborationOpen: true}, storyboard: published, userID: "reader", wantReason: ForkPermissionSourceNotVisible},
		{name: "invited contributor can continue private draft node", story: &domain.Story{BaseModel: common.BaseModel{ID: "story-1"}, UserID: "owner", Status: "draft", Visibility: string(domain.StoryVisibilityPrivate)}, storyboard: &domain.Storyboard{BaseModel: common.BaseModel{ID: "board-1"}, StoryID: "story-1", WorkflowStatus: domain.WorkflowStatusDraft}, userID: "contributor", contributors: []*domain.StoryContributor{{UserID: "contributor"}}, wantAllowed: true, wantReason: ForkPermissionAllowed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := serviceFor(tc.story, tc.storyboard, tc.contributors...)
			got, err := svc.CanForkStoryboard(context.Background(), "board-1", tc.userID)
			if err != nil {
				t.Fatalf("CanForkStoryboard returned error: %v", err)
			}
			if got.Allowed != tc.wantAllowed || got.Reason != tc.wantReason {
				t.Fatalf("permission = %+v, want allowed=%v reason=%q", got, tc.wantAllowed, tc.wantReason)
			}
		})
	}
}

func TestCanCreateRootStoryboardIsOwnerOnly(t *testing.T) {
	publicStory := func(status string, open bool) *domain.Story {
		return &domain.Story{
			BaseModel:           common.BaseModel{ID: "story-1"},
			UserID:              "owner",
			Status:              status,
			Visibility:          string(domain.StoryVisibilityPublic),
			IsCollaborationOpen: open,
		}
	}

	tests := []struct {
		name         string
		story        *domain.Story
		userID       string
		contributors []*domain.StoryContributor
		want         bool
	}{
		{name: "owner can create in draft story", story: publicStory("draft", false), userID: "owner", want: true},
		{name: "open published story does not grant root creation", story: publicStory("published", true), userID: "reader"},
		{name: "open draft story denies non-owner", story: publicStory("draft", true), userID: "reader"},
		{name: "listed contributor cannot create a root", story: publicStory("published", false), userID: "contributor", contributors: []*domain.StoryContributor{{UserID: "contributor"}}},
		{name: "closed story denies ordinary reader", story: publicStory("published", false), userID: "reader"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &forkPermissionRepository{story: tc.story, contributors: tc.contributors}
			svc := &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
			got, err := svc.CanCreateStoryboard(context.Background(), "story-1", tc.userID)
			if err != nil {
				t.Fatalf("CanCreateStoryboard returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("CanCreateStoryboard = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCollaboratorHierarchyIncludesDraftsOnlyForAuthoringBoundary(t *testing.T) {
	story := &domain.Story{BaseModel: common.BaseModel{ID: "story-1"}, UserID: "owner"}
	tests := []struct {
		name         string
		userID       string
		contributors []*domain.StoryContributor
		want         bool
	}{
		{name: "owner", userID: "owner", want: true},
		{name: "invited contributor", userID: "contributor", contributors: []*domain.StoryContributor{{UserID: "contributor"}}, want: true},
		{name: "ordinary reader", userID: "reader", want: false},
		{name: "guest", userID: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &forkPermissionRepository{story: story, contributors: tc.contributors}
			svc := &Service{repo: repo, log: zap.NewNop(), logger: zap.NewNop()}
			includeUnpublished := svc.CanViewUnpublishedStoryboards(context.Background(), story, tc.userID)
			if includeUnpublished != tc.want {
				t.Fatalf("CanViewUnpublishedStoryboards = %v, want %v", includeUnpublished, tc.want)
			}
			if _, err := svc.ListRootStoryboards(context.Background(), story.ID, 20, 0, includeUnpublished); err != nil {
				t.Fatalf("ListRootStoryboards returned error: %v", err)
			}
			if repo.includeDraft != tc.want {
				t.Fatalf("repository includeUnpublished = %v, want %v", repo.includeDraft, tc.want)
			}
		})
	}
}
