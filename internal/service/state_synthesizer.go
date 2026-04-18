package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// StateSynthesizer handles character state synthesis for parallel universe continuation
// It combines the "soul" (base character) with all context deltas from scenes
type StateSynthesizer struct {
	repo   domain.Repository
	logger *zap.Logger
}

// NewStateSynthesizer creates a new StateSynthesizer instance
func NewStateSynthesizer(repo domain.Repository, logger *zap.Logger) *StateSynthesizer {
	return &StateSynthesizer{
		repo:   repo,
		logger: logger,
	}
}

// SynthesizeStates combines character base states with all scene context deltas
// Returns the list of characters and their synthesized fate states
func (ss *StateSynthesizer) SynthesizeStates(
	ctx context.Context,
	parentStoryboard *domain.Storyboard,
	ancestorScenes []*domain.StoryboardScene,
	requestedCharacterIDs []string,
) ([]domain.Character, map[string]CharacterFateState, error) {
	ss.logger.Debug("starting state synthesis",
		zap.String("parentStoryboardId", parentStoryboard.ID),
		zap.Int("ancestorScenes", len(ancestorScenes)),
		zap.Int("requestedCharacters", len(requestedCharacterIDs)))

	// Step 1: Extract character IDs from storyboard's CharacterRefs
	var characterIDs []string
	if len(requestedCharacterIDs) > 0 {
		// Use requested characters
		characterIDs = requestedCharacterIDs
	} else {
		// Use all characters referenced in the storyboard
		characterIDs = make([]string, 0, len(parentStoryboard.CharacterRefs))
		for _, ref := range parentStoryboard.CharacterRefs {
			characterIDs = append(characterIDs, ref.CharacterID)
		}
	}

	if len(characterIDs) == 0 {
		ss.logger.Warn("no characters found for storyboard",
			zap.String("storyboardId", parentStoryboard.ID))
		return []domain.Character{}, make(map[string]CharacterFateState), nil
	}

	// Step 3: Fetch all character "souls" (base state)
	characters := make([]domain.Character, 0, len(characterIDs))
	characterMap := make(map[string]*domain.Character)

	for _, charID := range characterIDs {
		character, err := ss.repo.CharacterByID(ctx, charID)
		if err != nil {
			ss.logger.Warn("failed to fetch character",
				zap.String("characterId", charID),
				zap.Error(err))
			continue
		}
		characters = append(characters, *character)
		characterMap[charID] = character
	}

	// Step 4: Initialize fate states from character souls
	fateStates := make(map[string]CharacterFateState)
	for _, character := range characters {
		fateStates[character.ID] = ss.extractBaseState(character)
	}

	// Step 5: Apply all context deltas from ancestor scenes
	for _, scene := range ancestorScenes {
		if scene.ContextSnapshot == "" {
			continue
		}

		var contextDelta map[string]json.RawMessage
		if err := json.Unmarshal([]byte(scene.ContextSnapshot), &contextDelta); err != nil {
			ss.logger.Warn("failed to parse context snapshot",
				zap.String("sceneId", scene.ID),
				zap.Error(err))
			continue
		}

		// Apply delta to each character's fate state
		for characterID, deltaJSON := range contextDelta {
			if _, exists := fateStates[characterID]; !exists {
				// Skip if character not in our fate state map
				continue
			}

			var delta map[string]interface{}
			if err := json.Unmarshal(deltaJSON, &delta); err != nil {
				ss.logger.Warn("failed to parse context delta for character",
					zap.String("characterId", characterID),
					zap.String("sceneId", scene.ID),
					zap.Error(err))
				continue
			}

			ss.applyDelta(fateStates[characterID], delta)
		}
	}

	ss.logger.Debug("state synthesis completed",
		zap.String("parentStoryboardId", parentStoryboard.ID),
		zap.Int("characters", len(characters)),
		zap.Int("fateStates", len(fateStates)))

	return characters, fateStates, nil
}

// extractBaseState extracts the base state from a character (the "soul")
func (ss *StateSynthesizer) extractBaseState(character domain.Character) CharacterFateState {
	state := CharacterFateState{
		CharacterID: character.ID,
		Name:        character.Name,
		Metadata:    make(map[string]interface{}),
	}

	// Parse personality JSON if available
	if character.Personality != "" {
		var personality map[string]interface{}
		if err := json.Unmarshal([]byte(character.Personality), &personality); err == nil {
			for k, v := range personality {
				state.Metadata[k] = v
			}
		}
	}

	// Parse background JSON if available
	if character.Background != "" {
		var background map[string]interface{}
		if err := json.Unmarshal([]byte(character.Background), &background); err == nil {
			for k, v := range background {
				state.Metadata[k] = v
			}
		}
	}

	// Add goals
	if character.ShortTermGoal != "" {
		state.Metadata["shortTermGoal"] = character.ShortTermGoal
	}
	if character.LongTermGoal != "" {
		state.Metadata["longTermGoal"] = character.LongTermGoal
	}

	// Add traits directly (already []string in domain model)
	if len(character.Traits) > 0 {
		state.Metadata["traits"] = character.Traits
	}

	// Add skills directly as knowledge
	if len(character.Skills) > 0 {
		state.Knowledge = character.Skills
	}

	return state
}

// applyDelta applies a context delta to a character's fate state
func (ss *StateSynthesizer) applyDelta(state CharacterFateState, delta map[string]interface{}) {
	for key, value := range delta {
		switch key {
		case "health":
			if v, ok := value.(float64); ok {
				state.Health = int(v)
			}
		case "mood":
			if v, ok := value.(string); ok {
				state.Mood = v
			}
		case "location":
			if v, ok := value.(string); ok {
				state.Location = v
			}
		case "relationships":
			if v, ok := value.(map[string]interface{}); ok {
				if state.Relationships == nil {
					state.Relationships = make(map[string]string)
				}
				for k, val := range v {
					if s, ok := val.(string); ok {
						state.Relationships[k] = s
					}
				}
			}
		case "knowledge":
			if v, ok := value.([]interface{}); ok {
				for _, item := range v {
					if s, ok := item.(string); ok {
						state.Knowledge = append(state.Knowledge, s)
					}
				}
			}
		case "goals":
			if v, ok := value.([]interface{}); ok {
				for _, item := range v {
					if s, ok := item.(string); ok {
						state.Goals = append(state.Goals, s)
					}
				}
			}
		default:
			// Store in metadata
			if state.Metadata == nil {
				state.Metadata = make(map[string]interface{})
			}
			state.Metadata[key] = value
		}
	}
}

// GetCharacterStateAtScene returns the state of a character at a specific scene
// This is useful for debugging and for showing character states in the UI
func (ss *StateSynthesizer) GetCharacterStateAtScene(
	ctx context.Context,
	characterID string,
	storyboardID string,
	sceneID string,
) (CharacterFateState, error) {
	ss.logger.Debug("getting character state at scene",
		zap.String("characterId", characterID),
		zap.String("storyboardId", storyboardID),
		zap.String("sceneId", sceneID))

	// Fetch character
	character, err := ss.repo.CharacterByID(ctx, characterID)
	if err != nil {
		return CharacterFateState{}, fmt.Errorf("failed to fetch character: %w", err)
	}

	// Initialize with base state
	state := ss.extractBaseState(*character)

	// Trace path and apply deltas up to the target scene
	pathTracer := NewPathTracer(ss.repo, ss.logger)
	ancestorScenes, err := pathTracer.TracePath(ctx, storyboardID)
	if err != nil {
		return CharacterFateState{}, fmt.Errorf("failed to trace path: %w", err)
	}

	// Apply deltas from all scenes before the target
	for _, scene := range ancestorScenes {
		if scene.ContextSnapshot == "" {
			continue
		}

		// Stop if we've reached the target scene
		if scene.ID == sceneID {
			break
		}

		var contextDelta map[string]json.RawMessage
		if err := json.Unmarshal([]byte(scene.ContextSnapshot), &contextDelta); err != nil {
			ss.logger.Warn("failed to parse context snapshot",
				zap.String("sceneId", scene.ID),
				zap.Error(err))
			continue
		}

		if deltaJSON, exists := contextDelta[characterID]; exists {
			var delta map[string]interface{}
			if err := json.Unmarshal(deltaJSON, &delta); err != nil {
				ss.logger.Warn("failed to parse context delta for character",
					zap.String("characterId", characterID),
					zap.String("sceneId", scene.ID),
					zap.Error(err))
				continue
			}

			ss.applyDelta(state, delta)
		}
	}

	return state, nil
}

// CreateFateSnapshotJSON creates a JSON representation of the fate snapshot
// This can be stored in the storyboard's fate_snapshot field
func (ss *StateSynthesizer) CreateFateSnapshotJSON(fateStates map[string]CharacterFateState) (string, error) {
	data, err := json.Marshal(fateStates)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fate snapshot: %w", err)
	}
	return string(data), nil
}
