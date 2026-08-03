package mysql

import (
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestStoryboardWorkflowPromptSnapshotsRoundTrip(t *testing.T) {
	source := &domain.Storyboard{
		WorkflowReleaseID: "wfr_storyboard_1",
		WorkflowChecksum:  "workflow-checksum",
		PromptSnapshots: map[string]domain.PromptTemplateVersion{
			"generate_storyboard": {ID: "ptv_bible_1", Key: "storyboard.bible", Checksum: "prompt-checksum"},
		},
	}
	model := StoryboardToModel(source)
	if model.WorkflowReleaseID != source.WorkflowReleaseID || model.PromptSnapshotsJSON == "" {
		t.Fatalf("workflow context was not serialized: %+v", model)
	}
	roundTrip := ModelToStoryboard(model)
	if roundTrip.WorkflowChecksum != source.WorkflowChecksum || roundTrip.PromptSnapshots["generate_storyboard"].ID != "ptv_bible_1" {
		t.Fatalf("workflow context did not round trip: %+v", roundTrip)
	}
}
