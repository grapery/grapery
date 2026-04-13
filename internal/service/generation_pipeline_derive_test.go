package service

import (
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestDeriveWizardPipelineSteps_ContentFailed(t *testing.T) {
	sb := &domain.Storyboard{SceneCount: 4, WorkflowStatus: domain.WorkflowStatusDraft}
	cg := &domain.StoryboardContentGeneration{Status: domain.GenerationStatusFailed, ErrorMessage: "quota exceeded"}
	steps, sug := deriveWizardPipelineSteps(sb, nil, cg, nil, nil, false)
	if len(steps) != 3 {
		t.Fatalf("steps len: %d", len(steps))
	}
	if steps[0].Phase != domain.PipelinePhaseContent || steps[0].Status != domain.PipelineStepFailed {
		t.Fatalf("content step: %+v", steps[0])
	}
	if sug != domain.SuggestedResumeRegenerateContent {
		t.Fatalf("suggested: %s", sug)
	}
}

func TestDeriveWizardPipelineSteps_ImagesRetry(t *testing.T) {
	sb := &domain.Storyboard{
		SceneCount:     2,
		WorkflowStatus: domain.WorkflowStatusDraft,
		Content:        "hello",
	}
	scenes := []*domain.StoryboardScene{
		{BaseModel: common.BaseModel{ID: "s1"}, Title: "A", Sequence: 1},
		{BaseModel: common.BaseModel{ID: "s2"}, Title: "B", Sequence: 2},
	}
	imgs := []*domain.StoryboardImageGeneration{
		{SceneID: "s1", Status: domain.GenerationStatusCompleted, GeneratedImageURL: "http://x", CreatedAt: 1},
		{SceneID: "s2", Status: domain.GenerationStatusFailed, ErrorMessage: "timeout", CreatedAt: 2},
	}
	steps, sug := deriveWizardPipelineSteps(sb, scenes, nil, nil, imgs, false)
	if steps[2].Status != domain.PipelineStepFailed {
		t.Fatalf("images step: %+v", steps[2])
	}
	if sug != domain.SuggestedResumeRetryFailedImages {
		t.Fatalf("suggested: %s", sug)
	}
}

func TestDeriveWizardPipelineSteps_AllImagesDone(t *testing.T) {
	sb := &domain.Storyboard{SceneCount: 1, WorkflowStatus: domain.WorkflowStatusImagesReady, Content: "c"}
	scenes := []*domain.StoryboardScene{
		{BaseModel: common.BaseModel{ID: "s1"}, Title: "A", Image: "http://img", Sequence: 1},
	}
	steps, sug := deriveWizardPipelineSteps(sb, scenes, nil, nil, nil, false)
	if steps[2].Status != domain.PipelineStepCompleted {
		t.Fatalf("images: %+v", steps[2])
	}
	if sug != domain.SuggestedResumeNone {
		t.Fatalf("suggested: %s", sug)
	}
}
