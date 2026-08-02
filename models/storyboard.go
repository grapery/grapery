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
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	Title       string              `gorm:"column:title;index" json:"title,omitempty"`           // 故事板标题
	Description string              `gorm:"column:description" json:"description,omitempty"`     // 描述
	CreatorID   int64               `gorm:"column:creator_id;index" json:"creator_id,omitempty"` // 创建者ID
	StoryID     int64               `gorm:"column:story_id;index" json:"story_id,omitempty"`     // 所属故事ID
	PrevId      int64               `gorm:"column:prev_id;index" json:"prev_id,omitempty"`       // 上一个故事板ID
	Avatar      string              `gorm:"column:avatar" json:"avatar,omitempty"`               // 封面
	Status      int                 `gorm:"column:status" json:"status,omitempty"`
	Stage       gen.StoryboardStage `gorm:"column:stage" json:"stage,omitempty"`         //
	Params      string              `gorm:"column:params" json:"params,omitempty"`       // 生成参数
	IsAiGen     bool                `gorm:"column:is_ai_gen" json:"is_ai_gen,omitempty"` // 是否AI生成
	ForkAble    bool                `gorm:"column:fork_able" json:"fork_able,omitempty"` // 是否可被fork

	TaskId string `gorm:"column:task_id" json:"task_id,omitempty"` // 任务ID,关联故事板的uuid
	UUID   string `gorm:"column:uuid" json:"uuid,omitempty"`       // 唯一标识，对接三方平台的uuid
	Seed   int    `gorm:"column:seed" json:"seed,omitempty"`       // 故事版在生成过程中，需要保持种子

	ForkNum    int `gorm:"column:fork_num" json:"fork_num,omitempty"`       // fork数
	LikeNum    int `gorm:"column:like_num" json:"like_num,omitempty"`       // 点赞数
	CommentNum int `gorm:"column:comment_num" json:"comment_num,omitempty"` // 评论数
	RoleNum    int `gorm:"column:role_num" json:"role_num,omitempty"`       // 角色数
	ShareNum   int `gorm:"column:share_num" json:"share_num,omitempty"`     // 分享数
	TokenNum   int `gorm:"column:token_num" json:"token_num,omitempty"`     // 消耗token数

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

func UpdateStoryBoardStage(ctx context.Context, id int64, stage gen.StoryboardStage) error {
	return DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id = ?", id).
		Update("stage = ?", stage).Error
}

func UpdateStoryBoardParams(ctx context.Context, id int64, params string) error {
	return DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
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
	return int64(board.ID), nil
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
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
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
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
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
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
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
		Where("id = ?", board.ID).
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
		Where("status >= 0").
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
		Where("story_id = ? and prev_id = ? and status >= 0", storyID, prevId).
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED)

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

func UpdateStoryBoardRoleNum(ctx context.Context, id int64, num int) error {
	return DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id = ?", id).
		Update("role_num", num).
		Error
}

// 关联到故事板的角色，只做关联，更新故事角色是主数据表，不在这里
type StoryBoardRole struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	CreatorId   int64  `gorm:"column:creator_id" json:"creator_id,omitempty"`     // 创建者ID
	StoryId     int64  `gorm:"column:story_id" json:"story_id,omitempty"`         // 故事ID
	BoardId     int64  `gorm:"column:board_id" json:"board_id,omitempty"`         // 故事板ID
	RoleId      int64  `gorm:"column:role_id" json:"role_id,omitempty"`           // 角色ID
	Name        string `gorm:"column:name" json:"name,omitempty"`                 // 角色名
	Avatar      string `gorm:"column:avatar" json:"avatar,omitempty"`             // 头像
	Desc        string `gorm:"column:desc" json:"desc,omitempty"`                 // 角色描述
	Status      int    `gorm:"column:status" json:"status,omitempty"`             // 状态
	IsMain      int    `gorm:"column:is_main" json:"is_main,omitempty"`           // 是否主线角色
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
	return int64(role.ID), nil
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

func GetStoryBoardRolesID(ctx context.Context, boardId int64) ([]int64, error) {
	roleIds := make([]int64, 0)
	err := DataBase().Model(&StoryBoardRole{}).
		WithContext(ctx).
		Select("id").
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Scan(&roleIds).Error
	if err != nil {
		return nil, err
	}
	return roleIds, nil
}

func BatchUpdateStoryBoardRoles(ctx context.Context, roleIds []int64, isPublished bool) error {
	return DataBase().Model(&StoryBoardRole{}).
		WithContext(ctx).
		Where("id IN (?)", roleIds).
		Updates(map[string]interface{}{
			"is_published": isPublished,
		}).Error
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
		Where("id = ?", role.ID).
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
func GetStoryBoardsByRoleID(ctx context.Context, roleID int64, page int, pageSize int) ([]*StoryBoard, int64, error) {
	var boards []*StoryBoard
	var boardsIDs []int64
	if err := DataBase().Select("board_id").Model(&StoryBoardRole{}).
		Where("role_id = ?", roleID).
		Where("status >= 0").
		Find(&boardsIDs).Error; err != nil {
		return nil, 0, err
	}
	// 去重ID，避免重复查询
	boardsIDs = UniqueInt64s(boardsIDs)
	if len(boardsIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id in (?)", boardsIDs).
		Where("status >= 0").
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}
	err = DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id in (?)", boardsIDs).
		Where("status >= 0").
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&boards).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, 0, err
	}
	return boards, total, nil
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
	// 去重ID，避免重复查询
	boardsIDs = UniqueInt64s(boardsIDs)
	if len(boardsIDs) == 0 {
		return nil, nil
	}
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id in (?)", boardsIDs).
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
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

func GetStoryBoardsByStoryIds(ctx context.Context, storyIds []int64, page int, pageSize int, orderBy string) ([]*StoryBoard, int64, error) {
	// 去重ID，避免重复查询
	storyIds = UniqueInt64s(storyIds)
	if len(storyIds) == 0 {
		return nil, 0, nil
	}

	var boards []*StoryBoard
	var total int64
	query := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("story_id in (?)", storyIds).
		Where("status >= 0").
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}
	query = query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("create_at desc").
		Find(&boards)
	if err := query.Error; err != nil && err != gorm.ErrRecordNotFound {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return boards, total, nil
}

func GetStoryBoardsByRolesID(ctx context.Context, rolesIDs []int64, page int, pageSize int, orderBy string) ([]*StoryBoard, []*StoryBoardRole, error) {
	// 去重ID，避免重复查询
	rolesIDs = UniqueInt64s(rolesIDs)
	if len(rolesIDs) == 0 {
		return nil, nil, nil
	}

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
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
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
		Where("stage != ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
		Order("create_at desc").
		Offset(page * pageSize).
		Limit(pageSize).
		Find(&boards).Error
	if err != nil {
		return nil, err
	}
	return boards, nil
}

func GetUserStoryboardDrafts(ctx context.Context, userId int64, storyId int64, page int, pageSize int) ([]*StoryBoard, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	query := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("creator_id = ?", userId).
		Where("status >= 0").
		Where("stage != ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED)
	if storyId > 0 {
		query = query.Where("story_id = ?", storyId)
	}
	countDB := query.Session(&gorm.Session{})
	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	boards := make([]*StoryBoard, 0)
	if total == 0 {
		return boards, 0, nil
	}
	listDB := query.Session(&gorm.Session{})
	if err := listDB.
		Order("create_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&boards).Error; err != nil {
		return nil, 0, err
	}
	return boards, total, nil
}

func DeleteUserStoryboardDraft(ctx context.Context, userId int64, draftId int64) error {
	return DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status": -1,
			"stage":  gen.StoryboardStage_STORYBOARD_STAGE_DRAFT,
		}
		res := tx.Model(&StoryBoard{}).
			Where("id = ?", draftId).
			Where("creator_id = ?", userId).
			Where("status >= 0").
			Where("stage != ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Model(&StoryBoardScene{}).
			Where("board_id = ?", draftId).
			Where("status >= 0").
			Update("status", -1).Error; err != nil {
			return err
		}
		if err := tx.Model(&StoryBoardRole{}).
			Where("board_id = ?", draftId).
			Where("status >= 0").
			Update("status", -1).Error; err != nil {
			return err
		}
		return nil
	})
}

// 新增：分页获取StoryBoard列表
func GetStoryBoardList(ctx context.Context, offset, limit int) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&boards).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return boards, nil
}

func GetStoryBoardsByIds(ctx context.Context, ids []int64) ([]*StoryBoard, error) {
	// 去重ID，避免重复查询
	ids = UniqueInt64s(ids)
	if len(ids) == 0 {
		return nil, nil
	}

	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
		Where("id in (?)", ids).
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
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED).
		Where("title = ?", title).
		First(board).Error
	if err != nil {
		return nil, err
	}
	return board, nil
}
