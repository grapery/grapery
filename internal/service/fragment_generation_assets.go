package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

func (s *FragmentGenerationService) persistFragmentGenerationAssets(ctx context.Context, fragmentID, taskID string, req domain.FragmentGenerationRequest, result *domain.FragmentGenerationResult) {
	if s == nil || s.repo == nil || result == nil || strings.TrimSpace(fragmentID) == "" {
		return
	}
	assets := buildFragmentGenerationAssets(fragmentID, taskID, domain.FragmentGenerationAssetSourceAIGeneration, req.AspectRatio, req.ImageUrls, result, nil)
	if len(assets) == 0 {
		return
	}
	if err := s.repo.CreateFragmentGenerationAssets(ctx, assets); err != nil {
		s.logger.Warn("failed to persist fragment generation assets", zap.String("fragmentID", fragmentID), zap.String("taskID", taskID), zap.Error(err))
	}
}

func (s *FragmentPanelGenerationService) persistPanelGenerationAssets(ctx context.Context, fragmentID string, task *domain.FragmentPanelGenerationTask, aspectRatio string, policy *domain.FragmentConsistencyPolicy) {
	if s == nil || s.repo == nil || task == nil || task.Result == nil || strings.TrimSpace(fragmentID) == "" {
		return
	}
	var userRefs []string
	if u := strings.TrimSpace(task.Request.ReferenceImageURL); u != "" {
		userRefs = []string{u}
	}
	result := &domain.FragmentGenerationResult{
		ImageUrls:         panelResultImageURLs(task.Result.Panels),
		AspectRatio:       aspectRatio,
		AnchorImages:      task.AnchorImages,
		ScenePlan:         panelPlanItemsToScenePlans(task.Plan),
		ConsistencyPolicy: policy,
		GenerationTrace:   task.Result.GenerationTrace,
	}
	assets := buildFragmentGenerationAssets(fragmentID, task.ID, domain.FragmentGenerationAssetSourcePanelGeneration, aspectRatio, userRefs, result, task.Result.Panels)
	if len(assets) == 0 {
		return
	}
	if err := s.repo.CreateFragmentGenerationAssets(ctx, assets); err != nil {
		s.logger.Warn("failed to persist panel generation assets", zap.String("fragmentID", fragmentID), zap.String("taskID", task.ID), zap.Error(err))
	}
}

func buildFragmentGenerationAssets(fragmentID, taskID, source, aspectRatio string, userRefURLs []string, result *domain.FragmentGenerationResult, panels []domain.FragmentPanelResultItem) []*domain.FragmentGenerationAsset {
	if result == nil {
		return nil
	}
	ar := strings.TrimSpace(aspectRatio)
	if ar == "" {
		ar = strings.TrimSpace(result.AspectRatio)
	}
	seriesSeed := 0
	if result.ConsistencyPolicy != nil {
		seriesSeed = result.ConsistencyPolicy.SeriesSeed
	}
	bible := result.VisualBible
	if bible == nil && result.GenerationTrace != nil {
		bible = result.GenerationTrace.VisualBible
	}
	sceneMetric := traceMetricByName(result.GenerationTrace, "scene_images")
	panelMetricByIndex := panelMetricsByIndex(result.GenerationTrace)
	taskTokenTotal := result.TokensUsed
	var assets []*domain.FragmentGenerationAsset
	order := 0
	for _, u := range fragmentPrefillHTTPImageURLs(userRefURLs, len(userRefURLs)) {
		assets = append(assets, &domain.FragmentGenerationAsset{
			FragmentID:   fragmentID,
			Source:       source,
			TaskID:       taskID,
			Kind:         domain.FragmentGenerationAssetKindUserReference,
			URL:          u,
			AspectRatio:  ar,
			DisplayOrder: order,
			MetadataJSON: "{}",
		})
		order++
	}
	for i, asset := range result.ReferenceAssets {
		if strings.TrimSpace(asset.ImageURL) == "" {
			continue
		}
		assets = append(assets, &domain.FragmentGenerationAsset{
			FragmentID:   fragmentID,
			Source:       source,
			TaskID:       taskID,
			Kind:         domain.FragmentGenerationAssetKindReferenceAsset,
			EntityKind:   strings.TrimSpace(asset.Kind),
			EntityKey:    strings.TrimSpace(asset.Key),
			URL:          strings.TrimSpace(asset.ImageURL),
			AspectRatio:  ar,
			TokensUsed:   asset.TokensUsed,
			Provider:     traceMetricProvider(result.GenerationTrace, "reference_assets"),
			Model:        traceMetricModel(result.GenerationTrace, "reference_assets"),
			SeriesSeed:   seriesSeed,
			DisplayOrder: 100 + i,
			MetadataJSON: fragmentAssetMetadata(asset, bible, asset.Kind, asset.Key),
		})
	}
	for i, anchor := range result.AnchorImages {
		if strings.TrimSpace(anchor.ImageURL) == "" {
			continue
		}
		assets = append(assets, &domain.FragmentGenerationAsset{
			FragmentID:   fragmentID,
			Source:       source,
			TaskID:       taskID,
			Kind:         domain.FragmentGenerationAssetKindAnchorImage,
			EntityKind:   strings.TrimSpace(anchor.Kind),
			EntityKey:    strings.TrimSpace(anchor.Key),
			URL:          strings.TrimSpace(anchor.ImageURL),
			AspectRatio:  ar,
			Provider:     traceMetricProvider(result.GenerationTrace, "anchor_images"),
			Model:        traceMetricModel(result.GenerationTrace, "anchor_images"),
			SeriesSeed:   seriesSeed,
			DisplayOrder: 200 + i,
			MetadataJSON: fragmentAssetMetadata(anchor, bible, anchor.Kind, anchor.Key),
		})
	}
	if len(panels) > 0 {
		for i, panel := range panels {
			if strings.TrimSpace(panel.ImageURL) == "" {
				continue
			}
			idx := panel.Index
			panelMetric := panelMetricByIndex[idx]
			metadata := map[string]interface{}{
				"panel":       panel,
				"tokensScope": "asset_estimated",
			}
			tokenUsed := 0
			provider := ""
			model := ""
			if panelMetric != nil {
				tokenUsed = panelMetric.Tokens
				provider = strings.TrimSpace(panelMetric.Provider)
				model = strings.TrimSpace(panelMetric.Model)
				metadata["metricStep"] = panelMetric.Name
			} else if sceneMetric != nil {
				metadata["tokensScope"] = "task_total"
				metadata["taskTokensUsed"] = sceneMetric.Tokens
				provider = strings.TrimSpace(sceneMetric.Provider)
				model = strings.TrimSpace(sceneMetric.Model)
			} else {
				metadata["tokensScope"] = "task_total"
				metadata["taskTokensUsed"] = taskTokenTotal
			}
			assets = append(assets, &domain.FragmentGenerationAsset{
				FragmentID:   fragmentID,
				Source:       source,
				TaskID:       taskID,
				Kind:         domain.FragmentGenerationAssetKindSceneFinal,
				EntityKind:   domain.FragmentGenerationAssetEntityScene,
				SceneIndex:   &idx,
				URL:          strings.TrimSpace(panel.ImageURL),
				AspectRatio:  ar,
				TokensUsed:   tokenUsed,
				Provider:     provider,
				Model:        model,
				SeriesSeed:   seriesSeed,
				DisplayOrder: 1000 + i,
				MetadataJSON: assetMustJSON(metadata),
			})
		}
		return assets
	}
	for i, scene := range result.ScenePlan {
		u := strings.TrimSpace(scene.GeneratedImageURL)
		if u == "" && i < len(result.ImageUrls) {
			u = strings.TrimSpace(result.ImageUrls[i])
		}
		if u == "" {
			continue
		}
		idx := scene.Index
		metadata := map[string]interface{}{
			"scene":          scene,
			"tokensScope":    "task_total",
			"taskTokensUsed": taskTokenTotal,
		}
		provider := ""
		model := ""
		if sceneMetric != nil {
			metadata["taskTokensUsed"] = sceneMetric.Tokens
			if p := strings.TrimSpace(sceneMetric.Provider); p != "" {
				provider = p
			}
			if m := strings.TrimSpace(sceneMetric.Model); m != "" {
				model = m
			}
		}
		assets = append(assets, &domain.FragmentGenerationAsset{
			FragmentID:   fragmentID,
			Source:       source,
			TaskID:       taskID,
			Kind:         domain.FragmentGenerationAssetKindSceneFinal,
			EntityKind:   domain.FragmentGenerationAssetEntityScene,
			SceneIndex:   &idx,
			URL:          u,
			AspectRatio:  ar,
			Provider:     provider,
			Model:        model,
			SeriesSeed:   seriesSeed,
			SceneSeed:    scene.Seed,
			DisplayOrder: 1000 + i,
			MetadataJSON: assetMustJSON(metadata),
		})
	}
	return assets
}

func panelResultImageURLs(panels []domain.FragmentPanelResultItem) []string {
	out := make([]string, 0, len(panels))
	for _, panel := range panels {
		if u := strings.TrimSpace(panel.ImageURL); u != "" {
			out = append(out, u)
		}
	}
	return out
}

func assetMustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 {
		return "{}"
	}
	return string(b)
}

func fragmentAssetMetadata(raw interface{}, bible *domain.FragmentVisualBible, kind, key string) string {
	meta := map[string]interface{}{"raw": raw}
	if bible != nil && strings.TrimSpace(kind) == domain.FragmentGenerationAssetEntityCharacter {
		for _, ch := range bible.Characters {
			if strings.TrimSpace(ch.Key) == strings.TrimSpace(key) {
				meta["name"] = ch.Name
				meta["immutableTraits"] = ch.ImmutableTraits
				meta["negativeTraits"] = ch.NegativeTraits
				break
			}
		}
	}
	return assetMustJSON(meta)
}

func traceMetricByName(trace *domain.FragmentGenerationTrace, name string) *domain.FragmentGenerationStepMetric {
	if trace == nil {
		return nil
	}
	target := strings.TrimSpace(name)
	for i := range trace.Metrics {
		step := &trace.Metrics[i]
		if strings.TrimSpace(step.Name) == target {
			return step
		}
	}
	return nil
}

func traceMetricProvider(trace *domain.FragmentGenerationTrace, name string) string {
	step := traceMetricByName(trace, name)
	if step == nil {
		return ""
	}
	return strings.TrimSpace(step.Provider)
}

func traceMetricModel(trace *domain.FragmentGenerationTrace, name string) string {
	step := traceMetricByName(trace, name)
	if step == nil {
		return ""
	}
	return strings.TrimSpace(step.Model)
}

func panelMetricsByIndex(trace *domain.FragmentGenerationTrace) map[int]*domain.FragmentGenerationStepMetric {
	out := map[int]*domain.FragmentGenerationStepMetric{}
	if trace == nil {
		return out
	}
	for i := range trace.Metrics {
		step := &trace.Metrics[i]
		index, ok := parsePanelMetricIndex(step.Name)
		if !ok {
			continue
		}
		out[index] = step
	}
	return out
}

func parsePanelMetricIndex(step string) (int, bool) {
	name := strings.TrimSpace(step)
	const prefix = "generating_panel_"
	if !strings.HasPrefix(name, prefix) {
		return 0, false
	}
	var idx int
	if _, err := fmt.Sscanf(name, prefix+"%d", &idx); err != nil {
		return 0, false
	}
	return idx, true
}
