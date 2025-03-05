package story

import (
	"context"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils"
)

func (s *StoryService) GetStoryRoleCurrentUserStatus(ctx context.Context, roleId int64) (*api.WhatCurrentUserStatus, error) {
	// 查询用户ID
	if roleId == 0 {
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否关注了角色
	follow, err := models.GetWatchItemByStoryRoleAndUser(ctx, roleId, int64(userID))
	if err != nil {
		return nil, err
	}
	if follow != nil && follow.Deleted == false {
		cu.IsFollowed = true
	}
	// 查询用户是否点赞了角色
	like, err := models.GetLikeItemByStoryRoleAndUser(ctx, roleId, int(userID))
	if err != nil {
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
	}
	return cu, nil
}

func (s *StoryService) GetStoryCurrentUserStatus(ctx context.Context, storyId int64) (*api.WhatCurrentUserStatus, error) {
	// 查询用户ID
	if storyId == 0 {
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否关注了角色
	follow, err := models.GetWatchItemByStoryAndUser(ctx, storyId, int(userID))
	if err != nil {
		return nil, err
	}
	if follow != nil && follow.Deleted == false {
		cu.IsFollowed = true
	}
	// 查询用户是否点赞了角色
	like, err := models.GetLikeItemByStoryAndUser(ctx, storyId, int(userID))
	if err != nil {
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
	}
	return cu, nil
}

func (s *StoryService) GetStoryboardCurrentUserStatus(ctx context.Context, storyboardId int64) (*api.WhatCurrentUserStatus, error) {
	// 查询用户ID
	if storyboardId == 0 {
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否点赞了角色
	like, err := models.GetLikeItemByStoryBoardAndUser(ctx, storyboardId, int(userID))
	if err != nil {
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
	}
	return cu, nil
}
