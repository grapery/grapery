package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestFragmentGenerationHTTPErrorMapsDomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid request",
			err:        domain.NewFragmentGenerationError(domain.FragmentGenerationErrorInvalidRequest, "bad request", nil),
			wantStatus: http.StatusBadRequest,
			wantCode:   domain.FragmentGenerationErrorInvalidRequest,
		},
		{
			name:       "active task conflict",
			err:        domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "active task", nil),
			wantStatus: http.StatusConflict,
			wantCode:   domain.FragmentGenerationErrorConflict,
		},
		{
			name:       "unexpected error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotCode, _ := fragmentGenerationHTTPError(tt.err)
			if gotStatus != tt.wantStatus || gotCode != tt.wantCode {
				t.Fatalf("got status=%d code=%q, want status=%d code=%q", gotStatus, gotCode, tt.wantStatus, tt.wantCode)
			}
		})
	}
}

func TestFragmentGenerationSlotModeTreatsAppendPlaceholderSnapshotAsFull(t *testing.T) {
	task := &domain.FragmentGenerationTask{
		Request: domain.FragmentGenerationRequest{
			TargetDraftFragmentID: "fragment-1",
			ImageCount:            1,
		},
		Result: &domain.FragmentGenerationResult{
			ExpectedImageCount: 5,
			ImageSlots: []domain.FragmentGenerationImageSlot{
				{Index: 1, Status: "completed", ImageURL: "https://img.example/1.png"},
				{Index: 2, Status: "completed", ImageURL: "https://img.example/2.png"},
				{Index: 3, Status: "completed", ImageURL: "https://img.example/3.png"},
				{Index: 4, Status: "completed", ImageURL: "https://img.example/4.png"},
				{Index: 5, Status: "planned"},
			},
		},
	}

	if got := fragmentGenerationSlotMode(task); got != "full" {
		t.Fatalf("expected append placeholder snapshot to be full, got %q", got)
	}
}

func TestFragmentGenerationSlotModeKeepsReplacementDelta(t *testing.T) {
	task := &domain.FragmentGenerationTask{
		Request: domain.FragmentGenerationRequest{
			TargetDraftFragmentID: "fragment-1",
			ReplaceImageIndex:     3,
			ImageCount:            1,
		},
		Result: &domain.FragmentGenerationResult{
			ExpectedImageCount: 1,
			ImageSlots:         []domain.FragmentGenerationImageSlot{{Index: 1, Status: "generating"}},
		},
	}

	if got := fragmentGenerationSlotMode(task); got != "delta" {
		t.Fatalf("expected replacement snapshot to remain delta, got %q", got)
	}
}

func TestFragmentGenerationResultResponseUsesEmptyImageArrayWhileProcessing(t *testing.T) {
	task := &domain.FragmentGenerationTask{
		Result: &domain.FragmentGenerationResult{
			Content:   "正在生成",
			ImageUrls: nil,
		},
	}

	response := fragmentGenerationResultResponse(task)
	imageURLs, ok := response["imageUrls"].([]string)
	if !ok {
		t.Fatalf("imageUrls type = %T, want []string", response["imageUrls"])
	}
	if imageURLs == nil || len(imageURLs) != 0 {
		t.Fatalf("imageUrls = %#v, want a non-nil empty array", imageURLs)
	}
}
