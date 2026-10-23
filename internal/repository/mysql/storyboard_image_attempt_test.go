package mysql

import (
	"context"
	"errors"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func imageAttemptRepository(t *testing.T) (*Repository, *domain.StoryboardImageGeneration) {
	t.Helper()
	r, _, _ := storyboardRevisionRepository(t)
	if err := r.db.Migrator().CreateTable(&StoryboardImageGeneration{}); err != nil {
		t.Fatal(err)
	}
	return r, &domain.StoryboardImageGeneration{ID: "attempt-1", StoryboardID: "board", SceneID: "old", Status: domain.GenerationStatusPending, CreatedAt: 100}
}

func enqueueImageAttempt(t *testing.T, r *Repository, gen *domain.StoryboardImageGeneration, previous *string) error {
	t.Helper()
	var board Storyboard
	var scene StoryboardScene
	if err := r.db.First(&board, "id = ?", gen.StoryboardID).Error; err != nil {
		t.Fatal(err)
	}
	if err := r.db.First(&scene, "id = ?", gen.SceneID).Error; err != nil {
		t.Fatal(err)
	}
	return r.CreateStoryboardImageAttempt(context.Background(), gen, "owner", r.storyboardImageRevision(&board, scene), previous)
}

func TestStoryboardImageAttemptEnqueueCASAndActiveGuard(t *testing.T) {
	r, gen := imageAttemptRepository(t)
	empty := ""
	if err := enqueueImageAttempt(t, r, gen, &empty); err != nil {
		t.Fatal(err)
	}
	second := *gen
	second.ID = "attempt-2"
	if err := enqueueImageAttempt(t, r, &second, nil); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("active work replayed: %v", err)
	}
	gen.Status = domain.GenerationStatusFailed
	if err := r.UpdateImageGeneration(context.Background(), gen); err != nil {
		t.Fatal(err)
	}
	if err := enqueueImageAttempt(t, r, &second, &empty); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("stale retry enqueued: %v", err)
	}
	previous := gen.ID
	if err := enqueueImageAttempt(t, r, &second, &previous); err != nil {
		t.Fatal(err)
	}
	if !domain.StoryboardImageAttemptAfter(&second, gen) {
		t.Fatal("same-second retry order was lost")
	}
}

func TestStoryboardImageRetryCannotRaceAnAcceptedSceneImage(t *testing.T) {
	r, gen := imageAttemptRepository(t)
	if err := r.db.Model(&StoryboardScene{}).Where("id = ?", gen.SceneID).Update("image", "https://stored/accepted.png").Error; err != nil {
		t.Fatal(err)
	}
	empty := ""
	if err := enqueueImageAttempt(t, r, gen, &empty); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("retry raced accepted image: %v", err)
	}
}

func TestStoryboardImageResultCheckpointRecoversAfterSceneWriteFailure(t *testing.T) {
	r, gen := imageAttemptRepository(t)
	if err := enqueueImageAttempt(t, r, gen, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.db.Exec("CREATE TRIGGER fail_image BEFORE UPDATE OF image ON storyboard_scenes BEGIN SELECT RAISE(ABORT, 'scene unavailable'); END;").Error; err != nil {
		t.Fatal(err)
	}
	gen.GeneratedImageURL = "https://stored/paid-result.png"
	if accepted, err := r.FinalizeStoryboardImageGeneration(context.Background(), gen); err == nil || accepted {
		t.Fatalf("injected write failure: %v %v", accepted, err)
	}
	persisted, err := r.GetImageGeneration(context.Background(), gen.ID)
	if err != nil || persisted.GeneratedImageURL != gen.GeneratedImageURL || persisted.Status == domain.GenerationStatusCompleted {
		t.Fatalf("provider result lost or falsely completed: %+v %v", persisted, err)
	}
	var scene StoryboardScene
	r.db.First(&scene, "id = ?", "old")
	if scene.Image != "" {
		t.Fatal("failed transaction attached image")
	}
	if err := r.db.Exec("DROP TRIGGER fail_image").Error; err != nil {
		t.Fatal(err)
	}
	if accepted, err := r.FinalizeStoryboardImageGeneration(context.Background(), persisted); err != nil || !accepted {
		t.Fatalf("checkpoint not recoverable: %v %v", accepted, err)
	}
	r.db.First(&scene, "id = ?", "old")
	if scene.Image != gen.GeneratedImageURL {
		t.Fatalf("recovered image = %q", scene.Image)
	}
	if accepted, err := r.FinalizeStoryboardImageGeneration(context.Background(), persisted); err != nil || !accepted {
		t.Fatalf("finalize replay is not idempotent: %v %v", accepted, err)
	}
}

func TestStoryboardImageResultRejectsChangedAcceptedState(t *testing.T) {
	for _, tc := range []string{"scene edit", "new image", "published", "owner changed", "deleted", "superseded", "cancelled"} {
		t.Run(tc, func(t *testing.T) {
			r, gen := imageAttemptRepository(t)
			if err := enqueueImageAttempt(t, r, gen, nil); err != nil {
				t.Fatal(err)
			}
			switch tc {
			case "scene edit":
				r.db.Model(&StoryboardScene{}).Where("id = ?", "old").Update("description", "new accepted plan")
			case "new image":
				r.db.Model(&StoryboardScene{}).Where("id = ?", "old").Update("image", "https://stored/accepted.png")
			case "published":
				r.db.Model(&Storyboard{}).Where("id = ?", "board").Update("workflow_status", domain.WorkflowStatusPublished)
			case "owner changed":
				r.db.Model(&Storyboard{}).Where("id = ?", "board").Update("creator_id", "another-owner")
			case "deleted":
				r.db.Where("id = ?", "old").Delete(&StoryboardScene{})
			case "superseded":
				gen.Status = domain.GenerationStatusFailed
				if err := r.UpdateImageGeneration(context.Background(), gen); err != nil {
					t.Fatal(err)
				}
				other := *gen
				other.ID = "attempt-2"
				other.Status = domain.GenerationStatusPending
				if err := enqueueImageAttempt(t, r, &other, nil); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				gen.Status = domain.GenerationStatusFailed
				gen.ErrorMessage = "[cancelled] user stopped work"
				if err := r.UpdateImageGeneration(context.Background(), gen); err != nil {
					t.Fatal(err)
				}
			}
			gen.GeneratedImageURL = "https://stored/stale-candidate.png"
			if accepted, err := r.FinalizeStoryboardImageGeneration(context.Background(), gen); accepted || !errors.Is(err, domain.ErrAlreadyExists) {
				t.Fatalf("stale accepted: %v %v", accepted, err)
			}
			stored, err := r.GetImageGeneration(context.Background(), gen.ID)
			if err != nil || stored.GeneratedImageURL != gen.GeneratedImageURL || stored.Status != domain.GenerationStatusFailed {
				t.Fatalf("candidate disappeared: %+v %v", stored, err)
			}
			var scene StoryboardScene
			r.db.First(&scene, "id = ?", "old")
			if scene.Image == gen.GeneratedImageURL {
				t.Fatal("stale result overwrote scene")
			}
		})
	}
}

func TestStoryboardImageExecutionCannotBeReplacedByPromptUpdate(t *testing.T) {
	r, gen := imageAttemptRepository(t)
	if err := enqueueImageAttempt(t, r, gen, nil); err != nil {
		t.Fatal(err)
	}
	revision := gen.PromptDetails.Execution.Revision
	gen.PromptDetails = &domain.ImagePromptDetails{ArtStyle: "watercolor", Execution: &domain.StoryboardImageExecution{OwnerID: "forged", Revision: "fake"}}
	if err := r.UpdateImageGeneration(context.Background(), gen); err != nil {
		t.Fatal(err)
	}
	stored, err := r.GetImageGeneration(context.Background(), gen.ID)
	if err != nil || stored.PromptDetails.Execution.OwnerID != "owner" || stored.PromptDetails.Execution.Revision != revision {
		t.Fatalf("execution metadata replaced: %+v %v", stored, err)
	}
}

func TestStoryboardWorkflowCannotDowngradePublishedBoard(t *testing.T) {
	r, _ := imageAttemptRepository(t)
	if err := r.UpdateStoryboardWorkflow(context.Background(), "board", domain.WorkflowStatusPublished, 5); err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateStoryboardWorkflow(context.Background(), "board", domain.WorkflowStatusImagesReady, 3); err != nil {
		t.Fatal(err)
	}
	var board Storyboard
	if err := r.db.First(&board, "id = ?", "board").Error; err != nil {
		t.Fatal(err)
	}
	if board.WorkflowStatus != domain.WorkflowStatusPublished || board.CurrentStep != 5 {
		t.Fatalf("published board was downgraded: status=%s step=%d", board.WorkflowStatus, board.CurrentStep)
	}
}
