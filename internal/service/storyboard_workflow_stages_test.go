package service

import (
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestStoryboardBiblePlanFromRunRestoresSplitCheckpoint(t *testing.T) {
	run := &domain.StoryboardGenerationRun{
		StoryboardBibleJSON: `{"continuityRules":["keep coat red"]}`,
		BeatsJSON:           `[{"index":1,"purpose":"setup","summary":"arrival"}]`,
	}
	plan, err := storyboardBiblePlanFromRun(run)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Beats) != 1 || plan.Beats[0].Summary != "arrival" || len(plan.StoryboardBible.ContinuityRules) != 1 {
		t.Fatalf("unexpected restored plan: %#v", plan)
	}
}

func TestMergeStoryboardRunMetricsPreservesPreviousStage(t *testing.T) {
	got := mergeStoryboardRunMetrics(`{"biblePlanTokens":12}`, map[string]any{"scenePlanTokens": 34})
	if got != `{"biblePlanTokens":12,"scenePlanTokens":34}` && got != `{"scenePlanTokens":34,"biblePlanTokens":12}` {
		t.Fatalf("unexpected metrics: %s", got)
	}
}

func TestStoryboardMetricIntReadsJSONNumber(t *testing.T) {
	if got := storyboardMetricInt(`{"scenePlanRepairAttempts":1}`, "scenePlanRepairAttempts"); got != 1 {
		t.Fatalf("unexpected metric: %d", got)
	}
}

func TestStoryboardScenePlanCorrectionContextIncludesIssuesAndCandidate(t *testing.T) {
	got := storyboardScenePlanCorrectionContext(`{"scenes":[]}`, []domain.FragmentConsistencyIssue{{Severity: "high", Detail: "scene count mismatch"}})
	if !strings.Contains(got, "scene count mismatch") || !strings.Contains(got, `{"scenes":[]}`) {
		t.Fatalf("correction context lost evidence: %s", got)
	}
}

func TestStoryboardWorkflowRevisionRunMatchesStableRequestIdentity(t *testing.T) {
	run := &domain.StoryboardGenerationRun{RequestJSON: `{"clientRequestId":"turn_2","regenerateStructure":true,"userDirective":"改成雨夜"}`}
	if !storyboardWorkflowRunMatchesRequest(run, StoryboardWorkflowStageOptions{ClientRequestID: "turn_2", RegenerateStructure: true, UserDirective: "different retry text"}) {
		t.Fatal("same client request must reuse the durable revision run")
	}
	if storyboardWorkflowRunMatchesRequest(run, StoryboardWorkflowStageOptions{ClientRequestID: "turn_3", RegenerateStructure: true, UserDirective: "改成雨夜"}) {
		t.Fatal("a new client request must create a new revision run")
	}
	if !storyboardWorkflowRunIsRevision(run) {
		t.Fatal("revision marker was not restored from request checkpoint")
	}
}

func TestStoryboardWorkflowOutputLanguageUsesCheckpoint(t *testing.T) {
	run := &domain.StoryboardGenerationRun{RequestJSON: `{"language":"ja"}`}
	if got := storyboardWorkflowOutputLanguage(run); got != "ja" {
		t.Fatalf("unexpected output language: %q", got)
	}
	if got := storyboardWorkflowOutputLanguage(&domain.StoryboardGenerationRun{}); got != "zh-Hans" {
		t.Fatalf("unexpected default output language: %q", got)
	}
}
