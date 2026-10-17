package service

import (
	"context"
	"fmt"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"strings"
)

func applyGenerationCharge(ctx context.Context, repo domain.Repository, userID, operationID, relatedID, source string, tokens int) (*domain.TokenTransaction, error) {
	ledger, ok := repo.(domain.GenerationChargeRepository)
	if !ok {
		return nil, fmt.Errorf("generation charge ledger unavailable")
	}
	return ledger.ApplyGenerationCharge(ctx, userID, operationID, relatedID, source, tokens)
}

// This is an estimate at the existing image tariff, not a reservation or a final bill.
func (s *FragmentGenerationService) ensureFragmentRenderBalance(ctx context.Context, userID string, scenes []domain.FragmentScenePlan, assets []domain.FragmentReferenceAsset) error {
	if domain.IsSponsoredGeneration(ctx) {
		return nil
	}
	remaining := 0
	for _, scene := range scenes {
		if strings.TrimSpace(scene.GeneratedImageURL) != "" {
			continue
		}
		if scene.ComicPage != nil && len(scene.ComicPage.Panels) > 0 {
			for i := range scene.ComicPage.Panels {
				if i >= len(scene.PanelImageURLs) || strings.TrimSpace(scene.PanelImageURLs[i]) == "" {
					remaining++
				}
			}
		} else {
			remaining++
		}
	}
	if remaining == 0 {
		return nil
	}
	balance, err := s.repo.GetTokenBalance(ctx, userID)
	if err != nil {
		return err
	}
	estimate := remaining * common.AIImageBillingUnitTokens
	if balance < estimate {
		return fmt.Errorf("insufficient token balance for remaining panels: estimated %d tokens, available %d", estimate, balance)
	}
	return nil
}

type FragmentGenerationCostSnapshot struct {
	Amount *float64 `json:"amount,omitempty"`
	Tokens *int     `json:"tokens,omitempty"`
	Unit   string   `json:"unit"`
	State  string   `json:"state"`
	Text   string   `json:"text"`
}

func (s *FragmentGenerationService) GenerationCostSnapshot(ctx context.Context, task *domain.FragmentGenerationTask) FragmentGenerationCostSnapshot {
	result := FragmentGenerationCostSnapshot{Unit: "点数", State: "unavailable", Text: "历史费用暂无法核对"}
	if task == nil || task.Request.BillingVersion < 1 {
		return result
	}
	ledger, ok := s.repo.(domain.GenerationChargeRepository)
	if !ok {
		result.Text = "费用核对中"
		return result
	}
	tokens, err := ledger.GenerationChargedTokens(ctx, task.UserID, task.ID)
	if err != nil {
		result.Text = "费用核对中"
		return result
	}
	points := float64(tokens) / common.CreditToTokenRatio
	result.Amount = &points
	result.Tokens = &tokens
	result.State = "in_progress"
	result.Text = fmt.Sprintf("已结算 %.4f 点数 · 最终以账单为准", points)
	if string(task.Status) == "completed" || string(task.Status) == "failed" || string(task.Status) == "cancelled" {
		result.State = "recorded"
	}
	if strings.HasPrefix(task.ErrorMessage, "billing_unsettled:") || (task.Result != nil && task.Result.GenerationTrace != nil && task.Result.GenerationTrace.BillingStatus == "pending") {
		result.State = "unsettled"
		result.Text = fmt.Sprintf("已结算 %.4f 点数 · 还有费用待核对", points)
	}
	return result
}
