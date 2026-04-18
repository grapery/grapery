package service

import (
	"context"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const (
	plazaTopTopicsMinCount = 2
	plazaTopTopicsLimit    = 8
	plazaTopTopicsCacheTTL = 2 * time.Minute
)

// TopPublicFragmentTopicLabelsForPlaza returns top public fragment topic labels with Redis read-through (JSON array).
func (s *Service) TopPublicFragmentTopicLabelsForPlaza(ctx context.Context) ([]string, error) {
	key := cache.PlazaTopFragmentTopicsKeyV1()
	if c := s.getCache(); c != nil {
		var cached []string
		if err := c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}
	topics, err := s.repo.ListTopPublicFragmentTopicLabels(ctx, plazaTopTopicsMinCount, plazaTopTopicsLimit)
	if err != nil {
		return nil, err
	}
	if c := s.getCache(); c != nil && len(topics) > 0 {
		if err := c.Set(ctx, key, topics, plazaTopTopicsCacheTTL); err != nil {
			s.logger.Debug("plaza top topics cache set failed", zap.Error(err))
		}
	}
	return topics, nil
}

// ListPublicFragmentPreviews returns up to `limit` public non-draft fragments for plaza / embed previews.
func (s *Service) ListPublicFragmentPreviews(ctx context.Context, limit int) ([]*domain.Fragment, error) {
	if limit <= 0 {
		limit = 12
	}
	if limit > 100 {
		limit = 100
	}
	frags, _, err := s.repo.ListPublicNonDraftFragments(ctx, limit, 0)
	return frags, err
}

// ListPublicFragmentsByTopicPreview returns up to `limit` public non-draft fragments for an exact topic label.
func (s *Service) ListPublicFragmentsByTopicPreview(ctx context.Context, topic string, limit int) ([]*domain.Fragment, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 12
	}
	if limit > 100 {
		limit = 100
	}
	frags, _, err := s.repo.ListPublicFragmentsByTopic(ctx, topic, limit, 0)
	return frags, err
}

// ListPublishedStoriesPreview returns recently published stories for plaza rails (newest first).
func (s *Service) ListPublishedStoriesPreview(ctx context.Context, limit int) ([]*domain.Story, error) {
	if limit <= 0 {
		limit = 12
	}
	if limit > 100 {
		limit = 100
	}
	stories, _, err := s.repo.ListStories(ctx, domain.StoryFilter{
		Status: "published",
		Limit:  limit,
		Offset: 0,
	})
	return stories, err
}
