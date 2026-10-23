package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"go.uber.org/zap"
)

type characterRecoveryImageProvider struct{ calls int }

func (p *characterRecoveryImageProvider) Name() string { return "huoshan" }
func (p *characterRecoveryImageProvider) GenerateImage(context.Context, *genapi.GenerateRequest) (*genapi.GenerateResponse, error) {
	p.calls++
	return &genapi.GenerateResponse{ImageURLs: []string{"https://stored/turnaround.png"}, Usage: &genapi.Usage{TotalTokens: common.AIImageBillingUnitTokens, ImageCount: 1}}, nil
}

type characterRecoveryImageRepo struct {
	domain.Repository
	character    *domain.Character
	record       *domain.AIGenerationRecord
	balance      int
	balanceReads int
	debits       map[string]bool
	failVisual   bool
}

func (r *characterRecoveryImageRepo) CharacterByID(context.Context, string) (*domain.Character, error) {
	raw, _ := json.Marshal(r.character)
	var copy domain.Character
	_ = json.Unmarshal(raw, &copy)
	return &copy, nil
}
func (r *characterRecoveryImageRepo) StoryByID(context.Context, string) (*domain.Story, error) {
	return &domain.Story{}, nil
}
func (r *characterRecoveryImageRepo) GetTokenBalance(context.Context, string) (int, error) {
	r.balanceReads++
	return r.balance, nil
}
func (r *characterRecoveryImageRepo) GetAIGenerationRecord(_ context.Context, id string) (*domain.AIGenerationRecord, error) {
	if r.record == nil || r.record.ID != id {
		return nil, domain.ErrNotFound
	}
	return r.record, nil
}
func (r *characterRecoveryImageRepo) CreateAIGenerationRecord(_ context.Context, record *domain.AIGenerationRecord) error {
	r.record = record
	return nil
}
func (r *characterRecoveryImageRepo) UpdateAIGenerationRecord(_ context.Context, record *domain.AIGenerationRecord) error {
	r.record = record
	return nil
}
func (r *characterRecoveryImageRepo) ApplyGenerationCharge(_ context.Context, user, operation, related, source string, tokens int) (*domain.TokenTransaction, error) {
	if !r.debits[operation] {
		r.debits[operation] = true
		r.balance -= tokens
	}
	return &domain.TokenTransaction{}, nil
}
func (r *characterRecoveryImageRepo) GenerationChargedTokens(context.Context, string, string) (int, error) {
	return len(r.debits) * common.AIImageBillingUnitTokens, nil
}
func (r *characterRecoveryImageRepo) UpdateCharacterVisualAssets(_ context.Context, _ *domain.CharacterGenerationTask, ch *domain.Character, _ map[string]any) error {
	if r.failVisual {
		return errors.New("simulated write failure after paid image")
	}
	copy := *ch
	r.character = &copy
	return nil
}
func TestCharacterThreeViewsRecoversPaidImageAtZeroBalanceWithoutProviderOrSecondDebit(t *testing.T) {
	provider := &characterRecoveryImageProvider{}
	registry := genapi.NewGenAPI()
	registry.RegisterProvider(provider)
	repo := &characterRecoveryImageRepo{character: &domain.Character{BaseModel: common.BaseModel{ID: "character"}, Author: &domain.User{BaseModel: common.BaseModel{ID: "owner"}}, Name: "雨", Portrait: "https://stored/portrait.png"}, balance: common.AIImageBillingUnitTokens, debits: map[string]bool{}, failVisual: true}
	svc := &Service{repo: repo, logger: zap.NewNop(), imageProvider: "huoshan", genAPI: registry, aiGenService: &AIGenerationService{repo: repo, logger: zap.NewNop(), genAPI: registry}}
	ctx := context.WithValue(context.Background(), characterTaskContextKey{}, &domain.CharacterGenerationTask{ID: "task"})
	if _, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", GenerateCharacterThreeViewsRequest{}); err == nil {
		t.Fatal("first visual persistence failure missing")
	} else {
		t.Log(err)
	}
	if provider.calls != 1 || repo.balance != 0 || len(repo.debits) != 1 {
		t.Fatal(provider.calls, repo.balance, repo.debits)
	}
	repo.failVisual = false
	repo.balanceReads = 0
	result, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", GenerateCharacterThreeViewsRequest{})
	if err != nil || result.Views.Sheet != "https://stored/turnaround.png" {
		t.Fatal(result, err)
	}
	if repo.balanceReads != 0 || provider.calls != 1 || repo.balance != 0 || len(repo.debits) != 1 {
		t.Fatal("paid result gated/regenerated/recharged", repo.balanceReads, provider.calls, repo.balance, repo.debits)
	}
}
