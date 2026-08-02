package story

import (
	"context"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/cache"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

var storyHelper HelperServer

func init() {
	storyHelper = NewStoryHelper()
}

func GetStoryHelper() HelperServer {
	if storyHelper == nil {
		storyHelper = NewStoryHelper()
	}
	return storyHelper
}

func NewStoryHelper() *HelperService {
	return &HelperService{}
}

type HelperService struct {
}

type HelperServer interface {
	GetStoryRoleCurrentUserStatus(ctx context.Context, roleId int64) (*api.WhatCurrentUserStatus, error)
	GetStoryCurrentUserStatus(ctx context.Context, storyId int64) (*api.WhatCurrentUserStatus, error)
	GetStoryboardCurrentUserStatus(ctx context.Context, storyboardId int64) (*api.WhatCurrentUserStatus, error)
	GetGroupCurrentUserStatus(ctx context.Context, groupId int64) (*api.WhatCurrentUserStatus, error)
}

func (s *HelperService) GetGroupCurrentUserStatus(ctx context.Context, groupId int64) (*api.WhatCurrentUserStatus, error) {
	if groupId == 0 {
		log.Log().Warn("[GetGroupCurrentUserStatus] groupId为0，直接返回nil", zap.Int64("groupId", groupId))
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		log.Log().Error("[GetGroupCurrentUserStatus] 获取用户ID失败", zap.Error(err), zap.Int64("groupId", groupId))
		return nil, err
	}
	log.Log().Info("[GetGroupCurrentUserStatus] 获取用户ID成功", zap.Int64("userID", int64(userID)), zap.Int64("groupId", groupId))

	// 尝试从缓存获取用户状态
	userStatusCache := cache.GetUserStatusCache()
	cacheKey := userStatusCache.GetGroupUserStatusKey(groupId, int64(userID))
	cachedStatus, err := userStatusCache.GetCachedUserStatus(ctx, cacheKey)
	if err == nil && cachedStatus != nil {
		log.Log().Info("[GetGroupCurrentUserStatus] 从缓存获取用户状态成功", zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
		return cachedStatus, nil
	}

	// 缓存未命中，从数据库查询
	cu := new(api.WhatCurrentUserStatus)
	// follow, err := models.GetGroupFollowItemByGroupAndUser(ctx, groupId, int64(userID))
	// if err != nil {
	// 	log.Log().Error("[GetGroupCurrentUserStatus] 查询关注关系失败", zap.Error(err), zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
	// 	return nil, err
	// }
	// if follow != nil && follow.Deleted == false {
	// 	cu.IsFollowed = true
	// 	log.Log().Info("[GetGroupCurrentUserStatus] 用户已关注小组", zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
	// } else {
	// 	log.Log().Info("[GetGroupCurrentUserStatus] 用户未关注小组", zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
	// }
	// like, err := models.GetGroupLikeItemByGroupAndUser(ctx, groupId, int64(userID))
	// if err != nil {
	// 	log.Log().Error("[GetGroupCurrentUserStatus] 查询点赞关系失败", zap.Error(err), zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
	// 	return nil, err
	// }
	// if like != nil && like.Deleted == false {
	// 	cu.IsLiked = true
	// 	log.Log().Info("[GetGroupCurrentUserStatus] 用户已点赞小组", zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
	// } else {
	// 	log.Log().Info("[GetGroupCurrentUserStatus] 用户未点赞小组", zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
	// }

	// // 将结果缓存
	// err = userStatusCache.CacheUserStatus(ctx, cacheKey, cu)
	// if err != nil {
	// 	log.Log().Warn("[GetGroupCurrentUserStatus] 缓存用户状态失败", zap.Error(err), zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
	// }

	// log.Log().Info("[GetGroupCurrentUserStatus] 返回用户状态", zap.Any("status", cu), zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
	return cu, nil
}

func (s *HelperService) GetStoryRoleCurrentUserStatus(ctx context.Context, roleId int64) (*api.WhatCurrentUserStatus, error) {
	if roleId == 0 {
		log.Log().Warn("[GetStoryRoleCurrentUserStatus] roleId为0，直接返回nil", zap.Int64("roleId", roleId))
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		log.Log().Error("[GetStoryRoleCurrentUserStatus] 获取用户ID失败", zap.Error(err), zap.Int64("roleId", roleId))
		return nil, err
	}
	log.Log().Info("[GetStoryRoleCurrentUserStatus] 获取用户ID成功", zap.Int64("userID", int64(userID)), zap.Int64("roleId", roleId))

	// 尝试从缓存获取用户状态
	userStatusCache := cache.GetUserStatusCache()
	cacheKey := userStatusCache.GetStoryRoleUserStatusKey(roleId, int64(userID))
	cachedStatus, err := userStatusCache.GetCachedUserStatus(ctx, cacheKey)
	if err == nil && cachedStatus != nil {
		log.Log().Info("[GetStoryRoleCurrentUserStatus] 从缓存获取用户状态成功", zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
		return cachedStatus, nil
	}

	// 缓存未命中，从数据库查询
	cu := new(api.WhatCurrentUserStatus)
	follow, err := models.GetWatchItemByStoryRoleAndUser(ctx, roleId, int64(userID))
	if err != nil {
		log.Log().Error("[GetStoryRoleCurrentUserStatus] 查询关注关系失败", zap.Error(err), zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
		return nil, err
	}
	if follow != nil && follow.Deleted == false {
		cu.IsFollowed = true
		log.Log().Info("[GetStoryRoleCurrentUserStatus] 用户已关注角色", zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
	} else {
		log.Log().Info("[GetStoryRoleCurrentUserStatus] 用户未关注角色", zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
	}
	like, err := models.GetLikeItemByStoryRoleAndUser(ctx, roleId, int(userID))
	if err != nil {
		log.Log().Error("[GetStoryRoleCurrentUserStatus] 查询点赞关系失败", zap.Error(err), zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
		log.Log().Info("[GetStoryRoleCurrentUserStatus] 用户已点赞角色", zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
	} else {
		log.Log().Info("[GetStoryRoleCurrentUserStatus] 用户未点赞角色", zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
	}

	// 将结果缓存
	err = userStatusCache.CacheUserStatus(ctx, cacheKey, cu)
	if err != nil {
		log.Log().Warn("[GetStoryRoleCurrentUserStatus] 缓存用户状态失败", zap.Error(err), zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
	}

	log.Log().Info("[GetStoryRoleCurrentUserStatus] 返回用户状态", zap.Any("status", cu), zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
	return cu, nil
}

func (s *HelperService) GetStoryCurrentUserStatus(ctx context.Context, storyId int64) (*api.WhatCurrentUserStatus, error) {
	if storyId == 0 {
		log.Log().Warn("[GetStoryCurrentUserStatus] storyId为0，直接返回nil", zap.Int64("storyId", storyId))
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		log.Log().Error("[GetStoryCurrentUserStatus] 获取用户ID失败", zap.Error(err), zap.Int64("storyId", storyId))
		return nil, err
	}
	log.Log().Info("[GetStoryCurrentUserStatus] 获取用户ID成功", zap.Int64("userID", int64(userID)), zap.Int64("storyId", storyId))

	// 尝试从缓存获取用户状态
	userStatusCache := cache.GetUserStatusCache()
	cacheKey := userStatusCache.GetStoryUserStatusKey(storyId, int64(userID))
	cachedStatus, err := userStatusCache.GetCachedUserStatus(ctx, cacheKey)
	if err == nil && cachedStatus != nil {
		log.Log().Info("[GetStoryCurrentUserStatus] 从缓存获取用户状态成功", zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
		return cachedStatus, nil
	}

	// 缓存未命中，从数据库查询
	cu := new(api.WhatCurrentUserStatus)
	follow, err := models.GetWatchItemByStoryAndUser(ctx, storyId, int(userID))
	if err != nil {
		log.Log().Error("[GetStoryCurrentUserStatus] 查询关注关系失败", zap.Error(err), zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
		return nil, err
	}
	if follow != nil && follow.Deleted == false {
		cu.IsFollowed = true
		log.Log().Info("[GetStoryCurrentUserStatus] 用户已关注故事", zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
	} else {
		log.Log().Info("[GetStoryCurrentUserStatus] 用户未关注故事", zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
	}
	like, err := models.GetLikeItemByStoryAndUser(ctx, storyId, int(userID))
	if err != nil {
		log.Log().Error("[GetStoryCurrentUserStatus] 查询点赞关系失败", zap.Error(err), zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
		log.Log().Info("[GetStoryCurrentUserStatus] 用户已点赞故事", zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
	} else {
		log.Log().Info("[GetStoryCurrentUserStatus] 用户未点赞故事", zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
	}

	// 将结果缓存
	err = userStatusCache.CacheUserStatus(ctx, cacheKey, cu)
	if err != nil {
		log.Log().Warn("[GetStoryCurrentUserStatus] 缓存用户状态失败", zap.Error(err), zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
	}

	log.Log().Info("[GetStoryCurrentUserStatus] 返回用户状态", zap.Any("status", cu), zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
	return cu, nil
}

func (s *HelperService) GetStoryboardCurrentUserStatus(ctx context.Context, storyboardId int64) (*api.WhatCurrentUserStatus, error) {
	if storyboardId == 0 {
		log.Log().Warn("[GetStoryboardCurrentUserStatus] storyboardId为0，直接返回nil", zap.Int64("storyboardId", storyboardId))
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		log.Log().Error("[GetStoryboardCurrentUserStatus] 获取用户ID失败", zap.Error(err), zap.Int64("storyboardId", storyboardId))
		return nil, err
	}
	log.Log().Info("[GetStoryboardCurrentUserStatus] 获取用户ID成功", zap.Int64("userID", int64(userID)), zap.Int64("storyboardId", storyboardId))

	// 尝试从缓存获取用户状态
	userStatusCache := cache.GetUserStatusCache()
	cacheKey := userStatusCache.GetStoryboardUserStatusKey(storyboardId, int64(userID))
	cachedStatus, err := userStatusCache.GetCachedUserStatus(ctx, cacheKey)
	if err == nil && cachedStatus != nil {
		log.Log().Info("[GetStoryboardCurrentUserStatus] 从缓存获取用户状态成功", zap.Int64("storyboardId", storyboardId), zap.Int64("userID", int64(userID)))
		return cachedStatus, nil
	}

	// 缓存未命中，从数据库查询
	cu := new(api.WhatCurrentUserStatus)
	like, err := models.GetLikeItemByStoryBoardAndUser(ctx, storyboardId, int(userID))
	if err != nil {
		log.Log().Error("[GetStoryboardCurrentUserStatus] 查询点赞关系失败", zap.Error(err), zap.Int64("storyboardId", storyboardId), zap.Int64("userID", int64(userID)))
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
		log.Log().Info("[GetStoryboardCurrentUserStatus] 用户已点赞分镜", zap.Int64("storyboardId", storyboardId), zap.Int64("userID", int64(userID)))
	} else {
		log.Log().Info("[GetStoryboardCurrentUserStatus] 用户未点赞分镜", zap.Int64("storyboardId", storyboardId), zap.Int64("userID", int64(userID)))
	}

	// 将结果缓存
	err = userStatusCache.CacheUserStatus(ctx, cacheKey, cu)
	if err != nil {
		log.Log().Warn("[GetStoryboardCurrentUserStatus] 缓存用户状态失败", zap.Error(err), zap.Int64("storyboardId", storyboardId), zap.Int64("userID", int64(userID)))
	}

	log.Log().Info("[GetStoryboardCurrentUserStatus] 返回用户状态", zap.Any("status", cu), zap.Int64("storyboardId", storyboardId), zap.Int64("userID", int64(userID)))
	return cu, nil
}
