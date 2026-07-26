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
