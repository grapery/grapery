package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"strings"
)

func fragmentSceneExpansionTokenBudget(count int) int32 {
	if count < 1 {
		count = 1
	}
	if count > 8 {
		count = 8
	}
	return int32(1024 + count*768)
}

func requestFragmentSceneExpansion(ctx context.Context, prompt string, count int, generate func(string, int32) (string, int, error), validators ...func([]fragmentExpandedScene) error) ([]fragmentExpandedScene, int, error) {
	total := 0
	currentPrompt := prompt
	for attempt := 0; attempt < 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, total, err
		}
		raw, tokens, err := generate(currentPrompt, fragmentSceneExpansionTokenBudget(count))
		total += tokens
		var scenes []fragmentExpandedScene
		if err == nil {
			scenes, err = parseStrictFragmentSceneExpansion(raw, count)
		}
		if err == nil {
			for _, validate := range validators {
				if err = validate(scenes); err != nil {
					break
				}
			}
		}
		if err == nil {
			return scenes, total, nil
		}
		if attempt == 1 {
			return nil, total, err
		}
		currentPrompt = fmt.Sprintf("%s\n\nSCHEMA REPAIR: The previous page-beat plan is invalid: %s. Preserve every story fact and return exactly %d complete, distinct page beats using the original JSON contract. Do not insert generic placeholders.\nPrevious output:\n%s", prompt, err, count, truncateRunes(raw, 14000))
	}
	return nil, total, fmt.Errorf("page-beat plan unavailable")
}

func parseStrictFragmentSceneExpansion(raw string, count int) ([]fragmentExpandedScene, error) {
	cleaned := cleanFragmentModelJSON(raw)
	var envelope struct {
		Scenes []fragmentExpandedScene `json:"scenes"`
	}
	if err := json.Unmarshal([]byte(cleaned), &envelope); err != nil {
		if e := json.Unmarshal([]byte(cleaned), &envelope.Scenes); e != nil {
			return nil, fmt.Errorf("invalid page-beat JSON: %w", err)
		}
	}
	if len(envelope.Scenes) != count {
		return nil, fmt.Errorf("expected %d page beats, got %d", count, len(envelope.Scenes))
	}
	seen := map[string]bool{}
	for i := range envelope.Scenes {
		scene := &envelope.Scenes[i]
		scene.SceneDesc = strings.TrimSpace(scene.SceneDesc)
		scene.ImagePrompt = strings.TrimSpace(scene.ImagePrompt)
		if scene.Index != i || scene.SceneDesc == "" || scene.ImagePrompt == "" {
			return nil, fmt.Errorf("page beat %d requires index %d, narrative and imagePrompt", i, i)
		}
		key := fragmentComicReviewKey(scene.SceneDesc)
		if seen[key] {
			return nil, fmt.Errorf("page beat %d repeats an earlier beat", i)
		}
		seen[key] = true
	}
	return envelope.Scenes, nil
}

func validateFragmentPageBeatEntities(scenes []fragmentExpandedScene, bible *domain.FragmentVisualBible) error {
	valid, _, kinds := fragmentVisualBibleKeySets(bible)
	for i, scene := range scenes {
		for _, key := range scene.ReferenceKeys {
			if _, ok := valid[key]; !ok {
				return fmt.Errorf("page %d references unknown entity %s", i, key)
			}
		}
		for _, binding := range scene.EntityBindings {
			if _, ok := valid[binding.Key]; !ok {
				return fmt.Errorf("page %d binds unknown entity %s", i, binding.Key)
			}
			if binding.Kind != "" && kinds[binding.Key] != binding.Kind {
				return fmt.Errorf("page %d swaps entity kind for %s", i, binding.Key)
			}
		}
	}
	return nil
}
