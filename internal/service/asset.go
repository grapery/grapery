package service

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// ListAssets 获取用户的资产列表
func (s *Service) ListAssets(ctx context.Context, userID, assetType string, limit, offset int) ([]*domain.Asset, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	assets, err := s.repo.AssetsByUser(ctx, userID, assetType, limit, offset)
	if err != nil {
		s.logger.Error("failed to list assets", zap.Error(err), zap.String("userId", userID))
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	return assets, nil
}

// GetAsset 获取资产详情
func (s *Service) GetAsset(ctx context.Context, assetID, userID string) (*domain.Asset, error) {
	asset, err := s.repo.AssetByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %w", err)
	}

	// 验证权限（只能访问自己的资产）
	if asset.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	return asset, nil
}

// CreateAsset 创建资产
func (s *Service) CreateAsset(ctx context.Context, asset *domain.Asset) error {
	if err := s.repo.CreateAsset(ctx, asset); err != nil {
		s.logger.Error("failed to create asset", zap.Error(err))
		return fmt.Errorf("failed to create asset: %w", err)
	}

	s.logger.Info("asset created", zap.String("assetId", asset.ID), zap.String("userId", asset.UserID))
	return nil
}

// UpdateAsset 更新资产
func (s *Service) UpdateAsset(ctx context.Context, assetID, userID string, updates *AssetUpdateRequest) (*domain.Asset, error) {
	asset, err := s.repo.AssetByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %w", err)
	}

	// 验证权限
	if asset.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// 更新字段
	if updates.Name != nil {
		asset.Name = *updates.Name
	}
	if updates.Tags != nil {
		asset.Tags = *updates.Tags
	}

	if err := s.repo.UpdateAsset(ctx, asset); err != nil {
		s.logger.Error("failed to update asset", zap.Error(err), zap.String("assetId", assetID))
		return nil, fmt.Errorf("failed to update asset: %w", err)
	}

	s.logger.Info("asset updated", zap.String("assetId", assetID))
	return asset, nil
}

// DeleteAsset 删除资产
func (s *Service) DeleteAsset(ctx context.Context, assetID, userID string) error {
	asset, err := s.repo.AssetByID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("asset not found: %w", err)
	}

	// 验证权限
	if asset.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	if err := s.repo.DeleteAsset(ctx, assetID); err != nil {
		s.logger.Error("failed to delete asset", zap.Error(err), zap.String("assetId", assetID))
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	s.logger.Info("asset deleted", zap.String("assetId", assetID))
	return nil
}

// AssetUpdateRequest 资产更新请求
type AssetUpdateRequest struct {
	Name *string   `json:"name"`
	Tags *[]string `json:"tags"`
}
