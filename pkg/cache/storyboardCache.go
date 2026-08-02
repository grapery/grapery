package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/cache"
	"go.uber.org/zap"
)

// 故事板缓存键前缀
const (
	StoryBoardDetailPrefix      = "storyboard:detail:"       // 故事板详情
	StoryBoardListPrefix        = "storyboard:list:"         // 故事板列表
	StoryBoardScenesPrefix      = "storyboard:scenes:"       // 故事板场景
	StoryBoardRolesPrefix       = "storyboard:roles:"        // 故事板角色
	StoryBoardUserCreatedPrefix = "storyboard:user_created:" // 用户创建的故事板
	StoryBoardUserStatusPrefix  = "storyboard:user_status:"  // 故事板用户状态
	StoryBoardLikeCountPrefix   = "storyboard:like_count:"   // 故事板点赞数
	StoryBoardNextPrefix        = "storyboard:next:"         // 下一个故事板
	StoryBoardActivePrefix      = "storyboard:active:"       // 活跃故事板
	StoryBoardSearchPrefix      = "storyboard:search:"       // 故事板搜索结果
)

// 缓存过期时间（秒）
const (
	StoryBoardDetailTTL      = 600 // 10分钟
	StoryBoardListTTL        = 300 // 5分钟
	StoryBoardScenesTTL      = 300 // 5分钟
	StoryBoardRolesTTL       = 300 // 5分钟
	StoryBoardUserCreatedTTL = 180 // 3分钟
	StoryBoardUserStatusTTL  = 120 // 2分钟
	StoryBoardLikeCountTTL   = 60  // 1分钟
	StoryBoardNextTTL        = 300 // 5分钟
	StoryBoardActiveTTL      = 180 // 3分钟
	StoryBoardSearchTTL      = 120 // 2分钟
)

// StoryBoardCache 故事板缓存管理器
type StoryBoardCache struct{}

// NewStoryBoardCache 创建故事板缓存管理器
func NewStoryBoardCache() *StoryBoardCache {
	return &StoryBoardCache{}
}

// GetStoryBoardDetail 获取故事板详情缓存
func (sbc *StoryBoardCache) GetStoryBoardDetail(ctx context.Context, boardID int64) (*models.StoryBoard, error) {
	key := fmt.Sprintf("%s%d", StoryBoardDetailPrefix, boardID)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var board models.StoryBoard
	if err := json.Unmarshal(data, &board); err != nil {
		logger.Error("unmarshal storyboard detail failed", zap.Error(err))
		return nil, err
	}

	return &board, nil
}

// SetStoryBoardDetail 设置故事板详情缓存
func (sbc *StoryBoardCache) SetStoryBoardDetail(ctx context.Context, boardID int64, board *models.StoryBoard) error {
	key := fmt.Sprintf("%s%d", StoryBoardDetailPrefix, boardID)
	data, err := json.Marshal(board)
	if err != nil {
		logger.Error("marshal storyboard detail failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryBoardDetailTTL)
}

// DeleteStoryBoardDetail 删除故事板详情缓存
func (sbc *StoryBoardCache) DeleteStoryBoardDetail(ctx context.Context, boardID int64) error {
	key := fmt.Sprintf("%s%d", StoryBoardDetailPrefix, boardID)
	return cache.DelCache(ctx, key)
}

// GetStoryBoardList 获取故事列表的故事板缓存
func (sbc *StoryBoardCache) GetStoryBoardList(ctx context.Context, storyID int64, page, pageSize int) ([]*models.StoryBoard, error) {
	key := fmt.Sprintf("%s%d:%d:%d", StoryBoardListPrefix, storyID, page, pageSize)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var boards []*models.StoryBoard
	if err := json.Unmarshal(data, &boards); err != nil {
		logger.Error("unmarshal storyboard list failed", zap.Error(err))
		return nil, err
	}

	return boards, nil
}

// SetStoryBoardList 设置故事列表的故事板缓存
func (sbc *StoryBoardCache) SetStoryBoardList(ctx context.Context, storyID int64, page, pageSize int, boards []*models.StoryBoard) error {
	key := fmt.Sprintf("%s%d:%d:%d", StoryBoardListPrefix, storyID, page, pageSize)
	data, err := json.Marshal(boards)
	if err != nil {
		logger.Error("marshal storyboard list failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryBoardListTTL)
}

// DeleteStoryBoardList 删除故事列表的故事板缓存
func (sbc *StoryBoardCache) DeleteStoryBoardList(ctx context.Context, storyID int64) error {
	// 简化处理，删除指定story下的所有故事板列表缓存
	key := fmt.Sprintf("%s%d", StoryBoardListPrefix, storyID)
	return cache.DelCache(ctx, key)
}

// GetStoryBoardScenes 获取故事板场景缓存
func (sbc *StoryBoardCache) GetStoryBoardScenes(ctx context.Context, boardID int64) ([]*models.StoryBoardScene, error) {
	key := fmt.Sprintf("%s%d", StoryBoardScenesPrefix, boardID)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var scenes []*models.StoryBoardScene
	if err := json.Unmarshal(data, &scenes); err != nil {
		logger.Error("unmarshal storyboard scenes failed", zap.Error(err))
		return nil, err
	}

	return scenes, nil
}

// SetStoryBoardScenes 设置故事板场景缓存
func (sbc *StoryBoardCache) SetStoryBoardScenes(ctx context.Context, boardID int64, scenes []*models.StoryBoardScene) error {
	key := fmt.Sprintf("%s%d", StoryBoardScenesPrefix, boardID)
	data, err := json.Marshal(scenes)
	if err != nil {
		logger.Error("marshal storyboard scenes failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryBoardScenesTTL)
}

// DeleteStoryBoardScenes 删除故事板场景缓存
func (sbc *StoryBoardCache) DeleteStoryBoardScenes(ctx context.Context, boardID int64) error {
	key := fmt.Sprintf("%s%d", StoryBoardScenesPrefix, boardID)
	return cache.DelCache(ctx, key)
}

// GetStoryBoardRoles 获取故事板角色缓存
func (sbc *StoryBoardCache) GetStoryBoardRoles(ctx context.Context, boardID int64) ([]*models.StoryRole, error) {
	key := fmt.Sprintf("%s%d", StoryBoardRolesPrefix, boardID)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var roles []*models.StoryRole
	if err := json.Unmarshal(data, &roles); err != nil {
		logger.Error("unmarshal storyboard roles failed", zap.Error(err))
		return nil, err
	}

	return roles, nil
}

// SetStoryBoardRoles 设置故事板角色缓存
func (sbc *StoryBoardCache) SetStoryBoardRoles(ctx context.Context, boardID int64, roles []*models.StoryRole) error {
	key := fmt.Sprintf("%s%d", StoryBoardRolesPrefix, boardID)
	data, err := json.Marshal(roles)
	if err != nil {
		logger.Error("marshal storyboard roles failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryBoardRolesTTL)
}

// DeleteStoryBoardRoles 删除故事板角色缓存
func (sbc *StoryBoardCache) DeleteStoryBoardRoles(ctx context.Context, boardID int64) error {
	key := fmt.Sprintf("%s%d", StoryBoardRolesPrefix, boardID)
	return cache.DelCache(ctx, key)
}

// GetUserCreatedStoryBoards 获取用户创建的故事板缓存
func (sbc *StoryBoardCache) GetUserCreatedStoryBoards(ctx context.Context, userID int64, page, pageSize int) ([]*models.StoryBoard, int64, error) {
	key := fmt.Sprintf("%s%d:%d:%d", StoryBoardUserCreatedPrefix, userID, page, pageSize)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, 0, err
	}

	var result struct {
		Boards []*models.StoryBoard `json:"boards"`
		Total  int64                `json:"total"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		logger.Error("unmarshal user created storyboards failed", zap.Error(err))
		return nil, 0, err
	}

	return result.Boards, result.Total, nil
}

// SetUserCreatedStoryBoards 设置用户创建的故事板缓存
func (sbc *StoryBoardCache) SetUserCreatedStoryBoards(ctx context.Context, userID int64, page, pageSize int, boards []*models.StoryBoard, total int64) error {
	key := fmt.Sprintf("%s%d:%d:%d", StoryBoardUserCreatedPrefix, userID, page, pageSize)

	result := struct {
		Boards []*models.StoryBoard `json:"boards"`
		Total  int64                `json:"total"`
	}{
		Boards: boards,
		Total:  total,
	}

	data, err := json.Marshal(result)
	if err != nil {
		logger.Error("marshal user created storyboards failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryBoardUserCreatedTTL)
}

// DeleteUserCreatedStoryBoards 删除用户创建的故事板缓存
func (sbc *StoryBoardCache) DeleteUserCreatedStoryBoards(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%s%d", StoryBoardUserCreatedPrefix, userID)
	return cache.DelCache(ctx, key)
}

// GetStoryBoardLikeCount 获取故事板点赞数缓存
func (sbc *StoryBoardCache) GetStoryBoardLikeCount(ctx context.Context, boardID int64) (int64, error) {
	key := fmt.Sprintf("%s%d", StoryBoardLikeCountPrefix, boardID)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := json.Unmarshal(data, &count); err != nil {
		logger.Error("unmarshal storyboard like count failed", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// SetStoryBoardLikeCount 设置故事板点赞数缓存
func (sbc *StoryBoardCache) SetStoryBoardLikeCount(ctx context.Context, boardID int64, count int64) error {
	key := fmt.Sprintf("%s%d", StoryBoardLikeCountPrefix, boardID)
	data, err := json.Marshal(count)
	if err != nil {
		logger.Error("marshal storyboard like count failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryBoardLikeCountTTL)
}

// DeleteStoryBoardLikeCount 删除故事板点赞数缓存
func (sbc *StoryBoardCache) DeleteStoryBoardLikeCount(ctx context.Context, boardID int64) error {
	key := fmt.Sprintf("%s%d", StoryBoardLikeCountPrefix, boardID)
	return cache.DelCache(ctx, key)
}

// GetNextStoryBoards 获取下一个故事板列表缓存
func (sbc *StoryBoardCache) GetNextStoryBoards(ctx context.Context, storyID, prevBoardID int64, offset, pageSize int, orderBy string) ([]*models.StoryBoard, error) {
	key := fmt.Sprintf("%s%d:%d:%d:%d:%s", StoryBoardNextPrefix, storyID, prevBoardID, offset, pageSize, orderBy)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var boards []*models.StoryBoard
	if err := json.Unmarshal(data, &boards); err != nil {
		logger.Error("unmarshal next storyboards failed", zap.Error(err))
		return nil, err
	}

	return boards, nil
}

// SetNextStoryBoards 设置下一个故事板列表缓存
func (sbc *StoryBoardCache) SetNextStoryBoards(ctx context.Context, storyID, prevBoardID int64, offset, pageSize int, orderBy string, boards []*models.StoryBoard) error {
	key := fmt.Sprintf("%s%d:%d:%d:%d:%s", StoryBoardNextPrefix, storyID, prevBoardID, offset, pageSize, orderBy)
	data, err := json.Marshal(boards)
	if err != nil {
		logger.Error("marshal next storyboards failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryBoardNextTTL)
}

// DeleteNextStoryBoards 删除下一个故事板列表缓存
func (sbc *StoryBoardCache) DeleteNextStoryBoards(ctx context.Context, storyID int64) error {
	key := fmt.Sprintf("%s%d", StoryBoardNextPrefix, storyID)
	return cache.DelCache(ctx, key)
}

// InvalidateStoryBoardCache 清除故事板相关所有缓存
func (sbc *StoryBoardCache) InvalidateStoryBoardCache(ctx context.Context, boardID int64) error {
	// 删除故事板详情缓存
	if err := sbc.DeleteStoryBoardDetail(ctx, boardID); err != nil {
		logger.Warn("delete storyboard detail cache failed", zap.Error(err))
	}

	// 删除故事板场景缓存
	if err := sbc.DeleteStoryBoardScenes(ctx, boardID); err != nil {
		logger.Warn("delete storyboard scenes cache failed", zap.Error(err))
	}

	// 删除故事板角色缓存
	if err := sbc.DeleteStoryBoardRoles(ctx, boardID); err != nil {
		logger.Warn("delete storyboard roles cache failed", zap.Error(err))
	}

	// 删除故事板点赞数缓存
	if err := sbc.DeleteStoryBoardLikeCount(ctx, boardID); err != nil {
		logger.Warn("delete storyboard like count cache failed", zap.Error(err))
	}

	return nil
}

// InvalidateStoryRelatedCache 清除故事相关的故事板缓存
func (sbc *StoryBoardCache) InvalidateStoryRelatedCache(ctx context.Context, storyID int64) error {
	// 删除故事板列表缓存
	if err := sbc.DeleteStoryBoardList(ctx, storyID); err != nil {
		logger.Warn("delete storyboard list cache failed", zap.Error(err))
	}

	// 删除下一个故事板缓存
	if err := sbc.DeleteNextStoryBoards(ctx, storyID); err != nil {
		logger.Warn("delete next storyboards cache failed", zap.Error(err))
	}

	return nil
}

// InvalidateUserStoryBoardCache 清除用户故事板相关缓存
func (sbc *StoryBoardCache) InvalidateUserStoryBoardCache(ctx context.Context, userID int64) error {
	// 删除用户创建的故事板缓存
	if err := sbc.DeleteUserCreatedStoryBoards(ctx, userID); err != nil {
		logger.Warn("delete user created storyboards cache failed", zap.Error(err))
	}

	return nil
}

func (sbc *StoryBoardCache) InvalidateStoryBoardScenes(ctx context.Context, boardID int64) error {
	key := fmt.Sprintf("%s%d", StoryBoardScenesPrefix, boardID)
	return cache.DelCache(ctx, key)
}
