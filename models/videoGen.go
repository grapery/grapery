package models

import (
	"context"

	"gorm.io/gorm"
)

// VideoGen 视频生成任务记录
type VideoGen struct {
	IDBase
	OriginID   int64  `gorm:"column:origin_id" json:"origin_id,omitempty"`     // 源故事ID
	BoardID    int64  `gorm:"column:board_id" json:"board_id,omitempty"`       // 故事板ID
	RoleID     int64  `gorm:"column:role_id" json:"role_id,omitempty"`         // 角色ID
	TaskID     string `gorm:"column:task_id" json:"task_id,omitempty"`         // 任务ID
	Uuid       string `gorm:"column:uuid" json:"uuid,omitempty"`               // 唯一标识
	Status     int    `gorm:"column:status" json:"status,omitempty"`           // 状态
	Prompt     string `gorm:"column:prompt" json:"prompt,omitempty"`           // 提示词
	VideoUrl   string `gorm:"column:video_url" json:"video_url,omitempty"`     // 视频URL
	Timelength int    `gorm:"column:timelength" json:"timelength,omitempty"`   // 时长
	FisrtFrame string `gorm:"column:fisrt_frame" json:"fisrt_frame,omitempty"` // 首帧
	EndFrame   string `gorm:"column:end_frame" json:"end_frame,omitempty"`     // 末帧
	Code       string `gorm:"column:code" json:"code,omitempty"`               // 错误码
	Message    string `gorm:"column:message" json:"message,omitempty"`         // 错误信息
	Deleted    int    `gorm:"column:deleted" json:"deleted,omitempty"`         // 是否删除
}

func (v VideoGen) TableName() string {
	return "video_gen"
}

func CreateVideoGen(ctx context.Context, videoGen *VideoGen) (int64, error) {
	err := DataBase().Table(videoGen.TableName()).Create(videoGen).Error
	if err != nil {
		return 0, err
	}
	return int64(videoGen.ID), nil
}

func GetVideoGen(ctx context.Context, id int64) (*VideoGen, error) {
	var videoGen VideoGen
	err := DataBase().Table(videoGen.TableName()).Where("id = ?", id).First(&videoGen).Error
	if err != nil {
		return nil, err
	}
	return &videoGen, nil
}

func UpdateVideoGen(ctx context.Context, videoGen *VideoGen) error {
	return DataBase().Table(videoGen.TableName()).Where("id = ?", videoGen.ID).Updates(videoGen).Error
}

func DeleteVideoGen(ctx context.Context, id int64) error {
	return DataBase().Table(VideoGen{}.TableName()).
		Where("id = ?", id).
		Update("deleted", 1).Error
}

func GetVideoGenList(ctx context.Context, page, pageSize int) ([]*VideoGen, error) {
	var videoGenList []*VideoGen
	err := DataBase().Table(VideoGen{}.TableName()).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&videoGenList).Error
	if err != nil {
		return nil, err
	}
	return videoGenList, nil
}

func GetVideoGenListByStatus(ctx context.Context, status int) ([]*VideoGen, error) {
	var videoGenList []*VideoGen
	err := DataBase().Table(VideoGen{}.TableName()).Where("status = ?", status).Find(&videoGenList).Error
	if err != nil {
		return nil, err
	}
	return videoGenList, nil
}

// 新增：分页获取VideoGen列表
func GetVideoGenListPage(ctx context.Context, offset, limit int) ([]*VideoGen, error) {
	var videos []*VideoGen
	err := DataBase().Model(&VideoGen{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&videos).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return videos, nil
}

// 新增：通过TaskID唯一查询
func GetVideoGenByTaskID(ctx context.Context, taskID string) (*VideoGen, error) {
	video := &VideoGen{}
	err := DataBase().Model(video).
		WithContext(ctx).
		Where("task_id = ?", taskID).
		First(video).Error
	if err != nil {
		return nil, err
	}
	return video, nil
}
