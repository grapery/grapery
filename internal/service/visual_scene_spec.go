package service

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

const visualSceneContractVersion = "visual_scene_v2"

// visualSceneSpec is the provider-neutral contract between narrative planning
// and image-prompt compilation. Story decisions belong here; provider wording
// is derived from it and must not invent new story facts.
type visualSceneSpec struct {
	ContentLanguage   string
	LetteringLanguage string
	Title             string
	NarrativeBeat     string
	VisualPrompt      string
	ContinuityNote    string
	ArtStyle          string
	LayoutIntent      string
	CompositionPlan   string
	ShotType          string
	VisualHierarchy   string
	IsTransition      bool
	References        []domain.StoryboardImageReference
	ComicTexts        []visualComicText
}

type visualComicText struct {
	Type     string
	Text     string
	Speaker  string
	Position string
}

func normalizeGenerationLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "en", "en-us", "en-gb", "english":
		return "en"
	case "ja", "ja-jp", "japanese", "日本語":
		return "ja"
	default:
		return "zh-Hans"
	}
}

func inferGenerationLanguage(text string) string {
	for _, r := range text {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana) {
			return "ja"
		}
	}
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return "zh-Hans"
		}
	}
	return "en"
}

func generationLanguageName(language string) string {
	switch normalizeGenerationLanguage(language) {
	case "en":
		return "English"
	case "ja":
		return "Japanese"
	default:
		return "Simplified Chinese"
	}
}

func comicTextRuneLimit(language string) int {
	if normalizeGenerationLanguage(language) == "en" {
		return 32
	}
	return 12
}

func visualSceneLetteringPolicy(language string, texts []visualComicText) string {
	langName := generationLanguageName(language)
	if len(texts) == 0 {
		return "Lettering policy: this scene is wordless. Do not draw speech balloons, thought bubbles, caption boxes, SFX lettering, signs, logos, or any other readable or pseudo-readable text."
	}
	return fmt.Sprintf("Lettering policy: render only the supplied %s text. Never invent, translate, paraphrase, or add text. Attempt the exact glyphs; if exact lettering is unreliable, omit both the glyphs and the empty bubble/box outline while preserving clean negative space. Never draw approximate, garbled, random, or substitute text.", langName)
}

func storyboardComicTextsToVisual(texts []domain.StoryboardComicText, language string) []visualComicText {
	texts = normalizeStoryboardComicTextsForLanguage(texts, language)
	out := make([]visualComicText, 0, len(texts))
	for _, item := range texts {
		out = append(out, visualComicText{Type: item.Type, Text: item.Text, Speaker: item.Speaker, Position: item.Position})
	}
	return out
}

func fragmentComicTextsToVisual(texts []domain.FragmentComicText, language string) []visualComicText {
	texts = normalizeFragmentComicTextsForLanguage(texts, language)
	out := make([]visualComicText, 0, len(texts))
	for _, item := range texts {
		out = append(out, visualComicText{Type: item.Type, Text: item.Text, Speaker: item.Speaker, Position: item.Position})
	}
	return out
}

func storyboardVisualSceneSpec(gen *domain.StoryboardImageGeneration) visualSceneSpec {
	if gen == nil {
		return visualSceneSpec{}
	}
	language := normalizeGenerationLanguage(gen.ContentLanguage)
	if strings.TrimSpace(gen.ContentLanguage) == "" {
		language = inferGenerationLanguage(gen.SceneTitle + "\n" + gen.SceneDescription)
	}
	spec := visualSceneSpec{
		ContentLanguage:   language,
		LetteringLanguage: language,
		Title:             strings.TrimSpace(gen.SceneTitle),
		NarrativeBeat:     strings.TrimSpace(gen.SceneDescription),
		IsTransition:      gen.IsTransitionScene,
		References:        append([]domain.StoryboardImageReference(nil), gen.ReferenceManifest...),
	}
	if gen.StoryStyle != nil {
		spec.ArtStyle = strings.TrimSpace(gen.StoryStyle.Description)
		if spec.ArtStyle == "" {
			spec.ArtStyle = strings.TrimSpace(gen.StoryStyle.Style)
		}
	}
	if planned := gen.PlannedScene; planned != nil {
		spec.VisualPrompt = strings.TrimSpace(planned.ImagePrompt)
		spec.ContinuityNote = strings.TrimSpace(planned.ContinuityNote)
		spec.LayoutIntent = strings.TrimSpace(planned.LayoutIntent)
		spec.CompositionPlan = strings.TrimSpace(planned.CompositionPlan)
		spec.ShotType = strings.TrimSpace(planned.ShotType)
		spec.VisualHierarchy = strings.TrimSpace(planned.VisualHierarchy)
		spec.ComicTexts = storyboardComicTextsToVisual(planned.ComicTexts, language)
	}
	return spec
}

func visualSceneSpecPromptInput(spec visualSceneSpec) map[string]any {
	refs := make([]map[string]string, 0, len(spec.References))
	for _, ref := range spec.References {
		if strings.TrimSpace(ref.URL) == "" {
			continue
		}
		refs = append(refs, map[string]string{"role": ref.Role, "key": ref.Key, "url": ref.URL})
	}
	texts := make([]map[string]string, 0, len(spec.ComicTexts))
	for _, item := range spec.ComicTexts {
		texts = append(texts, map[string]string{"type": item.Type, "text": item.Text, "speaker": item.Speaker, "position": item.Position})
	}
	return map[string]any{
		"contentLanguage": spec.ContentLanguage, "letteringLanguage": spec.LetteringLanguage,
		"title": spec.Title, "narrativeBeat": spec.NarrativeBeat, "plannedVisualPrompt": spec.VisualPrompt,
		"continuityNote": spec.ContinuityNote, "artStyle": spec.ArtStyle,
		"layoutIntent": spec.LayoutIntent, "compositionPlan": spec.CompositionPlan,
		"shotType": spec.ShotType, "visualHierarchy": spec.VisualHierarchy,
		"isTransitionScene": spec.IsTransition, "references": refs, "comicTexts": texts,
	}
}

func mergeStoryboardPlannedSceneForImage(description string, planned *domain.StoryboardScene) string {
	description = strings.TrimSpace(description)
	if planned == nil {
		return description
	}
	var sections []string
	add := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" || strings.Contains(description, value) {
			return
		}
		sections = append(sections, label+"\n"+value)
	}
	add("[Planned visual execution — authoritative]", planned.ImagePrompt)
	add("[Continuity requirement]", planned.ContinuityNote)
	var layoutParts []string
	for _, item := range []struct{ key, value string }{
		{"layout_intent", planned.LayoutIntent},
		{"shot_type", planned.ShotType},
		{"composition_plan", planned.CompositionPlan},
		{"visual_hierarchy", planned.VisualHierarchy},
	} {
		if value := strings.TrimSpace(item.value); value != "" {
			layoutParts = append(layoutParts, item.key+"="+value)
		}
	}
	if layout := strings.Join(layoutParts, "; "); layout != "" {
		add("[Planned layout — authoritative]", layout)
	}
	if len(sections) == 0 {
		return description
	}
	if description == "" {
		return strings.Join(sections, "\n\n")
	}
	return description + "\n\n" + strings.Join(sections, "\n\n")
}

func appendStoryboardImageReference(refs []domain.StoryboardImageReference, seen map[string]struct{}, url, role, key string, max int) []domain.StoryboardImageReference {
	url = strings.TrimSpace(url)
	if url == "" || len(refs) >= max {
		return refs
	}
	if _, ok := seen[url]; ok {
		return refs
	}
	seen[url] = struct{}{}
	return append(refs, domain.StoryboardImageReference{URL: url, Role: role, Key: strings.TrimSpace(key)})
}

func storyboardReferenceURLs(refs []domain.StoryboardImageReference) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if url := strings.TrimSpace(ref.URL); url != "" {
			out = append(out, url)
		}
	}
	return out
}

func (s *Service) storyboardSceneContentLanguage(ctx context.Context, planned *domain.StoryboardScene, fallbackText string) string {
	if s != nil && s.repo != nil && planned != nil && strings.TrimSpace(planned.GenerationRunID) != "" {
		if run, err := s.repo.GetStoryboardGenerationRun(ctx, planned.GenerationRunID); err == nil && run != nil {
			return normalizeGenerationLanguage(storyboardWorkflowOutputLanguage(run))
		}
	}
	return inferGenerationLanguage(fallbackText)
}

func compileStoryboardImagePromptFromPlan(svc *Service, gen *domain.StoryboardImageGeneration) bool {
	if svc == nil || gen == nil || gen.PlannedScene == nil || strings.TrimSpace(gen.PlannedScene.ImagePrompt) == "" {
		return false
	}
	planned := gen.PlannedScene
	var composition []string
	for _, item := range []string{planned.ShotType, planned.LayoutIntent, planned.CompositionPlan, planned.VisualHierarchy} {
		if item = strings.TrimSpace(item); item != "" {
			composition = append(composition, item)
		}
	}
	details := &domain.ImagePromptDetails{
		Composition:     strings.Join(composition, "; "),
		KeyElements:     []string{strings.TrimSpace(planned.ImagePrompt)},
		Mood:            strings.TrimSpace(planned.Mood),
		AdditionalNotes: strings.TrimSpace(planned.ContinuityNote),
		ComicTexts:      normalizeStoryboardComicTextsForLanguage(planned.ComicTexts, gen.ContentLanguage),
	}
	if gen.StoryStyle != nil {
		details.ArtStyle = strings.TrimSpace(gen.StoryStyle.Description)
		if details.ArtStyle == "" {
			details.ArtStyle = strings.TrimSpace(gen.StoryStyle.Style)
		}
	}
	gen.PromptDetails = details
	gen.GeneratedPrompt = svc.combineImagePrompt(details, gen.SceneTitle, gen.SceneDescription, gen.ContentLanguage)
	return strings.TrimSpace(gen.GeneratedPrompt) != ""
}
