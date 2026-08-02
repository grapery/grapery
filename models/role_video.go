package models

import (
	"context"

	"github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RoleVideo struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	RoleId      int64         `gorm:"column:role_id" json:"role_id,omitempty"`
	VideoUrl    string        `gorm:"column:video_url" json:"video_url,omitempty"`
	RefImages   string        `gorm:"column:ref_images" json:"ref_images,omitempty"`
	Prompt      string        `gorm:"column:prompt" json:"prompt,omitempty"`
	IsBanned    bool          `gorm:"column:is_banned" json:"is_banned,omitempty"`
	CreatorId   int64         `gorm:"column:creator_id" json:"creator_id,omitempty"`
	LikedCount  int64         `gorm:"column:liked_count" json:"liked_count,omitempty"`
	SharedCount int64         `gorm:"column:shared_count" json:"shared_count,omitempty"`
	TokenCost   int64         `gorm:"column:token_cost" json:"token_cost,omitempty"`
	TokenSourse string        `gorm:"column:token_sourse" json:"token_sourse,omitempty"`
	IsPublished bool          `gorm:"column:is_published" json:"is_published,omitempty"`
	Stage       gen.TaskStage `gorm:"column:stage" json:"stage,omitempty"`
	TaskId      string        `gorm:"column:task_id" json:"task_id,omitempty"`
}

func (r RoleVideo) TableName() string {
	return "role_video"
}

func CreateRoleVideo(ctx context.Context, roleId int64, videoUrl string, prompt string, creatorId int64, tokenCost int64, tokenSourse string) error {
	roleVideo := &RoleVideo{
		RoleId:      roleId,
		VideoUrl:    videoUrl,
		Prompt:      prompt,
		CreatorId:   creatorId,
		TokenCost:   tokenCost,
		TokenSourse: tokenSourse,
	}
	if err := DataBase().Create(roleVideo).Error; err != nil {
		log.Log().Error("CreateRoleVideo failed: create role video error", zap.Error(err), zap.Any("roleVideo", roleVideo))
		return err
	}
	log.Log().Info("CreateRoleVideo success", zap.Any("roleVideo", roleVideo))
	return nil
}

func GetRoleVideoByRoleId(ctx context.Context, roleId int64) ([]*RoleVideo, int, bool, error) {
	var roleVideos []*RoleVideo
	var total int64
	var hasMore = false
	if err := DataBase().Model(&RoleVideo{}).
		Where("role_id = ?", roleId).
		Where("is_banned = ?", false).
		Count(&total).Error; err != nil {
		return nil, 0, hasMore, err
	}
	log.Log().Info("Total role videos found:", zap.Int64("total", total))
	if err := DataBase().Model(&RoleVideo{}).
		Where("role_id = ?", roleId).
		Where("is_banned = ?", false).
		Find(&roleVideos).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, hasMore, nil
		}
		return nil, 0, hasMore, err
	}
	log.Log().Info("Role videos found:", zap.Int64("roleId", roleId))
	return roleVideos, int(total), hasMore, nil
}

func GetRoleVideoByCreatorId(ctx context.Context, creatorId int64) ([]*RoleVideo, int, bool, error) {
	var roleVideos []*RoleVideo
	var total int64
	var hasMore = false
	if err := DataBase().Model(&RoleVideo{}).
		Where("creator_id = ?", creatorId).
		Where("is_banned = ?", false).
		Count(&total).Error; err != nil {
		return nil, 0, hasMore, err
	}
	log.Log().Info("Total role videos found:", zap.Int64("total", total))
	if err := DataBase().Model(&RoleVideo{}).
		Where("creator_id = ?", creatorId).
		Where("is_banned = ?", false).
		Find(&roleVideos).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, hasMore, nil
		}
		return nil, 0, hasMore, err
	}
	log.Log().Info("Role videos found:", zap.Int64("creatorId", creatorId))
	return roleVideos, int(total), hasMore, nil
}

func GetRoleVideoById(ctx context.Context, id int64) (*RoleVideo, error) {
	var roleVideo RoleVideo
	if err := DataBase().
		Where("id = ?", id).
		Where("is_banned = ?", false).
		First(&roleVideo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	log.Log().Info("Role video found:", zap.Int64("id", id))
	return &roleVideo, nil
}

func GetRoleVideoByLikedCount(ctx context.Context, offset int64, pageSize int64) ([]*RoleVideo, int, bool, error) {
	var roleVideos []*RoleVideo
	var total int64
	var hasMore = false
	if err := DataBase().
		Model(&RoleVideo{}).
		Where("liked_count > 0").
		Where("is_banned = ?", false).
		Count(&total).Error; err != nil {
		return nil, 0, hasMore, err
	}
	log.Log().Info("Total role videos found:", zap.Int64("total", total))
	if err := DataBase().
		Model(&RoleVideo{}).
		Where("liked_count > 0").
		Where("is_banned = ?", false).
		Offset(int(offset * pageSize)).
		Limit(int(pageSize)).
		Order("create_at desc").Find(&roleVideos).Error; err != nil {
		return nil, 0, hasMore, err
	}
	hasMore = len(roleVideos) >= int(pageSize)
	log.Log().Info("Role videos found:", zap.Int("count", len(roleVideos)))
	return roleVideos, int(total), hasMore, nil
}

func BanRoleVideo(ctx context.Context, id int64) error {
	roleVideo := &RoleVideo{
		ID: uint(id),
	}
	if err := DataBase().Model(roleVideo).Update("is_banned", true).Error; err != nil {
		log.Log().Error("BanRoleVideo failed: ban role video error", zap.Error(err), zap.Int64("id", id))
		return err
	}
	log.Log().Info("BanRoleVideo success", zap.Int64("id", id))
	return nil
}

func UnbanRoleVideo(ctx context.Context, id int64) error {
	roleVideo := &RoleVideo{
		ID: uint(id),
	}
	if err := DataBase().Model(roleVideo).Update("is_banned", false).Error; err != nil {
		log.Log().Error("UnbanRoleVideo failed: unban role video error", zap.Error(err), zap.Int64("id", id))
		return err
	}
	log.Log().Info("UnbanRoleVideo success", zap.Int64("id", id))
	return nil
}

type RoleVideoLiked struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	RoleVideoId int64 `gorm:"column:role_video_id" json:"role_video_id,omitempty"`
	UserId      int64 `gorm:"column:user_id" json:"user_id,omitempty"`
}

func (r RoleVideoLiked) TableName() string {
	return "role_video_liked"
}

func CreateRoleVideoLiked(ctx context.Context, roleVideoId int64, userId int64) error {
	roleVideoLiked := &RoleVideoLiked{
		RoleVideoId: roleVideoId,
		UserId:      userId,
	}
	if err := DataBase().Create(roleVideoLiked).Error; err != nil {
		log.Log().Error("CreateRoleVideoLiked failed: create role video liked error", zap.Error(err), zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
		return err
	}
	log.Log().Info("CreateRoleVideoLiked success", zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
	return nil
}

func DeleteRoleVideoLiked(ctx context.Context, roleVideoId int64, userId int64) error {
	roleVideoLiked := &RoleVideoLiked{
		RoleVideoId: roleVideoId,
		UserId:      userId,
	}
	if err := DataBase().Delete(roleVideoLiked).Error; err != nil {
		log.Log().Error("DeleteRoleVideoLiked failed: delete role video liked error", zap.Error(err), zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
		return err
	}
	log.Log().Info("DeleteRoleVideoLiked success", zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
	return nil
}

func IsRoleVideoLiked(ctx context.Context, roleVideoId int64, userId int64) (bool, error) {
	roleVideoLiked := &RoleVideoLiked{
		RoleVideoId: roleVideoId,
		UserId:      userId,
	}
	if err := DataBase().
		Where(roleVideoLiked).
		Where("is_banned = ?", false).
		First(roleVideoLiked).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		log.Log().Error("IsRoleVideoLiked failed: find role video liked error", zap.Error(err), zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
		return false, err
	}
	log.Log().Info("IsRoleVideoLiked success", zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
	return true, nil
}

type RoleVideoShared struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	RoleVideoId int64 `gorm:"column:role_video_id" json:"role_video_id,omitempty"`
	UserId      int64 `gorm:"column:user_id" json:"user_id,omitempty"`
}

func (r RoleVideoShared) TableName() string {
	return "role_video_shared"
}

func CreateRoleVideoShared(ctx context.Context, roleVideoId int64, userId int64) error {
	roleVideoShared := &RoleVideoShared{
		RoleVideoId: roleVideoId,
		UserId:      userId,
	}
	if err := DataBase().Create(roleVideoShared).Error; err != nil {
		log.Log().Error("CreateRoleVideoShared failed: create role video shared error", zap.Error(err), zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
		return err
	}
	log.Log().Info("CreateRoleVideoShared success", zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
	return nil
}

func DeleteRoleVideoShared(ctx context.Context, roleVideoId int64, userId int64) error {
	roleVideoShared := &RoleVideoShared{
		RoleVideoId: roleVideoId,
		UserId:      userId,
	}
	if err := DataBase().Delete(roleVideoShared).Error; err != nil {
		log.Log().Error("DeleteRoleVideoShared failed: delete role video shared error", zap.Error(err), zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
		return err
	}
	log.Log().Info("DeleteRoleVideoShared success", zap.Int64("roleVideoId", roleVideoId), zap.Int64("userId", userId))
	return nil
}
