package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

const fragmentComicDocumentSchemaVersion = 2

func syncFragmentComicDocumentPageAssets(document *domain.FragmentComicDocument, scenes []domain.FragmentScenePlan, offset int) {
	if document == nil {
		return
	}
	for i := range document.Pages {
		page := &document.Pages[i]
		sceneIndex := page.PageIndex - offset
		if sceneIndex < 0 || sceneIndex >= len(scenes) {
			continue
		}
		scene := scenes[sceneIndex]
		page.PanelImageURLs = append([]string(nil), scene.PanelImageURLs...)
		page.BackgroundImageURL = strings.TrimSpace(scene.GeneratedImageURL)
		page.FlattenedImageURL = page.BackgroundImageURL
		page.Status = "planned"
		if page.BackgroundImageURL == "" {
			continue
		}
		page.Status = "rendered"
		for _, item := range page.TextLayers {
			if item.RenderMode == "overlay" {
				page.Status = "rendered_editable"
				page.FlattenedImageURL = ""
				break
			}
		}
	}
}

func buildFragmentComicDocument(
	req domain.FragmentGenerationRequest,
	fragmentID, content, aspectRatio string,
	bible *domain.FragmentVisualBible,
	evidence []domain.FragmentVisualEvidence,
	scenes []domain.FragmentScenePlan,
	assets []domain.FragmentReferenceAsset,
) *domain.FragmentComicDocument {
	doc := &domain.FragmentComicDocument{
		SchemaVersion: fragmentComicDocumentSchemaVersion,
		Revision:      1,
		FragmentID:    strings.TrimSpace(fragmentID),
		CreativeContext: domain.FragmentCreativeContext{
			UserText: strings.TrimSpace(req.UserInput),
			Language: normalizeGenerationLanguage(req.Language),
			Style:    strings.TrimSpace(req.Style),
			Mood:     strings.TrimSpace(req.Mood),
			Inputs:   fragmentComicDocumentInputs(req, evidence),
			Facts:    fragmentComicDocumentFacts(req, content, evidence),
		},
		VisualBible:     bible,
		StoryState:      initialFragmentStoryState(bible, scenes),
		ReferenceAssets: append([]domain.FragmentReferenceAsset(nil), assets...),
		Pages:           make([]domain.FragmentComicPageDocument, 0, len(scenes)),
	}
	for i := range scenes {
		if scenes[i].ComicPage == nil {
			continue
		}
		plan := *scenes[i].ComicPage
		plan.Panels = cloneFragmentComicPanels(plan.Panels)
		if plan.Layout == nil {
			layout := fragmentComicLayoutForPanelCount(len(plan.Panels), aspectRatio)
			plan.Layout = &layout
		}
		texts := make([]domain.FragmentComicText, 0)
		hasOverlayText := false
		for panelIndex := range plan.Panels {
			for textIndex := range plan.Panels[panelIndex].ComicTexts {
				item := plan.Panels[panelIndex].ComicTexts[textIndex]
				if strings.TrimSpace(item.ID) == "" {
					item.ID = fmt.Sprintf("page-%d-panel-%d-text-%d", i, panelIndex, textIndex)
				}
				if strings.TrimSpace(item.RenderMode) == "" {
					item.RenderMode = "overlay"
				}
				plan.Panels[panelIndex].ComicTexts[textIndex] = item
				texts = append(texts, item)
				if item.RenderMode == "overlay" {
					hasOverlayText = true
				}
			}
		}
		status := "planned"
		if strings.TrimSpace(scenes[i].GeneratedImageURL) != "" {
			status = "rendered"
		}
		flattenedURL := strings.TrimSpace(scenes[i].GeneratedImageURL)
		if hasOverlayText {
			flattenedURL = ""
			if status == "rendered" {
				status = "rendered_editable"
			}
		}
		doc.Pages = append(doc.Pages, domain.FragmentComicPageDocument{
			PageIndex:          i,
			PageIntent:         strings.TrimSpace(scenes[i].SceneDesc),
			Plan:               plan,
			TextLayers:         texts,
			PanelImageURLs:     append([]string(nil), scenes[i].PanelImageURLs...),
			BackgroundImageURL: strings.TrimSpace(scenes[i].GeneratedImageURL),
			FlattenedImageURL:  flattenedURL,
			Status:             status,
		})
	}
	return doc
}

func fragmentComicDocumentInputs(req domain.FragmentGenerationRequest, evidence []domain.FragmentVisualEvidence) []domain.FragmentInputAsset {
	byURL := map[string]domain.FragmentInputAsset{}
	for _, slot := range req.ReferenceSlots {
		url := strings.TrimSpace(slot.ImageURL)
		if url == "" {
			continue
		}
		byURL[url] = domain.FragmentInputAsset{URL: url, Role: fragmentInputRole(slot.Kind), ReferenceKey: strings.TrimSpace(slot.Key), UserDeclared: true, Confidence: 1}
	}
	for _, raw := range req.ImageUrls {
		url := strings.TrimSpace(raw)
		if url == "" {
			continue
		}
		if _, ok := byURL[url]; !ok {
			byURL[url] = domain.FragmentInputAsset{URL: url, Role: "general_inspiration"}
		}
	}
	for _, item := range evidence {
		url := strings.TrimSpace(item.ImageURL)
		if url == "" {
			continue
		}
		asset := byURL[url]
		asset.URL = url
		if asset.Role == "" || asset.Role == "general_inspiration" {
			asset.Role = inferFragmentInputRole(item)
			asset.Confidence = item.Confidence
		}
		byURL[url] = asset
	}
	out := make([]domain.FragmentInputAsset, 0, len(byURL))
	for _, raw := range append(fragmentGenerationReferenceImageURLs(req), req.ImageUrls...) {
		url := strings.TrimSpace(raw)
		if item, ok := byURL[url]; ok {
			out = append(out, item)
			delete(byURL, url)
		}
	}
	remaining := make([]string, 0, len(byURL))
	for url := range byURL {
		remaining = append(remaining, url)
	}
	sort.Strings(remaining)
	for _, url := range remaining {
		out = append(out, byURL[url])
	}
	return out
}

func mergeFragmentComicDocuments(previous, current *domain.FragmentComicDocument, pageOffset int) *domain.FragmentComicDocument {
	if previous == nil {
		return current
	}
	if current == nil {
		copy := *previous
		return &copy
	}
	merged := *current
	merged.Revision = previous.Revision + 1
	if merged.Revision < 2 {
		merged.Revision = 2
	}
	if merged.FragmentID == "" {
		merged.FragmentID = previous.FragmentID
	}
	merged.CreativeContext.Inputs = mergeFragmentInputAssets(previous.CreativeContext.Inputs, current.CreativeContext.Inputs)
	merged.CreativeContext.Facts = mergeFragmentCreativeFacts(previous.CreativeContext.Facts, current.CreativeContext.Facts)
	pages := append([]domain.FragmentComicPageDocument(nil), previous.Pages...)
	for _, page := range pages {
		if pageOffset <= page.PageIndex {
			pageOffset = page.PageIndex + 1
		}
	}
	for _, page := range current.Pages {
		page.Plan.Panels = cloneFragmentComicPanels(page.Plan.Panels)
		page.PageIndex += pageOffset
		for panelIndex := range page.Plan.Panels {
			for textIndex := range page.Plan.Panels[panelIndex].ComicTexts {
				item := &page.Plan.Panels[panelIndex].ComicTexts[textIndex]
				item.ID = fmt.Sprintf("page-%d-panel-%d-text-%d", page.PageIndex, panelIndex, textIndex)
			}
		}
		page.TextLayers = nil
		for _, panel := range page.Plan.Panels {
			page.TextLayers = append(page.TextLayers, panel.ComicTexts...)
		}
		pages = append(pages, page)
	}
	merged.Pages = pages
	merged.ReferenceAssets = mergeFragmentReferenceAssets(previous.ReferenceAssets, current.ReferenceAssets)
	return &merged
}

func applyFragmentComicDocumentEdit(previous, current *domain.FragmentComicDocument, pageOffset, replaceImageIndex int) *domain.FragmentComicDocument {
	if replaceImageIndex <= 0 || previous == nil || current == nil || len(current.Pages) == 0 {
		return mergeFragmentComicDocuments(previous, current, pageOffset)
	}
	merged := *previous
	merged.Revision = previous.Revision + 1
	merged.CreativeContext.Inputs = mergeFragmentInputAssets(previous.CreativeContext.Inputs, current.CreativeContext.Inputs)
	merged.CreativeContext.Facts = mergeFragmentCreativeFacts(previous.CreativeContext.Facts, current.CreativeContext.Facts)
	merged.VisualBible = current.VisualBible
	merged.StoryState = current.StoryState
	merged.ReferenceAssets = mergeFragmentReferenceAssets(previous.ReferenceAssets, current.ReferenceAssets)
	merged.Pages = append([]domain.FragmentComicPageDocument(nil), previous.Pages...)
	index := replaceImageIndex - 1
	page := current.Pages[0]
	page.Plan.Panels = cloneFragmentComicPanels(page.Plan.Panels)
	page.PageIndex = index
	for panelIndex := range page.Plan.Panels {
		for textIndex := range page.Plan.Panels[panelIndex].ComicTexts {
			page.Plan.Panels[panelIndex].ComicTexts[textIndex].ID = fmt.Sprintf("page-%d-panel-%d-text-%d", index, panelIndex, textIndex)
		}
	}
	page.TextLayers = nil
	for _, panel := range page.Plan.Panels {
		page.TextLayers = append(page.TextLayers, panel.ComicTexts...)
	}
	for i := range merged.Pages {
		if merged.Pages[i].PageIndex == index {
			merged.Pages[i] = page
			return &merged
		}
	}
	merged.Pages = append(merged.Pages, page)
	sort.Slice(merged.Pages, func(i, j int) bool { return merged.Pages[i].PageIndex < merged.Pages[j].PageIndex })
	return &merged
}

func fragmentComicDocumentFromExistingDraft(
	request domain.FragmentGenerationRequest,
	fragmentID, content, aspectRatio string,
	imageURLs []string,
	trace *domain.FragmentGenerationTrace,
) *domain.FragmentComicDocument {
	if trace == nil {
		return nil
	}
	if trace.ComicDocument != nil {
		return reconcileFragmentComicDocumentImages(trace.ComicDocument, imageURLs)
	}
	scenes := append([]domain.FragmentScenePlan(nil), trace.Scenes...)
	for index := range scenes {
		if index < len(imageURLs) && strings.TrimSpace(scenes[index].GeneratedImageURL) == "" {
			scenes[index].GeneratedImageURL = strings.TrimSpace(imageURLs[index])
		}
		// Pre-document drafts are flattened pages. Their lettering is already
		// in the bitmap and their model-generated geometry is not deterministic.
		if scenes[index].ComicPage != nil && len(scenes[index].PanelImageURLs) == 0 {
			plan := *scenes[index].ComicPage
			plan.Panels = cloneFragmentComicPanels(plan.Panels)
			for panelIndex := range plan.Panels {
				for textIndex := range plan.Panels[panelIndex].ComicTexts {
					plan.Panels[panelIndex].ComicTexts[textIndex].RenderMode = "image"
				}
			}
			scenes[index].ComicPage = &plan
		}
	}
	return buildFragmentComicDocument(
		request, fragmentID, content, aspectRatio,
		trace.VisualBible, trace.VisualEvidence, scenes, trace.ReferenceAssets,
	)
}

// Image edits can reorder/delete pages while the historical generation trace
// remains unchanged. Reconcile by asset identity before using it for continuation.
func reconcileFragmentComicDocumentImages(document *domain.FragmentComicDocument, imageURLs []string) *domain.FragmentComicDocument {
	if document == nil {
		return nil
	}
	out := *document
	out.Pages = nil
	for index, url := range imageURLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		for _, page := range document.Pages {
			if page.BackgroundImageURL == url || page.FlattenedImageURL == url {
				page.PageIndex = index
				out.Pages = append(out.Pages, page)
				break
			}
		}
	}
	return &out
}

// Text IDs and render modes are assigned when merging/migrating documents;
// never mutate the previous generation's shared slice backing arrays.
func cloneFragmentComicPanels(panels []domain.FragmentComicPanelPlan) []domain.FragmentComicPanelPlan {
	out := append([]domain.FragmentComicPanelPlan(nil), panels...)
	for i := range out {
		out[i].ComicTexts = append([]domain.FragmentComicText(nil), panels[i].ComicTexts...)
	}
	return out
}

func mergeFragmentInputAssets(previous, current []domain.FragmentInputAsset) []domain.FragmentInputAsset {
	out := append([]domain.FragmentInputAsset(nil), previous...)
	seen := map[string]struct{}{}
	for _, item := range out {
		seen[item.URL+"|"+item.Role+"|"+item.ReferenceKey] = struct{}{}
	}
	for _, item := range current {
		key := item.URL + "|" + item.Role + "|" + item.ReferenceKey
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func mergeFragmentCreativeFacts(previous, current []domain.FragmentCreativeFact) []domain.FragmentCreativeFact {
	out := append([]domain.FragmentCreativeFact(nil), previous...)
	seen := map[string]struct{}{}
	for _, item := range out {
		seen[item.Source+"|"+item.Content] = struct{}{}
	}
	for _, item := range current {
		key := item.Source + "|" + item.Content
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func fragmentInputRole(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "character":
		return "character_reference"
	case "style":
		return "style_reference"
	case "location", "scene":
		return "scene_reference"
	case "prop", "object":
		return "prop_reference"
	case "existing_story_page":
		return "existing_story_page"
	default:
		return "general_inspiration"
	}
}

func inferFragmentInputRole(item domain.FragmentVisualEvidence) string {
	for _, entity := range item.Entities {
		switch strings.ToLower(strings.TrimSpace(entity.Kind)) {
		case "character":
			return "character_reference"
		case "location":
			return "scene_reference"
		case "prop":
			return "prop_reference"
		}
	}
	return "general_inspiration"
}

func fragmentComicDocumentFacts(req domain.FragmentGenerationRequest, content string, evidence []domain.FragmentVisualEvidence) []domain.FragmentCreativeFact {
	out := make([]domain.FragmentCreativeFact, 0, 2+len(evidence))
	if text := strings.TrimSpace(req.UserInput); text != "" {
		out = append(out, domain.FragmentCreativeFact{Content: text, Source: "user_text", Mutable: false})
	}
	for _, item := range evidence {
		if summary := strings.TrimSpace(item.Summary); summary != "" {
			out = append(out, domain.FragmentCreativeFact{Content: summary, Source: "user_image", Mutable: false})
		}
	}
	if story := strings.TrimSpace(content); story != "" && story != strings.TrimSpace(req.UserInput) {
		out = append(out, domain.FragmentCreativeFact{Content: story, Source: "agent_expansion", Mutable: true})
	}
	return out
}

func initialFragmentStoryState(bible *domain.FragmentVisualBible, scenes []domain.FragmentScenePlan) domain.FragmentStoryState {
	state := domain.FragmentStoryState{}
	characterIndexes := map[string]int{}
	locationKeys := map[string]struct{}{}
	if bible != nil {
		state.Characters = make([]domain.FragmentCharacterState, 0, len(bible.Characters))
		for _, character := range bible.Characters {
			key := strings.TrimSpace(character.Key)
			if key == "" {
				continue
			}
			characterIndexes[key] = len(state.Characters)
			state.Characters = append(state.Characters, domain.FragmentCharacterState{CharacterKey: key})
		}
		for _, location := range bible.Locations {
			if key := strings.TrimSpace(location.Key); key != "" {
				locationKeys[key] = struct{}{}
			}
		}
	}
	for _, scene := range scenes {
		for _, key := range scene.ReferenceKeys {
			if _, ok := locationKeys[key]; ok {
				state.CurrentLocation = key
			}
		}
		bindings := append([]domain.FragmentEntityBinding(nil), scene.EntityBindings...)
		if scene.ComicPage != nil {
			for _, panel := range scene.ComicPage.Panels {
				bindings = append(bindings, panel.EntityBindings...)
			}
		}
		for _, binding := range bindings {
			if index, ok := characterIndexes[binding.Key]; ok {
				character := &state.Characters[index]
				if emotion := strings.TrimSpace(binding.Emotion); emotion != "" {
					character.Emotion = emotion
				}
				if state.CurrentLocation != "" {
					character.CurrentLocationKey = state.CurrentLocation
				}
			}
			if ownerIndex, ok := characterIndexes[binding.OwnerKey]; ok && binding.Kind == "prop" {
				state.Characters[ownerIndex].Holding = normalizeFragmentKeyList(append(state.Characters[ownerIndex].Holding, binding.Key))
			}
		}
	}
	if len(scenes) > 0 {
		state.LastPageResult = strings.TrimSpace(scenes[len(scenes)-1].SceneDesc)
	}
	return state
}

func fragmentComicLayoutForPanelCount(panelCount int, aspectRatio string) domain.FragmentComicLayout {
	panelCount = clampFragmentComicPagePanelCount(panelCount)
	gutter := 0.018
	rows := fragmentComicLayoutRows(panelCount)
	panels := make([]domain.FragmentComicPanelLayout, 0, panelCount)
	y := gutter
	index := 0
	availableHeight := 1 - gutter*float64(len(rows)+1)
	rowHeight := availableHeight / float64(len(rows))
	for _, columns := range rows {
		availableWidth := 1 - gutter*float64(columns+1)
		width := availableWidth / float64(columns)
		x := gutter
		for column := 0; column < columns && index < panelCount; column++ {
			importance := "narrative"
			if index == 0 {
				importance = "establishing"
			} else if index == panelCount-1 {
				importance = "consequence"
			}
			panels = append(panels, domain.FragmentComicPanelLayout{Index: index, Rect: domain.FragmentComicRect{X: x, Y: y, Width: width, Height: rowHeight}, Importance: importance})
			x += width + gutter
			index++
		}
		y += rowHeight + gutter
	}
	order := make([]int, panelCount)
	for i := range order {
		order[i] = i
	}
	return domain.FragmentComicLayout{PageAspectRatio: firstNonBlank(domain.NormalizeFragmentAspectRatio(aspectRatio), domain.FragmentComicPageAspectDefault), Gutter: gutter, ReadingOrder: order, Panels: panels}
}

func fragmentComicLayoutRows(panelCount int) []int {
	switch panelCount {
	case 2:
		return []int{1, 1}
	case 3:
		return []int{1, 2}
	case 4:
		return []int{2, 2}
	case 5:
		return []int{1, 2, 2}
	case 6:
		return []int{2, 2, 2}
	case 7:
		return []int{1, 3, 3}
	case 8:
		return []int{2, 3, 3}
	case 9:
		return []int{3, 3, 3}
	default:
		return []int{2, 3, 3, 2}
	}
}
