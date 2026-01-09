package service

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func (s *Service) AddStoryTags(ctx context.Context, storyID string, tagNames []string) error {
	for _, name := range tagNames {
		tag, err := s.repo.GetOrCreateTag(ctx, name)
		if err != nil {
			return err
		}
		if err := s.repo.AddStoryTag(ctx, storyID, tag.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RemoveStoryTag(ctx context.Context, storyID, tagID string) error {
	return s.repo.RemoveStoryTag(ctx, storyID, tagID)
}

func (s *Service) GetStoryTags(ctx context.Context, storyID string) ([]*domain.Tag, error) {
	return s.repo.StoryTags(ctx, storyID)
}

func (s *Service) GetStoriesByTag(ctx context.Context, tagID string, limit, offset int) ([]*domain.Story, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.StoriesByTag(ctx, tagID, limit, offset)
}

func (s *Service) GetPopularTags(ctx context.Context, limit int) ([]*domain.Tag, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.PopularTags(ctx, limit)
}

func (s *Service) AddCharacterTags(ctx context.Context, characterID string, tagNames []string) error {
	for _, name := range tagNames {
		tag, err := s.repo.GetOrCreateTag(ctx, name)
		if err != nil {
			return err
		}
		if err := s.repo.AddCharacterTag(ctx, characterID, tag.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetCharacterTags(ctx context.Context, characterID string) ([]*domain.Tag, error) {
	return s.repo.CharacterTags(ctx, characterID)
}
