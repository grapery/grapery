package service

import (
	"context"
	"errors"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"testing"
)

type imageSettlementRepo struct {
	domain.Repository
	record      *domain.AIGenerationRecord
	calls       int
	chargeError error
	tokens      int
}

func (r *imageSettlementRepo) GetAIGenerationRecord(context.Context, string) (*domain.AIGenerationRecord, error) {
	return r.record, nil
}
func (r *imageSettlementRepo) ApplyGenerationCharge(_ context.Context, user, op, related, source string, tokens int) (*domain.TokenTransaction, error) {
	r.calls++
	r.tokens = tokens
	return &domain.TokenTransaction{}, r.chargeError
}
func (r *imageSettlementRepo) GenerationChargedTokens(context.Context, string, string) (int, error) {
	return r.tokens, nil
}
func TestImageSettlementRecoversStoredOutputWithoutProvider(t *testing.T) {
	req := &GenerateImageRequest{UserID: "u", OperationID: "stable-op", Prompt: "accepted", RelatedEntityID: "task", OutputCount: 1}
	record := &domain.AIGenerationRecord{ID: "record", UserID: "u", OriginalPrompt: req.Prompt, RelatedEntityID: "task", TotalTokens: 4096, OutputResult: `{"imageURLs":["saved.png"]}`, Metadata: map[string]interface{}{"operationFingerprint": imageOperationFingerprint(req), "sponsored": false}}
	repo := &imageSettlementRepo{record: record}
	svc := &AIGenerationService{repo: repo}
	result, err := svc.GenerateImage(context.Background(), req)
	if err != nil || result.ImageURLs[0] != "saved.png" || repo.calls != 1 {
		t.Fatalf("recovery %v %v calls=%d", result, err, repo.calls)
	}
	repo.chargeError = errors.New("balance unavailable")
	if _, err := svc.GenerateImage(context.Background(), req); err == nil {
		t.Fatal("settlement failure hidden")
	}
	var billing *domain.GenerationBillingError
	_, err = svc.GenerateImage(context.Background(), req)
	if !errors.As(err, &billing) {
		t.Fatal(err)
	}
	req.ReferenceImages = []string{"different-identity.png"}
	before := repo.calls
	if _, err := svc.GenerateImage(context.Background(), req); err == nil || repo.calls != before {
		t.Fatal("changed intent reused old operation")
	}
}
func TestImageSponsoredReplayCannotBecomeUserCharge(t *testing.T) {
	req := &GenerateImageRequest{UserID: "u", OperationID: "repair", Prompt: "accepted"}
	repo := &imageSettlementRepo{record: &domain.AIGenerationRecord{UserID: "u", OriginalPrompt: req.Prompt, OutputResult: `{"imageURLs":["saved.png"]}`, Metadata: map[string]interface{}{"operationFingerprint": imageOperationFingerprint(req), "sponsored": true}}}
	svc := &AIGenerationService{repo: repo}
	if _, err := svc.GenerateImage(domain.WithSponsoredGeneration(context.Background()), req); err != nil || repo.calls != 0 {
		t.Fatalf("sponsored charged: %v", err)
	}
	if _, err := svc.GenerateImage(context.Background(), req); err == nil {
		t.Fatal("sponsorship switch accepted")
	}
}
func TestHistoricalFragmentCostIsUnknownNotInvented(t *testing.T) {
	svc := &FragmentGenerationService{}
	snapshot := svc.GenerationCostSnapshot(context.Background(), &domain.FragmentGenerationTask{})
	if snapshot.Amount != nil || snapshot.State != "unavailable" {
		t.Fatalf("invented historical charge: %+v", snapshot)
	}
}
