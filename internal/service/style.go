package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// GetStyleConfigByID retrieves a style configuration by ID（带缓存）
func (s *Service) GetStyleConfigByID(ctx context.Context, id string) (*domain.StyleConfig, error) {
	s.logger.Debug("getting style config by ID",
		zap.String("id", id))

	if id == "" {
		s.logger.Warn("style config ID is empty")
		return nil, fmt.Errorf("style config ID cannot be empty")
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		key := cache.StyleConfigByIDKey(id)
		var cachedConfig domain.StyleConfig
		if err := c.Get(ctx, key, &cachedConfig); err == nil {
			s.logger.Debug("style config cache hit",
				zap.String("id", id))
			return &cachedConfig, nil
		} else {
			s.logger.Debug("style config cache miss",
				zap.String("id", id),
				zap.Error(err))
		}
	}

	// 从数据库获取
	config, err := s.repo.GetStyleConfigByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get style config",
			zap.String("id", id),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存
	if c != nil {
		key := cache.StyleConfigByIDKey(id)
		if err := c.Set(ctx, key, config, styleConfigCacheTTL); err != nil {
			s.logger.Warn("failed to cache style config",
				zap.String("id", id),
				zap.Error(err))
		} else {
			s.logger.Debug("style config cached",
				zap.String("id", id))
		}
	}

	return config, nil
}

// GetStyleConfigByStyle retrieves a style configuration by style name（带缓存）
func (s *Service) GetStyleConfigByStyle(ctx context.Context, styleName string) (*domain.StyleConfig, error) {
	s.logger.Debug("getting style config by style name",
		zap.String("styleName", styleName))

	if styleName == "" {
		s.logger.Warn("style name is empty")
		return nil, fmt.Errorf("style name cannot be empty")
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		key := cache.StyleConfigByStyleKey(styleName)
		var cachedConfig domain.StyleConfig
		if err := c.Get(ctx, key, &cachedConfig); err == nil {
			s.logger.Debug("style config cache hit",
				zap.String("styleName", styleName))
			return &cachedConfig, nil
		} else {
			s.logger.Debug("style config cache miss",
				zap.String("styleName", styleName),
				zap.Error(err))
		}
	}

	// 从数据库获取
	config, err := s.repo.GetStyleConfigByStyle(ctx, styleName)
	if err != nil {
		s.logger.Error("failed to get style config",
			zap.String("styleName", styleName),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存（同时缓存 ID 和 style name）
	if c != nil {
		idKey := cache.StyleConfigByIDKey(config.ID)
		styleKey := cache.StyleConfigByStyleKey(styleName)
		if err := c.Set(ctx, idKey, config, styleConfigCacheTTL); err != nil {
			s.logger.Warn("failed to cache style config by ID",
				zap.String("id", config.ID),
				zap.Error(err))
		}
		if err := c.Set(ctx, styleKey, config, styleConfigCacheTTL); err != nil {
			s.logger.Warn("failed to cache style config by style name",
				zap.String("styleName", styleName),
				zap.Error(err))
		} else {
			s.logger.Debug("style config cached",
				zap.String("id", config.ID),
				zap.String("styleName", styleName))
		}
	}

	return config, nil
}

// ListStyleConfigs retrieves all style configurations with pagination（带缓存）
// If groupID is provided, group-specific styles are prioritized
func (s *Service) ListStyleConfigs(ctx context.Context, groupID string, limit, offset int) ([]*domain.StyleConfig, int64, error) {
	s.logger.Debug("listing style configs",
		zap.String("groupID", groupID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// Set default limits
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.StyleConfigsListKey(groupID, limit, offset)
		var cachedConfigs []*domain.StyleConfig
		var cachedTotal int64
		if err := c.Get(ctx, cacheKey, &cachedConfigs); err == nil {
			// 尝试获取总数缓存
			totalKey := cacheKey + ":total"
			_ = c.Get(ctx, totalKey, &cachedTotal)
			s.logger.Debug("style configs list cache hit",
				zap.String("groupID", groupID),
				zap.Int("count", len(cachedConfigs)))
			return cachedConfigs, cachedTotal, nil
		} else {
			s.logger.Debug("style configs list cache miss",
				zap.String("groupID", groupID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	configs, total, err := s.repo.ListStyleConfigs(ctx, groupID, limit, offset)
	if err != nil {
		s.logger.Error("failed to list style configs",
			zap.String("groupID", groupID),
			zap.Error(err))
		return nil, 0, err
	}

	// 写入缓存
	if c != nil && len(configs) > 0 {
		cacheKey := cache.StyleConfigsListKey(groupID, limit, offset)
		if err := c.Set(ctx, cacheKey, configs, styleConfigCacheTTL); err != nil {
			s.logger.Warn("failed to cache style configs list",
				zap.String("groupID", groupID),
				zap.Error(err))
		} else {
			// 缓存总数
			totalKey := cacheKey + ":total"
			_ = c.Set(ctx, totalKey, total, styleConfigCacheTTL)
			s.logger.Debug("style configs list cached",
				zap.String("groupID", groupID),
				zap.Int("count", len(configs)))
		}
	}

	return configs, total, nil
}

// SearchStyleConfigs searches style configurations by keyword with pagination
// If groupID is provided, group-specific styles are prioritized in results
func (s *Service) SearchStyleConfigs(ctx context.Context, keyword, groupID string, limit, offset int) ([]*domain.StyleConfig, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("search keyword cannot be empty")
	}

	// Set default limits
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.SearchStyleConfigs(ctx, keyword, groupID, limit, offset)
}

// CreateStyleConfig creates a new style configuration
func (s *Service) CreateStyleConfig(ctx context.Context, styleConfig *domain.StyleConfig) error {
	if styleConfig == nil {
		return fmt.Errorf("style config cannot be nil")
	}

	if styleConfig.Style == "" {
		return fmt.Errorf("style name cannot be empty")
	}

	// Check if style with the same name already exists
	existingStyle, err := s.repo.GetStyleConfigByStyle(ctx, styleConfig.Style)
	if err == nil && existingStyle != nil {
		return fmt.Errorf("style config with name '%s' already exists", styleConfig.Style)
	}

	// Set timestamps
	now := time.Now().Unix()
	styleConfig.CreatedAt = now
	styleConfig.UpdatedAt = now

	if err := s.repo.CreateStyleConfig(ctx, styleConfig); err != nil {
		return err
	}

	// 使相关缓存失效
	c := s.getCache()
	if c != nil {
		// 清除列表缓存
		for limit := 50; limit <= 200; limit += 50 {
			for offset := 0; offset < 500; offset += limit {
				_ = c.Delete(ctx, cache.StyleConfigsListKey(styleConfig.GroupID, limit, offset))
				_ = c.Delete(ctx, cache.StyleConfigsListKey("", limit, offset)) // 清除全局列表
			}
		}
		s.logger.Debug("style config cache invalidated after create",
			zap.String("styleConfigID", styleConfig.ID))
	}

	return nil
}

// UpdateStyleConfig updates an existing style configuration
func (s *Service) UpdateStyleConfig(ctx context.Context, styleConfig *domain.StyleConfig) error {
	if styleConfig == nil {
		return fmt.Errorf("style config cannot be nil")
	}

	if styleConfig.ID == "" {
		return fmt.Errorf("style config ID cannot be empty")
	}

	if styleConfig.Style == "" {
		return fmt.Errorf("style name cannot be empty")
	}

	// Check if style config exists
	existingStyle, err := s.repo.GetStyleConfigByID(ctx, styleConfig.ID)
	if err != nil {
		return fmt.Errorf("style config not found: %w", err)
	}

	// If style name is being changed, check for duplicates
	if existingStyle.Style != styleConfig.Style {
		existingByName, err := s.repo.GetStyleConfigByStyle(ctx, styleConfig.Style)
		if err == nil && existingByName != nil && existingByName.ID != styleConfig.ID {
			return fmt.Errorf("style config with name '%s' already exists", styleConfig.Style)
		}
	}

	// Set updated timestamp
	styleConfig.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateStyleConfig(ctx, styleConfig); err != nil {
		return err
	}

	// 使相关缓存失效并重新缓存
	c := s.getCache()
	if c != nil {
		// 清除 ID 和 style name 缓存
		_ = c.Delete(ctx, cache.StyleConfigByIDKey(styleConfig.ID))
		_ = c.Delete(ctx, cache.StyleConfigByStyleKey(styleConfig.Style))
		// 如果 style name 改变，清除旧的 style name 缓存
		if existingStyle.Style != styleConfig.Style {
			_ = c.Delete(ctx, cache.StyleConfigByStyleKey(existingStyle.Style))
		}
		// 重新缓存
		_ = c.Set(ctx, cache.StyleConfigByIDKey(styleConfig.ID), styleConfig, styleConfigCacheTTL)
		_ = c.Set(ctx, cache.StyleConfigByStyleKey(styleConfig.Style), styleConfig, styleConfigCacheTTL)
		// 清除列表缓存
		for limit := 50; limit <= 200; limit += 50 {
			for offset := 0; offset < 500; offset += limit {
				_ = c.Delete(ctx, cache.StyleConfigsListKey(styleConfig.GroupID, limit, offset))
				_ = c.Delete(ctx, cache.StyleConfigsListKey("", limit, offset))
			}
		}
		s.logger.Debug("style config cache invalidated after update",
			zap.String("styleConfigID", styleConfig.ID))
	}

	return nil
}

// DeleteStyleConfig deletes a style configuration by ID
func (s *Service) DeleteStyleConfig(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("style config ID cannot be empty")
	}

	// Check if style config exists
	config, err := s.repo.GetStyleConfigByID(ctx, id)
	if err != nil {
		return fmt.Errorf("style config not found: %w", err)
	}

	if err := s.repo.DeleteStyleConfig(ctx, id); err != nil {
		return err
	}

	// 使相关缓存失效
	c := s.getCache()
	if c != nil {
		// 清除 ID 和 style name 缓存
		_ = c.Delete(ctx, cache.StyleConfigByIDKey(id))
		if config != nil {
			_ = c.Delete(ctx, cache.StyleConfigByStyleKey(config.Style))
		}
		// 清除列表缓存
		for limit := 50; limit <= 200; limit += 50 {
			for offset := 0; offset < 500; offset += limit {
				_ = c.Delete(ctx, cache.StyleConfigsListKey(config.GroupID, limit, offset))
				_ = c.Delete(ctx, cache.StyleConfigsListKey("", limit, offset))
			}
		}
		s.logger.Debug("style config cache invalidated after delete",
			zap.String("styleConfigID", id))
	}

	return nil
}

// BatchCreateStyleConfigs creates multiple style configurations in batch
func (s *Service) BatchCreateStyleConfigs(ctx context.Context, styleConfigs []*domain.StyleConfig) error {
	if len(styleConfigs) == 0 {
		return nil
	}

	// Validate all style configs
	for i, styleConfig := range styleConfigs {
		if styleConfig == nil {
			return fmt.Errorf("style config at index %d cannot be nil", i)
		}

		if styleConfig.Style == "" {
			return fmt.Errorf("style name at index %d cannot be empty", i)
		}

		// Set timestamps
		now := time.Now().Unix()
		styleConfig.CreatedAt = now
		styleConfig.UpdatedAt = now
	}

	return s.repo.BatchCreateStyleConfigs(ctx, styleConfigs)
}

// GetStyleOptions retrieves a simplified list of style options (ID and Style name)
// Returns public styles only (no group filtering)
func (s *Service) GetStyleOptions(ctx context.Context) ([]*domain.StyleConfig, error) {
	styleConfigs, _, err := s.repo.ListStyleConfigs(ctx, "", 200, 0) // Get all public up to 200
	if err != nil {
		return nil, fmt.Errorf("failed to get style options: %w", err)
	}

	// Return simplified style configs with description and sample image
	options := make([]*domain.StyleConfig, len(styleConfigs))
	for i, styleConfig := range styleConfigs {
		options[i] = &domain.StyleConfig{
			ID:             styleConfig.ID,
			Style:          styleConfig.Style,
			Description:    styleConfig.Description,
			SampleImageURL: styleConfig.SampleImageURL,
			GroupID:        styleConfig.GroupID,
		}
	}

	return options, nil
}

// InitializeDefaultStyles initializes the database with default style configurations
func (s *Service) InitializeDefaultStyles(ctx context.Context) error {
	// Check if styles already exist
	existingStyles, _, err := s.repo.ListStyleConfigs(ctx, "", 1, 0)
	if err != nil {
		return fmt.Errorf("failed to check existing styles: %w", err)
	}

	// If styles already exist, skip initialization
	if len(existingStyles) > 0 {
		return nil
	}

	// Create default style configurations from the provided list
	defaultStyles := s.getDefaultStyleConfigs()

	return s.BatchCreateStyleConfigs(ctx, defaultStyles)
}

// getDefaultStyleConfigs returns the list of default style configurations
func (s *Service) getDefaultStyleConfigs() []*domain.StyleConfig {
	return []*domain.StyleConfig{
		{ID: "1", Style: "吉卜力风格", Description: "以宫崎骏动画工作室为代表的日式动画风格，融合细腻的手绘质感和温暖的情感表达，适合创作充满童真与想象力的故事场景，具有独特的东方美学韵味和人文关怀。"},
		{ID: "2", Style: "像素风格", Description: "复古电子游戏时代的数字化美学，通过方块化像素点阵构建图像，营造怀旧氛围和数字化质感，适合创作8位机时代的游戏场景、复古科技主题和数字化艺术表达。"},
		{ID: "3", Style: "蒸汽朋克风格", Description: "融合维多利亚时代工业美学与科幻元素的独特风格，以黄铜齿轮、蒸汽管道、机械装置为核心视觉元素，营造复古未来主义氛围，适合创作架空历史、机械文明和工业革命主题。"},
		{ID: "4", Style: "水墨风格", Description: "中国传统绘画的数字化再现，强调墨色浓淡变化和笔触韵味，通过留白和意境营造东方美学氛围，适合创作中国风主题、古典文学场景和哲学思辨题材。"},
		{ID: "5", Style: "赛博朋克风格", Description: "未来主义与反乌托邦的视觉融合，以霓虹灯、机械义肢、虚拟现实为核心元素，营造高科技低生活的都市氛围，适合创作科幻题材、反乌托邦故事和未来都市场景。"},
		{ID: "6", Style: "低多边形风格", Description: "现代数字艺术的几何美学，通过简化的多边形面片构建立体模型，营造简约而富有科技感的视觉效果，适合创作现代设计、科技产品和抽象艺术表达。"},
		{ID: "7", Style: "霓虹灯风格", Description: "都市夜生活的视觉美学，以高饱和度荧光色彩和发光效果为核心，营造炫目而充满活力的城市氛围，适合创作夜店场景、都市夜景和电子音乐主题。"},
		{ID: "8", Style: "手绘线稿风格", Description: "保留创作过程痕迹的原始艺术感，通过铅笔或墨线勾勒展现艺术家的创作思路，营造人文关怀和手工质感，适合创作概念设计、艺术草图和创意表达。"},
		{ID: "9", Style: "浮世绘风格", Description: "日本传统木版画的平面美学，以平面色块和流畅曲线为特征，展现东方艺术的装饰性和叙事性，适合创作日本文化主题、传统故事和东方美学表达。"},
		{ID: "10", Style: "故障风格", Description: "数字时代的视觉叛逆，通过色块错位、信号干扰等效果表现科技时代的失真感，营造前卫而具有冲击力的视觉效果，适合创作数字艺术、科技主题和实验性表达。"},
		// Note: This is a truncated list for demonstration.
		// The full 200 styles would be included in the actual implementation.
	}
}
