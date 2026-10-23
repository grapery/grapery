package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestStoryboardImageProviderUncertaintyRequiresReconciliation(t *testing.T) {
	message := markStoryboardImageOutcomeUnknown(errors.New("provider timeout after dispatch"))
	gen := &domain.StoryboardImageGeneration{Status: domain.GenerationStatusFailed, ErrorMessage: message}
	if !storyboardImageOutcomeRequiresReconciliation(gen) {
		t.Fatal("unknown provider outcome was authorized for automatic retry")
	}
	if !strings.Contains(message, string(GenerationErrorTimeout)) {
		t.Fatalf("error classification lost: %q", message)
	}
}
