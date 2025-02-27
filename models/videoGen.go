package models

import "context"

type VideoGen struct {
	IDBase
	OriginID   int64
	BoardID    int64
	RoleID     int64
	TaskID     string
	Uuid       string
	Status     int
	Prompt     string
	VideoUrl   string
	Timelength int
	FisrtFrame string
	EndFrame   string
	Code       string
	Message    string
	Deleted    int
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
