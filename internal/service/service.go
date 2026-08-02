package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
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
	recoCfg          config.RecommendationConfig
	comicStyleSvc    *FragmentComicStyleService // 碎片漫画风格目录（与创作页同源）
	aiTextAdmission           *AITextAdmissionGate           // optional: global outbound LLM text concurrency (Redis)
	accountDeletionCfg        config.AccountDeletionConfig // grace period & system anon user ID
	terminationFragmentRepo   *repository.FragmentRepository
	shareSigner               *ShareLinkSigner
	shareEvents               *repository.ShareEventRepository

	// structureResumeLocks serializes POST .../generate/structure per storyboard ID (TryLock = busy).
	structureResumeLocks sync.Map // string -> *sync.Mutex
}

type FragmentAssetQuery struct {
	Kind       string
	EntityKind string
	EntityKey  string
}

// New creates a new service instance
func New(repo domain.Repository, log *zap.Logger, recoCfg config.RecommendationConfig) *Service {
	return &Service{
		repo:          repo,
		log:           log,
		logger:        log,
		cache:         nil,       // 稍后通过 SetCache 设置
		imageProvider: "huoshan", // Default image provider
		videoProvider: "huoshan", // Default video provider (火山优先)
		recoCfg:       recoCfg,
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

// SetFragmentComicStyleService 注入碎片风格目录服务（用于预填等按 value 解析展示名）。
func (s *Service) SetFragmentComicStyleService(svc *FragmentComicStyleService) {
	s.comicStyleSvc = svc
}

// ConfigureAITextAdmission sets the global outbound text LLM concurrency gate (shared by AIGenerationService and AIService).
// maxConcurrent <= 0 disables the gate (fail-open).
func (s *Service) ConfigureAITextAdmission(c cache.Cache, maxConcurrent int) {
	var gate *AITextAdmissionGate
	if maxConcurrent > 0 && !cache.IsEffectivelyNil(c) {
		gate = NewAITextAdmissionGate(c, maxConcurrent, s.logger)
		s.logger.Info("AI text admission gate configured",
			zap.Int("maxConcurrent", maxConcurrent),
			zap.Bool("disabled", gate == nil))
	} else {
		s.logger.Info("AI text admission gate disabled (maxConcurrent<=0 or redis nil)",
			zap.Int("maxConcurrent", maxConcurrent))
	}
	s.aiTextAdmission = gate
	if s.aiGenService != nil {
		s.aiGenService.SetAITextAdmission(gate)
	}
}

// SetShareLinkSigner wires HMAC signing for public share URLs.
func (s *Service) SetShareLinkSigner(signer *ShareLinkSigner) {
	s.shareSigner = signer
}

// SetShareEventRepository wires share issue/open event persistence for admin analytics.
func (s *Service) SetShareEventRepository(repo *repository.ShareEventRepository) {
	s.shareEvents = repo
}

// shareOpenDedupeTTL collapses repeated opens of the same link by the same viewer
// (page reloads, client retries, the landing page refetching content) into one event.
const shareOpenDedupeTTL = 30 * time.Minute

// RecordShareEvent persists a share funnel event and updates Prometheus counters.
// Persistence happens off the request goroutine: this sits on anonymous read paths and
// must never add latency or fail the response.
func (s *Service) RecordShareEvent(ctx context.Context, eventType domain.ShareEventType, kind ShareKind, contentID, userID, platform, source string) {
	if s == nil {
		return
	}
	if eventType == domain.ShareEventOpen && s.shareOpenAlreadyCounted(ctx, kind, contentID, userID, platform) {
		return
	}
	s.RecordShareMetric(eventType, kind, platform, source)
	if s.shareEvents == nil {
		return
	}
	ev := &domain.ShareEvent{
		EventType: eventType,
		Kind:      string(kind),
		ContentID: contentID,
		UserID:    userID,
		Platform:  platform,
		Source:    source,
	}
	go func() {
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.shareEvents.Create(writeCtx, ev); err != nil && s.logger != nil {
			s.logger.Warn("share event persist failed",
				zap.String("eventType", string(ev.EventType)),
				zap.String("kind", ev.Kind),
				zap.String("contentID", ev.ContentID),
				zap.Error(err))
		}
	}()
}

// shareOpenAlreadyCounted returns true when this viewer already produced an open for
// the same content within the dedupe window. Without a cache it never dedupes.
func (s *Service) shareOpenAlreadyCounted(ctx context.Context, kind ShareKind, contentID, userID, platform string) bool {
	c := s.getCache()
	if c == nil || contentID == "" {
		return false
	}
	viewer := userID
	if viewer == "" {
		// Anonymous opens can only be grouped by platform; good enough to absorb reloads.
		viewer = "anon:" + platform
	}
	key := fmt.Sprintf("share:open:%s:%s:%s", kind, contentID, viewer)
	n, err := c.Incr(ctx, key)
	if err != nil {
		return false
	}
	if n == 1 {
		_ = c.Expire(ctx, key, shareOpenDedupeTTL)
		return false
	}
	return true
}

// RecordShareMetric increments Prometheus share counters (no DB write).
func (s *Service) RecordShareMetric(eventType domain.ShareEventType, kind ShareKind, platform, source string) {
	if s == nil || s.metrics == nil {
		return
	}
	s.metrics.RecordShareEvent(string(eventType), string(kind), platform, source)
}

// HasValidShareGrant returns true when query token matches kind/id/exp.
func (s *Service) HasValidShareGrant(kind ShareKind, id, token string, exp int64) bool {
	if s.shareSigner == nil {
		return false
	}
	return s.shareSigner.Verify(kind, id, token, exp)
}

// SetAccountDeletionDeps wires phased account deletion (grace window, reassignment holder) plus fragment teardown.
func (s *Service) SetAccountDeletionDeps(cfg config.AccountDeletionConfig, frag *repository.FragmentRepository) {
	s.accountDeletionCfg = cfg
	s.terminationFragmentRepo = frag
	s.logger.Info("account deletion deps configured",
		zap.String("systemAnonymousUserId", cfg.SystemAnonymousUserID),
		zap.Int("gracePeriodSeconds", cfg.GracePeriodSeconds),
		zap.Bool("fragmentsAttached", frag != nil))
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
		if s.aiTextAdmission != nil {
			s.aiGenService.SetAITextAdmission(s.aiTextAdmission)
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
	return NewAIService(s.genAPI, s.geminiClient, s.aiGenService, s.imageProvider, s.videoProvider, s.repo, s.logger, s.aiTextAdmission)
}

// UserStatsService 获取用户统计服务
func (s *Service) UserStatsService() *UserStatisticsService {
	return s.userStatsService
}

// Health returns service health status
func (s *Service) Health() map[string]string {
	return map[string]string{"status": "ok", "service": "grapery-api"}
}
