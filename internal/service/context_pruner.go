package service

import (
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// ContextPruner handles context pruning for AI prompt optimization
// It implements a three-layer strategy: Core + Memory + Summary
type ContextPruner struct {
	logger *zap.Logger
}

// NewContextPruner creates a new ContextPruner instance
func NewContextPruner(logger *zap.Logger) *ContextPruner {
	return &ContextPruner{
		logger: logger,
	}
}

// PrunedContext represents the pruned context ready for AI generation
type PrunedContext struct {
	CoreLayer    []SceneWithContext `json:"coreLayer"`    // Character souls + final states
	MemoryLayer  []SceneWithContext `json:"memoryLayer"`  // Recent N scenes
	SummaryLayer []SceneSummary    `json:"summaryLayer"` // Ancient scenes as summaries

	// Total character count for reference
	Characters []domain.Character `json:"characters"`

	// Token estimation
	EstimatedTokenCount int `json:"estimatedTokenCount"`
}

// SceneWithContext represents a scene with full context
type SceneWithContext struct {
	Scene          domain.StoryboardScene `json:"scene"`
	CharacterNames []string              `json:"characterNames"`
	ContextSummary string                `json:"contextSummary,omitempty"`
}

// SceneSummary represents a summarized scene
type SceneSummary struct {
	SceneID       string `json:"sceneId"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	KeyEvents     []string `json:"keyEvents,omitempty"`
	CharacterMentions []string `json:"characterMentions,omitempty"`
}

// PruneContext prunes the full context into three layers for efficient AI generation
// maxRecentScenes specifies how many recent scenes to keep in full detail (Memory Layer)
func (cp *ContextPruner) PruneContext(
	allScenes []domain.StoryboardScene,
	characters []domain.Character,
	maxRecentScenes int,
) *PrunedContext {
	cp.logger.Debug("starting context pruning",
		zap.Int("totalScenes", len(allScenes)),
		zap.Int("maxRecentScenes", maxRecentScenes))

	result := &PrunedContext{
		Characters: characters,
	}

	totalScenes := len(allScenes)

	// If we have few scenes, return all as Memory Layer
	if totalScenes <= maxRecentScenes {
		result.MemoryLayer = cp.createMemoryLayer(allScenes, characters)
		result.EstimatedTokenCount = cp.estimateTokens(result)
		cp.logger.Debug("context pruned (all scenes fit in memory layer)",
			zap.Int("memoryLayerScenes", len(result.MemoryLayer)),
			zap.Int("estimatedTokens", result.EstimatedTokenCount))
		return result
	}

	// Split scenes: recent (Memory Layer) and ancient (Summary Layer)
	splitIndex := totalScenes - maxRecentScenes
	ancientScenes := allScenes[:splitIndex]
	recentScenes := allScenes[splitIndex:]

	// Create Memory Layer (recent scenes in full detail)
	result.MemoryLayer = cp.createMemoryLayer(recentScenes, characters)

	// Create Summary Layer (ancient scenes as summaries)
	result.SummaryLayer = cp.createSummaryLayer(ancientScenes, characters)

	// Core Layer is implicit (character souls are always included)
	// The character list in PrunedContext serves as the Core Layer

	result.EstimatedTokenCount = cp.estimateTokens(result)

	cp.logger.Debug("context pruning completed",
		zap.Int("coreLayerCharacters", len(result.Characters)),
		zap.Int("memoryLayerScenes", len(result.MemoryLayer)),
		zap.Int("summaryLayerScenes", len(result.SummaryLayer)),
		zap.Int("estimatedTokens", result.EstimatedTokenCount))

	return result
}

// createMemoryLayer creates the memory layer with full scene details
func (cp *ContextPruner) createMemoryLayer(scenes []domain.StoryboardScene, characters []domain.Character) []SceneWithContext {
	layer := make([]SceneWithContext, 0, len(scenes))

	characterNameMap := make(map[string]string)
	for _, char := range characters {
		characterNameMap[char.ID] = char.Name
	}

	for _, scene := range scenes {
		sceneWithCtx := SceneWithContext{
			Scene:          scene,
			CharacterNames: cp.extractCharacterNames(scene, characterNameMap),
		}

		// Add context summary if available
		if scene.ContextSnapshot != "" {
			sceneWithCtx.ContextSummary = cp.summarizeContextSnapshot(scene.ContextSnapshot)
		}

		layer = append(layer, sceneWithCtx)
	}

	return layer
}

// createSummaryLayer creates the summary layer with condensed scene information
func (cp *ContextPruner) createSummaryLayer(scenes []domain.StoryboardScene, characters []domain.Character) []SceneSummary {
	summaries := make([]SceneSummary, 0, len(scenes))

	characterNameMap := make(map[string]string)
	for _, char := range characters {
		characterNameMap[char.ID] = char.Name
	}

	for _, scene := range scenes {
		summary := SceneSummary{
			SceneID:   scene.ID,
			Title:     scene.Title,
			Summary:   cp.generateSceneSummary(scene),
			KeyEvents: cp.extractKeyEvents(scene),
		}

		// Track which characters appear in this scene
		characterNames := cp.extractCharacterNames(scene, characterNameMap)
		if len(characterNames) > 0 {
			summary.CharacterMentions = characterNames
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

// extractCharacterNames extracts character names from a scene
func (cp *ContextPruner) extractCharacterNames(scene domain.StoryboardScene, characterNameMap map[string]string) []string {
	// This is a simplified implementation
	// In production, this would parse scene.Characters JSON
	return []string{}
}

// summarizeContextSnapshot creates a human-readable summary of context snapshot
func (cp *ContextPruner) summarizeContextSnapshot(contextSnapshot string) string {
	// Simplified: return first 100 chars
	if len(contextSnapshot) > 100 {
		return contextSnapshot[:100] + "..."
	}
	return contextSnapshot
}

// generateSceneSummary generates a summary of a scene
func (cp *ContextPruner) generateSceneSummary(scene domain.StoryboardScene) string {
	// Simplified: use description truncated
	maxLen := 200
	description := scene.Description
	if len(description) > maxLen {
		return description[:maxLen] + "..."
	}
	return description
}

// extractKeyEvents extracts key events from a scene description
func (cp *ContextPruner) extractKeyEvents(scene domain.StoryboardScene) []string {
	// In production, this would use AI or NLP to extract key events
	// For now, return empty
	return []string{}
}

// estimateTokens estimates the token count for the pruned context
func (cp *ContextPruner) estimateTokens(context *PrunedContext) int {
	// Rough estimation: ~4 characters per token
	totalChars := 0

	// Count Core Layer (characters)
	for _, char := range context.Characters {
		totalChars += len(char.Name)
		totalChars += len(char.Description)
		totalChars += len(char.Personality)
	}

	// Count Memory Layer
	for _, scene := range context.MemoryLayer {
		totalChars += len(scene.Scene.Title)
		totalChars += len(scene.Scene.Description)
	}

	// Count Summary Layer
	for _, summary := range context.SummaryLayer {
		totalChars += len(summary.Title)
		totalChars += len(summary.Summary)
	}

	// Convert to tokens (rough estimate)
	return totalChars / 4
}

// PruneForMaxTokens prunes context to fit within a maximum token budget
// This is useful when you need to ensure the prompt doesn't exceed model limits
func (cp *ContextPruner) PruneForMaxTokens(
	allScenes []domain.StoryboardScene,
	characters []domain.Character,
	maxTokens int,
) *PrunedContext {
	cp.logger.Debug("pruning context for max tokens",
		zap.Int("totalScenes", len(allScenes)),
		zap.Int("maxTokens", maxTokens))

	// Start with all scenes in memory layer
	maxRecentScenes := len(allScenes)
	pruned := cp.PruneContext(allScenes, characters, maxRecentScenes)

	// If already under budget, return
	if pruned.EstimatedTokenCount <= maxTokens {
		return pruned
	}

	// Binary search to find the right number of recent scenes
	low, high := 0, len(allScenes)
	for low < high {
		mid := (low + high) / 2
		testPruned := cp.PruneContext(allScenes, characters, mid)
		if testPruned.EstimatedTokenCount <= maxTokens {
			low = mid + 1
			pruned = testPruned
		} else {
			high = mid
		}
	}

	cp.logger.Debug("pruned to fit token budget",
		zap.Int("finalMemoryLayerScenes", len(pruned.MemoryLayer)),
		zap.Int("finalEstimatedTokens", pruned.EstimatedTokenCount),
		zap.Int("maxTokens", maxTokens))

	return pruned
}
