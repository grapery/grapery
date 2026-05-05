package service

import (
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func buildStoryboardBiblePlanSystemPrompt() string {
	return `You are a senior storyboard continuity planner. Produce strict JSON only.
Your job is to create a storyboard visual bible and narrative beats before scene writing.
Rules:
- Preserve character identity and plot continuity over novelty.
- Character three-view sheets are identity authority for face, body, hairstyle, costume proportions, color blocking, and silhouette.
- Do not invent IDs. Use provided character IDs and location keys when available.
- Every beat referenceKeys value must point to a key declared in storyboardBible.characters, locations, or props.
- Output only JSON with keys storyboardBible and beats.`
}

func buildStoryboardBiblePlanUserPrompt(story *domain.Story, storyboard *domain.Storyboard, snapshot storyboardGenerationContextSnapshot, sceneCount int) string {
	var b strings.Builder
	b.WriteString("Create a storyboard bible and exactly ")
	b.WriteString(fmt.Sprintf("%d beats.\n\n", sceneCount))
	b.WriteString("Story:\n")
	b.WriteString(fmt.Sprintf("- id: %s\n- title: %s\n- genre: %s\n- description: %s\n", story.ID, story.Title, story.Genre, story.Description))
	if story.Style != nil {
		b.WriteString(fmt.Sprintf("- style: %s / %s\n", story.Style.Style, story.Style.Description))
	}
	b.WriteString("\nUser raw input:\n")
	b.WriteString(storyboard.RawInput)
	b.WriteString("\n\nContinuation/context facts:\n")
	b.WriteString(snapshot.ContextText)
	b.WriteString("\n\nCharacters with identity references:\n")
	if len(snapshot.Characters) == 0 {
		b.WriteString("- none explicitly selected; infer relationships only when necessary.\n")
	}
	for _, ch := range snapshot.Characters {
		b.WriteString(fmt.Sprintf("- key=%s id=%s name=%s\n", ch.Key, ch.ID, ch.Name))
		b.WriteString(fmt.Sprintf("  description=%s\n  personality=%s\n  appearance=%s\n", ch.Description, ch.Personality, ch.Appearance))
		if ch.Views != nil {
			b.WriteString(fmt.Sprintf("  threeViewSheet=%s front=%s side=%s back=%s\n", ch.Views.Sheet, ch.Views.Front, ch.Views.Side, ch.Views.Back))
		}
	}
	b.WriteString("\nAvailable story locations:\n")
	for _, sc := range snapshot.Scenes {
		b.WriteString(fmt.Sprintf("- key=%s id=%s title=%s location=%s time=%s description=%s\n", sc.Key, sc.ID, sc.Title, sc.Location, sc.TimeOfDay, sc.Description))
	}
	b.WriteString(`\nJSON schema:
{
  "storyboardBible": {
    "styleBible": {"artStyle":"English executable art direction","lineQuality":"","palette":"","lightingMood":"","cameraGrammar":""},
    "characters": [{"key":"char_1","id":"character id","name":"","narrativeRole":"","immutableTraits":[""],"currentState":"","turnaroundAssetKeys":["char_1"],"primaryIdentityImage":""}],
    "locations": [{"key":"loc_1","id":"story scene id","name":"","immutableTraits":[""],"currentState":""}],
    "props": [{"key":"prop_key","name":"","immutableTraits":[""],"currentState":""}],
    "continuityRules": ["identity and plot rule"]
  },
  "beats": [{"index":0,"beatId":"beat_0","purpose":"","summary":"","characters":["char_1"],"locationKey":"loc_1","referenceKeys":["char_1","loc_1"],"continuityNote":""}]
}
Hard requirements:
- beats length exactly matches requested count.
- Character entries must include turnaroundAssetKeys when three-view URLs are available.
- All immutableTraits must be visual facts, not vague personality labels.
- Beat summaries must form one continuous plot chain.`)
	return b.String()
}

func buildStoryboardSceneWriterSystemPrompt() string {
	return `You are a cinematic storyboard writer. Convert a validated bible and beat list into final storyboard content and scenes.
Do not change visual bible facts, character IDs, character keys, location keys, or beat order.
For every scene, include an English imagePrompt that can drive image generation with reference images.
Use the character three-view sheet as identity authority, but compose a new scene pose and camera according to the beat.
Output strict JSON only.`
}

func buildStoryboardSceneWriterUserPrompt(story *domain.Story, storyboard *domain.Storyboard, snapshot storyboardGenerationContextSnapshot, plan *domain.StoryboardBiblePlan, sceneCount int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Write final storyboard content and exactly %d scenes.\n", sceneCount))
	b.WriteString(fmt.Sprintf("Story title: %s\nGenre: %s\nUser raw input: %s\n\n", story.Title, story.Genre, storyboard.RawInput))
	b.WriteString("Validated bible and beats JSON:\n")
	b.WriteString(mustJSON(plan, "{}"))
	b.WriteString("\n\nParent tail scenes for continuity:\n")
	for _, sc := range snapshot.ParentTail {
		b.WriteString(fmt.Sprintf("- %s: %s image=%s\n", sc.Title, sc.Description, sc.Image))
	}
	b.WriteString(`\nOutput schema:
{
  "content": "Chinese polished storyboard summary, <= 420 Unicode chars recommended",
  "scenes": [
    {
      "sequence": 0,
      "title": "short Chinese scene title",
      "description": "Chinese visual scene description, 100-220 chars, concrete camera, light, action, expression, environment",
      "location": "place",
      "timeOfDay": "time",
      "characters": ["character display names visible in scene"],
      "mood": "mood",
      "referenceKeys": ["char_1","loc_1"],
      "continuityNote": "what must carry forward from previous scene or parent",
      "beatPurpose": "same purpose as beat",
      "imagePrompt": "English final image prompt. Include narrative, styleBible art direction, immutable character traits, location traits, camera, lighting, color, mood. Explicitly say three-view references define identity, but do not copy the turnaround pose.",
      "visualState": {"characters":"wardrobe/emotion/injuries/props after this scene"}
    }
  ]
}
Hard requirements:
- scenes length exactly matches requested count.
- referenceKeys must be declared in the bible.
- imagePrompt must be English and include identity, action, environment, composition, lighting, palette, mood, texture.
- Do not output Markdown or commentary.`)
	return b.String()
}

func validateStoryboardBiblePlan(plan *domain.StoryboardBiblePlan, sceneCount int) error {
	if plan == nil {
		return fmt.Errorf("empty storyboard bible plan")
	}
	if plan.StoryboardBible.StyleBible == nil || strings.TrimSpace(plan.StoryboardBible.StyleBible.ArtStyle) == "" {
		return fmt.Errorf("storyboard bible missing styleBible.artStyle")
	}
	if len(plan.Beats) != sceneCount {
		return fmt.Errorf("beat count mismatch: expected %d got %d", sceneCount, len(plan.Beats))
	}
	keys := storyboardBibleReferenceKeySet(&plan.StoryboardBible)
	for i, beat := range plan.Beats {
		if strings.TrimSpace(beat.Purpose) == "" || strings.TrimSpace(beat.Summary) == "" {
			return fmt.Errorf("beat %d missing purpose or summary", i)
		}
		for _, key := range beat.ReferenceKeys {
			if _, ok := keys[key]; !ok {
				return fmt.Errorf("beat %d references undeclared key %q", i, key)
			}
		}
	}
	return nil
}

func validateStoryboardScenePlan(scenePlan *domain.StoryboardScenePlan, biblePlan *domain.StoryboardBiblePlan, sceneCount int) error {
	if scenePlan == nil {
		return fmt.Errorf("empty storyboard scene plan")
	}
	if strings.TrimSpace(scenePlan.Content) == "" {
		return fmt.Errorf("storyboard scene plan missing content")
	}
	if len(scenePlan.Scenes) != sceneCount {
		return fmt.Errorf("scene count mismatch: expected %d got %d", sceneCount, len(scenePlan.Scenes))
	}
	keys := storyboardBibleReferenceKeySet(&biblePlan.StoryboardBible)
	for i, scene := range scenePlan.Scenes {
		if strings.TrimSpace(scene.Title) == "" || strings.TrimSpace(scene.Description) == "" {
			return fmt.Errorf("scene %d missing title or description", i)
		}
		if strings.TrimSpace(scene.ImagePrompt) == "" {
			return fmt.Errorf("scene %d missing imagePrompt", i)
		}
		for _, key := range scene.ReferenceKeys {
			if _, ok := keys[key]; !ok {
				return fmt.Errorf("scene %d references undeclared key %q", i, key)
			}
		}
	}
	return nil
}

func storyboardBibleReferenceKeySet(bible *domain.StoryboardVisualBible) map[string]struct{} {
	keys := make(map[string]struct{})
	if bible == nil {
		return keys
	}
	for _, ch := range bible.Characters {
		if strings.TrimSpace(ch.Key) != "" {
			keys[ch.Key] = struct{}{}
		}
	}
	for _, loc := range bible.Locations {
		if strings.TrimSpace(loc.Key) != "" {
			keys[loc.Key] = struct{}{}
		}
	}
	for _, prop := range bible.Props {
		if strings.TrimSpace(prop.Key) != "" {
			keys[prop.Key] = struct{}{}
		}
	}
	return keys
}

func (s *Service) auditStoryboardGenerationConsistency(plan *domain.StoryboardBiblePlan, scenePlan *domain.StoryboardScenePlan, assets []*domain.StoryboardGenerationAsset, sceneCount int) []domain.FragmentConsistencyIssue {
	var issues []domain.FragmentConsistencyIssue
	if plan == nil || scenePlan == nil {
		return []domain.FragmentConsistencyIssue{{Severity: "high", Detail: "missing plan or scene plan"}}
	}
	if len(plan.Beats) != sceneCount {
		issues = append(issues, domain.FragmentConsistencyIssue{Severity: "high", Detail: fmt.Sprintf("beat count mismatch: expected %d got %d", sceneCount, len(plan.Beats))})
	}
	if len(scenePlan.Scenes) != sceneCount {
		issues = append(issues, domain.FragmentConsistencyIssue{Severity: "high", Detail: fmt.Sprintf("scene count mismatch: expected %d got %d", sceneCount, len(scenePlan.Scenes))})
	}
	keys := storyboardBibleReferenceKeySet(&plan.StoryboardBible)
	assetKeys := make(map[string]struct{})
	for _, asset := range assets {
		if asset != nil {
			assetKeys[asset.AssetKey] = struct{}{}
		}
	}
	for i, scene := range scenePlan.Scenes {
		for _, key := range scene.ReferenceKeys {
			if _, ok := keys[key]; !ok {
				issues = append(issues, domain.FragmentConsistencyIssue{SceneIndex: i, Severity: "high", Detail: "scene references undeclared key: " + key})
			}
			if strings.HasPrefix(key, "char_") {
				if _, ok := assetKeys[key]; !ok {
					issues = append(issues, domain.FragmentConsistencyIssue{SceneIndex: i, Severity: "medium", Detail: "character key has no persisted three-view asset: " + key})
				}
			}
		}
		if !strings.Contains(strings.ToLower(scene.ImagePrompt), "three-view") && len(scene.ReferenceKeys) > 0 {
			issues = append(issues, domain.FragmentConsistencyIssue{SceneIndex: i, Severity: "low", Detail: "imagePrompt does not explicitly mention three-view identity authority"})
		}
	}
	return issues
}
