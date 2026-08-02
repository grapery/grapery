package models

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RolePoster struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	RoleId              int64         `gorm:"column:role_id" json:"role_id,omitempty"`
	RefImages           string        `gorm:"column:ref_images" json:"ref_images,omitempty"`
	Prompt              string        `gorm:"column:prompt" json:"prompt,omitempty"`
	PosterURL           string        `gorm:"column:poster_url" json:"poster_url,omitempty"`
	AdditionalImageUrls string        `gorm:"column:additional_image_urls" json:"additional_image_urls,omitempty"`
	Style               string        `gorm:"column:style" json:"style,omitempty"`
	IsBanned            bool          `gorm:"column:is_banned" json:"is_banned,omitempty"`
	CreatorId           int64         `gorm:"column:creator_id" json:"creator_id,omitempty"`
	LikedCount          int64         `gorm:"column:liked_count" json:"liked_count,omitempty"`
	SharedCount         int64         `gorm:"column:shared_count" json:"shared_count,omitempty"`
	TokenCost           int64         `gorm:"column:token_cost" json:"token_cost,omitempty"`
	TokenSourse         string        `gorm:"column:token_sourse" json:"token_sourse,omitempty"`
	IsPublished         bool          `gorm:"column:is_published" json:"is_published,omitempty"`
	Stage               gen.TaskStage `gorm:"column:stage" json:"stage,omitempty"`
}

func (r RolePoster) TableName() string {
	return "role_poster"
}

func CreateRolePoster(ctx context.Context, roleId int64, posterURL string,
	creatorId int64, refImages string, prompt string, style string, additionalImageUrls []string,
	tokenCost int64, tokenSourse string) (int64, error) {
	additionalImageUrlsStr, _ := json.Marshal(additionalImageUrls)
	rolePoster := &RolePoster{
		RoleId:              roleId,
		PosterURL:           posterURL,
		CreatorId:           creatorId,
		TokenCost:           tokenCost,
		TokenSourse:         tokenSourse,
		RefImages:           refImages,
		Prompt:              prompt,
		IsPublished:         false,
		Style:               style,
		AdditionalImageUrls: string(additionalImageUrlsStr),
		Stage:               gen.TaskStage_Init,
	}
	if err := DataBase().Create(rolePoster).Error; err != nil {
		log.Log().Error("CreateRolePoster failed: create role poster error", zap.Error(err), zap.Any("rolePoster", rolePoster))
		return 0, err
	}
	log.Log().Info("CreateRolePoster success", zap.Any("rolePoster", rolePoster))
	return int64(rolePoster.ID), nil
}

func SaveRolePosterUrl(ctx context.Context, rolePosterId int64, posterURL string) error {
	needUpdate := map[string]interface{}{
		"poster_url": posterURL,
	}
	if err := DataBase().
		Model(&RolePoster{}).
		WithContext(ctx).
		Where("id = ?", rolePosterId).
		Updates(needUpdate).Error; err != nil {
		log.Log().Error("SaveRolePosterUrl failed: update role poster error", zap.Error(err), zap.Any("rolePosterId", rolePosterId))
		return err
	}
	return nil
}

func UpdateRolePosterTokenCost(ctx context.Context, rolePosterId int64, tokenCost int64) error {
	needUpdate := map[string]interface{}{
		"token_cost": tokenCost,
	}
	if err := DataBase().
		Model(&RolePoster{}).
		WithContext(ctx).
		Where("id = ?", rolePosterId).
		Updates(needUpdate).Error; err != nil {
		log.Log().Error("UpdateRolePosterTokenCost failed: update role poster token cost error", zap.Error(err), zap.Int64("rolePosterId", rolePosterId), zap.Int64("tokenCost", tokenCost))
		return err
	}
	return nil
}

func PublishRolePoster(ctx context.Context, rolePosterId int64) error {
	needUpdate := map[string]interface{}{
		"is_published": true,
	}
	if err := DataBase().
		Model(RolePoster{}).
		WithContext(ctx).
		Where("id = ?", rolePosterId).
		Updates(needUpdate).Error; err != nil {
		log.Log().Error("PublishRolePoster failed: publish role poster error", zap.Error(err), zap.Int64("id", rolePosterId))
		return err
	}
	return nil
}

func GetRolePosterById(ctx context.Context, id int64) (*RolePoster, error) {
	var rolePoster RolePoster
	if err := DataBase().
		Where("id = ?", id).
		Where("is_banned = ?", false).
		First(&rolePoster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("GetRolePosterById failed: no role poster found", zap.Int64("id", id))
			return nil, nil
		}
		log.Log().Error("GetRolePosterById failed: find role poster error", zap.Error(err), zap.Int64("id", id))
		return nil, err
	}
	log.Log().Info("Role poster found:", zap.Int64("id", id))
	return &rolePoster, nil
}

func GetRolePosterByRoleId(ctx context.Context, roleId int64, offset int64, pageSize int64) ([]*RolePoster, int, bool, error) {
	var rolePosters []*RolePoster
	var total int64
	var hasMore = false
	if err := DataBase().Model(&RolePoster{}).
		Where("role_id = ?", roleId).
		Where("is_banned = ?", false).
		Where("is_published = ?", true).
		Count(&total).Error; err != nil {
		log.Log().Error("GetRolePosterByRoleId failed: count role posters error", zap.Error(err), zap.Int64("roleId", roleId))
		return nil, 0, hasMore, err
	}
	log.Log().Info("Total role posters found:", zap.Int64("total", total))
	if err := DataBase().Where("role_id = ?", roleId).
		Where("is_banned = ?", false).
		Where("is_published = ?", true).
		Offset(int(offset * pageSize)).
		Limit(int(pageSize)).
		Order("create_at desc").
		Find(&rolePosters).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("GetRolePosterByRoleId failed: no role posters found", zap.Int64("roleId", roleId))
			return nil, 0, hasMore, nil
		}
		log.Log().Error("GetRolePosterByRoleId failed: find role posters error", zap.Error(err), zap.Int64("roleId", roleId))
		return nil, 0, hasMore, err
	}
	hasMore = len(rolePosters) >= int(pageSize)
	log.Log().Info("Role posters found:", zap.Int("count", len(rolePosters)))
	return rolePosters, int(total), hasMore, nil
}

func GetUserLikedPostersWithPosterIds(ctx context.Context, userId int64, posterIds []int64) ([]*RolePoster, error) {
	// 去重ID，避免重复查询
	posterIds = UniqueInt64s(posterIds)
	if len(posterIds) == 0 {
		return nil, nil
	}

	var rolePosters []*RolePoster
	if err := DataBase().Model(&RolePoster{}).
		Where("id in (?)", posterIds).
		Where("creator_id = ?", userId).
		Where("is_banned = ?", false).
		Where("is_published = ?", true).
		Find(&rolePosters).Error; err != nil {
		return nil, err
	}
	return rolePosters, nil
}

func GetRolePosterByCreatorId(ctx context.Context, creatorId int64, offset int64, pageSize int64) ([]*RolePoster, int, bool, error) {
	var rolePosters []*RolePoster
	var total int64
	var hasMore = false
	if err := DataBase().Model(&RolePoster{}).
		Where("creator_id = ?", creatorId).
		Where("is_banned = ?", false).
		Where("is_published = ?", true).
		Count(&total).Error; err != nil {
		log.Log().Error("GetRolePosterByCreatorId failed: count role posters error", zap.Error(err), zap.Int64("creatorId", creatorId))
		return nil, 0, hasMore, err
	}
	log.Log().Info("Total role posters found:", zap.Int64("total", total))
	if err := DataBase().
		Where("creator_id = ?", creatorId).
		Where("is_banned = ?", false).
		Where("is_published = ?", true).
		Offset(int(offset * pageSize)).
		Limit(int(pageSize)).
		Order("create_at desc").
		Find(&rolePosters).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("GetRolePosterByCreatorId failed: no role posters found", zap.Int64("creatorId", creatorId))
			return nil, 0, hasMore, nil
		}
		log.Log().Error("GetRolePosterByCreatorId failed: find role posters error", zap.Error(err), zap.Int64("creatorId", creatorId))
		return nil, 0, hasMore, err
	}
	hasMore = len(rolePosters) >= int(pageSize)
	log.Log().Info("Role posters found:", zap.Int("count", len(rolePosters)))
	return rolePosters, int(total), hasMore, nil
}

func GetRolePosterByLikedCount(ctx context.Context, offset int64, pageSize int64) ([]*RolePoster, int, bool, error) {
	var rolePosters []*RolePoster
	var total int64
	var hasMore = false
	if err := DataBase().Model(&RolePoster{}).
		Where("liked_count > 0").
		Where("is_banned = ?", false).
		Count(&total).Error; err != nil {
		log.Log().Error("GetRolePosterByLikedCount failed: count role posters error", zap.Error(err))
		return nil, 0, hasMore, err
	}
	log.Log().Info("Total role posters found:", zap.Int64("total", total))
	if err := DataBase().
		Where("liked_count > 0").
		Where("is_banned = ?", false).
		Offset(int(offset * pageSize)).
		Limit(int(pageSize)).
		Order("create_at desc").
		Find(&rolePosters).Error; err != nil {
		return nil, 0, hasMore, err
	}
	hasMore = len(rolePosters) >= int(pageSize)
	log.Log().Info("Role posters found:", zap.Int("count", len(rolePosters)))
	return rolePosters, int(total), hasMore, nil
}

func LikeRolePoster(ctx context.Context, rolePosterId int64, userId int64) (bool, error) {
	var created bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var liked RolePosterLiked
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("role_poster_id = ? AND user_id = ?", rolePosterId, userId).
			First(&liked).Error
		switch {
		case err == nil:
			if !liked.Deleted {
				log.Log().Info("LikeRolePoster skipped: already liked", zap.Int64("rolePosterId", rolePosterId), zap.Int64("userId", userId))
				return nil
			}
			if err := tx.Model(&RolePosterLiked{}).
				Where("id = ?", liked.ID).
				Updates(map[string]interface{}{"deleted": false}).Error; err != nil {
				return err
			}
			created = true
		case errors.Is(err, gorm.ErrRecordNotFound):
			liked = RolePosterLiked{
				RolePosterId: rolePosterId,
				UserId:       userId,
			}
			if err := tx.Create(&liked).Error; err != nil {
				return err
			}
			created = true
		default:
			return err
		}

		if !created {
			return nil
		}

		if err := tx.Model(&RolePoster{}).
			Where("id = ?", rolePosterId).
			Update("liked_count", gorm.Expr("CASE WHEN liked_count IS NULL THEN 1 ELSE liked_count + 1 END")).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Log().Error("LikeRolePoster failed", zap.Error(err), zap.Int64("rolePosterId", rolePosterId), zap.Int64("userId", userId))
		return false, err
	}
	if created {
		log.Log().Info("LikeRolePoster success", zap.Int64("rolePosterId", rolePosterId), zap.Int64("userId", userId))
	}
	return created, nil
}

func DislikeRolePoster(ctx context.Context, rolePosterId int64, userId int64) (bool, error) {
	var removed bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var liked RolePosterLiked
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("role_poster_id = ? AND user_id = ?", rolePosterId, userId).
			First(&liked).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Log().Info("DislikeRolePoster skipped: not liked", zap.Int64("rolePosterId", rolePosterId), zap.Int64("userId", userId))
			return nil
		}
		if err != nil {
			return err
		}
		if liked.Deleted {
			log.Log().Info("DislikeRolePoster skipped: already unliked", zap.Int64("rolePosterId", rolePosterId), zap.Int64("userId", userId))
			return nil
		}

		if err := tx.Model(&RolePosterLiked{}).
			Where("id = ?", liked.ID).
			Updates(map[string]interface{}{"deleted": true}).Error; err != nil {
			return err
		}

		if err := tx.Model(&RolePoster{}).
			Where("id = ?", rolePosterId).
			Update("liked_count", gorm.Expr("CASE WHEN liked_count > 0 THEN liked_count - 1 ELSE 0 END")).Error; err != nil {
			return err
		}
		removed = true
		return nil
	})
	if err != nil {
		log.Log().Error("DislikeRolePoster failed", zap.Error(err), zap.Int64("rolePosterId", rolePosterId), zap.Int64("userId", userId))
		return false, err
	}
	if removed {
		log.Log().Info("DislikeRolePoster success", zap.Int64("rolePosterId", rolePosterId), zap.Int64("userId", userId))
	}
	return removed, nil
}

func IsRolePosterLiked(ctx context.Context, rolePosterId int64, userId int64) (bool, error) {
	rolePosterLiked := &RolePosterLiked{
		RolePosterId: rolePosterId,
		UserId:       userId,
	}
	if err := DataBase().WithContext(ctx).
		Model(&RolePosterLiked{}).
		Where("role_poster_id = ? AND user_id = ? AND deleted = ?", rolePosterId, userId, false).
		First(rolePosterLiked).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		log.Log().Error("IsRolePosterLiked failed: find role poster liked error", zap.Error(err), zap.Int64("rolePosterId", rolePosterId), zap.Int64("userId", userId))
		return false, err
	}
	return true, nil
}

func BanRolePoster(ctx context.Context, rolePosterId int64) error {
	rolePoster := &RolePoster{
		ID: uint(rolePosterId),
	}
	if err := DataBase().Model(rolePoster).Update("is_banned", true).Error; err != nil {
		log.Log().Error("BanRolePoster failed: ban role poster error", zap.Error(err), zap.Int64("rolePosterId", rolePosterId))
		return err
	}
	return nil
}

func UnbanRolePoster(ctx context.Context, rolePosterId int64) error {
	rolePoster := &RolePoster{
		ID: uint(rolePosterId),
	}
	if err := DataBase().Model(rolePoster).Update("is_banned", false).Error; err != nil {
		log.Log().Error("UnbanRolePoster failed: unban role poster error", zap.Error(err), zap.Int64("rolePosterId", rolePosterId))
		return err
	}
	return nil
}

type RolePosterLiked struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	RolePosterId int64 `gorm:"column:role_poster_id;uniqueIndex:uniq_role_poster_like,priority:1" json:"role_poster_id,omitempty"`
	UserId       int64 `gorm:"column:user_id;uniqueIndex:uniq_role_poster_like,priority:2" json:"user_id,omitempty"`
}

type RolePosterShared struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	RolePosterId int64 `gorm:"column:role_poster_id" json:"role_poster_id,omitempty"`
	UserId       int64 `gorm:"column:user_id" json:"user_id,omitempty"`
}

func (r RolePosterLiked) TableName() string {
	return "role_poster_liked"
}

func (r RolePosterShared) TableName() string {
	return "role_poster_shared"
}

func CreateRolePosterLiked(ctx context.Context, rolePosterId int64, userId int64) error {
	rolePosterLiked := &RolePosterLiked{
		RolePosterId: rolePosterId,
		UserId:       userId,
	}
	if err := DataBase().Create(rolePosterLiked).Error; err != nil {
		log.Log().Error("CreateRolePosterLiked failed: create role poster liked error", zap.Error(err), zap.Any("rolePosterLiked", rolePosterLiked))
		return err
	}
	return nil
}

func CreateRolePosterShared(ctx context.Context, rolePosterId int64, userId int64) error {
	rolePosterShared := &RolePosterShared{
		RolePosterId: rolePosterId,
		UserId:       userId,
	}
	if err := DataBase().Create(rolePosterShared).Error; err != nil {
		log.Log().Error("CreateRolePosterShared failed: create role poster shared error", zap.Error(err), zap.Any("rolePosterShared", rolePosterShared))
		return err
	}
	return nil
}

func DeleteRolePosterLiked(ctx context.Context, rolePosterId int64, userId int64) error {
	rolePosterLiked := &RolePosterLiked{
		RolePosterId: rolePosterId,
		UserId:       userId,
	}
	if err := DataBase().Delete(rolePosterLiked).Error; err != nil {
		log.Log().Error("DeleteRolePosterLiked failed: delete role poster liked error", zap.Error(err), zap.Any("rolePosterLiked", rolePosterLiked))
		return err
	}
	return nil
}

func DeleteRolePosterShared(ctx context.Context, rolePosterId int64, userId int64) error {
	rolePosterShared := &RolePosterShared{
		RolePosterId: rolePosterId,
		UserId:       userId,
	}
	if err := DataBase().Delete(rolePosterShared).Error; err != nil {
		log.Log().Error("DeleteRolePosterShared failed: delete role poster shared error", zap.Error(err), zap.Any("rolePosterShared", rolePosterShared))
		return err
	}
	return nil
}
