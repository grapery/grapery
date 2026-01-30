package service

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"go.uber.org/zap"
)

// Service exposes business logic
type Service struct {
	repo             domain.Repository
	log              *zap.Logger
	logger           *zap.Logger // 别名，用于新代码
	cache            interface{} // Cache 接口
	genAPI           *genapi.GenAPI
	geminiClient     *gemini.Client
	aiGenService     *AIGenerationService   // AI生成服务（统一管理AI能力使用）
	imageProvider    string                 // Provider for image generation (gemini, huoshan)
	videoProvider    string                 // Provider for video generation (gemini, huoshan, hailuo)
	metrics          *telemetry.Metrics     // Prometheus metrics (optional)
	userStatsService *UserStatisticsService // 用户统计服务
}

// New creates a new service instance
func New(repo domain.Repository, log *zap.Logger) *Service {
	return &Service{
		repo:          repo,
		log:           log,
		logger:        log,
		cache:         nil,       // 稍后通过 SetCache 设置
		imageProvider: "huoshan", // Default image provider
		videoProvider: "hailuo",  // Default video provider
	}
}

// SetAIConfig sets the AI provider configuration
func (s *Service) SetAIConfig(cfg config.AIConfig) {
	if cfg.ImageProvider != "" {
		s.imageProvider = cfg.ImageProvider
	}
	if cfg.VideoProvider != "" {
		s.videoProvider = cfg.VideoProvider
	}
	s.logger.Info("AI provider configuration set",
		zap.String("imageProvider", s.imageProvider),
		zap.String("videoProvider", s.videoProvider))
}

// SetCache 设置缓存实例
func (s *Service) SetCache(cache interface{}) {
	s.cache = cache
}

// SetAIClients 设置 AI 客户端
func (s *Service) SetAIClients(genAPI *genapi.GenAPI, geminiClient *gemini.Client) {
	s.genAPI = genAPI
	s.geminiClient = geminiClient

	// 初始化AI生成服务
	if genAPI != nil || geminiClient != nil {
		s.aiGenService = NewAIGenerationService(s.repo, geminiClient, genAPI, s.logger)
		// Set metrics if available
		if s.metrics != nil {
			s.aiGenService.SetMetrics(s.metrics)
		}
		s.logger.Info("AI generation service initialized")

		// 恢复未完成的视频生成任务
		go s.RecoverPendingVideoGenerations(context.Background())
	}
}

// SetMetrics 设置 Prometheus metrics
func (s *Service) SetMetrics(metrics *telemetry.Metrics) {
	s.metrics = metrics
	// 初始化用户统计服务
	if s.cache != nil {
		if cache, ok := s.cache.(cache.Cache); ok {
			s.userStatsService = NewUserStatisticsService(s.repo, cache, s.logger, metrics)
		}
	}
}

// AIGenerationService 获取AI生成服务
func (s *Service) AIGenerationService() *AIGenerationService {
	return s.aiGenService
}

// AIService 获取AI服务（用于 FragmentGenerationService）
func (s *Service) AIService() *AIService {
	return NewAIService(s.genAPI, s.geminiClient, s.repo, s.logger)
}

// UserStatsService 获取用户统计服务
func (s *Service) UserStatsService() *UserStatisticsService {
	return s.userStatsService
}

// Health returns service health status
func (s *Service) Health() map[string]string {
	return map[string]string{"status": "ok", "service": "grapery-api"}
}
