package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const (
	fragmentComicPageMinPanels     = 2
	fragmentComicPageDefaultPanels = 4
	fragmentComicPageMaxPanels     = 10
	fragmentComicPagePlanMaxTokens = 4096
)

type fragmentComicPagePlanningCallback func(pageIndex int, pages []domain.FragmentScenePlan)

// planFragmentComicPages turns every outer image slot into one complete comic
// page. Each page gets its own bounded draft/revision cycle: the text endpoint
// has a compact response limit, so one giant response for all pages could
// truncate later pages and silently lose panels.
func (s *FragmentGenerationService) planFragmentComicPages(
	ctx context.Context,
	userID, taskID string,
	req domain.FragmentGenerationRequest,
	storyContent string,
	bible *domain.FragmentVisualBible,
	pages []domain.FragmentScenePlan,
	callbacks ...fragmentComicPagePlanningCallback,
) ([]domain.FragmentScenePlan, int) {
	out := append([]domain.FragmentScenePlan(nil), pages...)
	totalTokens := 0
	for i := range out {
		if ctx.Err() != nil {
			break
		}
		fallbackPanelCount := fragmentComicPageFallbackPanelCount(out[i])
		prompt := buildFragmentComicPagePlanPrompt(req, storyContent, bible, out, i)
		payload, _ := json.Marshal(map[string]interface{}{
			"prompt": prompt,
			"step":   "fragment_comic_page_plan",
			"page":   i,
		})
		aiReq := domain.AITask{
			ID:                uuid.New().String(),
			UserID:            userID,
			Type:              domain.AITaskGenerateFragmentContent,
			Status:            domain.AITaskStatusProcessing,
			Input:             string(payload),
			RelatedEntityID:   taskID,
			RelatedEntityType: "fragment_generation",
		}

		var raw string
		var tokens int
		var err error
		if s == nil || s.aiService == nil {
			err = fmt.Errorf("fragment comic page planner unavailable")
		} else {
			raw, tokens, err = s.aiService.GenerateFragmentStructuredTextJSON(
				ctx,
				&aiReq,
				fragmentComicPagePlanMaxTokens,
				0.35,
				"fragment_comic_page_plan",
			)
		}
		totalTokens += tokens
		var plan *domain.FragmentComicPagePlan
		planningStatus := "planned"
		planningError := ""
		if err == nil {
			plan, err = parseFragmentComicPagePlan(raw)
			if err == nil {
				err = validateFragmentComicPagePlanCount(plan)
			}
			if err == nil {
				reviewIssues := reviewFragmentComicPagePlan(plan)
				if len(reviewIssues) > 0 {
					initialPlan := plan
					revisionPrompt := buildFragmentComicPageRevisionPrompt(prompt, raw, reviewIssues)
					revisionPayload, _ := json.Marshal(map[string]interface{}{
						"prompt": revisionPrompt,
						"step":   "fragment_comic_page_revision",
						"page":   i,
					})
					revisionTask := aiReq
					revisionTask.ID = uuid.New().String()
					revisionTask.Input = string(revisionPayload)
					revisionRaw, revisionTokens, revisionErr := s.aiService.GenerateFragmentStructuredTextJSON(
						ctx, &revisionTask, fragmentComicPagePlanMaxTokens, 0.25, "fragment_comic_page_revision",
					)
					totalTokens += revisionTokens
					if revisionErr == nil {
						var revised *domain.FragmentComicPagePlan
						revised, revisionErr = parseFragmentComicPagePlan(revisionRaw)
						if revisionErr == nil {
							revisionErr = validateFragmentComicPagePlanCount(revised)
						}
						if revisionErr == nil && len(reviewFragmentComicPagePlan(revised)) == 0 {
							plan = revised
							planningStatus = "revised"
						} else {
							plan = initialPlan
						}
					}
					if planningStatus != "revised" {
						planningStatus = "planned_with_review_notes"
						planningError = truncateRunes(strings.Join(reviewIssues, "; "), 180)
					}
				}
			}
		}
		if err != nil || plan == nil {
			if s != nil && s.logger != nil {
				s.logger.Warn("fragment comic page planning fell back to deterministic plan",
					zap.String("task_id", taskID), zap.Int("page_index", i), zap.Error(err))
			}
			fallback := buildFallbackFragmentComicPagePlan(out[i], fallbackPanelCount, req.Language)
			plan = &fallback
			planningStatus = "fallback"
			if err != nil {
				planningError = truncateRunes(err.Error(), 180)
			}
		}
		panelCount := len(plan.Panels)
		normalized := normalizeFragmentComicPagePlan(*plan, out[i], panelCount, bible, req.Language)
		layout := fragmentComicLayoutForPanelCount(panelCount, req.AspectRatio)
		normalized.Layout = &layout
		normalized.PlanningStatus = planningStatus
		normalized.PlanningError = planningError
		out[i].ComicPage = &normalized
		out[i].ReferenceKeys = mergeFragmentPageReferenceKeys(out[i].ReferenceKeys, normalized.Panels)
		out[i].EntityBindings = mergeFragmentPageEntityBindings(out[i].EntityBindings, normalized.Panels)
		for _, callback := range callbacks {
			if callback != nil {
				callback(i, append([]domain.FragmentScenePlan(nil), out...))
			}
		}
	}
	return out, totalTokens
}

// fragmentComicPageFallbackPanelCount is used only when semantic page planning
// is unavailable. It derives a compact count from visible narrative clauses so
// a thin beat is not padded into a fixed high-density layout.
func fragmentComicPageFallbackPanelCount(page domain.FragmentScenePlan) int {
	beat := strings.TrimSpace(page.SceneDesc)
	if beat == "" {
		return fragmentComicPageDefaultPanels
	}
	clauses := strings.FieldsFunc(beat, func(r rune) bool {
		switch r {
		case '。', '！', '？', '；', '，', '.', '!', '?', ';', ',', '\n':
			return true
		default:
			return false
		}
	})
	meaningfulClauses := 0
	for _, clause := range clauses {
		if len([]rune(strings.TrimSpace(clause))) >= 4 {
			meaningfulClauses++
		}
	}
	if meaningfulClauses == 0 {
		meaningfulClauses = 1
	}
	// A page needs room for setup and consequence, but never more panels than
	// the beat can justify. Ten panels require at least nine distinct clauses.
	return clampFragmentComicPagePanelCount(meaningfulClauses + 1)
}

func buildFragmentComicPagePlanPrompt(
	req domain.FragmentGenerationRequest,
	storyContent string,
	bible *domain.FragmentVisualBible,
	pages []domain.FragmentScenePlan,
	pageIndex int,
) string {
	language := normalizeGenerationLanguage(req.Language)
	page := pages[pageIndex]
	previous := ""
	next := ""
	if pageIndex > 0 {
		previous = pages[pageIndex-1].SceneDesc
	}
	if pageIndex+1 < len(pages) {
		next = pages[pageIndex+1].SceneDesc
	}
	return renderPromptDSL(PromptDSL{
		Role: "你是一位漫画页导演与结构化分镜规划器。",
		Task: "先识别当前页不可合并的有效剧情节拍，再自行决定所需内部漫画格数；外层仍只生成一张完整漫画页图片。",
		Inputs: map[string]any{
			"pageIndex": pageIndex, "pageCount": len(pages),
			"minimumPanelCount": fragmentComicPageMinPanels, "maximumPanelCount": fragmentComicPageMaxPanels,
			"pageAspectRatio": req.AspectRatio, "style": req.Style, "mood": req.Mood,
			"storyContext": truncateRunes(storyContent, 600), "pageBeat": page.SceneDesc,
			"pageVisualDirection": page.ImagePrompt, "previousPageBeat": previous, "nextPageBeat": next,
			"allowedReferenceKeys": formatVisualBibleKeyListForPrompt(bible),
			"contentLanguage":      language, "letteringLanguage": language,
		},
		GlobalConfig: structuredStoryPanelGuidance(),
		Sections: []PromptDSLSection{{Title: "Complete Comic Page Rules", Kind: "text", Body: fmt.Sprintf(`
- 这是第 %d/%d 张最终图片；它本身必须是一张完整漫画页，不是单幅插图，也不是把多张外部图片合成到一起。
- 先从 pageBeat 中识别独立的动作、发现、反应、对白、转折和结果，再在 %d～%d 格之间选择能够完整且紧凑表达本页的最少格数。
- 不得先选择高格数再用重复姿态、重复环境、无结果的过渡动作填满；相邻格若没有新增信息、动作结果、情绪变化、对白或关键视角，必须合并。
- panelCount 必须等于 panels 长度，index 从 0 连续递增；每格只承担一个主要节拍，全页形成清楚的因果推进。
- 第 1 格建立本页人物与空间，末格产生结果或翻页钩子；中间格交替使用远景、中景、近景、特写、俯拍、仰拍或倾斜机位。
- imagePrompt 只用英文，具体描述本格主体外观、姿态、动作、环境前中后景、镜头、构图、光源、材质和运动效果。
- 每格必须填写 newInformation 和 dramaticIntent；没有新增信息、动作结果、认知/情绪变化或必要节奏价值的格必须与相邻格合并。
- 每个角色通过 entityBindings 描述 narrativeRole、region、depth、relativeScale、facing、gazeTarget、pose、expression、emotion、emotionIntensity 和 stagingIntent；人物相对位置服务剧情，可使用非现实的漫画夸张。
- relations 描述角色/道具间的戏剧关系（dominates、looks_at、between、shields、follows、overlaps），required 关系必须进入最终构图。
- sceneDesc、beatPurpose 和 comicTexts 使用%s；对白、心理活动、语气词和拟声词必须短且服务于动作，不得发明 pageBeat 之外的新剧情。
- 有语言行为时使用 comicTexts，type 为 narration、dialogue、thought、sfx 或 interjection；补充 speaker、target、tone、volume、rhythm、balloonStyle、tailTarget。对白/心理/旁白默认 renderMode=overlay。
- 无文字格必须填写 silentIntent，说明沉默如何通过表情、姿态或构图推进叙事；禁止因为遗漏 comicTexts 而默认无字。
- referenceKeys 只能使用 allowedReferenceKeys；角色、服装、道具归属与地点结构在整页及前后页保持一致。
- layoutPreset 必须与自行选择的 panelCount 对应；layoutDescription 必须说明每一行的宽窄格、内部白色 gutter、粗黑分格线和从左到右、从上到下的阅读顺序。`, pageIndex+1, len(pages), fragmentComicPageMinPanels, fragmentComicPageMaxPanels, generationLanguageName(language))}},
		OutputContract: fmt.Sprintf(`只输出一个 JSON 对象：
{"panelCount":4,"layoutPreset":"manga_page_4_dynamic","layoutDescription":"页面行列、宽窄格及叙事重点","readingOrder":"left_to_right_top_to_bottom","panels":[{"index":0,"beatPurpose":"contentLanguage purpose","newInformation":"本格新增信息","dramaticIntent":"本格戏剧目的","silentIntent":"仅无文字时填写","sceneDesc":"contentLanguage visible beat","imagePrompt":"English executable panel prompt without dialogue glyphs","shotType":"wide_shot","cameraAngle":"eye_level","composition":"English subject placement and visual flow","referenceKeys":[],"entityBindings":[{"key":"character_key","kind":"character","narrativeRole":"dramatic role","region":"right_bottom","depth":"background","relativeScale":"small","facing":"target key","gazeTarget":"target key","pose":"visible pose","expression":"visible expression","emotion":"emotion","emotionIntensity":0.7,"stagingIntent":"dramatic composition purpose","allowComicExaggeration":true}],"relations":[{"subject":"character_key","relation":"looks_at","object":["target_key"],"visualExpression":"visible staging","priority":"required"}],"comicTexts":[{"type":"dialogue","text":"短对白","speaker":"character_key","target":"target_key","tone":"语气","volume":"normal","rhythm":"short","balloonStyle":"oval","tailTarget":"character_key","position":"upper_right","renderMode":"overlay"}]}]}
示例中的 4 不是默认值；必须根据 pageBeat 选择 %d～%d，并令 panels 数量与 panelCount 完全一致。不得输出 markdown、解释或 JSON 之外的文字。`, fragmentComicPageMinPanels, fragmentComicPageMaxPanels),
	})
}

func validateFragmentComicPagePlanCount(plan *domain.FragmentComicPagePlan) error {
	if plan == nil {
		return fmt.Errorf("fragment comic page plan is nil")
	}
	panelCount := plan.PanelCount
	if panelCount == 0 {
		panelCount = len(plan.Panels)
		plan.PanelCount = panelCount
	}
	if panelCount < fragmentComicPageMinPanels || panelCount > fragmentComicPageMaxPanels {
		return fmt.Errorf("planner selected panel count %d outside adaptive range %d-%d", panelCount, fragmentComicPageMinPanels, fragmentComicPageMaxPanels)
	}
	if len(plan.Panels) != panelCount {
		return fmt.Errorf("planner returned %d panels for selected count %d", len(plan.Panels), panelCount)
	}
	for i, panel := range plan.Panels {
		if strings.TrimSpace(panel.NewInformation) == "" {
			return fmt.Errorf("planner panel %d omitted newInformation", i)
		}
		if strings.TrimSpace(panel.DramaticIntent) == "" {
			return fmt.Errorf("planner panel %d omitted dramaticIntent", i)
		}
		if len(panel.ComicTexts) == 0 && strings.TrimSpace(panel.SilentIntent) == "" {
			return fmt.Errorf("planner panel %d omitted comicTexts without an intentional silentIntent", i)
		}
	}
	return nil
}

func reviewFragmentComicPagePlan(plan *domain.FragmentComicPagePlan) []string {
	if plan == nil {
		return []string{"missing page plan"}
	}
	issues := make([]string, 0)
	seenInformation := map[string]int{}
	seenScenes := map[string]int{}
	for index, panel := range plan.Panels {
		informationKey := fragmentComicReviewKey(panel.NewInformation)
		if previous, ok := seenInformation[informationKey]; informationKey != "" && ok {
			issues = append(issues, fmt.Sprintf("panel %d repeats panel %d new information", index+1, previous+1))
		} else if informationKey != "" {
			seenInformation[informationKey] = index
		}
		sceneKey := fragmentComicReviewKey(panel.SceneDesc)
		if previous, ok := seenScenes[sceneKey]; sceneKey != "" && ok {
			issues = append(issues, fmt.Sprintf("panel %d duplicates panel %d visible beat", index+1, previous+1))
		} else if sceneKey != "" {
			seenScenes[sceneKey] = index
		}
		if index > 0 {
			previous := plan.Panels[index-1]
			currentStaging := fragmentComicReviewKey(panel.ShotType + "|" + panel.CameraAngle + "|" + panel.Composition)
			previousStaging := fragmentComicReviewKey(previous.ShotType + "|" + previous.CameraAngle + "|" + previous.Composition)
			if currentStaging != "||" && currentStaging == previousStaging {
				issues = append(issues, fmt.Sprintf("panels %d and %d use the same staging", index, index+1))
			}
		}
		for _, binding := range panel.EntityBindings {
			if binding.Kind == "character" && strings.TrimSpace(binding.Pose) == "" && strings.TrimSpace(binding.Expression) == "" {
				issues = append(issues, fmt.Sprintf("panel %d character %s has no visible performance", index+1, binding.Key))
			}
		}
	}
	return issues
}

func fragmentComicReviewKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), ""))
}

func buildFragmentComicPageRevisionPrompt(originalPrompt, rawPlan string, issues []string) string {
	return fmt.Sprintf(`%s

EDITOR REVIEW
The first storyboard draft below failed narrative review:
%s

Issues that must be fixed:
- %s

Revise the draft rather than merely rewording it. Merge redundant panels, create distinct visible actions/reactions/results, vary staging, and preserve every required story fact, identity, spatial relation, and structured text intent. The selected panel count may change within the allowed adaptive range. Return only the corrected JSON object using the original output contract.`,
		originalPrompt, truncateRunes(rawPlan, 5000), strings.Join(issues, "\n- "))
}

func parseFragmentComicPagePlan(raw string) (*domain.FragmentComicPagePlan, error) {
	cleaned := cleanFragmentModelJSON(raw)
	var plan domain.FragmentComicPagePlan
	if err := json.Unmarshal([]byte(cleaned), &plan); err == nil && len(plan.Panels) > 0 {
		return &plan, nil
	}
	var envelope struct {
		ComicPage domain.FragmentComicPagePlan `json:"comicPage"`
	}
	if err := json.Unmarshal([]byte(cleaned), &envelope); err != nil {
		return nil, fmt.Errorf("parse fragment comic page plan: %w", err)
	}
	if len(envelope.ComicPage.Panels) == 0 {
		return nil, fmt.Errorf("fragment comic page plan returned no panels")
	}
	return &envelope.ComicPage, nil
}

func normalizeFragmentComicPagePlan(
	plan domain.FragmentComicPagePlan,
	page domain.FragmentScenePlan,
	want int,
	bible *domain.FragmentVisualBible,
	language string,
) domain.FragmentComicPagePlan {
	want = clampFragmentComicPagePanelCount(want)
	fallback := buildFallbackFragmentComicPagePlan(page, want, language)
	panels := append([]domain.FragmentComicPanelPlan(nil), plan.Panels...)
	if len(panels) > want {
		panels = panels[:want]
	}
	for len(panels) < want {
		panels = append(panels, fallback.Panels[len(panels)])
	}
	valid, characters, kinds := fragmentVisualBibleKeySets(bible)
	for i := range panels {
		panels[i].Index = i
		panels[i].BeatPurpose = strings.TrimSpace(panels[i].BeatPurpose)
		panels[i].NewInformation = firstNonBlank(strings.TrimSpace(panels[i].NewInformation), fallback.Panels[i].NewInformation)
		panels[i].DramaticIntent = firstNonBlank(strings.TrimSpace(panels[i].DramaticIntent), fallback.Panels[i].DramaticIntent)
		panels[i].SilentIntent = strings.TrimSpace(panels[i].SilentIntent)
		panels[i].SceneDesc = firstNonBlank(strings.TrimSpace(panels[i].SceneDesc), fallback.Panels[i].SceneDesc)
		panels[i].ImagePrompt = firstNonBlank(strings.TrimSpace(panels[i].ImagePrompt), fallback.Panels[i].ImagePrompt)
		panels[i].ShotType = firstNonBlank(strings.TrimSpace(panels[i].ShotType), fallback.Panels[i].ShotType)
		panels[i].CameraAngle = firstNonBlank(strings.TrimSpace(panels[i].CameraAngle), fallback.Panels[i].CameraAngle)
		panels[i].Composition = firstNonBlank(strings.TrimSpace(panels[i].Composition), fallback.Panels[i].Composition)
		panels[i].ReferenceKeys = filterFragmentReferenceKeys(panels[i].ReferenceKeys, valid)
		panels[i].EntityBindings = normalizeFragmentComicPanelBindings(panels[i].EntityBindings, valid, characters, kinds)
		panels[i].Relations = normalizeFragmentSpatialRelations(panels[i].Relations, valid)
		panels[i].ComicTexts = normalizeFragmentComicTextsForLanguage(panels[i].ComicTexts, language)
		for j := range panels[i].ComicTexts {
			item := &panels[i].ComicTexts[j]
			if item.Type != "dialogue" && item.Type != "thought" && item.Type != "interjection" {
				item.Speaker = ""
			} else if _, ok := characters[item.Speaker]; !ok {
				item.Speaker = ""
			}
			if _, ok := valid[item.Target]; !ok {
				item.Target = ""
			}
			if _, ok := characters[item.TailTarget]; !ok {
				item.TailTarget = item.Speaker
			}
			if item.RenderMode == "" {
				item.RenderMode = "overlay"
			}
		}
		if len(panels[i].ComicTexts) == 0 && panels[i].SilentIntent == "" {
			panels[i].SilentIntent = fallback.Panels[i].SilentIntent
		}
	}
	plan.PanelCount = want
	plan.LayoutPreset = firstNonBlank(strings.TrimSpace(plan.LayoutPreset), fragmentComicPageLayoutPreset(want))
	plan.LayoutDescription = firstNonBlank(strings.TrimSpace(plan.LayoutDescription), fragmentComicPageLayoutDescription(want))
	plan.ReadingOrder = firstNonBlank(strings.TrimSpace(plan.ReadingOrder), "left_to_right_top_to_bottom")
	plan.Panels = panels
	return plan
}

func buildFallbackFragmentComicPagePlan(page domain.FragmentScenePlan, panelCount int, language string) domain.FragmentComicPagePlan {
	panelCount = clampFragmentComicPagePanelCount(panelCount)
	roles := []struct{ purpose, shot, angle string }{
		{"建立本页空间与目标", "ultra_wide_shot", "eye_level"},
		{"人物观察并接近目标", "medium_shot", "over_the_shoulder"},
		{"阻碍进入画面", "wide_action_shot", "low_angle"},
		{"人物作出反应", "close_up", "eye_level"},
		{"动作开始升级", "dynamic_medium_shot", "dutch_angle"},
		{"关键动作产生碰撞", "close_action_shot", "low_angle"},
		{"局势发生变化", "wide_shot", "high_angle"},
		{"本页结果或翻页钩子", "cinematic_wide_shot", "rear_view"},
		{"代价或关键信息显现", "extreme_close_up", "eye_level"},
		{"以强烈结果收束本页", "ultra_wide_final_shot", "eye_level"},
	}
	panels := make([]domain.FragmentComicPanelPlan, 0, panelCount)
	for i := 0; i < panelCount; i++ {
		role := roles[i]
		if i == panelCount-1 {
			role = roles[7]
		}
		panels = append(panels, domain.FragmentComicPanelPlan{
			Index:          i,
			BeatPurpose:    role.purpose,
			NewInformation: fmt.Sprintf("第%d格必须呈现相对上一格可见的新变化", i+1),
			DramaticIntent: role.purpose,
			SilentIntent:   "用明确的表情、姿态和构图承担叙事；不绘制空气泡或伪文字",
			SceneDesc:      fmt.Sprintf("第%d格：%s。围绕「%s」呈现清楚的可见变化。", i+1, role.purpose, truncateRunes(page.SceneDesc, 44)),
			ImagePrompt: fmt.Sprintf("%s, %s, single comic panel %d of a %d-beat sequence, preserve the narrative: %s, distinct visible action and result, detailed environment, strong readable silhouette, coherent lighting and materials",
				role.shot, role.angle, i+1, panelCount, truncateRunes(page.ImagePrompt, 180)),
			ShotType:    role.shot,
			CameraAngle: role.angle,
			Composition: "clear focal subject, layered foreground middle ground and background, readable visual flow to the next panel",
		})
	}
	return domain.FragmentComicPagePlan{
		PanelCount:        panelCount,
		LayoutPreset:      fragmentComicPageLayoutPreset(panelCount),
		LayoutDescription: fragmentComicPageLayoutDescription(panelCount),
		ReadingOrder:      "left_to_right_top_to_bottom",
		Panels:            panels,
	}
}

func clampFragmentComicPagePanelCount(value int) int {
	if value < fragmentComicPageMinPanels {
		return fragmentComicPageMinPanels
	}
	if value > fragmentComicPageMaxPanels {
		return fragmentComicPageMaxPanels
	}
	return value
}

func fragmentComicPageLayoutPreset(panelCount int) string {
	return fmt.Sprintf("manga_page_%d_dynamic", clampFragmentComicPagePanelCount(panelCount))
}

func fragmentComicPageLayoutDescription(panelCount int) string {
	switch clampFragmentComicPagePanelCount(panelCount) {
	case 2:
		return "two-panel page: one establishing or setup panel and one consequence or hook panel; bold black divider and clean white gutter"
	case 3:
		return "three-panel page: setup, change or reaction, and consequence; varied panel sizes, bold black dividers and clean white gutters"
	case 4:
		return "four-panel page: compact setup, development, turning beat, and result; balanced panel geometry, bold black dividers and clean white gutters"
	case 5:
		return "five-panel page: one wide opener, three focused narrative beats, and one consequence panel; varied panel sizes, bold black dividers and clean white gutters"
	case 6:
		return "six-panel page: one wide establishing panel, two paired middle panels, one wide impact panel, and two closing panels; bold black dividers and clean white gutters"
	case 7:
		return "seven-panel page: one wide opener, two paired beats, three compact action or reaction panels, and one wide closing panel; bold black dividers and clean white gutters"
	case 8:
		return "eight-panel page: one wide opener, two paired narrative panels, three compact action panels, one wide consequence panel, and one wide closing panel; bold black dividers and clean white gutters"
	case 9:
		return "nine-panel page: one wide opener, two paired rows, three compact acceleration panels, and one wide closing panel; varied panel widths, bold black dividers and clean white gutters"
	default:
		return "ten-panel cinematic page: one ultra-wide opener, two paired narrative panels, two paired reaction panels, three narrow action panels, one wide consequence panel, and one ultra-wide closing panel; varied panel widths, bold black dividers and clean white gutters"
	}
}

func fragmentComicPageOutputDirective(plan *domain.FragmentComicPagePlan) string {
	if plan == nil {
		return ""
	}
	panelCount := clampFragmentComicPagePanelCount(plan.PanelCount)
	return fmt.Sprintf("[HARD OUTPUT REQUIREMENT] Generate ONE single image that is a complete comic page, NOT one standalone cinematic illustration and NOT multiple separate output images. The one image must contain exactly %d visibly separated internal comic panels using %s. Use %s. Reading order is %s. Draw deliberate black panel dividers and clean white gutters inside the page; the complete page fills the requested output aspect ratio with no UI chrome, mockup background, watermark, signature, or extra page outside the canvas. Do not add, remove, merge, or duplicate panels.",
		panelCount,
		firstNonBlank(plan.LayoutPreset, fragmentComicPageLayoutPreset(panelCount)),
		firstNonBlank(plan.LayoutDescription, fragmentComicPageLayoutDescription(panelCount)),
		firstNonBlank(plan.ReadingOrder, "left_to_right_top_to_bottom"),
	)
}

func writeFragmentComicPagePrompt(b *strings.Builder, plan *domain.FragmentComicPagePlan, language string) {
	if b == nil || plan == nil {
		return
	}
	fmt.Fprintf(b, "Comic page structure: panel_count=%d; layout_preset=%s; reading_order=%s.\n", plan.PanelCount, plan.LayoutPreset, plan.ReadingOrder)
	fmt.Fprintf(b, "Page layout geometry: %s.\n", plan.LayoutDescription)
	b.WriteString("Ordered internal panel directions:\n")
	for i, panel := range plan.Panels {
		fmt.Fprintf(b, "- Internal panel %d/%d: purpose=%s; narrative beat (%s)=%s; shot=%s; camera=%s; composition=%s; visual execution=%s.\n",
			i+1, len(plan.Panels), panel.BeatPurpose, generationLanguageName(language), panel.SceneDesc,
			panel.ShotType, panel.CameraAngle, panel.Composition, panel.ImagePrompt)
		if len(panel.ReferenceKeys) > 0 {
			fmt.Fprintf(b, "  Active identity/reference keys: %s.\n", strings.Join(panel.ReferenceKeys, ", "))
		}
		if len(panel.EntityBindings) > 0 {
			for _, binding := range panel.EntityBindings {
				fmt.Fprintf(b, "  Entity %s (%s): position=%s; action=%s; owner=%s; continuity=%s.\n",
					binding.Key, binding.Kind, binding.Position, binding.Action, binding.OwnerKey, binding.ConsistencyNote)
			}
		}
		if len(panel.ComicTexts) == 0 {
			fmt.Fprintf(b, "  Intentional silent panel: %s. Do not draw dialogue, narration, thought bubbles, or pseudo-text.\n", panel.SilentIntent)
			continue
		}
		b.WriteString("  Structured comic text directions for this panel:\n")
		for _, item := range normalizeFragmentComicTextsForLanguage(panel.ComicTexts, language) {
			if item.RenderMode == "image" {
				fmt.Fprintf(b, "  * IMAGE lettering: type=%s; exact_%s_text=%q; tone=%s; position=%s. Draw this expressive sound effect only.\n",
					item.Type, generationLanguageName(language), sanitizeComicPromptText(item.Text), item.Tone, item.Position)
				continue
			}
			fmt.Fprintf(b, "  * OVERLAY reservation: type=%s; speaker=%s; target=%s; tone=%s; balloon=%s; position=%s. Leave clean, unobstructed negative space there; do NOT draw a balloon, glyphs, subtitles, or pseudo-text because the compositor adds it later.\n",
				item.Type, item.Speaker, item.Target, item.Tone, item.BalloonStyle, item.Position)
		}
	}
	b.WriteString("Lettering policy for the complete page: only entries marked IMAGE may be painted into pixels. For OVERLAY entries, reserve the requested negative space but draw no balloon and no glyphs. Never invent, translate, paraphrase, duplicate, or move lettering between panels; never draw pseudo-readable text.\n")
}

func normalizeFragmentSpatialRelations(relations []domain.FragmentSpatialRelation, valid map[string]struct{}) []domain.FragmentSpatialRelation {
	out := make([]domain.FragmentSpatialRelation, 0, len(relations))
	for _, relation := range relations {
		relation.Subject = strings.TrimSpace(relation.Subject)
		relation.Relation = strings.TrimSpace(strings.ToLower(relation.Relation))
		if _, ok := valid[relation.Subject]; !ok || relation.Relation == "" {
			continue
		}
		objects := make([]string, 0, len(relation.Object))
		for _, object := range relation.Object {
			object = strings.TrimSpace(object)
			if _, ok := valid[object]; ok && object != relation.Subject {
				objects = append(objects, object)
			}
		}
		if len(objects) == 0 {
			continue
		}
		relation.Object = objects
		relation.VisualExpression = strings.TrimSpace(relation.VisualExpression)
		relation.Priority = firstNonBlank(strings.TrimSpace(relation.Priority), "preferred")
		out = append(out, relation)
	}
	return out
}

func fragmentVisualBibleKeySets(bible *domain.FragmentVisualBible) (map[string]struct{}, map[string]struct{}, map[string]string) {
	valid := map[string]struct{}{}
	characters := map[string]struct{}{}
	kinds := map[string]string{}
	if bible == nil {
		return valid, characters, kinds
	}
	for _, item := range bible.Characters {
		if key := strings.TrimSpace(item.Key); key != "" {
			valid[key] = struct{}{}
			characters[key] = struct{}{}
			kinds[key] = "character"
		}
	}
	for _, item := range bible.Props {
		if key := strings.TrimSpace(item.Key); key != "" {
			valid[key] = struct{}{}
			kinds[key] = "prop"
		}
	}
	for _, item := range bible.Locations {
		if key := strings.TrimSpace(item.Key); key != "" {
			valid[key] = struct{}{}
			kinds[key] = "location"
		}
	}
	return valid, characters, kinds
}

func normalizeFragmentComicPanelBindings(
	bindings []domain.FragmentEntityBinding,
	valid, characters map[string]struct{},
	kinds map[string]string,
) []domain.FragmentEntityBinding {
	out := make([]domain.FragmentEntityBinding, 0, len(bindings))
	seen := map[string]struct{}{}
	for _, binding := range bindings {
		binding.Key = strings.TrimSpace(binding.Key)
		if _, ok := valid[binding.Key]; !ok {
			continue
		}
		binding.Kind = kinds[binding.Key]
		binding.OwnerKey = strings.TrimSpace(binding.OwnerKey)
		if binding.OwnerKey != "" {
			if _, ok := characters[binding.OwnerKey]; !ok || binding.OwnerKey == binding.Key {
				binding.OwnerKey = ""
			}
		}
		binding.Role = strings.TrimSpace(binding.Role)
		binding.Position = strings.TrimSpace(binding.Position)
		binding.Action = strings.TrimSpace(binding.Action)
		binding.ConsistencyNote = strings.TrimSpace(binding.ConsistencyNote)
		binding.NarrativeRole = strings.TrimSpace(binding.NarrativeRole)
		binding.Region = strings.TrimSpace(binding.Region)
		binding.Depth = strings.TrimSpace(binding.Depth)
		binding.RelativeScale = strings.TrimSpace(binding.RelativeScale)
		binding.Facing = strings.TrimSpace(binding.Facing)
		binding.GazeTarget = strings.TrimSpace(binding.GazeTarget)
		if _, ok := valid[binding.GazeTarget]; !ok {
			binding.GazeTarget = ""
		}
		binding.Pose = strings.TrimSpace(binding.Pose)
		binding.Expression = strings.TrimSpace(binding.Expression)
		binding.Emotion = strings.TrimSpace(binding.Emotion)
		if binding.EmotionIntensity < 0 {
			binding.EmotionIntensity = 0
		} else if binding.EmotionIntensity > 1 {
			binding.EmotionIntensity = 1
		}
		binding.StagingIntent = strings.TrimSpace(binding.StagingIntent)
		key := binding.Key + "|" + binding.Position + "|" + binding.Action + "|" + binding.OwnerKey
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, binding)
	}
	return out
}

func filterFragmentReferenceKeys(keys []string, valid map[string]struct{}) []string {
	out := make([]string, 0, len(keys))
	for _, key := range normalizeFragmentKeyList(keys) {
		if _, ok := valid[key]; ok {
			out = append(out, key)
		}
	}
	return out
}

func mergeFragmentPageReferenceKeys(base []string, panels []domain.FragmentComicPanelPlan) []string {
	all := append([]string(nil), base...)
	for _, panel := range panels {
		all = append(all, panel.ReferenceKeys...)
	}
	return normalizeFragmentKeyList(all)
}

func mergeFragmentPageEntityBindings(base []domain.FragmentEntityBinding, panels []domain.FragmentComicPanelPlan) []domain.FragmentEntityBinding {
	out := append([]domain.FragmentEntityBinding(nil), base...)
	seen := map[string]struct{}{}
	for _, item := range out {
		seen[item.Key+"|"+item.Position+"|"+item.Action] = struct{}{}
	}
	for _, panel := range panels {
		for _, item := range panel.EntityBindings {
			key := item.Key + "|" + item.Position + "|" + item.Action
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, item)
		}
	}
	return out
}
