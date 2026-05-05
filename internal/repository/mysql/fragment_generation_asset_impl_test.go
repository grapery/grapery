package mysql

import (
	"context"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateFragmentGenerationAssets_ReplacesByFragmentSourceTask(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&FragmentGenerationAssetDB{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := NewRepository(db, zap.NewNop(), config.RecommendationConfig{})
	ctx := context.Background()

	scopeA1 := []*domain.FragmentGenerationAsset{
		{FragmentID: "f1", Source: "panel_generation", TaskID: "t1", Kind: domain.FragmentGenerationAssetKindSceneFinal, URL: "https://old.example/1.png"},
		{FragmentID: "f1", Source: "panel_generation", TaskID: "t1", Kind: domain.FragmentGenerationAssetKindSceneFinal, URL: "https://old.example/2.png"},
	}
	if err := repo.CreateFragmentGenerationAssets(ctx, scopeA1); err != nil {
		t.Fatalf("create initial: %v", err)
	}
	scopeOther := []*domain.FragmentGenerationAsset{
		{FragmentID: "f1", Source: "panel_generation", TaskID: "t2", Kind: domain.FragmentGenerationAssetKindSceneFinal, URL: "https://keep.example/1.png"},
	}
	if err := repo.CreateFragmentGenerationAssets(ctx, scopeOther); err != nil {
		t.Fatalf("create other scope: %v", err)
	}

	scopeA2 := []*domain.FragmentGenerationAsset{
		{FragmentID: "f1", Source: "panel_generation", TaskID: "t1", Kind: domain.FragmentGenerationAssetKindSceneFinal, URL: "https://new.example/1.png"},
	}
	if err := repo.CreateFragmentGenerationAssets(ctx, scopeA2); err != nil {
		t.Fatalf("create replacement: %v", err)
	}

	items, err := repo.ListFragmentGenerationAssets(ctx, "f1")
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	var oldCount, newCount, keepCount int
	for _, item := range items {
		switch item.URL {
		case "https://old.example/1.png", "https://old.example/2.png":
			oldCount++
		case "https://new.example/1.png":
			newCount++
		case "https://keep.example/1.png":
			keepCount++
		}
	}
	if oldCount != 0 {
		t.Fatalf("expected old scoped rows deleted, got %d", oldCount)
	}
	if newCount != 1 {
		t.Fatalf("expected one new row, got %d", newCount)
	}
	if keepCount != 1 {
		t.Fatalf("expected other scope row kept, got %d", keepCount)
	}
}
