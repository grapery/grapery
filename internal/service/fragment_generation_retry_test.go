package service

import (
	"errors"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestFragmentRetryPreservesCompletedStagesAndRejectsUnknownProviderOutcomes(t *testing.T) {
	task := &domain.FragmentGenerationTask{ID: "task", UserID: "owner", Status: "failed"}
	for _, tc := range []struct {
		checkpoint     string
		retry, unknown bool
	}{
		{`{"version":1,"stages":{"elements":{"state":"completed","payload":{"Content":"saved"}},"images":{"state":"inflight"}}}`, true, false},
		{`{"version":1,"stages":{"elements":{"state":"inflight"}}}`, false, true},
		{`{"version":1,"stages":{"page_plan:0":{"state":"inflight"}}}`, false, true},
		{`{"version":1,"stages":{"audit":{"state":"inflight"}}}`, false, true},
		{`{"version":1,"stages":{"recheck":{"state":"inflight"}}}`, false, true},
		{`{"version":1,"stages":{"prepared_commit":{"state":"completed","payload":{}}}}`, true, false},
		{``, false, true},
	} {
		task.CheckpointJSON = tc.checkpoint
		got := fragmentRetryEligibility(task, nil)
		if got.CanRetry != tc.retry || got.RequiresReconciliation != tc.unknown {
			t.Fatalf("eligibility %s: %#v", tc.checkpoint, got)
		}
		if task.CheckpointJSON != tc.checkpoint {
			t.Fatal("eligibility mutated durable checkpoint")
		}
	}
	for _, status := range []string{"completed", "cancelled", "pending", "processing"} {
		task.Status = status
		if fragmentRetryEligibility(task, nil).CanRetry {
			t.Fatal("nonfailed task marked retryable", status)
		}
	}
}

func TestFragmentRetryChecksImageOperationBeforeResuming(t *testing.T) {
	task := &domain.FragmentGenerationTask{ID: "task", UserID: "owner", Status: "failed", CheckpointJSON: `{"version":1,"stages":{"images":{"state":"inflight"},"image_output:op":{"state":"inflight"}}}`}
	for _, tc := range []struct {
		status, output string
		lookupErr      error
		retry          bool
	}{
		{"processing", "{}", nil, false},
		{"pending", "{}", nil, false},
		{"failed", "{}", nil, true},
		{"completed", `{"imageURLs":["https://saved/image"]}`, nil, true},
		{"", "", domain.ErrNotFound, true},
		{"", "", errors.New("database unavailable"), false},
	} {
		got := fragmentRetryEligibility(task, func(string) (*domain.AIGenerationRecord, error) {
			return &domain.AIGenerationRecord{UserID: "owner", RelatedEntityID: "task", Status: domain.AITaskStatus(tc.status), OutputResult: tc.output}, tc.lookupErr
		})
		if got.CanRetry != tc.retry {
			t.Fatalf("%s %s: %#v", tc.status, tc.output, got)
		}
	}
	task.ErrorMessage = "billing_unsettled: source unavailable"
	if got := fragmentRetryEligibility(task, nil); got.CanRetry || !got.RequiresReconciliation {
		t.Fatalf("unsettled failed task can bypass settlement: %#v", got)
	}
}

func TestFragmentRequestReplayValidatesIntentAndIgnoresTransportDefaults(t *testing.T) {
	req := domain.FragmentGenerationRequest{ClientMessageID: "key", UserInput: "saved story", ImageCount: 2, Style: "anime", TargetDraftFragmentID: "draft"}
	task := &domain.FragmentGenerationTask{UserID: "owner", Request: req}
	replay := req
	replay.WorkflowRunID = "new-transport-run"
	replay.WorkflowSystemPrompt = "ignored legacy template"
	replay.IncludeGenerationTrace = true
	replay.BillingVersion = 1
	if err := validateFragmentRequestReplay(task, "owner", replay); err != nil {
		t.Fatal(err)
	}
	for _, changed := range []domain.FragmentGenerationRequest{
		{ClientMessageID: "key", UserInput: "different story", ImageCount: 2, Style: "anime", TargetDraftFragmentID: "draft"},
		{ClientMessageID: "key", UserInput: "saved story", ImageCount: 3, Style: "anime", TargetDraftFragmentID: "draft"},
		{ClientMessageID: "key", UserInput: "saved story", ImageCount: 2, Style: "anime", TargetDraftFragmentID: "other-draft"},
	} {
		if validateFragmentRequestReplay(task, "owner", changed) == nil {
			t.Fatal("changed paid intent replay accepted")
		}
	}
	if validateFragmentRequestReplay(task, "another-owner", req) == nil {
		t.Fatal("cross-owner request replay accepted")
	}
}
