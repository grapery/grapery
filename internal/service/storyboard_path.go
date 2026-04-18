package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// StoryboardPathService 故事板路径服务
type StoryboardPathService struct {
	storyRepo domain.Repository
	logger    *zap.Logger
}

// NewStoryboardPathService 创建路径服务
func NewStoryboardPathService(
	storyRepo domain.Repository,
	logger *zap.Logger,
) *StoryboardPathService {
	return &StoryboardPathService{
		storyRepo: storyRepo,
		logger:    logger,
	}
}

// SetDefaultPath 手动设置默认路径
func (s *StoryboardPathService) SetDefaultPath(ctx context.Context, storyID string, userID string, nodeIDs []string) error {
	// 1. 获取故事
	story, err := s.storyRepo.StoryByID(ctx, storyID)
	if err != nil {
		return fmt.Errorf("story not found: %w", err)
	}

	// 2. 检查权限（只有作者可以设置）
	if story.UserID != userID {
		return fmt.Errorf("permission denied: only author can set default path")
	}

	// 3. 验证所有节点ID是否有效且属于该故事
	for _, nodeID := range nodeIDs {
		storyboard, err := s.storyRepo.StoryboardByID(ctx, nodeID)
		if err != nil {
			return fmt.Errorf("storyboard not found: %s", nodeID)
		}
		if storyboard.StoryID != storyID {
			return fmt.Errorf("storyboard %s does not belong to story %s", nodeID, storyID)
		}
	}

	// 4. 更新故事的默认路径
	now := time.Now().Unix()
	story.DefaultPathNodeIDs = nodeIDs
	story.DefaultPathUpdatedAt = &now
	story.DefaultPathType = "manual"

	if err := s.storyRepo.UpdateStory(ctx, story); err != nil {
		return fmt.Errorf("failed to update story default path: %w", err)
	}

	// 5. 更新 storyboard 的默认路径标记
	s.updateStoryboardDefaultPathMarks(ctx, storyID, nodeIDs)

	return nil
}

// CalculateAutoPath 基于点赞数自动计算默认路径
func (s *StoryboardPathService) CalculateAutoPath(ctx context.Context, storyID string, userID string) ([]string, error) {
	// 1. 获取故事
	story, err := s.storyRepo.StoryByID(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("story not found: %w", err)
	}

	// 2. 检查权限
	if story.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}

	// 3. 获取故事的所有故事板
	storyboards, err := s.storyRepo.StoryboardsByStory(ctx, storyID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get storyboards: %w", err)
	}

	// 4. 计算每个故事板的权重
	type weightedNode struct {
		nodeID string
		weight int
	}

	nodes := make([]weightedNode, 0, len(storyboards))
	for _, sb := range storyboards {
		// 点赞数以 storyboards.likes 为准（与 storyboard_likes / 专用点赞接口一致）
		likeCount := sb.Likes

		// 权重 = 点赞数 + 基础权重
		weight := likeCount + 1

		// 根故事板权重更高
		if sb.ParentID == domain.StoryboardRootMarker {
			weight += 100
		}

		nodes = append(nodes, weightedNode{
			nodeID: sb.ID,
			weight: weight,
		})
	}

	// 5. 按权重排序
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].weight > nodes[j].weight
	})

	// 6. 提取节点ID列表
	path := make([]string, 0, len(nodes))
	for _, node := range nodes {
		path = append(path, node.nodeID)
	}

	// 7. 更新故事的默认路径
	now := time.Now().Unix()
	story.DefaultPathNodeIDs = path
	story.DefaultPathUpdatedAt = &now
	story.DefaultPathType = "auto"

	if err := s.storyRepo.UpdateStory(ctx, story); err != nil {
		return nil, fmt.Errorf("failed to update story default path: %w", err)
	}

	// 8. 更新 storyboard 标记
	s.updateStoryboardDefaultPathMarks(ctx, storyID, path)

	return path, nil
}

// GetDefaultPath 获取故事的默认路径
func (s *StoryboardPathService) GetDefaultPath(ctx context.Context, storyID string) ([]*domain.Storyboard, error) {
	// 1. 获取故事
	story, err := s.storyRepo.StoryByID(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("story not found: %w", err)
	}

	// 2. 如果没有设置默认路径，返回空
	if len(story.DefaultPathNodeIDs) == 0 {
		return []*domain.Storyboard{}, nil
	}

	// 3. 按顺序获取故事板
	path := make([]*domain.Storyboard, 0, len(story.DefaultPathNodeIDs))
	for _, nodeID := range story.DefaultPathNodeIDs {
		storyboard, err := s.storyRepo.StoryboardByID(ctx, nodeID)
		if err != nil {
			s.logger.Warn("storyboard not found in default path",
				zap.String("storyboard_id", nodeID),
				zap.String("story_id", storyID))
			continue
		}
		path = append(path, storyboard)
	}

	return path, nil
}

// updateStoryboardDefaultPathMarks 更新故事板的默认路径标记
func (s *StoryboardPathService) updateStoryboardDefaultPathMarks(ctx context.Context, storyID string, nodeIDs []string) {
	// 获取故事的所有故事板
	storyboards, err := s.storyRepo.StoryboardsByStory(ctx, storyID, 1000, 0)
	if err != nil {
		s.logger.Error("failed to get storyboards for path update", zap.Error(err))
		return
	}

	// 创建节点ID到顺序的映射
	orderMap := make(map[string]int)
	for i, nodeID := range nodeIDs {
		orderMap[nodeID] = i + 1
	}

	// 更新每个故事板
	for _, sb := range storyboards {
		if order, exists := orderMap[sb.ID]; exists {
			sb.IsInDefaultPath = true
			sb.DefaultPathOrder = order
		} else {
			sb.IsInDefaultPath = false
			sb.DefaultPathOrder = 0
		}

		if err := s.storyRepo.UpdateStoryboard(ctx, sb); err != nil {
			s.logger.Warn("failed to update storyboard path mark",
				zap.String("storyboard_id", sb.ID),
				zap.Error(err))
		}
	}
}
