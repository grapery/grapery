package models

import (
	"context"
	"errors"

	"github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StoryBoardScene 代表故事板中的一个场景
// status: 1-有效, 0-无效
// gen_status: 0-未生成, 1-生成中, 2-完成, 3-失败
// task_id: 生成任务ID
// content: 场景文本内容
// character_ids: 角色ID列表
// image_prompts/audio_prompts/video_prompts: 多模态生成提示
// gen_result: 生成结果
// ...
type StoryBoardScene struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	Content         string             `gorm:"column:content" json:"content,omitempty"`                     // 场景内容文字描述
	CharacterIds    string             `gorm:"column:character_ids" json:"character_ids,omitempty"`         // 角色ID列表数组
	CreatorId       int64              `gorm:"column:creator_id" json:"creator_id,omitempty"`               // 创建者ID
	StoryId         int64              `gorm:"column:story_id" json:"story_id,omitempty"`                   // 故事ID
	BoardId         int64              `gorm:"column:board_id" json:"board_id,omitempty"`                   // 故事板ID
	ImagePrompts    string             `gorm:"column:image_prompts" json:"image_prompts,omitempty"`         // 图像生成提示
	AudioPrompts    string             `gorm:"column:audio_prompts" json:"audio_prompts,omitempty"`         // 音频生成提示
	VideoPrompts    string             `gorm:"column:video_prompts" json:"video_prompts,omitempty"`         // 视频生成提示
	GenStatus       gen.StoryGenStatus `gorm:"column:gen_status" json:"gen_status,omitempty"`               // 生成状态
	GenResult       string             `gorm:"column:gen_result" json:"gen_result,omitempty"`               // 生成结果
	Error           string             `gorm:"column:error" json:"error,omitempty"`                         // 最近一次生成错误信息
	VideoUrl        string             `gorm:"column:video_url" json:"video_url,omitempty"`                 // 视频URL
	ImageUrl        string             `gorm:"column:image_url" json:"image_url,omitempty"`                 // 图片URL
	AudioUrl        string             `gorm:"column:audio_url" json:"audio_url,omitempty"`                 // 音频URL
	SceneDefaultUrl string             `gorm:"column:scene_default_url" json:"scene_default_url,omitempty"` // 场景的默认URL
	Status          int                `gorm:"column:status" json:"status,omitempty"`                       // 记录状态
	Seed            int                `gorm:"column:seed" json:"seed,omitempty"`                           // 种子用来在生成视频和图片时，保持一致性
	TaskId          string             `gorm:"column:task_id" json:"task_id,omitempty"`                     // 任务ID,关联故事板的uuid
	UUID            string             `gorm:"column:uuid" json:"uuid,omitempty"`                           // 唯一标识，对接三方平台的uuid
}

func (board StoryBoardScene) TableName() string {
	return "story_board_sence"
}

func CreateStoryBoardScene(ctx context.Context, scene *StoryBoardScene) (int64, error) {
	scene.Status = 1
	if err := DataBase().Model(scene).
		WithContext(ctx).
		Create(scene).Error; err != nil {
		return 0, err
	}
	return int64(scene.ID), nil
}

func GetStoryBoardScene(ctx context.Context, id int64) (*StoryBoardScene, error) {
	scene := &StoryBoardScene{}
	err := DataBase().Model(scene).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		First(scene).Error
	if err != nil {
		return nil, err
	}
	return scene, nil
}

func GetStoryBoardSceneByBoard(ctx context.Context, boardId int64) ([]*StoryBoardScene, error) {
	scenes := make([]*StoryBoardScene, 0)
	err := DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Find(&scenes).Error
	if err != nil {
		return nil, err
	}
	return scenes, nil
}

func GetStoryBoardScenesByBoard(ctx context.Context, boardId int64) ([]*StoryBoardScene, error) {
	scenes := make([]*StoryBoardScene, 0)
	err := DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Find(&scenes).Error
	if err != nil {
		return nil, err
	}
	return scenes, nil
}

func DelStoryBoardScene(ctx context.Context, id int64) error {
	err := DataBase().Model(&StoryBoardScene{}).WithContext(ctx).
		Where("id = ?", id).
		Update("status = ?", -1).Error
	return err
}

func UpdateStoryBoardScene(ctx context.Context, scene *StoryBoardScene) error {
	return DataBase().Model(scene).WithContext(ctx).
		Where("id = ?", scene.ID).
		Where("status >= 0").
		Updates(scene).Error
}

func UpdateStoryBoardSceneMultiColumn(ctx context.Context, id int64, columns map[string]interface{}) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Updates(columns).Error
}

func SetGenResult(ctx context.Context, id int64, result string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("gen_result = ?", result).Error
}

func SetGenerating(ctx context.Context, id int64, isGenerating int) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("is_generating = ?", isGenerating).Error
}

func UpdateStoryBoardSceneStatus(ctx context.Context, id int64, status int) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("status = ?", status).Error
}

func DeleteStoryBoardScene(ctx context.Context, sceneID int64, userID int64) error {
	return DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var scene StoryBoardScene
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", sceneID).
			First(&scene).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if err := tx.Model(&StoryBoardScene{}).
			Where("id = ?", sceneID).
			Updates(map[string]interface{}{
				"status":  -1,
				"deleted": true,
			}).Error; err != nil {
			return err
		}
		profile := &UserProfile{UserId: userID}
		if err := profile.DecrementContributStoryNum(ctx); err != nil {
			log.Log().Warn("[DeleteStoryBoardScene] 减少贡献故事数失败", zap.Error(err), zap.Int64("userId", userID), zap.Int64("sceneId", sceneID))
		}
		return nil
	})
}

func BatchUpdateStoryBoardSceneStatus(ctx context.Context, ids []int64, status int) error {
	// 去重ID，避免重复操作
	ids = UniqueInt64s(ids)
	if len(ids) == 0 {
		return nil
	}

	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id in (?)", ids).
		Where("status >= 0").
		Update("status = ?", status).Error
}

func UpdateStoryBoardSceneContent(ctx context.Context, id int64, content string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("content = ?", content).Error
}

func UpdateStoryBoardSceneCharacterIds(ctx context.Context, id int64, characterIds string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("character_ids = ?", characterIds).Error
}

func UpdateStoryBoardSceneImagePrompts(ctx context.Context, id int64, imagePrompts string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("image_prompts = ?", imagePrompts).Error
}

func UpdateStoryBoardSceneAudioPrompts(ctx context.Context, id int64, audioPrompts string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("audio_prompts = ?", audioPrompts).Error
}

func UpdateStoryBoardSceneVideoPrompts(ctx context.Context, id int64, videoPrompts string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("video_prompts = ?", videoPrompts).Error
}

func UpdateStoryBoardSceneGenResult(ctx context.Context, id int64, genResult string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("gen_result = ?", genResult).Error
}

// StoryBoardRole 代表故事板中的角色
// is_main: 0-主线人物, 1~* 其他分支人物
// is_published: 1-发布, 其他-未发布
