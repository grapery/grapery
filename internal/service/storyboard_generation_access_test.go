package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

type storyboardAccessRepository struct {
	domain.Repository
	board  *domain.Storyboard
	scenes []*domain.StoryboardScene
	writes int
}

func (r *storyboardAccessRepository) StoryboardByID(context.Context, string) (*domain.Storyboard, error) {
	return r.board, nil
}
func (r *storyboardAccessRepository) StoryboardScenes(context.Context, string) ([]*domain.StoryboardScene, error) {
	return r.scenes, nil
}
func (r *storyboardAccessRepository) CreateImageGeneration(context.Context, *domain.StoryboardImageGeneration) error {
	r.writes++
	return errors.New("unexpected write")
}

func TestStoryboardGenerationRejectsUnauthorizedBeforeEnqueue(t *testing.T) {
	for _, tc := range []struct {
		name, actor, status, scene string
		want                       error
	}{
		{"anonymous", "", domain.WorkflowStatusDraft, "owned", domain.ErrUnauthorized},
		{"other creator", "other", domain.WorkflowStatusDraft, "owned", domain.ErrForbidden},
		{"published", "owner", domain.WorkflowStatusPublished, "owned", domain.ErrForbidden},
		{"foreign scene", "owner", domain.WorkflowStatusDraft, "foreign", domain.ErrInvalidInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &storyboardAccessRepository{board: &domain.Storyboard{BaseModel: common.BaseModel{ID: "board"}, UserID: "owner", WorkflowStatus: tc.status}, scenes: []*domain.StoryboardScene{{BaseModel: common.BaseModel{ID: "owned"}}}}
			svc := &Service{repo: repo, logger: zap.NewNop()}
			ctx := WithStoryboardGenerationCaller(context.Background(), tc.actor)
			if _, err := svc.GenerateSceneImage(ctx, &ImageGenerationRequest{StoryboardID: "board", SceneID: tc.scene}); !errors.Is(err, tc.want) {
				t.Fatalf("single-image error %v, want %v", err, tc.want)
			}
			if _, err := svc.GenerateStoryboardComicPage(ctx, &ComicPageGenerationRequest{StoryboardID: "board", SceneID: tc.scene}); !errors.Is(err, tc.want) {
				t.Fatalf("comic-page error %v, want %v", err, tc.want)
			}
			if repo.writes != 0 {
				t.Fatalf("unauthorized request enqueued %d jobs", repo.writes)
			}
		})
	}
}

func TestStoryboardGenerationOwnerInternalIdentity(t *testing.T) {
	repo := &storyboardAccessRepository{board: &domain.Storyboard{BaseModel: common.BaseModel{ID: "board"}, UserID: "owner", WorkflowStatus: domain.WorkflowStatusImagesReady}, scenes: []*domain.StoryboardScene{{BaseModel: common.BaseModel{ID: "scene"}}}}
	svc := &Service{repo: repo}
	board, scene, err := svc.authorizeStoryboardSceneGeneration(WithStoryboardGenerationCaller(context.Background(), "owner"), "board", "scene")
	if err != nil || board.ID != "board" || scene.ID != "scene" {
		t.Fatalf("legitimate internal caller denied: %v", err)
	}
}

func TestStoryboardLocalEditPreservesUntargetedAcceptedScene(t *testing.T) {
	original := domain.StoryboardScene{BaseModel: common.BaseModel{ID: "keep"}, Title: "accepted", Image: "accepted.png", ImagePrompt: "locked", GenerationRunID: "old"}
	changed := domain.StoryboardScene{BaseModel: common.BaseModel{ID: "edit"}, Sequence: 1, Title: "day"}
	baseline := &domain.StoryboardRevisionBaseline{Scenes: []domain.StoryboardScene{original, changed}, TargetSceneIDs: []string{"edit"}}
	generated := []domain.StoryboardScene{{Title: "model tried to rewrite locked"}, {Title: "night", Sequence: 1, GenerationRunID: "new"}}
	out, err := mergeStoryboardRevision(generated, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out[0], original) {
		t.Fatal("non-target scene changed")
	}
	if out[1].Title != "night" || out[1].ID != "" || out[1].Sequence != 1 {
		t.Fatalf("target not replaced: %+v", out[1])
	}
	if _, err := mergeStoryboardRevision(generated[:1], baseline); err == nil {
		t.Fatal("local edit changed page count")
	}
}

func TestStoryboardRevisionRejectsStaleClientAndForeignTargets(t *testing.T) {
	repo := &storyboardAccessRepository{board: &domain.Storyboard{BaseModel: common.BaseModel{ID: "board"}, Content: "accepted"}, scenes: []*domain.StoryboardScene{{BaseModel: common.BaseModel{ID: "scene"}}}}
	svc := &Service{repo: repo}
	if _, err := svc.captureStoryboardRevision(context.Background(), repo.board, nil, "stale"); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("stale revision accepted: %v", err)
	}
	if _, err := svc.captureStoryboardRevision(context.Background(), repo.board, []string{"foreign"}, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("foreign target accepted: %v", err)
	}
}

func TestStoryboardRetryRejectsOtherCreatorBeforeListingJobs(t *testing.T) {
	repo := &storyboardAccessRepository{board: &domain.Storyboard{BaseModel: common.BaseModel{ID: "board"}, UserID: "owner"}}
	svc := &Service{repo: repo}
	_, _, err := svc.RetryFailedStoryboardImages(WithStoryboardGenerationCaller(context.Background(), "stranger"), "board", nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("retry error=%v", err)
	}
}

func TestStoryboardComicPageRetryPreservesAcceptedLayout(t *testing.T) {
	options := ComicPagePipelineOptions{PanelCount: 4, LayoutPreset: "four_panel_2x2_grid", PageAspectRatio: "3:4", DialogueMode: "none"}
	gen := &domain.StoryboardImageGeneration{PromptDetails: &domain.ImagePromptDetails{ComicPagePlan: comicPagePlanSnapshot(options)}}
	retry := resolveComicPagePipeline(comicPageRetryOptions(gen), nil, "a silent transition", true, 3)
	if retry != options {
		t.Fatalf("retry changed confirmed page plan: %+v", retry)
	}
}

func TestStoryboardCreateIdempotencyWorksWithoutRedis(t *testing.T) {
	board := &domain.Storyboard{BaseModel: common.BaseModel{ID: storyboardCreateIdentity("owner", "request")}, UserID: "owner", StoryID: "story", ParentID: domain.StoryboardRootMarker, RawInput: "accepted request"}
	board.CreationRequestHash = storyboardCreationRequestHash(&domain.Storyboard{UserID: "owner", StoryID: "story", RawInput: "accepted request"})
	repo := &storyboardAccessRepository{board: board}
	svc := &Service{repo: repo}
	request := &domain.Storyboard{UserID: "owner", StoryID: "story", RawInput: "accepted request"}
	if err := svc.CreateStoryboardWithIdempotency(context.Background(), request, "request"); err != nil {
		t.Fatal(err)
	}
	if request.ID != board.ID {
		t.Fatal("retry did not reuse accepted draft")
	}
	other := &domain.Storyboard{UserID: "owner", StoryID: "different story", RawInput: "accepted request"}
	if err := svc.CreateStoryboardWithIdempotency(context.Background(), other, "request"); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("reused request key for different target: %v", err)
	}
	if storyboardCreateIdentity("owner", "request") == storyboardCreateIdentity("other", "request") {
		t.Fatal("idempotency identity was not scoped to user")
	}
}

func TestStoryboardCreationIdentityRejectsChangedIntents(t *testing.T) {
	original := &domain.Storyboard{UserID: "owner", StoryID: "story", ParentID: "parent", RawInput: "plot", Title: "title", SceneCount: 3, ContinuationComicStyle: "ink", CharacterRefs: []domain.StoryboardCharacterRef{{CharacterID: "character"}}, SceneRefs: []domain.StoryboardSceneRef{{StorySceneID: "location"}}}
	original.CreationRequestHash = storyboardCreationRequestHash(original)
	for name, mutate := range map[string]func(*domain.Storyboard){
		"story": func(b *domain.Storyboard) { b.StoryID = "other" }, "raw input": func(b *domain.Storyboard) { b.RawInput = "other" }, "parent": func(b *domain.Storyboard) { b.ParentID = "other" }, "count": func(b *domain.Storyboard) { b.SceneCount = 4 }, "style": func(b *domain.Storyboard) { b.ContinuationComicStyle = "oil" }, "character": func(b *domain.Storyboard) { b.CharacterRefs = []domain.StoryboardCharacterRef{{CharacterID: "other"}} }, "location": func(b *domain.Storyboard) { b.SceneRefs = []domain.StoryboardSceneRef{{StorySceneID: "other"}} }, "pipeline": func(b *domain.Storyboard) { b.UseComicPagePipeline = true },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := *original
			mutate(&candidate)
			candidate.CreationRequestHash = storyboardCreationRequestHash(&candidate)
			if sameStoryboardCreationTarget(original, &candidate) {
				t.Fatal("changed creation intent reused original idempotency key")
			}
		})
	}
	// Later generated content must never redefine the original immutable request.
	edited := *original
	edited.Content = "new accepted result"
	if !sameStoryboardCreationTarget(original, &edited) {
		t.Fatal("immutable creation fingerprint changed with later draft edits")
	}
}

type storyboardCreateFailureRepository struct {
	domain.Repository
	lookups int
}

func (r *storyboardCreateFailureRepository) StoryboardByID(context.Context, string) (*domain.Storyboard, error) {
	r.lookups++
	return nil, domain.ErrNotFound
}
func (r *storyboardCreateFailureRepository) StoryByID(context.Context, string) (*domain.Story, error) {
	return nil, domain.ErrForbidden
}
func TestStoryboardCreateDoesNotSwallowPermissionOrSchemaFailure(t *testing.T) {
	repo := &storyboardCreateFailureRepository{}
	svc := &Service{repo: repo, logger: zap.NewNop()}
	err := svc.CreateStoryboardWithIdempotency(context.Background(), &domain.Storyboard{UserID: "owner", StoryID: "story", RawInput: "plot"}, "request")
	if !errors.Is(err, domain.ErrForbidden) || repo.lookups != 1 {
		t.Fatalf("non-duplicate failure was swallowed/re-read: err=%v lookups=%d", err, repo.lookups)
	}
}
