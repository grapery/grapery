package models

import "context"

type ImageGen struct {
	IDBase
	OriginID int64
	BoardID  int64
	RoleID   int64
	TaskID   string
	Uuid     string
	Status   int
	Prompt   string
	ImageUrl string
	Code     string
	Message  string
	Deleted  int
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
