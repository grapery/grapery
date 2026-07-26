package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) CreateFragmentGenerationAssets(ctx context.Context, assets []*domain.FragmentGenerationAsset) error {
	if len(assets) == 0 {
		return nil
	}
	rows := make([]*FragmentGenerationAssetDB, 0, len(assets))
	scopes := make(map[string]domainFragmentAssetScope)
	for _, asset := range assets {
		row := domainFragmentGenerationAssetToDB(asset)
		if row != nil {
			rows = append(rows, row)
			key := domainBuildFragmentAssetScopeKey(row.FragmentID, row.Source, row.TaskID)
			scopes[key] = domainFragmentAssetScope{FragmentID: row.FragmentID, Source: row.Source, TaskID: row.TaskID}
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, scope := range scopes {
			q := tx.WithContext(ctx).Where("fragment_id = ? AND source = ?", scope.FragmentID, scope.Source)
			if scope.TaskID == "" {
				q = q.Where("task_id = '' OR task_id IS NULL")
			} else {
				q = q.Where("task_id = ?", scope.TaskID)
			}
			if err := q.Delete(&FragmentGenerationAssetDB{}).Error; err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).
			Clauses(clause.OnConflict{UpdateAll: true}).
			Create(&rows).Error
	})
}

func (r *Repository) ListFragmentGenerationAssets(ctx context.Context, fragmentID string) ([]*domain.FragmentGenerationAsset, error) {
	var rows []FragmentGenerationAssetDB
	if err := r.db.WithContext(ctx).
		Where("fragment_id = ?", fragmentID).
		Order("display_order ASC, scene_index ASC, created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.FragmentGenerationAsset, 0, len(rows))
	for i := range rows {
		out = append(out, domainFragmentGenerationAssetFromDB(&rows[i]))
	}
	return out, nil
}

func domainFragmentGenerationAssetToDB(asset *domain.FragmentGenerationAsset) *FragmentGenerationAssetDB {
	if asset == nil || asset.FragmentID == "" || asset.URL == "" || asset.Kind == "" {
		return nil
	}
	id := asset.ID
	if id == "" {
		id = stableDomainFragmentGenerationAssetID(asset)
	}
	metadata := asset.MetadataJSON
	if metadata == "" {
		metadata = "{}"
	}
	return &FragmentGenerationAssetDB{
		ID:           id,
		FragmentID:   asset.FragmentID,
		Source:       asset.Source,
		TaskID:       asset.TaskID,
		Kind:         asset.Kind,
		EntityKind:   asset.EntityKind,
		EntityKey:    asset.EntityKey,
		SlotID:       asset.SlotID,
		SlotIndex:    asset.SlotIndex,
		SceneIndex:   asset.SceneIndex,
		URL:          asset.URL,
		StorageKey:   asset.StorageKey,
		AspectRatio:  asset.AspectRatio,
		TokensUsed:   asset.TokensUsed,
		Provider:     asset.Provider,
		Model:        asset.Model,
		SeriesSeed:   asset.SeriesSeed,
		SceneSeed:    asset.SceneSeed,
		DisplayOrder: asset.DisplayOrder,
		MetadataJSON: metadata,
		CreatedAt:    asset.CreatedAt,
		UpdatedAt:    asset.UpdatedAt,
	}
}

func domainFragmentGenerationAssetFromDB(row *FragmentGenerationAssetDB) *domain.FragmentGenerationAsset {
	if row == nil {
		return nil
	}
	return &domain.FragmentGenerationAsset{
		ID:           row.ID,
		FragmentID:   row.FragmentID,
		Source:       row.Source,
		TaskID:       row.TaskID,
		Kind:         row.Kind,
		EntityKind:   row.EntityKind,
		EntityKey:    row.EntityKey,
		SlotID:       row.SlotID,
		SlotIndex:    row.SlotIndex,
		SceneIndex:   row.SceneIndex,
		URL:          row.URL,
		StorageKey:   row.StorageKey,
		AspectRatio:  row.AspectRatio,
		TokensUsed:   row.TokensUsed,
		Provider:     row.Provider,
		Model:        row.Model,
		SeriesSeed:   row.SeriesSeed,
		SceneSeed:    row.SceneSeed,
		DisplayOrder: row.DisplayOrder,
		MetadataJSON: row.MetadataJSON,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func stableDomainFragmentGenerationAssetID(asset *domain.FragmentGenerationAsset) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d|%s",
		asset.FragmentID, asset.Source, asset.TaskID, asset.Kind, asset.EntityKey,
		domainAssetSceneIndexValue(firstNonNilInt(asset.SlotIndex, asset.SceneIndex)), asset.DisplayOrder, asset.URL)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(raw)).String()
}

func domainAssetSceneIndexValue(idx *int) int {
	if idx == nil {
		return -1
	}
	return *idx
}

func firstNonNilInt(values ...*int) *int {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

type domainFragmentAssetScope struct {
	FragmentID string
	Source     string
	TaskID     string
}

func domainBuildFragmentAssetScopeKey(fragmentID, source, taskID string) string {
	return fragmentID + "|" + source + "|" + taskID
}
