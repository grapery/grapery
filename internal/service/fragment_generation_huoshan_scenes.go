package service

import (
	"strconv"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// narrative fragment：Huoshan 组图 — 单条合并 prompt + 合并参考图（与逐场景 Gemini 路径分离）。

func buildFragmentScenesBatchHuoshanPrompt(bible *domain.FragmentVisualBible, scenes []domain.FragmentScenePlan, n int, languages ...string) string {
	if n < 1 || len(scenes) == 0 {
		return ""
	}
	if n > len(scenes) {
		n = len(scenes)
	}
	var b strings.Builder
	comicPages := true
	for i := 0; i < n; i++ {
		if scenes[i].ComicPage == nil {
			comicPages = false
			break
		}
	}
	b.WriteString("Generate a coherent visual sequence of exactly ")
	b.WriteString(strconv.Itoa(n))
	if comicPages {
		b.WriteString(" separate complete comic-page images in strict order (page image 1, then page image 2, ...). ")
		b.WriteString("Every output is independently a complete multi-panel comic page; never return loose panels or combine several requested page images into one output. Maintain consistent characters, wardrobe, props, locations, lettering language, and world rules across all page images.\n")
	} else {
		b.WriteString(" separate full images in strict order (image 1, then image 2, ...). ")
		b.WriteString("Each output corresponds to one narrative scene. Maintain consistent characters, wardrobe, and world rules across scenes where continuity applies.\n")
		b.WriteString("The following canvas rule applies to every image in the batch. ")
		b.WriteString(fullBleedCanvasDirective())
	}
	b.WriteString("\n\n")
	for i := 0; i < n; i++ {
		if comicPages {
			b.WriteString("--- Complete Comic Page Image ")
		} else {
			b.WriteString("--- Scene ")
		}
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(" / ")
		b.WriteString(strconv.Itoa(n))
		b.WriteString(" ---\n")
		b.WriteString(buildFragmentSceneImagePromptCore(bible, scenes[i], languages...))
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

func mergeFragmentScenesBatchReferenceImages(userURLs []string, scenes []domain.FragmentScenePlan, assets []domain.FragmentReferenceAsset, maxTotal int) []string {
	if maxTotal <= 0 {
		maxTotal = fragmentMaxSceneReferenceImages
	}
	m := fragmentReferenceAssetMap(assets)
	seen := map[string]struct{}{}
	var out []string
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	for _, u := range userURLs {
		add(u)
		if len(out) >= maxTotal {
			return out
		}
	}
	for _, sc := range scenes {
		for _, k := range sc.ReferenceKeys {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			if u := m[key]; u != "" {
				add(u)
				if len(out) >= maxTotal {
					return out
				}
			}
		}
	}
	return out
}
