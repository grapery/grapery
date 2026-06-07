package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func popularTagsCacheKey(limit int) string {
	return fmt.Sprintf("tags:popular:%d", limit)
}

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
	if c := s.getCache(); c != nil {
		for i := 20; i <= 100; i += 20 {
			_ = c.Delete(ctx, popularTagsCacheKey(i))
		}
	}
	return nil
}

func (s *Service) RemoveStoryTag(ctx context.Context, storyID, tagID string) error {
	if err := s.repo.RemoveStoryTag(ctx, storyID, tagID); err != nil {
		return err
	}
	if c := s.getCache(); c != nil {
		for i := 20; i <= 100; i += 20 {
			_ = c.Delete(ctx, popularTagsCacheKey(i))
		}
	}
	return nil
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
	c := s.getCache()
	key := popularTagsCacheKey(limit)
	if c != nil {
		var cached []*domain.Tag
		if err := c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}
	tags, err := s.repo.PopularTags(ctx, limit)
	if err != nil {
		return nil, err
	}
	if c != nil {
		_ = c.Set(ctx, key, tags, 10*time.Minute)
	}
	return tags, nil
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
