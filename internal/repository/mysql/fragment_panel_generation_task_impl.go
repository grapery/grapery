package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func (r *Repository) GetFragmentPanelGenerationTask(ctx context.Context, taskID string) (*domain.FragmentPanelGenerationTask, error) {
	var row FragmentPanelGenerationTaskDB
	if err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&row).Error; err != nil {
		return nil, err
	}
	return panelTaskDomainFromDB(&row)
}

func panelTaskDomainFromDB(row *FragmentPanelGenerationTaskDB) (*domain.FragmentPanelGenerationTask, error) {
	task := &domain.FragmentPanelGenerationTask{
		ID:              row.ID,
		UserID:          row.UserID,
		Status:          row.Status,
		Progress:        row.Progress,
		CurrentStep:     row.CurrentStep,
		ErrorMessage:    row.ErrorMessage,
		DraftFragmentID: row.DraftFragmentID,
		CreatedAt:       row.CreatedAt,
		StartedAt:       row.StartedAt,
		CompletedAt:     row.CompletedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if strings.TrimSpace(row.RequestJSON) != "" {
		if err := json.Unmarshal([]byte(row.RequestJSON), &task.Request); err != nil {
			return nil, fmt.Errorf("unmarshal panel task request: %w", err)
		}
	}
	if strings.TrimSpace(row.PlanJSON) != "" {
		plan, bible, evidence, anchors, err := unmarshalPanelTaskPlanJSON(row.PlanJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshal panel task plan: %w", err)
		}
		task.Plan = plan
		task.VisualBible = bible
		task.VisualEvidence = evidence
		task.AnchorImages = anchors
	}
	if strings.TrimSpace(row.ResultJSON) != "" {
		var result domain.FragmentPanelResultData
		if err := json.Unmarshal([]byte(row.ResultJSON), &result); err != nil {
			return nil, fmt.Errorf("unmarshal panel task result: %w", err)
		}
		task.Result = &result
	}
	if strings.TrimSpace(row.MetricsJSON) != "" {
		var metrics domain.FragmentPanelMetricsData
		if err := json.Unmarshal([]byte(row.MetricsJSON), &metrics); err != nil {
			return nil, fmt.Errorf("unmarshal panel task metrics: %w", err)
		}
		task.Metrics = &metrics
	}
	return task, nil
}

func unmarshalPanelTaskPlanJSON(raw string) ([]domain.FragmentPanelPlanItem, *domain.FragmentVisualBible, []domain.FragmentVisualEvidence, []domain.FragmentAnchorImage, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil, nil, nil, nil
	}
	var payload struct {
		VisualBible    *domain.FragmentVisualBible     `json:"visualBible,omitempty"`
		VisualEvidence []domain.FragmentVisualEvidence `json:"visualEvidence,omitempty"`
		AnchorImages   []domain.FragmentAnchorImage    `json:"anchorImages,omitempty"`
		Panels         []domain.FragmentPanelPlanItem  `json:"panels"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err == nil && (len(payload.Panels) > 0 || payload.VisualBible != nil) {
		return payload.Panels, payload.VisualBible, payload.VisualEvidence, payload.AnchorImages, nil
	}
	var legacy []domain.FragmentPanelPlanItem
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return nil, nil, nil, nil, err
	}
	return legacy, nil, nil, nil, nil
}
