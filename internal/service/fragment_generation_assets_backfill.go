package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

type fragmentPanelTaskGetter interface {
	GetFragmentPanelGenerationTask(ctx context.Context, taskID string) (*domain.FragmentPanelGenerationTask, error)
}

// BackfillFragmentGenerationAssets best-effort backfill for legacy rows.
func (s *Service) BackfillFragmentGenerationAssets(ctx context.Context, fragmentID string) (int, error) {
	fragment, err := s.repo.FragmentByID(ctx, fragmentID)
	if err != nil {
		return 0, err
	}
	if fragment == nil {
		return 0, domain.ErrNotFound
	}
	assets, err := s.backfillFragmentGenerationAssets(ctx, fragment)
	if err != nil {
		return 0, err
	}
	return len(assets), nil
}

func (s *Service) backfillFragmentGenerationAssets(ctx context.Context, fragment *domain.Fragment) ([]*domain.FragmentGenerationAsset, error) {
	if s == nil || s.repo == nil || fragment == nil || strings.TrimSpace(fragment.ID) == "" {
		return nil, nil
	}
	assets := s.buildFragmentAssetsFromLegacy(fragment)
	if len(assets) == 0 && strings.TrimSpace(fragment.SourceType) == string(domain.FragmentSourcePanelGeneration) {
		assets = s.buildFragmentAssetsFromPanelTask(ctx, fragment)
	}
	if len(assets) == 0 {
		return nil, nil
	}
	if err := s.repo.CreateFragmentGenerationAssets(ctx, assets); err != nil {
		return nil, err
	}
	items, err := s.repo.ListFragmentGenerationAssets(ctx, fragment.ID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) buildFragmentAssetsFromLegacy(fragment *domain.Fragment) []*domain.FragmentGenerationAsset {
	if fragment == nil {
		return nil
	}
	var trace domain.FragmentGenerationTrace
	raw := strings.TrimSpace(fragment.GenerationMetadata)
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &trace)
	}
	taskID := strings.TrimSpace(fragment.GenerationTaskID)
	if taskID == "" {
		taskID = strings.TrimSpace(fragment.SourceID)
	}
	result := &domain.FragmentGenerationResult{
		ImageUrls:         fragmentMediaURLs(fragment),
		VisualBible:       trace.VisualBible,
		ReferenceAssets:   trace.ReferenceAssets,
		ScenePlan:         trace.Scenes,
		ConsistencyPolicy: trace.ConsistencyPolicy,
		GenerationTrace:   &trace,
	}
	assets := buildFragmentGenerationAssets(
		fragment.ID,
		taskID,
		fragmentAssetSourceForFragment(fragment),
		"",
		nil,
		result,
		nil,
	)
	if len(assets) > 0 {
		return assets
	}
	urls := fragmentMediaURLs(fragment)
	out := make([]*domain.FragmentGenerationAsset, 0, len(urls))
	for i, u := range urls {
		if strings.TrimSpace(u) == "" {
			continue
		}
		idx := i
		out = append(out, &domain.FragmentGenerationAsset{
			FragmentID:   fragment.ID,
			Source:       fragmentAssetSourceForFragment(fragment),
			TaskID:       taskID,
			Kind:         domain.FragmentGenerationAssetKindSceneFinal,
			EntityKind:   domain.FragmentGenerationAssetEntityScene,
			SceneIndex:   &idx,
			URL:          strings.TrimSpace(u),
			DisplayOrder: 1000 + i,
			MetadataJSON: `{"tokensScope":"unknown"}`,
		})
	}
	return out
}

func (s *Service) buildFragmentAssetsFromPanelTask(ctx context.Context, fragment *domain.Fragment) []*domain.FragmentGenerationAsset {
	if s == nil || fragment == nil {
		return nil
	}
	getter, ok := s.repo.(fragmentPanelTaskGetter)
	if !ok {
		return nil
	}
	sourceID := strings.TrimSpace(fragment.SourceID)
	if sourceID == "" {
		return nil
	}
	task, err := getter.GetFragmentPanelGenerationTask(ctx, sourceID)
	if err != nil || task == nil {
		if err != nil {
			s.logger.Warn("panel task fallback lookup failed", zap.String("fragmentID", fragment.ID), zap.String("sourceID", sourceID), zap.Error(err))
		}
		return nil
	}
	result := &domain.FragmentGenerationResult{
		AspectRatio:       task.Request.AspectRatio,
		VisualBible:       task.VisualBible,
		AnchorImages:      task.AnchorImages,
		ConsistencyPolicy: nil,
	}
	var panels []domain.FragmentPanelResultItem
	if task.Result != nil {
		result.ImageUrls = panelResultImageURLs(task.Result.Panels)
		if task.Result.VisualBible != nil && result.VisualBible == nil {
			result.VisualBible = task.Result.VisualBible
		}
		if len(task.Result.AnchorImages) > 0 && len(result.AnchorImages) == 0 {
			result.AnchorImages = task.Result.AnchorImages
		}
		result.GenerationTrace = task.Result.GenerationTrace
		panels = task.Result.Panels
	}
	if len(task.Plan) > 0 {
		result.ScenePlan = panelPlanItemsToScenePlans(task.Plan)
	}
	userRefs := []string{}
	if u := strings.TrimSpace(task.Request.ReferenceImageURL); u != "" {
		userRefs = append(userRefs, u)
	}
	return buildFragmentGenerationAssets(fragment.ID, task.ID, domain.FragmentGenerationAssetSourcePanelGeneration, task.Request.AspectRatio, fragmentUserReferenceAssetsFromURLs(userRefs), result, panels)
}

func fragmentAssetSourceForFragment(fragment *domain.Fragment) string {
	if fragment == nil {
		return domain.FragmentGenerationAssetSourceAIGeneration
	}
	switch strings.TrimSpace(fragment.SourceType) {
	case string(domain.FragmentSourcePanelGeneration):
		return domain.FragmentGenerationAssetSourcePanelGeneration
	default:
		return domain.FragmentGenerationAssetSourceAIGeneration
	}
}

func fragmentMediaURLs(fragment *domain.Fragment) []string {
	if fragment == nil {
		return nil
	}
	if len(fragment.MediaURLs) > 0 {
		return fragment.MediaURLs
	}
	var urls []string
	if strings.TrimSpace(fragment.ImageUrls) != "" {
		_ = json.Unmarshal([]byte(fragment.ImageUrls), &urls)
	}
	return urls
}
