package service

import (
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestBuildWorkflowContentProfileRecognizesBranchAndAction(t *testing.T) {
	profile := BuildWorkflowContentProfile("voyager.storyboard", "generate", map[string]any{
		"parentStoryboardId": "storyboard-parent",
		"chapterContent":     "她冲出车站，身后的玻璃突然爆炸，两个人一路追逐到桥边。",
		"characters":         []any{"a", "b"},
	})
	if profile["artifact"] != "storyboard" || profile["operation"] != "fork" || profile["continuityLevel"] != "strong" {
		t.Fatalf("unexpected branch profile: %+v", profile)
	}
	if profile["narrativeMode"] != "action" || profile["characterCount"] != 2 {
		t.Fatalf("unexpected narrative profile: %+v", profile)
	}
}

func TestResolveWorkflowEntriesUsesConditionalBindingBeforeFallback(t *testing.T) {
	entries := []*domain.WorkflowCatalogEntry{
		{
			Binding: domain.WorkflowBinding{ID: "action", Priority: 100, Conditions: map[string]any{
				"all": []any{
					map[string]any{"field": "narrativeMode", "op": "eq", "value": "action"},
					map[string]any{"field": "actionIntensity", "op": "gte", "value": 0.3},
				},
			}},
			Release: domain.WorkflowRelease{ID: "wfr_action"},
		},
		{
			Binding: domain.WorkflowBinding{ID: "default", Priority: 0},
			Release: domain.WorkflowRelease{ID: "wfr_default"},
		},
	}
	resolution, err := ResolveWorkflowEntries(entries, domain.WorkflowContentProfile{"narrativeMode": "action", "actionIntensity": 0.8})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Entry.Release.ID != "wfr_action" || resolution.Fallback || resolution.RouterVersion == "" {
		t.Fatalf("unexpected resolution: %+v", resolution)
	}

	resolution, err = ResolveWorkflowEntries(entries, domain.WorkflowContentProfile{"narrativeMode": "dialogue", "actionIntensity": 0.1})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Entry.Release.ID != "wfr_default" || !resolution.Fallback {
		t.Fatalf("expected default fallback: %+v", resolution)
	}
}

func TestResolveWorkflowEntriesSupportsAnyAndIn(t *testing.T) {
	entries := []*domain.WorkflowCatalogEntry{{
		Binding: domain.WorkflowBinding{ID: "continuity", Conditions: map[string]any{
			"any": []any{
				map[string]any{"field": "operation", "op": "in", "value": []any{"append", "fork"}},
				map[string]any{"field": "continuityLevel", "op": "eq", "value": "strong"},
			},
		}},
		Release: domain.WorkflowRelease{ID: "wfr_continuity"},
	}}
	resolution, err := ResolveWorkflowEntries(entries, domain.WorkflowContentProfile{"operation": "fork"})
	if err != nil || resolution.Entry.Release.ID != "wfr_continuity" {
		t.Fatalf("expected continuity route, resolution=%+v err=%v", resolution, err)
	}
}

func TestResolveWorkflowEntriesRejectsInvalidOperator(t *testing.T) {
	entries := []*domain.WorkflowCatalogEntry{{
		Binding: domain.WorkflowBinding{ID: "bad", Conditions: map[string]any{
			"all": []any{map[string]any{"field": "operation", "op": "regex", "value": ".*"}},
		}},
		Release: domain.WorkflowRelease{ID: "wfr_bad"},
	}}
	if _, err := ResolveWorkflowEntries(entries, domain.WorkflowContentProfile{"operation": "fork"}); err == nil {
		t.Fatal("expected invalid operator to fail closed")
	}
}

func TestBuildWorkflowContentProfileKeepsZeroScores(t *testing.T) {
	profile := BuildWorkflowContentProfile("voyager.fragment", "generate", map[string]any{"userInput": "安静的空房间"})
	if _, ok := profile["actionIntensity"]; !ok {
		t.Fatalf("zero action score must remain routable: %+v", profile)
	}
}

func TestValidateWorkflowBindingConditionsRejectsMalformedRulesBeforeRuntime(t *testing.T) {
	err := ValidateWorkflowBindingConditions(map[string]any{
		"all": []any{map[string]any{"field": "operation", "op": "regex", "value": ".*"}},
	})
	if err == nil {
		t.Fatal("expected malformed routing rules to be rejected")
	}
	if err := ValidateWorkflowBindingConditions(map[string]any{
		"any": []any{map[string]any{"field": "operation", "op": "in", "value": []any{"fork"}}},
	}); err != nil {
		t.Fatalf("expected valid conditions, got %v", err)
	}
}
