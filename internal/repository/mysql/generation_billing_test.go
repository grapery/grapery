package mysql

import (
	"context"
	"fmt"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"testing"
)

func billingTestRepo(t *testing.T) *Repository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { sql.Close() })
	for _, schema := range []string{
		`CREATE TABLE memberships(id TEXT PRIMARY KEY,user_id TEXT UNIQUE,tier TEXT,status TEXT,start_date DATETIME,end_date DATETIME,auto_renew BOOLEAN,token_quota INTEGER,token_used INTEGER,storage_quota INTEGER,storage_used INTEGER,created_at DATETIME,updated_at DATETIME,deleted_at DATETIME)`,
		`CREATE TABLE token_transactions(id TEXT PRIMARY KEY,user_id TEXT,type TEXT,amount INTEGER,balance INTEGER,source TEXT,related_id TEXT,description TEXT,created_at DATETIME,deleted_at DATETIME)`,
	} {
		if err := db.Exec(schema).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&Membership{ID: "m", UserID: "u", Tier: "premium", TokenQuota: 20000}).Error; err != nil {
		t.Fatal(err)
	}
	return &Repository{db: db}
}
func TestGenerationChargeConcurrentReplayAndConflict(t *testing.T) {
	r := billingTestRepo(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.ApplyGenerationCharge(ctx, "u", "image-result", "task", "ai_image_generation", 4096); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var m Membership
	r.db.Where("user_id = ?", "u").First(&m)
	if m.TokenUsed != 4096 {
		t.Fatalf("duplicate debit: %d", m.TokenUsed)
	}
	if _, err := r.ApplyGenerationCharge(ctx, "u", "image-result", "task", "ai_image_generation", 8192); err == nil {
		t.Fatal("same operation with different intent accepted")
	}
	if tokens, err := r.GenerationChargedTokens(ctx, "u", "task"); err != nil || tokens != 4096 {
		t.Fatalf("scope total %d: %v", tokens, err)
	}
	if tokens, err := r.GenerationChargedTokens(ctx, "other", "task"); err != nil || tokens != 0 {
		t.Fatalf("cross-user cost leak: %d %v", tokens, err)
	}
}
func TestGenerationChargeInsufficientBalanceRollsBack(t *testing.T) {
	r := billingTestRepo(t)
	ctx := context.Background()
	if _, err := r.ApplyGenerationCharge(ctx, "u", "expensive", "task", "ai_image_generation", 20001); err == nil {
		t.Fatal("overspend accepted")
	}
	var count int64
	r.db.Model(&TokenTransaction{}).Count(&count)
	var m Membership
	r.db.Where("user_id = ?", "u").First(&m)
	if count != 0 || m.TokenUsed != 0 {
		t.Fatalf("failed charge mutated ledger or balance: %d %d", count, m.TokenUsed)
	}
}
func TestFragmentStoredQualityCannotBeErasedOnPublish(t *testing.T) {
	r := billingTestRepo(t)
	if err := r.db.Exec(`CREATE TABLE fragments(id TEXT PRIMARY KEY, generation_metadata TEXT, deleted_at DATETIME)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := r.db.Exec(`INSERT INTO fragments(id,generation_metadata) VALUES(?,?)`, "draft", `{"quality":{"status":"unreviewed"}}`).Error; err != nil {
		t.Fatal(err)
	}
	err := r.UpdateFragment(context.Background(), &domain.Fragment{BaseModel: common.BaseModel{ID: "draft"}, IsDraft: false, GenerationMetadata: `{"quality":{"status":"passed"}}`})
	if err == nil {
		t.Fatal("forged client approval bypassed stored quality")
	}
	var metadata string
	r.db.Model(&FragmentDB{}).Where("id = ?", "draft").Select("generation_metadata").Scan(&metadata)
	if metadata != `{"quality":{"status":"unreviewed"}}` {
		t.Fatalf("rejected publication mutated draft: %s", metadata)
	}
}

func TestGenerationChargeLedgerWriteFailureRollsBackBalance(t *testing.T) {
	r := billingTestRepo(t)
	if err := r.db.Exec(`CREATE TRIGGER reject_charge BEFORE INSERT ON token_transactions BEGIN SELECT RAISE(ABORT, 'ledger unavailable'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApplyGenerationCharge(context.Background(), "u", "result", "task", "fragment_generation_text", 1024); err == nil {
		t.Fatal("ledger failure hidden")
	}
	var m Membership
	r.db.Where("user_id = ?", "u").First(&m)
	if m.TokenUsed != 0 {
		t.Fatalf("balance charged without ledger: %d", m.TokenUsed)
	}
}

func TestFailedImageOperationHasOneRetryClaim(t *testing.T) {
	r := billingTestRepo(t)
	if err := r.db.Exec(`CREATE TABLE ai_generation_records(id TEXT PRIMARY KEY,status TEXT,updated_at DATETIME,deleted_at DATETIME)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := r.db.Exec(`INSERT INTO ai_generation_records(id,status) VALUES('failed','failed'),('unknown','processing')`).Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	claims := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := r.ClaimFailedGenerationImage(context.Background(), "failed")
			if err != nil {
				t.Error(err)
			}
			if ok {
				mu.Lock()
				claims++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if claims != 1 {
		t.Fatalf("claimed %d retries", claims)
	}
	if ok, err := r.ClaimFailedGenerationImage(context.Background(), "unknown"); err != nil || ok {
		t.Fatalf("unknown in-flight provider outcome retried: %v %v", ok, err)
	}
}
