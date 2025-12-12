package service

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func (s *Service) SearchStories(ctx context.Context, query string, limit, offset int) ([]*domain.Story, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.SearchStories(ctx, query, limit, offset)
}

func (s *Service) SearchCharacters(ctx context.Context, query string, limit, offset int) ([]*domain.Character, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.SearchCharacters(ctx, query, limit, offset)
}

func (s *Service) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.SearchUsers(ctx, query, limit, offset)
}

func (s *Service) SearchGroups(ctx context.Context, query string, limit, offset int) ([]*domain.Group, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.SearchGroups(ctx, query, limit, offset)
}

func (s *Service) SearchAll(ctx context.Context, query string, limit int) (map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}
	stories, _ := s.repo.SearchStories(ctx, query, limit, 0)
	characters, _ := s.repo.SearchCharacters(ctx, query, limit, 0)
	users, _ := s.repo.SearchUsers(ctx, query, limit, 0)
	groups, _ := s.repo.SearchGroups(ctx, query, limit, 0)

	return map[string]interface{}{
		"stories":    stories,
		"characters": characters,
		"users":      users,
		"groups":     groups,
	}, nil
}
