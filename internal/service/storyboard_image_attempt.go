package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

var ErrStoryboardImageAttemptConflict = errors.New("storyboard_image_attempt_conflict")

const storyboardImageReconciliationMarker = "[reconciliation_required]"

type storyboardImageAttemptRepository interface {
	CreateStoryboardImageAttempt(context.Context, *domain.StoryboardImageGeneration, string, string, *string) error
	FinalizeStoryboardImageGeneration(context.Context, *domain.StoryboardImageGeneration) (bool, error)
}

func storyboardImageAttemptError(err error) error {
	if errors.Is(err, domain.ErrAlreadyExists) {
		return fmt.Errorf("%w: %w", ErrStoryboardImageAttemptConflict, err)
	}
	return err
}

func (s *Service) createStoryboardImageAttempt(ctx context.Context, gen *domain.StoryboardImageGeneration, owner, expectedRevision string, expectedPrevious *string) error {
	repo, ok := s.repo.(storyboardImageAttemptRepository)
	if !ok {
		return fmt.Errorf("storyboard image attempt persistence unavailable")
	}
	return storyboardImageAttemptError(repo.CreateStoryboardImageAttempt(ctx, gen, owner, expectedRevision, expectedPrevious))
}

func (s *Service) finalizeStoryboardImageAttempt(ctx context.Context, gen *domain.StoryboardImageGeneration) (bool, error) {
	repo, ok := s.repo.(storyboardImageAttemptRepository)
	if !ok {
		return false, fmt.Errorf("storyboard image attempt persistence unavailable")
	}
	accepted, err := repo.FinalizeStoryboardImageGeneration(ctx, gen)
	return accepted, storyboardImageAttemptError(err)
}

func storyboardSceneImageObjectKey(gen *domain.StoryboardImageGeneration, extension string) string {
	return fmt.Sprintf("storyboard/%s/scenes/%s/generations/%s.%s", gen.StoryboardID, gen.SceneID, gen.ID, strings.TrimPrefix(extension, "."))
}

func markStoryboardImageOutcomeUnknown(err error) string {
	return storyboardImageReconciliationMarker + " " + formatGenerationError(classifyGenerationError(err, GenerationErrorProvider), err.Error())
}

func storyboardImageOutcomeRequiresReconciliation(gen *domain.StoryboardImageGeneration) bool {
	return gen != nil && strings.Contains(gen.ErrorMessage, storyboardImageReconciliationMarker)
}

func (s *Service) completeStoryboardImageAttempt(ctx context.Context, gen *domain.StoryboardImageGeneration) (bool, error) {
	if strings.TrimSpace(gen.GeneratedImageURL) != "" {
		return s.finalizeStoryboardImageAttempt(ctx, gen)
	}
	gen.Status = domain.GenerationStatusFailed
	if gen.ErrorMessage == "" {
		gen.ErrorMessage = formatGenerationError(GenerationErrorProvider, "image generation failed: empty generated image url")
	}
	now := time.Now().Unix()
	gen.CompletedAt = &now
	return false, s.repo.UpdateImageGeneration(ctx, gen)
}
