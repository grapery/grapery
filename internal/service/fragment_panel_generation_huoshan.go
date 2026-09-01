package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// Huoshan Panel 组图：单条合并 prompt + 多参考 + 一次 API 返回 N 张（与 Gemini 逐张循环分离）。

// buildPanelBatchHuoshanPromptSlice 对 plan[sliceStart:sliceExclusive) 拼装组图文案；每张格的全局序号为全局 totalPanels。
func buildPanelBatchHuoshanPromptSlice(plan []domain.FragmentPanelPlanItem, sliceStart, sliceExclusive int, styleSlug, aspectRatio string, totalPanels int, languages ...string) string {
	batchLen := sliceExclusive - sliceStart
	if batchLen < 1 || sliceStart < 0 || sliceExclusive > len(plan) || sliceStart >= sliceExclusive || totalPanels < 1 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Generate a coherent visual sequence of exactly ")
	b.WriteString(strconv.Itoa(batchLen))
	b.WriteString(" separate full images in strict order (image 1, then image 2, ...). ")
	b.WriteString("Each output corresponds to one comic panel. Maintain consistent characters, wardrobe, and world rules across panels where the panel text implies continuity.\n\n")
	for i := sliceStart; i < sliceExclusive; i++ {
		b.WriteString("--- Panel ")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(" / ")
		b.WriteString(strconv.Itoa(totalPanels))
		b.WriteString(" ---\n")
		b.WriteString(buildPanelFinalImagePrompt(plan[i], styleSlug, aspectRatio, i, totalPanels, languages...))
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

func buildPanelBatchHuoshanPrompt(plan []domain.FragmentPanelPlanItem, styleSlug, aspectRatio string, totalPanels int, languages ...string) string {
	n := totalPanels
	if n > len(plan) {
		n = len(plan)
	}
	return buildPanelBatchHuoshanPromptSlice(plan, 0, n, styleSlug, aspectRatio, totalPanels, languages...)
}

func mergePanelBatchReferenceImages(userRef string, plan []domain.FragmentPanelPlanItem, anchorMap map[string]string, maxN int) []string {
	if maxN < 1 {
		maxN = 6
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		low := strings.ToLower(u)
		if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		if len(out) >= maxN {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	add(userRef)
	for _, p := range plan {
		for _, k := range p.ReferenceKeys {
			key := strings.TrimSpace(k)
			if key == "" || anchorMap == nil {
				continue
			}
			add(anchorMap[key])
			if len(out) >= maxN {
				return out
			}
		}
	}
	return out
}

// runPanelImageBatchHuoshan 单次组图生成全部格（startIdx==0）。
func (s *FragmentPanelGenerationService) runPanelImageBatchHuoshan(ctx context.Context, task *domain.FragmentPanelGenerationTask, taskID, draftID string, req domain.FragmentPanelGenerationRequest, imgProv string, n int, anchorMap map[string]string, policy *domain.FragmentConsistencyPolicy) error {
	return s.runPanelImageBatchHuoshanRange(ctx, task, taskID, draftID, req, imgProv, 0, n, anchorMap, policy)
}

// runPanelImageBatchHuoshanRange Huoshan 组图：单次 API 生成 [startIdx, n) 共 n-startIdx 张（resume 时用，避免逐张 loop）。
func (s *FragmentPanelGenerationService) runPanelImageBatchHuoshanRange(ctx context.Context, task *domain.FragmentPanelGenerationTask, taskID, draftID string, req domain.FragmentPanelGenerationRequest, imgProv string, startIdx, n int, anchorMap map[string]string, policy *domain.FragmentConsistencyPolicy) error {
	if n < 1 || len(task.Plan) < n || startIdx < 0 || startIdx >= n {
		return fmt.Errorf("invalid panel batch size or range")
	}
	batchLen := n - startIdx
	stepName := "generating_panel_batch_huoshan"
	if startIdx > 0 {
		stepName = fmt.Sprintf("generating_panel_batch_huoshan_resume_%d", startIdx)
	}
	task.CurrentStep = stepName
	den := max(n, 1)
	prog := 28 + (61*(startIdx+batchLen))/den
	if prog > 89 {
		prog = 89
	}
	task.Progress = prog
	task.UpdatedAt = time.Now().Unix()
	_ = s.panelRepo.Save(ctx, task)

	prompt := buildPanelBatchHuoshanPromptSlice(task.Plan, startIdx, n, req.Style, req.AspectRatio, n, req.Language)
	refURLs := mergePanelBatchReferenceImages(req.ReferenceImageURL, task.Plan[startIdx:n], anchorMap, panelMaxReferenceImages)

	ar := domain.NormalizeFragmentAspectRatio(req.AspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	seed := panelSeed(policy, startIdx, nil)
	options := cloneFragmentProviderOptions(policy)

	imgStart := time.Now()
	imgReq := &GenerateImageRequest{
		UserID:            task.UserID,
		Prompt:            prompt,
		Provider:          imgProv,
		Quality:           "standard",
		Style:             req.Style,
		OutputCount:       batchLen,
		ReferenceImages:   refURLs,
		Seed:              seed,
		Options:           options,
		RelatedEntityID:   taskID,
		RelatedEntityType: "fragment_panel_generation",
		Metadata: map[string]interface{}{
			"step":              stepName,
			"panel_batch":       true,
			"provider":          imgProv,
			"aspectRatio":       ar,
			"panel_count_total": n,
			"panel_batch_start": startIdx,
			"panel_batch_len":   batchLen,
			"reference_count":   len(refURLs),
			"seed":              seed,
			"max_images":        batchLen,
			"seedream50_mode":   "batch_panel",
		},
	}
	imgReq.Size = domain.FragmentImagePixelSizeForAspectRatio(ar)

	imgOut, genErr := s.aiGen.GenerateImage(ctx, imgReq)
	if genErr != nil {
		return fmt.Errorf("组图生成失败: %w", genErr)
	}
	var gotCnt int
	if imgOut != nil {
		gotCnt = len(imgOut.ImageURLs)
	}
	exp := imgReq.OutputCount
	if imgOut == nil || gotCnt < exp {
		return fmt.Errorf("组图返回不足: got %d need %d", gotCnt, exp)
	}
	modelLabel := strings.TrimSpace(imgReq.Model)
	dur := time.Since(imgStart).Milliseconds()
	if imgOut.DurationMs > 0 {
		dur = imgOut.DurationMs
	}
	appendPanelMetric(task, stepName, imgOut.TokensUsed, dur, imgProv, modelLabel)

	if startIdx == 0 {
		task.Result.Panels = make([]domain.FragmentPanelResultItem, 0, exp)
		for i := 0; i < exp; i++ {
			task.Result.Panels = append(task.Result.Panels, domain.FragmentPanelResultItem{
				Index:    i,
				ImageURL: imgOut.ImageURLs[i],
				Caption:  strings.TrimSpace(task.Plan[i].Caption),
			})
		}
		task.Result.BatchHuoshanPrompt = prompt
		task.Result.BatchSourceImageURLs = append([]string(nil), imgOut.ImageURLs[:exp]...)
	} else {
		for i := 0; i < exp; i++ {
			gi := startIdx + i
			task.Result.Panels = append(task.Result.Panels, domain.FragmentPanelResultItem{
				Index:    gi,
				ImageURL: imgOut.ImageURLs[i],
				Caption:  strings.TrimSpace(task.Plan[gi].Caption),
			})
		}
		task.Result.BatchHuoshanPrompt = prompt
		if task.Result.BatchSourceImageURLs == nil {
			task.Result.BatchSourceImageURLs = append([]string(nil), imgOut.ImageURLs[:exp]...)
		} else {
			task.Result.BatchSourceImageURLs = append(task.Result.BatchSourceImageURLs, imgOut.ImageURLs[:exp]...)
		}
	}

	task.UpdatedAt = time.Now().Unix()
	if err := s.panelRepo.Save(ctx, task); err != nil {
		s.logger.Error("panel gen: save after batch", zap.Error(err))
	}
	s.syncDraftFromTask(ctx, task)
	return nil
}
