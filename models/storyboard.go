package models

import (
	"context"
	"time"

	"github.com/grapery/common-protoc/gen"
	"gorm.io/gorm"
)

// 场景，剧情，故事板
// StoryBoard 代表一个故事板（漫画/剧情单元）
type StoryBoard struct {
	IDBase
	Title       string `gorm:"column:title" json:"title,omitempty"`             // 故事板标题
	Description string `gorm:"column:description" json:"description,omitempty"` // 描述
	CreatorID   int64  `gorm:"column:creator_id" json:"creator_id,omitempty"`   // 创建者ID
	StoryID     int64  `gorm:"column:story_id" json:"story_id,omitempty"`       // 所属故事ID
	PrevId      int64  `gorm:"column:prev_id" json:"prev_id,omitempty"`         // 上一个故事板ID
	Avatar      string `gorm:"column:avatar" json:"avatar,omitempty"`           // 封面
	Status      int    `gorm:"column:status" json:"status,omitempty"`           // 是否删除（1:有效, 0:无效）
	Stage       int    `gorm:"column:stage" json:"stage,omitempty"`             // 0:初始化,1:生成中,2:完成,3:失败
	Params      string `gorm:"column:params" json:"params,omitempty"`           // 生成参数
	ForkAble    bool   `gorm:"column:fork_able" json:"fork_able,omitempty"`     // 是否可被fork
	ForkNum     int    `gorm:"column:fork_num" json:"fork_num,omitempty"`       // fork数
	LikeNum     int    `gorm:"column:like_num" json:"like_num,omitempty"`       // 点赞数
	CommentNum  int    `gorm:"column:comment_num" json:"comment_num,omitempty"` // 评论数
	RoleNum     int    `gorm:"column:role_num" json:"role_num,omitempty"`       // 角色数
	ShareNum    int    `gorm:"column:share_num" json:"share_num,omitempty"`     // 分享数
	Level       int    `gorm:"column:level" json:"level,omitempty"`             // 层级
	IsAiGen     bool   `gorm:"column:is_ai_gen" json:"is_ai_gen,omitempty"`     // 是否AI生成
}

func (board StoryBoard) TableName() string {
	return "story_board"
}

func IsForkable(ctx context.Context, id int64) (bool, error) {
	board, err := GetStoryboard(ctx, id)
	if err != nil {
		return false, err
	}
	return board.ForkAble, nil
}

func UpdateStoryBoardForkAble(ctx context.Context, id int64, forkAble bool) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("fork_able = ?", forkAble).Error
}

func IncrementStoryBoardForkNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("fork_num", gorm.Expr("fork_num + ?", 1)).Error
}

func IncrementStoryBoardLikeNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("like_num", gorm.Expr("like_num + ?", 1)).Error
}

func IncrementStoryBoardCommentNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("comment_num", gorm.Expr("comment_num + ?", 1)).Error
}

func IncrementStoryBoardShareNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("share_num", gorm.Expr("share_num + ?", 1)).Error
}

func DecrementStoryBoardForkNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("fork_num", gorm.Expr("fork_num - ?", 1)).Error
}

func DecrementStoryBoardLikeNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("like_num", gorm.Expr("like_num - ?", 1)).Error
}

func DecrementStoryBoardCommentNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("comment_num", gorm.Expr("comment_num - ?", 1)).Error
}

func DecrementStoryBoardShareNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("share_num", gorm.Expr("share_num - ?", 1)).Error
}

func IncrementStoryBoardRoleNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("role_num", gorm.Expr("role_num + ?", 1)).Error
}

func DecrementStoryBoardRoleNum(ctx context.Context, id int64) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("role_num", gorm.Expr("role_num - ?", 1)).Error
}

func UpdateStoryBoardStage(ctx context.Context, id int64, stage int) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("stage = ?", stage).Error
}

func UpdateStoryBoardParams(ctx context.Context, id int64, params string) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("params = ?", params).Error
}

func UpdateStoryBoardTitle(ctx context.Context, id int64, title string) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("title = ?", title).Error
}

func UpdateStoryBoardDescription(ctx context.Context, id int64, description string) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("description = ?", description).Error
}

func UpdateStoryBoardAvatar(ctx context.Context, id int64, avatar string) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("avatar = ?", avatar).Error
}

func CreateStoryBoard(ctx context.Context, board *StoryBoard) (int64, error) {
	if err := DataBase().Model(board).
		WithContext(ctx).
		Create(board).Error; err != nil {
		return 0, err
	}
	return int64(board.IDBase.ID), nil
}

func GetStoryboard(ctx context.Context, id int64) (*StoryBoard, error) {
	board := &StoryBoard{}
	err := DataBase().Model(board).
		WithContext(ctx).
		Where("id = ? and status >= 0", id).
		First(board).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return board, nil
}

func GetStoryboardsByPrevId(ctx context.Context, prevId int64) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("prev_id = ? and status >= 0", prevId).
		Order("create_at desc").
		Find(&boards).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return boards, nil
}

func GetStoryboardsByStory(ctx context.Context, storyID int64) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("story_id = ? and status >= 0", storyID).
		Order("create_at desc").
		Find(&boards).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return boards, nil
}

func GetStoryboardsByStoryMultiPage(ctx context.Context, storyID int64, page int, pageSize int) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("story_id = ? and status >= 0", storyID).
		Order("create_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&boards).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return boards, nil
}

func GetStoryboardsByCreator(ctx context.Context, creatorID int64) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("creator_id = ? and status >= 0", creatorID).
		Order("create_at desc").
		Find(&boards).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return boards, nil
}

func DelStoryboard(ctx context.Context, id int64) error {
	err := DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).
		Update("status = ?", -1).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	return nil
}

func UpdateStoryboard(ctx context.Context, board *StoryBoard) error {
	return DataBase().Model(board).WithContext(ctx).
		Where("id = ?", board.IDBase.ID).
		Updates(board).Error
}

func UpdateStoryboardPublishedState(ctx context.Context, boardId int64, stage gen.StoryboardStage) error {
	return DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id = ?", boardId).
		Update("stage", stage).
		Error
}

func UpdateStoryboardMultiColumn(ctx context.Context, id int64, columns map[string]interface{}) error {
	if len(columns) == 0 {
		return nil
	}
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).Updates(columns).Error
}

func GetStoryContributors(ctx context.Context, storyID int64) ([]*User, error) {
	contributors := make([]*User, 0)
	err := DataBase().Model(&User{}).WithContext(ctx).
		Where("id in (select distinct(creator_id) from story_board where story_id = ?)", storyID).
		Find(&contributors).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return contributors, nil
}

func GetStoryBoardByStoryAndPrevId(ctx context.Context, storyID int64, prevId int64, page int, pageSize int, orderBy string) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	query := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("story_id = ? and prev_id = ? and status >= 0", storyID, prevId)

	if orderBy != "" {
		if orderBy == "create_at" {
			query = query.Order("create_at desc")
		} else if orderBy == "update_at" {
			query = query.Order("update_at desc")
		} else if orderBy == "fork_num" {
			query = query.Order("fork_num desc")
		} else if orderBy == "like" {
			query = query.Order("like desc")
		}
	}

	err := query.Offset(page * pageSize).Limit(pageSize).Find(&boards).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return boards, nil
}

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
	IDBase
	Content      string `gorm:"column:content" json:"content,omitempty"`             // 场景内容
	CharacterIds string `gorm:"column:character_ids" json:"character_ids,omitempty"` // 角色ID列表
	CreatorId    int64  `gorm:"column:creator_id" json:"creator_id,omitempty"`       // 创建者ID
	StoryId      int64  `gorm:"column:story_id" json:"story_id,omitempty"`           // 故事ID
	BoardId      int64  `gorm:"column:board_id" json:"board_id,omitempty"`           // 故事板ID
	ImagePrompts string `gorm:"column:image_prompts" json:"image_prompts,omitempty"` // 图像生成提示
	AudioPrompts string `gorm:"column:audio_prompts" json:"audio_prompts,omitempty"` // 音频生成提示
	VideoPrompts string `gorm:"column:video_prompts" json:"video_prompts,omitempty"` // 视频生成提示
	GenStatus    int    `gorm:"column:gen_status" json:"gen_status,omitempty"`       // 生成状态
	GenResult    string `gorm:"column:gen_result" json:"gen_result,omitempty"`       // 生成结果
	Status       int    `gorm:"column:status" json:"status,omitempty"`               // 记录状态
	TaskId       string `gorm:"column:task_id" json:"task_id,omitempty"`             // 任务ID
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
	return int64(scene.IDBase.ID), nil
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
		Where("id = ?", scene.IDBase.ID).
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

func BatchUpdateStoryBoardSceneStatus(ctx context.Context, ids []int64, status int) error {
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
type StoryBoardRole struct {
	IDBase
	CreatorId   int64  `gorm:"column:creator_id" json:"creator_id,omitempty"`     // 创建者ID
	StoryId     int64  `gorm:"column:story_id" json:"story_id,omitempty"`         // 故事ID
	BoardId     int64  `gorm:"column:board_id" json:"board_id,omitempty"`         // 故事板ID
	RoleId      int64  `gorm:"column:role_id" json:"role_id,omitempty"`           // 角色ID
	Name        string `gorm:"column:name" json:"name,omitempty"`                 // 角色名
	Avatar      string `gorm:"column:avatar" json:"avatar,omitempty"`             // 头像
	Desc        string `gorm:"column:desc" json:"desc,omitempty"`                 // 角色描述
	Status      int    `gorm:"column:status" json:"status,omitempty"`             // 状态
	IsMain      int    `gorm:"column:is_main" json:"is_main,omitempty"`           // 是否主线
	IsPublished int    `gorm:"column:is_published" json:"is_published,omitempty"` // 是否发布
}

func (board StoryBoardRole) TableName() string {
	return "story_board_role"
}

func CreateStoryBoardRole(ctx context.Context, role *StoryBoardRole) (int64, error) {
	role.Status = 1
	if err := DataBase().Model(role).
		WithContext(ctx).
		Create(role).Error; err != nil {
		return 0, err
	}
	return int64(role.IDBase.ID), nil
}

func UpdateStoryBoardRoleDescAndAvatar(ctx context.Context, id int64, desc string, avatar string) error {
	needUpdate := make(map[string]interface{})
	if desc != "" {
		needUpdate["desc"] = desc
	}
	if avatar != "" {
		needUpdate["avatar"] = avatar
	}
	return DataBase().Model(&StoryBoardRole{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Updates(needUpdate).Error
}

func GetStoryBoardRoles(ctx context.Context, boardId int64) ([]*StoryBoardRole, error) {
	role := make([]*StoryBoardRole, 0)
	err := DataBase().Model(role).
		WithContext(ctx).
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Scan(&role).Error
	if err != nil {
		return nil, err
	}
	return role, nil
}

func UpdateStoryBoardRolePublished(ctx context.Context, id int64, published int) error {
	now := time.Now()
	needUpdate := make(map[string]interface{})
	if published == 1 {
		needUpdate["published"] = published
		needUpdate["publish_at"] = now
	}
	return DataBase().Model(&StoryBoardRole{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Updates(needUpdate).Error
}

func GetStoryBoardRolesByBoard(ctx context.Context, boardId int64) ([]*StoryBoardRole, error) {
	roles := make([]*StoryBoardRole, 0)
	err := DataBase().Model(&StoryBoardRole{}).
		WithContext(ctx).
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func DelStoryBoardRole(ctx context.Context, id int64) error {
	err := DataBase().Model(&StoryBoardRole{}).WithContext(ctx).
		Where("id = ?", id).
		Update("status = ?", -1).Error
	return err
}

func UpdateStoryBoardRole(ctx context.Context, role *StoryBoardRole) error {
	return DataBase().Model(role).WithContext(ctx).
		Where("id = ?", role.IDBase.ID).
		Where("status >= 0").
		Updates(role).Error
}

func UpdateStoryBoardRoleMultiColumn(ctx context.Context, id int64, columns map[string]interface{}) error {
	return DataBase().Model(&StoryBoardRole{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Updates(columns).Error
}

// 获取角色参与的某一个故事的所有故事板
func GetStoryBoardsByRoleID(ctx context.Context, roleID int64, page int, pageSize int) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	var boardsIDs []int64
	if err := DataBase().Select("board_id").Model(&StoryBoardRole{}).
		Where("role_id = ?", roleID).
		Where("status >= 0").
		Find(&boardsIDs).Limit(pageSize).Offset(page * pageSize).Error; err != nil {
		return nil, err
	}
	if len(boardsIDs) == 0 {
		return nil, nil
	}
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id in (?)", boardsIDs).
		Where("status >= 0").
		Find(&boards).Limit(pageSize).Offset(page * pageSize).Error
	if err != nil {
		return nil, err
	}
	return boards, nil
}

func GetStoryBoardSencesByRoleID(ctx context.Context, roleID int64) ([]*StoryBoardScene, error) {
	var boards []*StoryBoard
	var boardsIDs []int64
	if err := DataBase().Select("board_id").Model(&StoryBoardRole{}).
		Where("role_id = ?", roleID).
		Where("status > 0").
		Find(&boardsIDs).Limit(10).Error; err != nil {
		return nil, err
	}
	if len(boardsIDs) == 0 {
		return nil, nil
	}
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id in (?)", boardsIDs).
		Where("status > 0").
		Find(&boards).Error
	if err != nil {
		return nil, err
	}
	var scenes []*StoryBoardScene
	err = DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("board_id in (?)", boardsIDs).
		Where("status > 0").
		Find(&scenes).Error
	if err != nil {
		return nil, err
	}
	return scenes, nil
}

func GetStoryBoardsByStoryIds(ctx context.Context, storyIds []int64, page int, pageSize int, orderBy string) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("story_id in (?)", storyIds).
		Where("status = ?", 1).
		Order("create_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&boards).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return boards, nil
}

func GetStoryBoardsByRolesID(ctx context.Context, rolesIDs []int64, page int, pageSize int, orderBy string) ([]*StoryBoard, []*StoryBoardRole, error) {
	var boards []*StoryBoard
	var boardsIDs []int64
	var roleModels []*StoryBoardRole
	if err := DataBase().Select("board_id").Model(&StoryBoardRole{}).
		Where("role_id in (?)", rolesIDs).
		Where("status > 0").
		Find(&roleModels).
		Offset(page * pageSize).
		Limit(pageSize).
		Error; err != nil {
		return nil, nil, err
	}
	if len(roleModels) == 0 {
		return nil, nil, nil
	}
	for _, role := range roleModels {
		boardsIDs = append(boardsIDs, role.BoardId)
	}
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id in (?)", boardsIDs).
		Where("status > 0").
		Order("create_at desc").
		Find(&boards).Error
	if err != nil {
		return nil, nil, err
	}
	return boards, roleModels, nil
}

func GetUnPublishedStoryBoardsByUserId(ctx context.Context, userId int64, page int, pageSize int, orderBy string) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("creator_id = ?", userId).
		Where("stage != ?", 2).
		Order("create_at desc").
		Offset(page * pageSize).
		Limit(pageSize).
		Find(&boards).Error
	if err != nil {
		return nil, err
	}
	return boards, nil
}

// 新增：分页获取StoryBoard列表
func GetStoryBoardList(ctx context.Context, offset, limit int) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&boards).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return boards, nil
}

// 新增：通过Title唯一查询
func GetStoryBoardByTitle(ctx context.Context, title string) (*StoryBoard, error) {
	board := &StoryBoard{}
	err := DataBase().Model(board).
		WithContext(ctx).
		Where("title = ?", title).
		First(board).Error
	if err != nil {
		return nil, err
	}
	return board, nil
}
