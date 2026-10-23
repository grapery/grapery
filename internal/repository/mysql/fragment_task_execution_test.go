package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFragmentAssetsRejectStaleLeaseBeforeReplacingCurrentAssets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{IgnoreRelationshipsWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&FragmentGenerationAssetDB{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(`CREATE TABLE fragment_generation_tasks (id TEXT PRIMARY KEY,status TEXT,lease_owner TEXT,lease_fence INTEGER,lease_expires_at INTEGER)`).Error; err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(time.Minute).UnixMilli()
	if err = db.Exec(`INSERT INTO fragment_generation_tasks(id,status,lease_owner,lease_fence,lease_expires_at) VALUES('task','processing','fresh',2,?)`, expires).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db, zap.NewNop(), config.RecommendationConfig{})
	ctx := context.Background()
	current := domain.WithFragmentTaskLease(ctx, &domain.FragmentGenerationTask{ID: "task", LeaseOwner: "fresh", LeaseFence: 2})
	stale := domain.WithFragmentTaskLease(ctx, &domain.FragmentGenerationTask{ID: "task", LeaseOwner: "old", LeaseFence: 1})
	asset := func(url string) []*domain.FragmentGenerationAsset {
		return []*domain.FragmentGenerationAsset{{FragmentID: "draft", TaskID: "task", Source: "ai_generation", Kind: domain.FragmentGenerationAssetKindSceneFinal, URL: url}}
	}
	if err = repo.CreateFragmentGenerationAssets(current, asset("fresh")); err != nil {
		t.Fatal(err)
	}
	if err = repo.CreateFragmentGenerationAssets(stale, asset("stale")); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("stale asset replaced current asset", err)
	}
	wrongTask := asset("other")
	wrongTask[0].TaskID = "other"
	if err = repo.CreateFragmentGenerationAssets(current, wrongTask); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("cross-task asset accepted", err)
	}
	assets, err := repo.ListFragmentGenerationAssets(ctx, "draft")
	if err != nil || len(assets) != 1 || assets[0].URL != "fresh" {
		t.Fatalf("stale asset side effect: %#v %v", assets, err)
	}
}
