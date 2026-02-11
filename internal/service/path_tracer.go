package service

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// PathTracer handles path tracing for parallel universe continuation
// It recursively collects all scenes along the path from root to a specific storyboard
type PathTracer struct {
	repo   domain.Repository
	logger *zap.Logger
}

// NewPathTracer creates a new PathTracer instance
func NewPathTracer(repo domain.Repository, logger *zap.Logger) *PathTracer {
	return &PathTracer{
		repo:   repo,
		logger: logger,
	}
}

// TracePath traces the path from root to the specified storyboard
// Returns all scenes along the path in order (root -> parent -> target)
func (pt *PathTracer) TracePath(ctx context.Context, storyboardID string) ([]*domain.StoryboardScene, error) {
	pt.logger.Debug("starting path trace",
		zap.String("storyboardId", storyboardID))

	// Start with the target storyboard
	currentStoryboard, err := pt.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch storyboard: %w", err)
	}

	var allScenes []*domain.StoryboardScene
	var visited []string

	// Trace back to root, collecting scenes along the way
	for currentStoryboard != nil {
		// Detect cycles
		for _, v := range visited {
			if v == currentStoryboard.ID {
				pt.logger.Warn("cycle detected in storyboard hierarchy",
					zap.String("storyboardId", storyboardID),
					zap.String("cycleStart", v))
				return nil, fmt.Errorf("cycle detected in storyboard hierarchy")
			}
		}
		visited = append(visited, currentStoryboard.ID)

		// Collect scenes from current storyboard
		scenes, err := pt.repo.StoryboardScenes(ctx, currentStoryboard.ID)
		if err != nil {
			pt.logger.Warn("failed to fetch scenes for storyboard",
				zap.String("storyboardId", currentStoryboard.ID),
				zap.Error(err))
			// Continue even if we can't fetch scenes
		} else {
			// Prepend scenes so they're in root -> target order
			allScenes = append(scenes, allScenes...)
			pt.logger.Debug("collected scenes from storyboard",
				zap.String("storyboardId", currentStoryboard.ID),
				zap.Int("sceneCount", len(scenes)))
		}

		// Check if we've reached the root
		if currentStoryboard.ParentID == "" ||
		   currentStoryboard.ParentID == domain.StoryboardRootMarker {
			pt.logger.Debug("reached root storyboard",
				zap.String("rootStoryboardId", currentStoryboard.ID))
			break
		}

		// Move to parent
		currentStoryboard, err = pt.repo.StoryboardByID(ctx, currentStoryboard.ParentID)
		if err != nil {
			pt.logger.Warn("failed to fetch parent storyboard",
				zap.String("parentId", currentStoryboard.ParentID),
				zap.String("childStoryboardId", storyboardID),
				zap.Error(err))
			// Stop if we can't find parent
			break
		}
	}

	pt.logger.Debug("path trace completed",
		zap.String("storyboardId", storyboardID),
		zap.Int("totalScenes", len(allScenes)),
		zap.Int("totalStoryboards", len(visited)))

	return allScenes, nil
}

// TracePathWithMetadata traces the path and returns metadata about each storyboard
// This is useful for debugging and for building rich context for AI generation
func (pt *PathTracer) TracePathWithMetadata(ctx context.Context, storyboardID string) ([]StoryboardPathNode, error) {
	pt.logger.Debug("starting path trace with metadata",
		zap.String("storyboardId", storyboardID))

	var pathNodes []StoryboardPathNode
	var visited []string

	currentStoryboard, err := pt.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch storyboard: %w", err)
	}

	for currentStoryboard != nil {
		// Detect cycles
		for _, v := range visited {
			if v == currentStoryboard.ID {
				pt.logger.Warn("cycle detected in storyboard hierarchy",
					zap.String("storyboardId", storyboardID),
					zap.String("cycleStart", v))
				return nil, fmt.Errorf("cycle detected in storyboard hierarchy")
			}
		}
		visited = append(visited, currentStoryboard.ID)

		// Collect scenes
		scenes, err := pt.repo.StoryboardScenes(ctx, currentStoryboard.ID)
		if err != nil {
			pt.logger.Warn("failed to fetch scenes for storyboard",
				zap.String("storyboardId", currentStoryboard.ID),
				zap.Error(err))
			scenes = []*domain.StoryboardScene{}
		}

		// Create path node
		node := StoryboardPathNode{
			Storyboard: currentStoryboard,
			Scenes:     scenes,
			Depth:     len(pathNodes),
		}

		// Prepend so root is first
		pathNodes = append([]StoryboardPathNode{node}, pathNodes...)

		// Check if we've reached the root
		if currentStoryboard.ParentID == "" ||
		   currentStoryboard.ParentID == domain.StoryboardRootMarker {
			break
		}

		// Move to parent
		currentStoryboard, err = pt.repo.StoryboardByID(ctx, currentStoryboard.ParentID)
		if err != nil {
			pt.logger.Warn("failed to fetch parent storyboard",
				zap.String("parentId", currentStoryboard.ParentID),
				zap.Error(err))
			break
		}
	}

	pt.logger.Debug("path trace with metadata completed",
		zap.String("storyboardId", storyboardID),
		zap.Int("pathNodes", len(pathNodes)))

	return pathNodes, nil
}

// GetPathDepth returns the depth of a storyboard in the tree
// Root storyboards have depth 0
func (pt *PathTracer) GetPathDepth(ctx context.Context, storyboardID string) (int, error) {
	depth := 0
	visited := make(map[string]bool)

	currentStoryboard, err := pt.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch storyboard: %w", err)
	}

	for currentStoryboard != nil {
		// Detect cycles
		if visited[currentStoryboard.ID] {
			return 0, fmt.Errorf("cycle detected in storyboard hierarchy")
		}
		visited[currentStoryboard.ID] = true

		// Check if we've reached the root
		if currentStoryboard.ParentID == "" ||
		   currentStoryboard.ParentID == domain.StoryboardRootMarker {
			break
		}

		depth++

		// Move to parent
		currentStoryboard, err = pt.repo.StoryboardByID(ctx, currentStoryboard.ParentID)
		if err != nil {
			return depth, nil // Return depth we've calculated so far
		}
	}

	return depth, nil
}

// StoryboardPathNode represents a storyboard in the path with its metadata
type StoryboardPathNode struct {
	Storyboard *domain.Storyboard       `json:"storyboard"`
	Scenes     []*domain.StoryboardScene `json:"scenes"`
	Depth      int                       `json:"depth"`
}

// TraceAncestors returns all ancestor storyboards (excluding the target)
func (pt *PathTracer) TraceAncestors(ctx context.Context, storyboardID string) ([]*domain.Storyboard, error) {
	pt.logger.Debug("tracing ancestors",
		zap.String("storyboardId", storyboardID))

	var ancestors []*domain.Storyboard
	visited := make(map[string]bool)

	currentStoryboard, err := pt.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch storyboard: %w", err)
	}

	for currentStoryboard != nil {
		// Detect cycles
		if visited[currentStoryboard.ID] {
			return nil, fmt.Errorf("cycle detected in storyboard hierarchy")
		}
		visited[currentStoryboard.ID] = true

		// Check if we've reached the root
		if currentStoryboard.ParentID == "" ||
		   currentStoryboard.ParentID == domain.StoryboardRootMarker {
			break
		}

		// Move to parent
		currentStoryboard, err = pt.repo.StoryboardByID(ctx, currentStoryboard.ParentID)
		if err != nil {
			pt.logger.Warn("failed to fetch parent storyboard",
				zap.String("parentId", currentStoryboard.ParentID),
				zap.Error(err))
			break
		}

		// Add to ancestors (excluding the original target)
		ancestors = append([]*domain.Storyboard{currentStoryboard}, ancestors...)
	}

	pt.logger.Debug("ancestors traced",
		zap.String("storyboardId", storyboardID),
		zap.Int("ancestorCount", len(ancestors)))

	return ancestors, nil
}
