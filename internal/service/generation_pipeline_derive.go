package service

import (
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// deriveWizardPipelineSteps builds content → scenes → images steps for Voyager wizard (through batch images).
// suggestedResumeAction guides the client to retry from the first failed / blocked phase.
func deriveWizardPipelineSteps(
	sb *domain.Storyboard,
	scenes []*domain.StoryboardScene,
	contentGen *domain.StoryboardContentGeneration,
	sceneGens []*domain.StoryboardSceneGeneration,
	imageGens []*domain.StoryboardImageGeneration,
	globalGenerating bool,
) ([]domain.GenerationPipelineStep, string) {
	if sb == nil {
		return nil, domain.SuggestedResumeNone
	}

	expectedScenes := sb.SceneCount
	if expectedScenes <= 0 {
		expectedScenes = 3
	}

	// --- Content step ---
	contentStep := domain.GenerationPipelineStep{
		Phase:  domain.PipelinePhaseContent,
		Order:  1,
		Title:  "故事内容",
		Status: domain.PipelineStepPending,
	}
	hasInferredContent := strings.TrimSpace(sb.Content) != "" ||
		sb.WorkflowStatus == domain.WorkflowStatusContentReady ||
		sb.WorkflowStatus == domain.WorkflowStatusImagesReady ||
		sb.WorkflowStatus == domain.WorkflowStatusVideoReady ||
		sb.WorkflowStatus == domain.WorkflowStatusPublished

	if contentGen != nil {
		switch contentGen.Status {
		case domain.GenerationStatusFailed:
			contentStep.Status = domain.PipelineStepFailed
			contentStep.ErrorMessage = strings.TrimSpace(contentGen.ErrorMessage)
			if contentStep.ErrorMessage == "" {
				contentStep.ErrorMessage = "内容生成失败"
			}
		case domain.GenerationStatusProcessing, domain.GenerationStatusPending:
			contentStep.Status = domain.PipelineStepRunning
			contentStep.Summary = "正在生成叙述内容"
		case domain.GenerationStatusCompleted:
			contentStep.Status = domain.PipelineStepCompleted
			contentStep.Summary = "叙述内容已就绪"
		default:
			contentStep.Status = domain.PipelineStepRunning
		}
	} else if hasInferredContent {
		contentStep.Status = domain.PipelineStepCompleted
		contentStep.Summary = "叙述内容已就绪"
	} else if globalGenerating {
		contentStep.Status = domain.PipelineStepRunning
		contentStep.Summary = "正在准备内容"
	} else {
		contentStep.Status = domain.PipelineStepPending
		contentStep.Summary = "等待生成内容"
	}

	contentDone := contentStep.Status == domain.PipelineStepCompleted

	// --- Scenes step ---
	scenesStep := domain.GenerationPipelineStep{
		Phase:  domain.PipelinePhaseScenes,
		Order:  2,
		Title:  "分镜与场景",
		Status: domain.PipelineStepPending,
	}
	nScenes := len(scenes)
	if !contentDone {
		scenesStep.Status = domain.PipelineStepPending
		scenesStep.Summary = "等待故事内容完成"
	} else if nScenes == 0 {
		if globalGenerating {
			scenesStep.Status = domain.PipelineStepRunning
			scenesStep.Summary = fmt.Sprintf("正在生成分镜（目标 %d 格）", expectedScenes)
		} else {
			scenesStep.Status = domain.PipelineStepFailed
			scenesStep.ErrorMessage = "分镜数据未就绪，请重试或重新发起生成"
			scenesStep.Summary = "分镜未生成"
		}
	} else {
		scenesStep.Summary = fmt.Sprintf("已生成 %d 格分镜", nScenes)
		anySceneGenFail := false
		var firstErr string
		var items []domain.GenerationPipelineSceneItem
		for _, sg := range sceneGens {
			if sg == nil {
				continue
			}
			if sg.Status == domain.GenerationStatusFailed {
				anySceneGenFail = true
				em := strings.TrimSpace(sg.ErrorMessage)
				if firstErr == "" {
					firstErr = em
				}
				items = append(items, domain.GenerationPipelineSceneItem{
					SceneID:      sg.SceneID,
					SceneTitle:   sg.SceneTitle,
					Status:       domain.PipelineStepFailed,
					ErrorMessage: em,
				})
			}
		}
		if anySceneGenFail {
			scenesStep.Status = domain.PipelineStepFailed
			scenesStep.ErrorMessage = firstErr
			if scenesStep.ErrorMessage == "" {
				scenesStep.ErrorMessage = "部分场景描述生成失败"
			}
			scenesStep.SceneItems = items
		} else if nScenes < expectedScenes {
			scenesStep.Status = domain.PipelineStepRunning
			scenesStep.Summary = fmt.Sprintf("分镜 %d/%d", nScenes, expectedScenes)
		} else {
			scenesStep.Status = domain.PipelineStepCompleted
		}
	}

	scenesDone := scenesStep.Status == domain.PipelineStepCompleted

	// --- Images step ---
	imagesStep := domain.GenerationPipelineStep{
		Phase:  domain.PipelinePhaseImages,
		Order:  3,
		Title:  "分镜配图",
		Status: domain.PipelineStepPending,
	}
	if !scenesDone {
		imagesStep.Status = domain.PipelineStepPending
		imagesStep.Summary = "等待分镜就绪"
	} else if nScenes == 0 {
		imagesStep.Status = domain.PipelineStepSkipped
		imagesStep.Summary = "无分镜可配图"
	} else {
		latestByScene := latestImageGenByScene(imageGens)
		var items []domain.GenerationPipelineSceneItem
		failedN, doneN, runN, pendN := 0, 0, 0, 0

		for _, sc := range scenes {
			if sc == nil {
				continue
			}
			g := latestByScene[sc.ID]
			st, errMsg := pipelineImageItemStatus(sc, g)
			switch st {
			case domain.PipelineStepFailed:
				failedN++
			case domain.PipelineStepCompleted:
				doneN++
			case domain.PipelineStepRunning:
				runN++
			default:
				pendN++
			}
			if st == domain.PipelineStepFailed || (st == domain.PipelineStepRunning && errMsg != "") {
				items = append(items, domain.GenerationPipelineSceneItem{
					SceneID:      sc.ID,
					SceneTitle:   sc.Title,
					Status:       st,
					ErrorMessage: errMsg,
				})
			}
		}

		imagesStep.SceneItems = items
		imagesStep.Summary = fmt.Sprintf("已完成 %d/%d 格配图", doneN, nScenes)

		switch {
		case failedN > 0:
			imagesStep.Status = domain.PipelineStepFailed
			imagesStep.ErrorMessage = fmt.Sprintf("%d 格配图失败", failedN)
			if len(items) > 0 && strings.TrimSpace(items[0].ErrorMessage) != "" {
				imagesStep.ErrorMessage = items[0].ErrorMessage
			}
		case runN > 0 || pendN > 0:
			imagesStep.Status = domain.PipelineStepRunning
			if runN > 0 {
				imagesStep.Summary = fmt.Sprintf("配图进行中（%d 格处理中）", runN)
			}
		case doneN >= nScenes:
			imagesStep.Status = domain.PipelineStepCompleted
		default:
			imagesStep.Status = domain.PipelineStepRunning
		}
	}

	steps := []domain.GenerationPipelineStep{contentStep, scenesStep, imagesStep}
	suggested := suggestedResumeFromSteps(steps)
	return steps, suggested
}

func latestImageGenByScene(gens []*domain.StoryboardImageGeneration) map[string]*domain.StoryboardImageGeneration {
	out := make(map[string]*domain.StoryboardImageGeneration)
	for _, g := range gens {
		if g == nil || g.SceneID == "" {
			continue
		}
		cur, ok := out[g.SceneID]
		if !ok || g.CreatedAt > cur.CreatedAt {
			out[g.SceneID] = g
		}
	}
	return out
}

func pipelineImageItemStatus(sc *domain.StoryboardScene, g *domain.StoryboardImageGeneration) (status string, errMsg string) {
	if sc != nil && strings.TrimSpace(sc.Image) != "" {
		return domain.PipelineStepCompleted, ""
	}
	if g == nil {
		return domain.PipelineStepPending, ""
	}
	switch g.Status {
	case domain.GenerationStatusFailed:
		em := strings.TrimSpace(g.ErrorMessage)
		if em == "" {
			em = "配图生成失败"
		}
		return domain.PipelineStepFailed, em
	case domain.GenerationStatusProcessing, domain.GenerationStatusPending:
		return domain.PipelineStepRunning, ""
	case domain.GenerationStatusCompleted:
		if strings.TrimSpace(g.GeneratedImageURL) == "" {
			return domain.PipelineStepFailed, "配图结果为空"
		}
		return domain.PipelineStepCompleted, ""
	default:
		return domain.PipelineStepRunning, ""
	}
}

func suggestedResumeFromSteps(steps []domain.GenerationPipelineStep) string {
	for _, s := range steps {
		if s.Status != domain.PipelineStepFailed {
			continue
		}
		switch s.Phase {
		case domain.PipelinePhaseContent:
			return domain.SuggestedResumeRegenerateContent
		case domain.PipelinePhaseScenes:
			return domain.SuggestedResumeRegenerateScenes
		case domain.PipelinePhaseImages:
			return domain.SuggestedResumeRetryFailedImages
		}
	}
	return domain.SuggestedResumeNone
}
