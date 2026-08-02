package models

import (
	"context"

	"gorm.io/gorm"

	"github.com/grapery/common-protoc/gen"
)

// ImageGen 图片生成任务记录
type ImageGen struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	StoryId int64 `gorm:"column:story_id;index:idx_story_id" json:"story_id,omitempty"` // 源故事ID
	BoardId int64 `gorm:"column:board_id;index:idx_board_id" json:"board_id,omitempty"` // 故事板ID
	SceneId int64 `gorm:"column:scene_id;index:idx_scene_id" json:"scene_id,omitempty"` // 场景ID
	RoleId  int64 `gorm:"column:role_id;index:idx_role_id" json:"role_id,omitempty"`    // 角色ID

	TaskType gen.RenderType `gorm:"column:gen_type" json:"gen_type,omitempty"` // 生成类型
	TaskId   string         `gorm:"column:task_id" json:"task_id,omitempty"`   // 任务ID
	UUID     string         `gorm:"column:uuid" json:"uuid,omitempty"`         // 唯一标识
	Seed     int64          `gorm:"column:seed" json:"seed,omitempty"`         // 随机数种子

	Prompt    string `gorm:"column:prompt" json:"prompt,omitempty"`         // 提示词
	ImageUrl  string `gorm:"column:image_url" json:"image_url,omitempty"`   // 图片URL
	RefImages string `gorm:"column:ref_images" json:"ref_images,omitempty"` // 参考图片

	UserID      int64              `gorm:"column:user_id" json:"user_id,omitempty"`
	GenStatus   gen.StoryGenStatus `gorm:"column:gen_status" json:"gen_status,omitempty"`     // 生成状态（0:未生成,1:生成中,2:完成,3:失败）
	Code        string             `gorm:"column:code" json:"code,omitempty"`                 // 错误码
	Message     string             `gorm:"column:message" json:"message,omitempty"`           // 错误信息
	Deleted     int                `gorm:"column:deleted" json:"deleted,omitempty"`           // 是否删除
	Tokens      int                `gorm:"column:tokens" json:"tokens,omitempty"`             // 消耗token数
	StartTime   int64              `gorm:"column:start_time" json:"start_time,omitempty"`     // 生成开始时间
	EndTime     int64              `gorm:"column:end_time" json:"end_time,omitempty"`         // 生成完成时间
	LLmPlatform string             `gorm:"column:llm_platform" json:"llm_platform,omitempty"` // 使用的大模型平台
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
	err := DataBase().Table(imageGen.TableName()).
		WithContext(ctx).
		Where("id = ?", id).First(&imageGen).Error
	if err != nil {
		return nil, err
	}
	return &imageGen, nil
}

func UpdateImageGen(ctx context.Context, imageGen *ImageGen) error {
	return DataBase().
		Table(imageGen.TableName()).
		WithContext(ctx).
		Where("id = ?", imageGen.ID).Updates(imageGen).Error
}

func DeleteImageGen(ctx context.Context, id int64) error {
	return DataBase().Table(ImageGen{}.TableName()).
		WithContext(ctx).
		Where("id = ?", id).
		Update("deleted", 1).Error
}

func GetImageGenList(ctx context.Context, page, pageSize int) ([]*ImageGen, error) {
	var imageGenList []*ImageGen
	err := DataBase().Table(ImageGen{}.TableName()).
		WithContext(ctx).
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
		WithContext(ctx).
		Where("gen_status = ?", status).
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
