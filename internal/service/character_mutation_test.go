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

func characterDirectVisualFixture() (*Service, *characterRecoveryImageRepo, *characterRecoveryImageProvider) {
	provider := &characterRecoveryImageProvider{}
	registry := genapi.NewGenAPI()
	registry.RegisterProvider(provider)
	repo := &characterRecoveryImageRepo{character: &domain.Character{BaseModel: common.BaseModel{ID: "character"}, StoryID: "story", Name: "雨", Author: &domain.User{BaseModel: common.BaseModel{ID: "owner"}}, Portrait: "https://stored/old-portrait.png"}, balance: common.AIImageBillingUnitTokens * 3, debits: map[string]bool{}}
	svc := &Service{repo: repo, logger: zap.NewNop(), imageProvider: "huoshan", genAPI: registry, aiGenService: &AIGenerationService{repo: repo, logger: zap.NewNop(), genAPI: registry}}
	return svc, repo, provider
}
func TestCharacterDirectPortraitResponseLostRetryReusesFrozenInputAndSingleDebit(t *testing.T) {
	svc, repo, provider := characterDirectVisualFixture()
	ctx := context.Background()
	req := GenerateCharacterPortraitRequest{ClientRequestID: "one-user-action", AspectRatio: "2:3"}
	first, err := svc.GenerateCharacterPortrait(ctx, "owner", "character", req)
	if err != nil {
		t.Fatal(err)
	}
	if repo.character.Portrait != "https://stored/turnaround.png" {
		t.Fatal(repo.character.Portrait)
	}
	repo.balance = 0
	repo.balanceReads = 0
	second, err := svc.GenerateCharacterPortrait(ctx, "owner", "character", req)
	if err != nil {
		t.Fatal("response-lost retry rejected new self-produced reference", err)
	}
	if first.RecordID != second.RecordID || first.PortraitURL != second.PortraitURL || provider.calls != 1 || len(repo.debits) != 1 || repo.balanceReads != 0 {
		t.Fatal(first, second, provider.calls, repo.debits, repo.balanceReads)
	}
	req.CustomPrompt = "different action with stolen request key"
	if _, err := svc.GenerateCharacterPortrait(ctx, "owner", "character", req); !errors.Is(err, domain.ErrCharacterTaskRequestConflict) {
		t.Fatal("changed intent accepted", err)
	}
	if provider.calls != 1 || len(repo.debits) != 1 {
		t.Fatal("conflicting intent generated/charged")
	}
}
func TestCharacterDirectThreeViewsResponseLostReplayAndChangedIntentConflict(t *testing.T) {
	svc, repo, provider := characterDirectVisualFixture()
	ctx := context.Background()
	req := GenerateCharacterThreeViewsRequest{ClientRequestID: "one-sheet-action", RegenerateAll: true}
	first, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", req)
	if err != nil {
		t.Fatal(err)
	}
	repo.balance = 0
	repo.balanceReads = 0
	second, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Views.Sheet != second.Views.Sheet || provider.calls != 1 || len(repo.debits) != 1 || repo.balanceReads != 0 {
		t.Fatal(provider.calls, repo.balanceReads)
	}
	req.ReferenceImage = "https://different-ref"
	if _, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", req); !errors.Is(err, domain.ErrCharacterTaskRequestConflict) {
		t.Fatal(err)
	}
}
func TestCharacterDirectThreeViewsRecoversPaidButUnattachedImageAtZeroBalance(t *testing.T) {
	svc, repo, provider := characterDirectVisualFixture()
	ctx := context.Background()
	req := GenerateCharacterThreeViewsRequest{ClientRequestID: "recover-paid-image"}
	repo.failVisual = true
	if _, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", req); err == nil {
		t.Fatal("missing persistence failure")
	}
	repo.failVisual = false
	repo.balance = 0
	repo.balanceReads = 0
	if _, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", req); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 || len(repo.debits) != 1 || repo.balanceReads != 0 {
		t.Fatal(provider.calls, repo.debits, repo.balanceReads)
	}
}
func TestCharacterDirectVisualNewActionKeyPermitsIntentionalRegeneration(t *testing.T) {
	svc, repo, provider := characterDirectVisualFixture()
	ctx := context.Background()
	for _, key := range []string{"first-action", "second-action"} {
		if _, err := svc.GenerateCharacterPortrait(ctx, "owner", "character", GenerateCharacterPortraitRequest{ClientRequestID: key}); err != nil {
			t.Fatal(err)
		}
	}
	if provider.calls != 2 || len(repo.debits) != 2 {
		t.Fatal("intentional regeneration deduped", provider.calls, repo.debits)
	}
}
func TestCharacterCropAvatarRejectsNonOwnerBeforeAnyWrite(t *testing.T) {
	repo := &characterPromptTestRepo{character: &domain.Character{BaseModel: common.BaseModel{ID: "character"}, Author: &domain.User{BaseModel: common.BaseModel{ID: "owner"}}, Portrait: "portrait", Avatar: "owner-avatar"}}
	svc := &Service{repo: repo, logger: zap.NewNop()}
	if _, err := svc.CropAvatarFromPortrait(context.Background(), "attacker", "character"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal(err)
	}
	if repo.character.Avatar != "owner-avatar" {
		t.Fatal("attacker changed avatar")
	}
}
func TestCharacterPartialRetryUsesCurrentConfirmedFieldsWithoutPaidRefill(t *testing.T) {
	var attrs GeneratedCharacterAttributes
	if err := json.Unmarshal([]byte(validCharacterAttributeJSON()), &attrs); err != nil {
		t.Fatal(err)
	}
	ch := &domain.Character{BaseModel: common.BaseModel{ID: "character"}, StoryID: "story", Author: &domain.User{BaseModel: common.BaseModel{ID: "owner"}}, Name: "owner rename", Portrait: "portrait", Views: &domain.CharacterThreeViews{Sheet: "sheet"}}
	applyCharacterAttributes(ch, attrs)
	repo := &characterFilledRetryRepo{characterPromptTestRepo: &characterPromptTestRepo{character: ch}}
	svc := &Service{repo: repo, logger: zap.NewNop()} // nil model: a paid refill would fail.
	task := &domain.CharacterGenerationTask{ID: "task", UserID: "owner", StoryID: "story", CharacterID: "character", RequestJSON: `{"name":"old name","description":"old description"}`, Stages: map[string]domain.CharacterGenerationStage{"attributes": {Status: "failed", Attempts: 1}}}
	if err := svc.executeCharacterGenerationTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.CharacterGenerationStatusCompleted || task.Stages["attributes"].Attempts != 1 || repo.character.Name != "owner rename" || repo.character.Description != attrs.Description {
		t.Fatal(task, repo.character)
	}
}

type characterUnknownRetryRepo struct {
	domain.Repository
	domain.CharacterTaskExecutionRepository
	task   *domain.CharacterGenerationTask
	queued int
}

func (r *characterUnknownRetryRepo) CharacterGenerationTaskByID(context.Context, string) (*domain.CharacterGenerationTask, error) {
	return r.task, nil
}
func (r *characterUnknownRetryRepo) QueueCharacterTaskRetry(context.Context, string, string) (bool, error) {
	r.queued++
	return true, nil
}
func TestCharacterUnknownStageRetryRejectedBeforeQueue(t *testing.T) {
	repo := &characterUnknownRetryRepo{task: &domain.CharacterGenerationTask{ID: "task", UserID: "owner", Status: domain.CharacterGenerationStatusPartial, Stages: map[string]domain.CharacterGenerationStage{"attributes": {Status: "failed", OutcomeUnknown: true}}}}
	svc := &Service{repo: repo}
	if _, err := svc.RetryCharacterGenerationTask(context.Background(), "owner", "task"); !errors.Is(err, domain.ErrCharacterStageOutcomeUnknown) {
		t.Fatal(err)
	}
	if repo.queued != 0 {
		t.Fatal("unknown provider outcome queued again")
	}
}

type characterFilledRetryRepo struct{ *characterPromptTestRepo }

func (r *characterFilledRetryRepo) CreateNotification(context.Context, *domain.Notification) error {
	return errors.New("test does not dispatch notifications")
}

func TestCharacterDirectVisualUnknownLedgerOutcomeNeverResendsProvider(t *testing.T) {
	svc, repo, provider := characterDirectVisualFixture()
	ctx := context.Background()
	req := GenerateCharacterThreeViewsRequest{ClientRequestID: "unknown-outcome"}
	repo.failVisual = true
	if _, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", req); err == nil {
		t.Fatal("missing simulated persistence error")
	}
	repo.failVisual = false
	// Reproduce a durable in-flight entry with no known output. Repeated retries
	// must stop even when the balance could pay for another provider call.
	repo.record.Status = domain.AITaskStatusProcessing
	repo.record.OutputResult = "{}"
	for i := 0; i < 2; i++ {
		if _, err := svc.GenerateCharacterThreeViews(ctx, "owner", "character", req); err == nil {
			t.Fatal("unknown operation was replayed")
		}
	}
	if provider.calls != 1 || len(repo.debits) != 1 {
		t.Fatal("unknown operation regenerated or recharged", provider.calls, repo.debits)
	}
}
