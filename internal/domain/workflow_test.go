package domain

import "testing"

func TestValidateWorkflowRelease(t *testing.T) {
	base := func() *WorkflowRelease {
		return &WorkflowRelease{
			ID: "wfr_storyboard_1", Key: "storyboard", Version: 1, Status: "released",
			Definition: WorkflowDefinition{Nodes: []WorkflowNode{
				{ID: "load", Type: "activity", Activity: "legacy.storyboard.generate"},
				{ID: "plan", Type: "activity", Activity: "legacy.storyboard.generate", DependsOn: []string{"load"}},
				{ID: "persist", Type: "persist", Activity: "legacy.storyboard.generate", DependsOn: []string{"plan"}},
			}},
		}
	}
	if err := ValidateWorkflowRelease(base()); err != nil {
		t.Fatalf("valid release rejected: %v", err)
	}
	split := base()
	split.Definition.Nodes = []WorkflowNode{
		{ID: "generate_storyboard", Type: "activity", Activity: "storyboard.ensure_draft"},
		{ID: "await_content", Type: "activity", Activity: "storyboard.await_content", DependsOn: []string{"generate_storyboard"}},
		{ID: "ensure_images", Type: "activity", Activity: "storyboard.ensure_images", DependsOn: []string{"await_content"}},
	}
	if err := ValidateWorkflowRelease(split); err != nil {
		t.Fatalf("split storyboard workflow rejected: %v", err)
	}

	cyclic := base()
	cyclic.Definition.Nodes[0].DependsOn = []string{"persist"}
	if err := ValidateWorkflowRelease(cyclic); err == nil {
		t.Fatal("expected cycle validation error")
	}

	unknown := base()
	unknown.Definition.Nodes[1].DependsOn = []string{"missing"}
	if err := ValidateWorkflowRelease(unknown); err == nil {
		t.Fatal("expected unknown dependency validation error")
	}
}
