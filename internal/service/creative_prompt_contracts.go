package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// buildFragmentSceneExpansionPrompt keeps narrative decisions, output shape and
// rendering instructions separate. The previous prompt repeated the same long
// manga handbook and used word count as a proxy for quality, which increased
// truncation risk without making a scene more drawable.
func buildFragmentSceneExpansionPrompt(
	req domain.FragmentGenerationRequest,
	elemResult *fragmentElementExtractionResult,
	sceneCount int,
	aspectRatio string,
	continuation *fragmentDraftContinuationContext,
) string {
	language := normalizeGenerationLanguage(req.Language)
	languageName := generationLanguageName(language)
	var elements any = []any{}
	var visualBible any = map[string]any{}
	if elemResult != nil {
		if b, err := json.Marshal(elemResult.Elements); err == nil {
			_ = json.Unmarshal(b, &elements)
		}
		if elemResult.VisualBible != nil {
			if b, err := json.Marshal(elemResult.VisualBible); err == nil {
				_ = json.Unmarshal(b, &visualBible)
			}
		}
	}
	content := ""
	keyHint := ""
	if elemResult != nil {
		content = elemResult.Content
		keyHint = formatVisualBibleKeyListForPrompt(elemResult.VisualBible)
	}

	continuationRule := "This is a new sequence of complete comic-page images."
	if continuation != nil {
		continuationRule = fmt.Sprintf("Continue after %d existing comic-page images. Plan only %d new page beats; do not repeat an existing page.", continuation.ExistingImageCount, sceneCount)
	}

	return renderPromptDSL(PromptDSL{
		Role: "你是一位漫画故事编辑与页面节奏规划器。",
		Task: "把故事转为恰好 sceneCount 张连续的完整漫画页剧情节拍。scenes 中每一项代表一张最终漫画页图片，不是页内单格。",
		Inputs: map[string]any{
			"contractVersion": visualSceneContractVersion,
			"sceneCount":      sceneCount, "aspectRatio": aspectRatio, "style": req.Style, "mood": req.Mood,
			"storyContent": content, "storyElements": elements, "visualBible": visualBible,
			"allowedReferenceKeys": keyHint, "continuation": continuationRule,
			"contentLanguage": language, "letteringLanguage": language,
		},
		GlobalConfig: structuredStoryPanelGuidance(),
		Sections: []PromptDSLSection{
			{Title: "Comic Page Beat Planning Rules", Kind: "text", Body: fmt.Sprintf(`
	- scenes 数组长度必须等于 sceneCount，index 从 0 连续递增；每项是一张最终漫画页图片的 page beat。
	- sceneDesc 用一至两句%s描述这一整页要完成的目标、阻碍、关键动作与页末结果/钩子。
	- imagePrompt 只用英文，描述本页统一媒介画风、主要人物与地点、核心动作、整体光线色彩及连续性；页内逐格镜头由下一阶段规划，不要把本页误写成一张单幅插图。
	- 页面形成完整叙事序列：首张漫画页建立人物、目标与空间；中间页升级行动、反应、转折或代价；末页呈现结果并回应开场。相邻页不得重复同一事件。
	- 每页必须保留该页独有的 must-show 事实；旧信、裂开的封蜡、雨具、姓名等关键线索不得被泛化成无关钥匙、手机或随机道具。
	- referenceKeys 只能从 allowedReferenceKeys 中选择；没有合适项就返回空数组。
	- entityBindings 仅描述本页跨格必须保持的实体归属与连续性，不复制 visualBible 全文。
	- comicTexts 在此页面节拍阶段保持为空；下一阶段会把精确对白、旁白和拟声词分配到页内各格。
	- 每张最终图片都会独立生成一张带内部粗黑分格线和白色 gutter 的完整漫画页；sceneCount 仍表示最终图片页数。`, languageName)}},
		OutputContract: `只输出一个 JSON 对象：
	{"scenes":[{"index":0,"sceneDesc":"contentLanguage complete comic-page beat","imagePrompt":"English page-wide visual direction","referenceKeys":[],"entityBindings":[],"comicTexts":[]}]}
不得输出 markdown、解释、占位符或 JSON 之外的文字。`,
	})
}

// buildFragmentPanelPlanUserPrompt is the production prompt for reference-led
// multi-panel planning.
func buildFragmentPanelPlanUserPrompt(userInput, style string, panelCount int, layoutAddon string, languages ...string) string {
	ui := strings.TrimSpace(userInput)
	st := strings.TrimSpace(style)
	if st == "" {
		st = "fantasy"
	}
	if panelCount < 1 {
		panelCount = 1
	}
	language := inferGenerationLanguage(ui)
	if len(languages) > 0 && strings.TrimSpace(languages[0]) != "" {
		language = normalizeGenerationLanguage(languages[0])
	}
	return renderPromptDSL(PromptDSL{
		Role: "你是一位漫画分镜导演与结构化提示词工程师。",
		Task: "以参考图事实为世界锚点，规划连续但不重复的 panels[]，并建立最小 visualBible。",
		Inputs: map[string]any{
			"contractVersion": visualSceneContractVersion,
			"userInput":       ui, "styleSlug": st, "styleDirection": fragmentStyleDesc(st),
			"panelCount": panelCount, "layoutAddon": strings.TrimSpace(layoutAddon),
			"narrativeRhythm": panelPlanNarrativeRhythm(panelCount),
			"contentLanguage": language, "letteringLanguage": language,
		},
		GlobalConfig: structuredStoryPanelGuidance(),
		Sections: []PromptDSLSection{
			{Title: "Paneling / Camera / Action / Comic Elements Rules", Kind: "text", Body: fmt.Sprintf(`
【自动布局决策】
- 第 0 格建立参考图中的身份与世界；后续格必须改变时刻、动作、景别或叙事结果，不得把参考图描 %d 遍。
- 每格输出 layout_intent、composition_plan、shot_type、visual_hierarchy。composition_plan 同时说明摆放方式和该布局服务的剧情功能。
- 默认使用单一连续画面；只有同一格确有两个以上节拍时才使用多个区域，并明确内部 gutter 与阅读顺序。
- 这是文本阶段的前置漫画规划；不允许“先不规划，后续生图再决定漫画元素”。
- image_prompt 只用英文并写成可执行视觉 brief，覆盖画风、主体与动作、环境、镜头构图、光线色彩和必要细节；不设最低词数。
- visualBible 只记录跨格必须稳定的可见事实。characters 每项必须有非空 name；所有 key 使用唯一 snake_case。
- reference_keys 只能引用 visualBible 已声明的 key。dialogue/thought 的 speaker 必须是本格角色 key。
- caption 与 comic_texts 使用%s；comic_texts 可为空，需要时保持短而少，并且是唯一允许出现的图中文字来源。
%s`, panelCount, generationLanguageName(language), fullBleedPlanningRule())},
		},
		OutputContract: fmt.Sprintf(`只输出一个 JSON 对象：
{"visualBible":{"styleBible":{"artStyle":"English executable art direction"},"characters":[],"props":[],"locations":[]},"panels":[{"index":0,"image_prompt":"English visual brief","caption":"contentLanguage caption","reference_keys":[],"layout_intent":"single_subject_focus","composition_plan":"构图及其叙事作用","shot_type":"medium_shot","visual_hierarchy":"主视觉、次视觉、背景","comic_texts":[]}]}
panels 必须恰好 %d 项，index 为 0..%d。caption 与 comic_texts 必须使用 %s。不得输出 markdown、解释或 JSON 之外的文字。`, panelCount, panelCount-1, generationLanguageName(language)),
	})
}

// buildImageGenerationPrompt normalizes an already planned storyboard scene.
// It must not invent a second story or re-run the full narrative handbook.
func (s *Service) buildImageGenerationPrompt(gen *domain.StoryboardImageGeneration) string {
	spec := storyboardVisualSceneSpec(gen)
	inputs := map[string]any{"contractVersion": visualSceneContractVersion, "sceneSpec": visualSceneSpecPromptInput(spec), "sceneCharacters": gen.SceneCharacters, "comicStyleSlug": strings.TrimSpace(gen.ComicStyle)}
	sections := []PromptDSLSection{{Title: "Normalization Rules", Kind: "text", Body: `
- Preserve the planned event, cast and continuity facts. Do not add a new character, prop, location, dialogue or plot beat.
- Convert only visible facts into concise fields. Prefer concrete camera, blocking, lighting and palette decisions over adjectives.
- Obey each references[].role exactly: previous_panel controls shot continuity, character_identity controls identity, and user_reference supplies user evidence. Never infer a role from array position.
- plannedVisualPrompt, continuityNote, layoutIntent, compositionPlan, shotType and visualHierarchy are authoritative when non-empty.
- For a transition scene, show environment only. Do not introduce characters or environmental signage that is absent from sceneSpec.
- comicTexts is authoritative. Do not invent, translate, paraphrase or add dialogue, interjections, signs or logos.
- For action, choose only the impact devices justified by the event. For quiet scenes, preserve stillness and negative space.
- Do not repeat the scene narrative in every field and do not include canvas borders or page margins.`}}
	if cs := strings.TrimSpace(gen.ComicStyle); cs != "" {
		sections = append(sections, PromptDSLSection{Title: "Comic Style Continuation", Kind: "text", Body: "Maintain the visual identity of comic style slug " + cs + ": " + fragmentStyleDesc(cs)})
	}
	if len(spec.References) > 0 {
		sections = append(sections, PromptDSLSection{Title: "Reference Policy", Kind: "text", Body: "The typed reference manifest is the sole authority for reference meaning. Preserve identity and continuity while creating the requested new beat and camera composition."})
	}
	return renderPromptDSL(PromptDSL{
		Role:         "You are a storyboard scene prompt normalizer.",
		Task:         "Convert the already planned scene into compact structured image controls without rewriting its story.",
		Inputs:       inputs,
		GlobalConfig: `Image-prompt field order: medium/style; narrative must-show facts; subject identity and visible action; environment; shot scale, camera angle and composition; lighting and palette; continuity; optional lettering negative space.`,
		Sections:     sections,
		OutputContract: `Return one JSON object only:
{"artStyle":"English medium and rendering method","lighting":"English concrete light source","colorPalette":"English concrete palette","composition":"English shot scale, camera angle, subject placement and depth","keyElements":["visible must-show anchor"],"mood":"English compound tone","additionalNotes":"English continuity, layout or material detail"}
Use concise strings. Do not output comicTexts; the persisted scene plan is the sole lettering authority. No markdown or commentary.`,
	})
}

func (s *Service) buildComicPageImageGenerationLLMPrompt(gen *domain.StoryboardImageGeneration, opts ComicPagePipelineOptions, plannedScene *domain.StoryboardScene, totalScenes int) string {
	NormalizeComicPagePipeline(&opts)
	if gen.PlannedScene == nil {
		gen.PlannedScene = plannedScene
	}
	spec := storyboardVisualSceneSpec(gen)
	return renderPromptDSL(PromptDSL{
		Role: "You are a manga/webtoon page director and structured image-prompt normalizer.",
		Task: "Convert one already-planned storyboard scene into one comic-page image with the exact requested internal panel count. Do not rewrite the story.",
		Inputs: map[string]any{
			"contractVersion": visualSceneContractVersion,
			"sceneSpec":       visualSceneSpecPromptInput(spec),
			"panelCount":      opts.PanelCount, "layoutPreset": opts.LayoutPreset,
			"pageAspectRatio": opts.PageAspectRatio, "dialogueMode": opts.DialogueMode,
			"storyboardSceneCount": totalScenes,
		},
		GlobalConfig: structuredStoryPanelGuidance(),
		Sections: []PromptDSLSection{{Title: "Comic Page Rules", Kind: "text", Body: `
- Produce visual controls for ONE output image containing exactly panelCount visibly separated internal panels in layoutPreset and pageAspectRatio.
- Split the authoritative narrativeBeat into a because/but/therefore sequence of distinct visible moments. Do not introduce a new plot event outside that beat.
- keyElements must contain exactly panelCount ordered panel descriptions. Each description includes a visible action/result plus an English shot-scale + camera-angle cue.
- Adjacent panels must not repeat the same shot-scale + camera-angle pair.
- Preserve plannedVisualPrompt, continuityNote, character identity, wardrobe, props and location facts.
- composition must map every internal panel to layoutPreset, including internal gutters and reading order; the outermost artwork still bleeds to the page edge.
- comicTexts from sceneSpec is authoritative. Never invent, translate, paraphrase or add lettering. Do not output comicTexts in the response.
- Quiet beats use stillness and negative space; impact devices appear only when the narrative justifies them.`}},
		OutputContract: fmt.Sprintf(`Return one JSON object only:
{"artStyle":"English page-wide medium and rendering method","lighting":"English page-level lighting with justified panel shifts","colorPalette":"English concrete palette","composition":"English %s layout geometry, internal gutters, reading order, panel mapping and full-bleed outer edge","keyElements":["exactly %d ordered panel descriptions"],"mood":"English compound tone","additionalNotes":"English identity, continuity, layout and material controls"}
keyElements length must equal %d. No markdown, comicTexts, explanations or trailing prose.`, opts.LayoutPreset, opts.PanelCount, opts.PanelCount),
	})
}
