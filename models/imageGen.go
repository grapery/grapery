package models

import (
	"context"

	"gorm.io/gorm"
)

// ImageGen 图片生成任务记录
type ImageGen struct {
	IDBase
	OriginID int64  `gorm:"column:origin_id" json:"origin_id,omitempty"` // 源故事ID
	BoardID  int64  `gorm:"column:board_id" json:"board_id,omitempty"`   // 故事板ID
	RoleID   int64  `gorm:"column:role_id" json:"role_id,omitempty"`     // 角色ID
	TaskID   string `gorm:"column:task_id" json:"task_id,omitempty"`     // 任务ID
	Uuid     string `gorm:"column:uuid" json:"uuid,omitempty"`           // 唯一标识
	Status   int    `gorm:"column:status" json:"status,omitempty"`       // 状态
	Prompt   string `gorm:"column:prompt" json:"prompt,omitempty"`       // 提示词
	ImageUrl string `gorm:"column:image_url" json:"image_url,omitempty"` // 图片URL
	Code     string `gorm:"column:code" json:"code,omitempty"`           // 错误码
	Message  string `gorm:"column:message" json:"message,omitempty"`     // 错误信息
	Deleted  int    `gorm:"column:deleted" json:"deleted,omitempty"`     // 是否删除
}

func (i ImageGen) TableName() string {
	return "image_gen"
}

func CreateImageGen(ctx context.Context, imageGen *ImageGen) (int64, error) {
	err := DataBase().Table(imageGen.TableName()).Create(imageGen).Error
	if err != nil {
		return 0, err
	}
	return int64(imageGen.ID), nil
}

func GetImageGen(ctx context.Context, id int64) (*ImageGen, error) {
	var imageGen ImageGen
	err := DataBase().Table(imageGen.TableName()).Where("id = ?", id).First(&imageGen).Error
	if err != nil {
		return nil, err
	}
	return &imageGen, nil
}

func UpdateImageGen(ctx context.Context, imageGen *ImageGen) error {
	return DataBase().Table(imageGen.TableName()).Where("id = ?", imageGen.ID).Updates(imageGen).Error
}

func DeleteImageGen(ctx context.Context, id int64) error {
	return DataBase().Table(ImageGen{}.TableName()).
		Where("id = ?", id).
		Update("deleted", 1).Error
}

func GetImageGenList(ctx context.Context, page, pageSize int) ([]*ImageGen, error) {
	var imageGenList []*ImageGen
	err := DataBase().Table(ImageGen{}.TableName()).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&imageGenList).Error
	if err != nil {
		return nil, err
	}
	return imageGenList, nil
}

func GetImageGenListByStatus(ctx context.Context, status int) ([]*ImageGen, error) {
	var imageGenList []*ImageGen
	err := DataBase().Table(ImageGen{}.TableName()).
		Where("status = ?", status).
		Find(&imageGenList).Error
	if err != nil {
		return nil, err
	}
	return imageGenList, nil
}

// 新增：分页获取ImageGen列表
func GetImageGenListPage(ctx context.Context, offset, limit int) ([]*ImageGen, error) {
	var images []*ImageGen
	err := DataBase().Model(&ImageGen{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&images).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return images, nil
}

// 新增：通过TaskID唯一查询
func GetImageGenByTaskID(ctx context.Context, taskID string) (*ImageGen, error) {
	img := &ImageGen{}
	err := DataBase().Model(img).
		WithContext(ctx).
		Where("task_id = ?", taskID).
		First(img).Error
	if err != nil {
		return nil, err
	}
	return img, nil
}
