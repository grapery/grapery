package http

import (
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestWorkflowReleaseManagesStoryboardStagesRequiresCompleteStageSet(t *testing.T) {
	definition := domain.WorkflowDefinition{Nodes: []domain.WorkflowNode{
		{Activity: "storyboard.ensure_draft"},
		{Activity: "storyboard.generate_bible_plan"},
		{Activity: "storyboard.generate_scene_plan"},
		{Activity: "storyboard.persist_content"},
	}}
	if !workflowReleaseManagesStoryboardStages(definition) {
		t.Fatal("expected complete staged workflow to suppress legacy generation")
	}
	definition.Nodes = definition.Nodes[:3]
	if workflowReleaseManagesStoryboardStages(definition) {
		t.Fatal("partial staged workflow must retain legacy generation compatibility")
	}
}
