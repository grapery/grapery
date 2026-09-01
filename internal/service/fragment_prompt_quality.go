package service

import (
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ensureFragmentVisualBibleFallback prevents a partially valid extraction
// response from silently disabling all downstream art-direction controls.
func ensureFragmentVisualBibleFallback(req domain.FragmentGenerationRequest, result *fragmentElementExtractionResult) {
	if result == nil {
		return
	}
	if result.VisualBible == nil {
		result.VisualBible = &domain.FragmentVisualBible{}
	}
	bible := result.VisualBible
	if bible.StyleBible == nil {
		bible.StyleBible = &domain.FragmentVisualStyleBible{}
	}
	if strings.TrimSpace(bible.StyleBible.ArtStyle) == "" {
		bible.StyleBible.ArtStyle = fallbackFragmentArtStyle(req.Style)
	}
	if strings.TrimSpace(bible.StyleBible.LineQuality) == "" {
		bible.StyleBible.LineQuality = "clean intentional line work, controlled texture, readable silhouettes, detailed faces and hands"
	}
	if strings.TrimSpace(bible.StyleBible.Palette) == "" {
		bible.StyleBible.Palette = fallbackFragmentPalette(req.Mood, result.Elements.Weather, result.Elements.TimeOfDay)
	}
	if strings.TrimSpace(bible.StyleBible.LightingMood) == "" {
		bible.StyleBible.LightingMood = "motivated cinematic lighting with a clear key light, readable facial planes, atmospheric depth, and restrained highlights"
	}

	if len(bible.Characters) == 0 && len(result.Elements.Characters) > 0 {
		for i, raw := range result.Elements.Characters {
			if i >= 3 || strings.TrimSpace(raw) == "" {
				break
			}
			importance := "supporting"
			if i == 0 {
				importance = "core"
			}
			bible.Characters = append(bible.Characters, domain.FragmentVisualCharacter{
				Key:             fmt.Sprintf("char_%d", i),
				Name:            fallbackFragmentEntityName(raw, fmt.Sprintf("故事主角%d", i+1)),
				ImmutableTraits: []string{strings.TrimSpace(raw)},
				NegativeTraits:  []string{"do not hide the character in every scene; do not replace the character with a disembodied hand"},
				RoleImportance:  importance,
			})
		}
	}
	if len(bible.Props) == 0 {
		for i, raw := range result.Elements.Objects {
			if i >= 5 || strings.TrimSpace(raw) == "" {
				break
			}
			owner := ""
			if len(bible.Characters) > 0 {
				owner = bible.Characters[0].Key
			}
			importance := "supporting"
			if i == 0 {
				importance = "core"
			}
			bible.Props = append(bible.Props, domain.FragmentVisualProp{
				Key:             fmt.Sprintf("prop_%d", i),
				Name:            fallbackFragmentEntityName(raw, fmt.Sprintf("关键道具%d", i+1)),
				ImmutableTraits: []string{strings.TrimSpace(raw)},
				Ownership:       owner,
				RoleImportance:  importance,
			})
		}
	}
	if len(bible.Locations) == 0 {
		location := strings.TrimSpace(result.Elements.Location)
		if location == "" && len(result.Elements.Scenes) > 0 {
			location = strings.TrimSpace(result.Elements.Scenes[0])
		}
		if location != "" {
			bible.Locations = []domain.FragmentVisualLocation{{
				Key: "loc_0", Name: fallbackFragmentEntityName(location, "主要场景"),
				ImmutableTraits: []string{location}, RoleImportance: "core",
			}}
		}
	}
	normalizeFragmentVisualBible(&result.VisualBible)
}

func fallbackFragmentEntityName(raw, fallback string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return fallback
	}
	for _, separator := range []string{"：", ":", "，", ",", "。", ";", "；"} {
		if p := strings.Index(s, separator); p > 0 {
			s = strings.TrimSpace(s[:p])
			break
		}
	}
	return truncateRunes(s, 12)
}

func fallbackFragmentArtStyle(style string) string {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "realistic":
		return "cinematic contemporary graphic novel illustration, grounded anatomy, natural materials, subtle painterly shading, no photorealistic camera artifacts"
	case "anime":
		return "cinematic anime illustration, expressive character acting, crisp controlled line art, nuanced cel shading, richly observed backgrounds"
	case "scifi":
		return "cinematic science-fiction graphic novel illustration, precise industrial design, controlled ink lines, atmospheric painted lighting"
	case "fantasy":
		return "cinematic fantasy graphic novel illustration, elegant ink contours, painterly color, tactile materials, believable character acting"
	default:
		return "cinematic narrative illustration, expressive character acting, clean authored line work, painterly color, tactile environmental detail"
	}
}

func fallbackFragmentPalette(mood, weather, timeOfDay string) string {
	base := "restrained cinematic palette with coherent skin, wardrobe, prop, and environment colors"
	if strings.EqualFold(strings.TrimSpace(mood), "mysterious") || strings.Contains(weather, "雨") || strings.Contains(strings.ToLower(weather), "rain") {
		return base + ", cool blue-gray shadows, muted amber practical lights, small warm focal accents, wet-surface reflections"
	}
	if strings.EqualFold(strings.TrimSpace(mood), "happy") {
		return base + ", warm daylight neutrals, gentle complementary accents, lively but controlled saturation"
	}
	if strings.EqualFold(strings.TrimSpace(mood), "sad") || strings.Contains(timeOfDay, "夜") {
		return base + ", low-saturation blue and charcoal, soft warm highlights, delicate tonal separation"
	}
	return base + ", balanced warm-cool contrast and one purposeful accent color"
}

// strengthenFragmentScenePlans repairs terse/repetitive model output before it
// reaches the image provider. Close-up privilege is deliberately limited.
func strengthenFragmentScenePlans(plans []domain.FragmentScenePlan, bible *domain.FragmentVisualBible, elements fragmentStoryElements, content, aspectRatio string) []domain.FragmentScenePlan {
	out := append([]domain.FragmentScenePlan(nil), plans...)
	for i := range out {
		profile := fragmentShotProfile(i, len(out))
		refs := normalizeFragmentKeyList(out[i].ReferenceKeys)
		if bible != nil {
			if len(bible.Characters) > 0 {
				refs = append(refs, bible.Characters[0].Key)
			}
			if len(bible.Locations) > 0 {
				refs = append(refs, bible.Locations[0].Key)
			}
			for p := 0; p < len(bible.Props) && p < 2; p++ {
				refs = append(refs, bible.Props[p].Key)
			}
		}
		out[i].ReferenceKeys = normalizeFragmentKeyList(refs)

		original := strings.TrimSpace(out[i].ImagePrompt)
		if original == "" {
			original = strings.TrimSpace(out[i].SceneDesc)
		}
		ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
		if ar == "" {
			ar = domain.FragmentAspectDefault
		}
		out[i].ImagePrompt = fmt.Sprintf(`%s. Narrative must-show: %s. Subject staging: %s. Environment: %s. Camera and composition: %s; build clear foreground, midground, and background separation for a %s canvas. Lighting and palette: %s; %s. Continuity: preserve the same protagonist identity, face, hair, wardrobe, key props, weather, architecture, and color script established by the visual bible. Quality controls: anatomically coherent hands, readable facial expression and body language, natural prop contact, intentional negative space, no duplicated composition from adjacent images, no disembodied hand as the primary subject, no random letters, no logo, no watermark.`,
			original, strings.TrimSpace(out[i].SceneDesc), profile.subject,
			fragmentEnvironmentBrief(elements), profile.camera, ar,
			fragmentLightingBrief(elements), profile.rhythm,
		)
		if strings.TrimSpace(out[i].SceneDesc) == "" {
			out[i].SceneDesc = fmt.Sprintf("第%d格：%s", i+1, truncateRunes(content, 64))
		}
	}
	return out
}

type fragmentShotDirection struct{ subject, camera, rhythm string }

func fragmentShotProfile(index, total int) fragmentShotDirection {
	profiles := []fragmentShotDirection{
		{"show the protagonist's recognizable face or three-quarter profile and at least three quarters of the body; the key prop is secondary to the person and situation", "medium-wide establishing shot at eye level, protagonist on a rule-of-thirds point, environment and route of movement clearly readable", "an establishing image with breathing room and a strong silhouette"},
		{"keep the protagonist's head, shoulders, hands, and the discovered prop in one spatially coherent frame; show the emotional change through posture and gaze", "over-the-shoulder medium shot with a new viewing axis and strong depth cues, never repeat the previous framing", "a discovery beat that moves the eye from the character to the evidence"},
		{"show the protagonist's face and upper body reacting while interacting naturally with the prop; hands remain attached and anatomically plausible", "medium close reaction shot from a three-quarter angle, face is the focal point and the prop occupies a smaller secondary plane", "the emotional turn, intimate but not an isolated object insert"},
		{"show the protagonist and the consequence together; reveal how the final clue changes their relationship to the location", "wide or medium-wide payoff shot from a fresh angle, character and final clue connected by lighting and leading lines", "a concluding reveal with visual payoff rather than another close-up"},
	}
	if total == 1 {
		return fragmentShotDirection{"show the protagonist's recognizable face, full pose, key action, and story-defining prop together", "medium-wide hero frame with layered environment and one clear focal hierarchy", "one self-contained narrative image with setup, tension, and clue visible at once"}
	}
	return profiles[index%len(profiles)]
}

func fragmentEnvironmentBrief(elements fragmentStoryElements) string {
	parts := compactNonEmptyStrings([]string{elements.Location, strings.Join(elements.Scenes, "; "), elements.Weather, elements.TimeOfDay})
	if len(parts) == 0 {
		return "a specific lived-in location with tactile materials, believable scale, and story-relevant foreground details"
	}
	return strings.Join(parts, "; ")
}

func fragmentLightingBrief(elements fragmentStoryElements) string {
	parts := compactNonEmptyStrings([]string{elements.TimeOfDay, elements.Weather})
	if len(parts) == 0 {
		return "motivated cinematic key light, readable face, controlled highlights, atmospheric depth"
	}
	return strings.Join(parts, "; ") + ", with motivated cinematic key light, readable facial planes, controlled highlights, and atmospheric depth"
}
