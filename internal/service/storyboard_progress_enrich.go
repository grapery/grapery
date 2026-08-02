package service

import (
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// enrichStoryboardProgressFineGrain fills stepKey/messageKey/stage/progressPercent
// from latestRun + pipelineSteps without changing legacy CurrentStep int.
func enrichStoryboardProgressFineGrain(progress *domain.StoryboardGenerationProgress) {
	if progress == nil {
		return
	}
	stepKey := ""
	percent := 0
	if progress.LatestRun != nil {
		stepKey = strings.TrimSpace(progress.LatestRun.CurrentStep)
		if progress.LatestRun.Progress > 0 {
			percent = progress.LatestRun.Progress
		}
		if strings.EqualFold(progress.LatestRun.Status, domain.PipelineStepCompleted) ||
			strings.EqualFold(progress.LatestRun.Status, "succeeded") {
			if stepKey == "" {
				stepKey = domain.StoryboardGenerationStepConsistency
			}
			if percent < 100 {
				percent = 100
			}
		}
	}
	if stepKey == "" {
		stepKey = deriveStoryboardStepKeyFromPipeline(progress.PipelineSteps, progress.IsGenerating)
	}
	if percent <= 0 {
		percent = deriveStoryboardProgressPercent(progress.PipelineSteps, progress.LatestRun, progress.IsGenerating)
	}
	progress.StepKey = stepKey
	progress.MessageKey = storyboardGenerationStepMessageKey(stepKey)
	progress.Stage = storyboardGenerationStage(stepKey, progress.IsGenerating, progress.WorkflowStatus)
	progress.ProgressPercent = percent
}

func deriveStoryboardStepKeyFromPipeline(steps []domain.GenerationPipelineStep, isGenerating bool) string {
	if len(steps) == 0 {
		if isGenerating {
			return domain.StoryboardGenerationStepContext
		}
		return ""
	}
	for _, step := range steps {
		if step.Status == domain.PipelineStepRunning || step.Status == domain.PipelineStepPending {
			return mapPipelinePhaseToStepKey(step.Phase)
		}
	}
	// All done — last phase maps to consistency if images done.
	last := steps[len(steps)-1]
	if last.Status == domain.PipelineStepCompleted {
		return domain.StoryboardGenerationStepConsistency
	}
	return mapPipelinePhaseToStepKey(last.Phase)
}

func mapPipelinePhaseToStepKey(phase string) string {
	switch strings.TrimSpace(phase) {
	case domain.PipelinePhaseContent:
		return domain.StoryboardGenerationStepBiblePlan
	case domain.PipelinePhaseScenes:
		return domain.StoryboardGenerationStepScenePlan
	case domain.PipelinePhaseImages:
		return domain.StoryboardGenerationStepImage
	case domain.PipelinePhaseAudit:
		return domain.StoryboardGenerationStepConsistency
	default:
		return domain.StoryboardGenerationStepContext
	}
}

func deriveStoryboardProgressPercent(steps []domain.GenerationPipelineStep, run *domain.StoryboardGenerationRun, isGenerating bool) int {
	if run != nil && run.Progress > 0 {
		return clampPercent(run.Progress)
	}
	if len(steps) == 0 {
		if isGenerating {
			return 5
		}
		return 0
	}
	completed := 0
	runningBonus := 0
	for _, step := range steps {
		switch step.Status {
		case domain.PipelineStepCompleted, domain.PipelineStepSkipped:
			completed++
		case domain.PipelineStepRunning:
			runningBonus = 1
		}
	}
	total := len(steps)
	if total == 0 {
		return 0
	}
	pct := (completed * 100) / total
	if runningBonus > 0 && pct < 95 {
		pct += 10 / total
		if pct < 5 {
			pct = 5
		}
	}
	return clampPercent(pct)
}

func clampPercent(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func storyboardGenerationStepMessageKey(stepKey string) string {
	switch strings.TrimSpace(stepKey) {
	case domain.StoryboardGenerationStepContext:
		return "storyboard_generation_reading_context"
	case domain.StoryboardGenerationStepBiblePlan:
		return "storyboard_generation_planning_bible"
	case domain.StoryboardGenerationStepScenePlan:
		return "storyboard_generation_writing_scenes"
	case domain.StoryboardGenerationStepImage:
		return "storyboard_generation_generating_images"
	case domain.StoryboardGenerationStepConsistency:
		return "storyboard_generation_checking_consistency"
	default:
		if stepKey == "" {
			return "storyboard_generation_preparing"
		}
		return "storyboard_generation_in_progress"
	}
}

func storyboardGenerationStage(stepKey string, isGenerating bool, workflowStatus string) string {
	if workflowStatus == domain.WorkflowStatusPublished ||
		workflowStatus == domain.WorkflowStatusImagesReady ||
		workflowStatus == domain.WorkflowStatusVideoReady {
		if !isGenerating {
			return "completed"
		}
	}
	switch strings.TrimSpace(stepKey) {
	case domain.StoryboardGenerationStepContext, domain.StoryboardGenerationStepBiblePlan:
		return "outline"
	case domain.StoryboardGenerationStepScenePlan:
		return "scenes"
	case domain.StoryboardGenerationStepImage:
		return "images"
	case domain.StoryboardGenerationStepConsistency:
		return "review"
	default:
		return "preparing"
	}
}
