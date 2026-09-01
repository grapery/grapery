package service

import (
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// applyStoryboardScenePlanFallbacks fills missing layout metadata and sanitizes referenceKeys before validate.
func applyStoryboardScenePlanFallbacks(scenePlan *domain.StoryboardScenePlan, biblePlan *domain.StoryboardBiblePlan) {
	if scenePlan == nil || biblePlan == nil {
		return
	}
	keySet := storyboardBibleReferenceKeySet(&biblePlan.StoryboardBible)
	for i := range scenePlan.Scenes {
		sc := &scenePlan.Scenes[i]
		var filtered []string
		for _, k := range sc.ReferenceKeys {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if _, ok := keySet[k]; ok {
				filtered = append(filtered, k)
			}
		}
		sc.ReferenceKeys = filtered
		if len(sc.ReferenceKeys) == 0 && i < len(biblePlan.Beats) {
			for _, k := range biblePlan.Beats[i].ReferenceKeys {
				k = strings.TrimSpace(k)
				if _, ok := keySet[k]; ok {
					sc.ReferenceKeys = append(sc.ReferenceKeys, k)
				}
			}
		}
		if strings.TrimSpace(sc.PanelShape) == "" {
			sc.PanelShape = "full"
		}
		if strings.TrimSpace(sc.LayoutIntent) == "" {
			sc.LayoutIntent = "comic_single_panel"
		}
		if strings.TrimSpace(sc.CompositionPlan) == "" {
			sc.CompositionPlan = "single vertical panel, centered subject, clean negative space for any supplied lettering"
		}
		if strings.TrimSpace(sc.ShotType) == "" {
			sc.ShotType = "medium_shot"
		}
		if strings.TrimSpace(sc.VisualHierarchy) == "" {
			sc.VisualHierarchy = "character_primary_environment_secondary"
		}
	}
}

// writeStoryboardTurnDirectives appends the two conversational inputs that the
// single RawInput string cannot express: this turn's revision instruction and the
// style the user picked on the planning card.
func writeStoryboardTurnDirectives(b *strings.Builder, storyboard *domain.Storyboard) {
	if directive := strings.TrimSpace(storyboard.TurnDirective); directive != "" {
		b.WriteString("\n\n【本轮修改要求（最高优先级）】\n")
		b.WriteString(directive)
		b.WriteString("\n约束：只改动本轮要求指向的部分；未被提及的角色设定、已确定情节、视觉基线保持不变，不要整体重写。\n")
	}
	if style := strings.TrimSpace(storyboard.ContinuationComicStyle); style != "" {
		b.WriteString("\n\n【视觉风格优先级】\n")
		b.WriteString(fmt.Sprintf("1. 用户本次选择的漫画风格 slug：%s（最高优先）\n", style))
		b.WriteString("2. 故事自身的 style 配置\n")
		b.WriteString("3. 题材默认风格\n")
	}
}

func buildStoryboardBiblePlanSystemPrompt() string {
	return `You are a senior storyboard continuity planner. Produce strict JSON only.
Your job is to create a storyboard visual bible and narrative beats before scene writing.
Rules:
- Preserve character identity and plot continuity over novelty.
- Character three-view sheets are identity authority for face, body, hairstyle, costume proportions, color blocking, and silhouette.
- Do not invent IDs. Use provided character IDs and location keys when available.
- Every beat referenceKeys value must point to a key declared in storyboardBible.characters, locations, or props.
- Plan in manga/comic language from the start: dialogue, inner monologue, interjections/SFX, emotional turning points, shock beats, anticipation beats, and celebration/release beats must be explicit beat functions when the story supports them.
- Do not make every beat a scenic illustration. A good comic sequence alternates establishing, action/reaction, dialogue/thought, emotional punctuation, and transition.
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
	writeStoryboardTurnDirectives(&b, storyboard)
	b.WriteString(`\nJSON schema:
{
  "storyboardBible": {
    "styleBible": {"artStyle":"English executable art direction","lineQuality":"","palette":"","lightingMood":"","cameraGrammar":""},
    "characters": [{"key":"char_1","id":"character id","name":"","narrativeRole":"","immutableTraits":[""],"currentState":"","turnaroundAssetKeys":["char_1"],"primaryIdentityImage":""}],
    "locations": [{"key":"loc_1","id":"story scene id","name":"","immutableTraits":[""],"currentState":""}],
    "props": [{"key":"prop_key","name":"","immutableTraits":[""],"currentState":""}],
    "continuityRules": ["identity and plot rule"]
  },
  "beats": [{"index":0,"beatId":"beat_0","purpose":"","summary":"","comicFunction":"establish","layoutHint":"wide_establishing + negative_space_tension","characters":["char_1"],"locationKey":"loc_1","referenceKeys":["char_1","loc_1"],"continuityNote":""}]
}
Hard requirements:
- beats length exactly matches requested count.
- Character entries must include turnaroundAssetKeys when three-view URLs are available.
- All immutableTraits must be visual facts, not vague personality labels.
- Beat summaries must form one continuous plot chain.
- For a continuation, continuityRules must preserve the parent ending and the first beat continuityNote must explain the handoff from that ending.
- Every beat must include comicFunction and a concise layoutHint so scene writing starts with comic grammar, not post-image patching.
- Allowed comicFunction values: establish|dialogue|inner_monologue|action_impact|reaction|turning_point|shock|anticipation|celebration|transition|atmosphere.
- Use comicFunction intentionally:
  * dialogue: character speech is narratively useful; later scene must need speech balloons.
  * inner_monologue: private thought/psychological hesitation; later scene must need thought bubbles.
  * shock: surprise, fear, revelation, sudden reversal; later scene should use close-up, reaction marks, SFX/interjection.
  * anticipation: before reveal/action/decision; later scene should use silence, negative space, caption or thought bubble.
  * celebration: relief, success, reunion, reward; later scene may use warm light, crowd reaction, short cheering text.
  * turning_point: main plot direction changes; later scene must visually emphasize the irreversible moment.
- For 4+ beats, include at least one non-atmospheric emotional punctuation beat (turning_point|shock|anticipation|celebration|inner_monologue|dialogue) when the input story supports it.
- layoutHint must mention concrete comic grammar, such as speech_balloon_safe_space, thought_bubble_close_up, jagged_shock_burst, cliffhanger_negative_space, celebration_wide_panel, border_breaking_impact, caption_box_transition.`)
	return b.String()
}

func buildStoryboardSceneWriterSystemPrompt() string {
	return `You are a cinematic storyboard writer and manga/comic panel director. Convert a validated bible and beat list into final storyboard content and scenes.
Do not change visual bible facts, character IDs, character keys, location keys, or beat order.
For every scene, include an English imagePrompt that can drive image generation with reference images.
Use the character three-view sheet as identity authority, but compose a new scene pose and camera according to the beat.
When the story style or beat requires it, populate comicTexts with on-image lettering (dialogue bubbles, thought bubbles, narration boxes, SFX/interjections) so the image model paints them directly — no app-side overlay.
Emotional manga beats must not become generic illustrations: turning_point, shock, anticipation, celebration, dialogue, and inner_monologue require visible comic staging and appropriate short text unless deliberately wordless for stronger effect.
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
	writeStoryboardTurnDirectives(&b, storyboard)
	b.WriteString(`\nOutput schema:
{
  "content": "polished storyboard summary in the requested content language, <= 420 Unicode chars recommended",
  "scenes": [
    {
      "sequence": 0,
      "title": "short title in the requested content language",
      "description": "visual scene description in the requested content language, 100-220 Unicode chars, concrete camera, light, action, expression, environment",
      "location": "place",
      "timeOfDay": "time",
      "characters": ["character display names visible in scene"],
      "mood": "mood",
      "referenceKeys": ["char_1","loc_1"],
      "continuityNote": "what must carry forward from previous scene or parent",
      "beatPurpose": "same purpose as beat",
      "imagePrompt": "English final image prompt. Include narrative, styleBible art direction, immutable character traits, location traits, camera, lighting, color, mood. Explicitly say three-view references define identity, but do not copy the turnaround pose. For manga-emphasis beats, include concrete staging: speech/thought balloon placement, tail direction, SFX/interjection typography, reaction marks, effect lines, caption boxes, and negative space reserved for lettering — so the image model renders them directly in-image.",
      "visualState": {"characters":"wardrobe/emotion/injuries/props after this scene"},
      "layoutIntent": "short snake_case layout intent such as comic_single_panel, split_screen_two_beat, diagonal_motion, detail_insert",
      "compositionPlan": "concise layout plan in the requested content language: panel/zones, gutters, reading order, lettering safe-space, focal flow",
      "shotType": "English shot type such as close_up, medium_shot, wide_shot, dutch_angle, overhead",
      "visualHierarchy": "what is primary, secondary, background information in this scene image",
      "panelShape": "one of: full | diagonal_left | diagonal_right | trapezoid_leading | trapezoid_trailing | triangle_tl | triangle_tr | triangle_bl | triangle_br | wide_panorama",
      "comicTexts": [
        {"type":"narration","text":"short exact narration in the requested language","speaker":"","position":"top-left"},
        {"type":"dialogue","text":"short exact dialogue","speaker":"char_1","position":"speech-bubble"},
        {"type":"thought","text":"short exact inner monologue","speaker":"char_1","position":"thought-bubble"},
        {"type":"sfx","text":"short exact SFX","speaker":"","position":"mid-frame"}
      ]
    }
  ]
}
Hard requirements:
- scenes length exactly matches requested count.
- For a continuation, the first scene continuityNote must explicitly explain how it follows the parent ending without resetting character state.
- referenceKeys must be declared in the bible.
- imagePrompt must be English and include identity, action, environment, composition, lighting, palette, mood, texture.
- layoutIntent/compositionPlan/shotType/visualHierarchy/panelShape are required for every scene and must be consistent with beat comicFunction/layoutHint.
- panelShape encodes the clipping shape used when the app assembles a multi-panel collage cover. Choose based on dramatic content:
  * full          — establish, atmosphere, anticipation, celebration, inner_monologue (contained or wide single-image beats)
  * wide_panorama — establish with sweeping landscape; crowd scenes; transition bookends
  * diagonal_left — action_impact, turning_point where energy flows top-right → bottom-left; the character/focal point sits in the LEFT portion
  * diagonal_right — action_impact, turning_point where energy flows top-left → bottom-right; the character/focal point sits in the RIGHT portion
  * trapezoid_leading — dialogue or reaction close-up: leading-edge trapezoid, left panel in a 2- or 3-panel row
  * trapezoid_trailing — dialogue or reaction close-up: trailing-edge trapezoid, right panel in a 2- or 3-panel row
  * triangle_tl   — shock, surprise reveal: triangular crop top-left (jagged energy)
  * triangle_tr   — shock, surprise reveal: triangular crop top-right
  * triangle_bl   — transition, fading out: triangular crop bottom-left
  * triangle_br   — transition, fading out: triangular crop bottom-right
- For a storyboard with N scenes, ensure the panelShape sequence creates a visually balanced collage (avoid N consecutive "full"; mix at least one non-rectangular shape per 3 scenes when the story supports it).
- comicTexts is the sole authority for visible lettering. It may be omitted or empty for deliberately wordless scenes. When used: preserve the requested content language; keep CJK text <= 12 characters and English text <= 32 characters; speaker must be a character name from the scene; per-panel cap ~1 narration, 1-2 dialogue, 1 sfx, 1 thought; type must be one of narration|dialogue|thought|sfx.
- For beats whose comicFunction is dialogue, inner_monologue, shock, anticipation, celebration, or turning_point, either provide comicTexts OR explicitly make imagePrompt describe a deliberate wordless comic device (large silence, empty balloon avoided, held breath, reaction-only close-up). Do not leave these as plain scenic descriptions.
- Do not invent filler dialogue, captions, signs, labels, or pseudo-text. If comicTexts is empty, imagePrompt must explicitly forbid all readable text and empty bubble/box outlines.
- imagePrompt must mirror any comicTexts as concrete English lettering instructions (exact supplied glyphs, balloon shape, tail direction, font weight, reserved white space). If exact lettering is unreliable, omit both glyphs and empty outlines rather than substituting garbled text.
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
		if strings.TrimSpace(beat.ComicFunction) == "" || strings.TrimSpace(beat.LayoutHint) == "" {
			return fmt.Errorf("beat %d missing comicFunction or layoutHint", i)
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
		if strings.TrimSpace(scene.LayoutIntent) == "" || strings.TrimSpace(scene.CompositionPlan) == "" || strings.TrimSpace(scene.ShotType) == "" || strings.TrimSpace(scene.VisualHierarchy) == "" {
			return fmt.Errorf("scene %d missing comic layout metadata", i)
		}
		if strings.TrimSpace(scene.PanelShape) == "" {
			return fmt.Errorf("scene %d missing panelShape", i)
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

func (s *Service) auditStoryboardGenerationConsistency(plan *domain.StoryboardBiblePlan, scenePlan *domain.StoryboardScenePlan, assets []*domain.StoryboardGenerationAsset, sceneCount int, continuation bool) []domain.FragmentConsistencyIssue {
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
	if continuation {
		if len(plan.StoryboardBible.ContinuityRules) == 0 {
			issues = append(issues, domain.FragmentConsistencyIssue{Severity: "high", Detail: "continuation storyboard has no inherited continuity rules"})
		}
		if len(plan.Beats) > 0 && strings.TrimSpace(plan.Beats[0].ContinuityNote) == "" {
			issues = append(issues, domain.FragmentConsistencyIssue{SceneIndex: 0, Severity: "high", Detail: "first continuation beat does not explain how it connects to the parent ending"})
		}
		if len(scenePlan.Scenes) > 0 && strings.TrimSpace(scenePlan.Scenes[0].ContinuityNote) == "" {
			issues = append(issues, domain.FragmentConsistencyIssue{SceneIndex: 0, Severity: "high", Detail: "first continuation scene has no parent-ending continuity note"})
		}
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

func firstHighStoryboardConsistencyIssue(issues []domain.FragmentConsistencyIssue) string {
	for _, issue := range issues {
		if issue.Severity == "high" {
			return issue.Detail
		}
	}
	return ""
}
