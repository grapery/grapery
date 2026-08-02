package models

import (
	"context"
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	_ "time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	api "github.com/grapery/common-protoc/gen"
)

type ChatSetting int

const (
	NoLimit           ChatSetting = 0
	AtleastOneGroup   ChatSetting = 1
	AtleastThreeGroup ChatSetting = 2
	Forbiden          ChatSetting = 4
)

/*
 */
// User 用户基础信息
type User struct {
	ID uint `gorm:"primary_key,column:id;autoIncrement:1000000" json:"id,omitempty"`
	IDBase
	Name      string         `gorm:"column:name;index" json:"name,omitempty"`       // 用户名
	Email     string         `gorm:"column:email;index" json:"email,omitempty"`     // 邮箱
	Phone     string         `gorm:"column:phone;index" json:"phone,omitempty"`     // 手机号
	Gender    int            `gorm:"column:gender" json:"gender,omitempty"`         // 性别
	BioID     string         `gorm:"column:bio_id" json:"bio_id,omitempty"`         // 简介ID
	Status    api.UserStatus `gorm:"column:status" json:"status,omitempty"`         // 用户状态
	Location  string         `gorm:"column:location" json:"location,omitempty"`     // 位置
	Avatar    string         `gorm:"column:avatar" json:"avatar,omitempty"`         // 头像
	ShortDesc string         `gorm:"column:short_desc" json:"short_desc,omitempty"` // 简短描述
}

func (u User) TableName() string {
	return "users"
}

func (u *User) Create() error {
	err := DataBase().Model(u).Create(u).First(u).Error
	if err != nil {
		log.Errorf("create user [%s/%s] failed [%s] ", u.Phone, u.Email, err.Error())
		return fmt.Errorf("create user failed")
	}
	return nil
}

func (u *User) UpdateName(ctx context.Context) error {
	err := DataBase().Model(u).Update("name", u.Name).Where("id = ?", u.ID).WithContext(ctx).Error
	if err != nil {
		log.Errorf("update user [%d] name failed ", u.ID)
		return fmt.Errorf("update user [%d] name failed ", u.ID)
	}
	return nil
}

func (u *User) UpdateBio(ctx context.Context) error {
	err := DataBase().Model(u).Update("bio", u.BioID).Where("id = ?", u.ID).WithContext(ctx).Error
	if err != nil {
		log.Errorf("update user [%d] bio [%s] failed ", u.ID, u.BioID)
		return fmt.Errorf("update user [%d] bio failed ", u.ID)
	}
	return nil
}

func (u *User) UpdateAvatar(ctx context.Context) error {
	err := DataBase().Model(u).Update("avatar", u.Avatar).Where("id = ?", u.ID).WithContext(ctx).Error
	if err != nil {
		log.Errorf("update user [%d] avatar [%s] failed ", u.ID, u.Avatar)
		return fmt.Errorf("update user [%d] avatar failed ", u.ID)
	}
	return nil
}

func (u *User) UpdateAll(ctx context.Context) error {
	err := DataBase().Model(u).
		Update("avatar", u.Avatar).
		Update("short_desc", u.ShortDesc).
		Update("name", u.Name).
		Where("id = ?", u.ID).WithContext(ctx).Error
	if err != nil {
		log.Errorf("update user [%d] all [%s] failed ", u.ID, u.Avatar)
		return fmt.Errorf("update user [%d] all failed ", u.ID)
	}
	return nil
}

func UpdateUserInfo(ctx context.Context, userId int64, needUpdates map[string]interface{}) error {
	err := DataBase().Model(User{}).WithContext(ctx).
		WithContext(ctx).
		Updates(needUpdates).
		Where("id = ?", userId).Error
	if err != nil {
		log.Errorf("update user info failed [%s]", err.Error())
		return fmt.Errorf("update user info failed")
	}
	log.Info("update user info success")
	return nil
}

func (u *User) GetById(ctx context.Context) error {
	err := DataBase().Model(u).Where("id = ? and deleted = ?", u.ID, 0).First(u).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return nil
	}
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%d] info failed ", u.ID)
	}
	log.Infof("get user [%d] info success ", u.ID)
	return nil
}

func (u *User) GetByName(ctx context.Context) error {
	err := DataBase().Model(u).Where("name = ? and deleted = ? ", u.Name, 0).First(u).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user [%s] info failed : [%s]", u.Name, err.Error())
		return nil
	}
	if err != nil {
		log.Errorf("get user [%s] info failed : [%s]", u.Name, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Name)
	}
	return nil
}

func (u *User) GetByPhone(ctx context.Context) error {
	err := DataBase().Model(u).Where("phone = ? and deleted = ?", u.Phone, 0).First(u).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return nil
	}
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Phone)
	}
	return nil
}

func (u *User) GetByEmail(ctx context.Context) error {
	err := DataBase().Model(u).Where("email = ? and deleted = ?", u.Email, 0).First(u).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return nil
	}
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Email)
	}
	return nil
}

func (u *User) Delete(ctx context.Context) error {
	err := DataBase().Model(u).Update("deleted", 1).Where("id = ? ", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] deleted failed ", u.ID)
		return fmt.Errorf("deleted user [%d] failed ", u.ID)
	}
	return nil
}

func GetUsersByIds(ctx context.Context, ids []int64) (users []*User, err error) {
	// 去重ID，避免重复查询
	ids = UniqueInt64s(ids)
	if len(ids) == 0 {
		return nil, nil
	}

	err = DataBase().Model(User{}).
		WithContext(ctx).
		Where("id in (?)", ids).
		Scan(&users).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get users by ids failed : [%s]", err.Error())
		return nil, nil
	}
	if err != nil {
		log.Errorf("get users by ids failed : [%s]", err.Error())
		return nil, err
	}
	return users, nil
}

func GetUsersByIdsMap(ctx context.Context, ids []int64) (map[int]*User, error) {
	// 去重ID，避免重复查询
	ids = UniqueInt64s(ids)
	if len(ids) == 0 {
		return nil, nil
	}

	var users = make([]*User, 0)
	err := DataBase().Model(User{}).
		WithContext(ctx).
		Where("id in (?)", ids).
		Scan(&users).Error
	if err != nil {
		log.Errorf("get users by ids map failed : [%s]", err.Error())
		return nil, err
	}
	var userMap = make(map[int]*User)
	for _, val := range users {
		userMap[int(val.ID)] = val
	}
	return userMap, nil
}

func GetUserById(ctx context.Context, id int64) (*User, error) {
	user := &User{}
	err := DataBase().
		Model(user).
		Where("id = ? and deleted = ?", id, 0).First(user).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user [%d] info failed : [%s]", id, err.Error())
		return nil, nil
	}
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", id, err.Error())
		return nil, fmt.Errorf("get user [%d] info failed ", id)
	}
	return user, nil
}

func GetUserByPhone(ctx context.Context, phone string) (*User, error) {
	user := &User{}
	err := DataBase().
		Model(user).
		Where("phone = ? and deleted = ?", phone, 0).First(user).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user [%s] info failed : [%s]", phone, err.Error())
		return nil, nil
	}
	if err != nil {
		log.Errorf("get user [%s] info failed : [%s]", phone, err.Error())
		return nil, fmt.Errorf("get user [%s] info failed ", phone)
	}
	return user, nil
}

func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := DataBase().
		Model(user).
		Where("email = ? and deleted = ?", email, 0).First(user).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user [%s] info failed : [%s]", email, err.Error())
		return nil, nil
	}
	if err != nil {
		log.Errorf("get user [%s] info failed : [%s]", email, err.Error())
		return nil, fmt.Errorf("get user [%s] info failed ", email)
	}
	return user, nil
}

func GetUserByName(ctx context.Context, name string) (*User, error) {
	user := &User{}
	err := DataBase().
		Model(user).
		Where("name = ? and deleted = ?", name, 0).First(user).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user [%s] info failed : [%s]", name, err.Error())
		return nil, nil
	}
	if err != nil {
		log.Errorf("get user [%s] info failed : [%s]", name, err.Error())
		return nil, fmt.Errorf("get user [%s] info failed ", name)
	}
	return user, nil
}

func GetUsersByName(ctx context.Context, name string, offset, number int) ([]*User, int64, error) {
	var users []*User
	var total int64
	err := DataBase().Model(User{}).
		WithContext(ctx).
		Where("name like ?", name+"%").
		Offset((offset - 1) * number).
		Limit(number).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UserProfile 用户扩展信息
type UserProfile struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	UserId               int64  `gorm:"column:user_id" json:"user_id,omitempty"`                                 // 用户ID
	Background           string `gorm:"column:background" json:"background,omitempty"`                           // 背景
	NumGroup             int    `gorm:"column:num_group" json:"num_group,omitempty"`                             // 加入群组数
	DefaultGroupID       int64  `gorm:"column:default_group_id" json:"default_group_id,omitempty"`               // 默认群组ID
	MinSameGroup         int    `gorm:"column:min_same_group" json:"min_same_group,omitempty"`                   // 最小同群组数
	Limit                int    `gorm:"column:limit" json:"limit,omitempty"`                                     // 限制
	UsedTokens           int    `gorm:"column:used_tokens" json:"used_tokens,omitempty"`                         // 已用token
	Status               int    `gorm:"column:status" json:"status,omitempty"`                                   // 状态
	CreatedGroupNum      int    `gorm:"column:created_group_num" json:"created_group_num,omitempty"`             // 创建群组数
	CreatedStoryNum      int    `gorm:"column:created_story_num" json:"created_story_num,omitempty"`             // 创建故事数
	CreatedRoleNum       int    `gorm:"column:created_role_num" json:"created_role_num,omitempty"`               // 创建角色数
	CreatedBoardNum      int    `gorm:"column:created_board_num" json:"created_board_num,omitempty"`             // 创建故事板数
	CreatedGenNum        int    `gorm:"column:created_gen_num" json:"created_gen_num,omitempty"`                 // 创建生成数
	WatchingStoryNum     int    `gorm:"column:watching_story_num" json:"watching_story_num,omitempty"`           // 关注故事数
	WatchingGroupNum     int    `gorm:"column:watching_group_num" json:"watching_group_num,omitempty"`           // 关注群组数
	WatchingStoryRoleNum int    `gorm:"column:watching_story_role_num" json:"watching_story_role_num,omitempty"` // 关注角色数
	ContributStoryNum    int    `gorm:"column:contribut_story_num" json:"contribut_story_num,omitempty"`         // 贡献故事数
	ContributRoleNum     int    `gorm:"column:contribut_role_num" json:"contribut_role_num,omitempty"`           // 贡献角色数
	LikedStoryNum        int    `gorm:"column:liked_story_num" json:"liked_story_num,omitempty"`                 // 点赞故事数
	LikedRoleNum         int    `gorm:"column:liked_role_num" json:"liked_role_num,omitempty"`                   // 点赞角色数
	FollowersNum         int    `gorm:"column:followers_num" json:"followers_num,omitempty"`                     // 粉丝数
	FollowingNum         int    `gorm:"column:following_num" json:"following_num,omitempty"`                     // 关注数
}

func (u *UserProfile) TableName() string {
	return "user_profile"
}

func (u *UserProfile) Create(ctx context.Context) error {
	err := DataBase().Model(u).Create(u).First(u).WithContext(ctx).Error
	if err != nil {
		log.Errorf("create user profile [%d] failed [%s] ", u.UserId, err.Error())
		return fmt.Errorf("create user profile failed")
	}
	return nil
}

func (u *UserProfile) Update(ctx context.Context) error {
	err := DataBase().Model(u).Updates(u).Where("id = ?", u.ID).WithContext(ctx).Error
	if err != nil {
		log.Errorf("update user profile [%d] failed ", u.ID)
		return fmt.Errorf("update user profile [%d] failed ", u.ID)
	}
	return nil
}

func (u *UserProfile) GetById(ctx context.Context) error {
	err := DataBase().Model(u).Where("id = ?", u.ID).First(u).WithContext(ctx).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user profile [%d] info failed : [%s]", u.ID, err.Error())
		return nil
	}
	if err != nil {
		log.Errorf("get user profile [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user profile [%d] info failed ", u.ID)
	}
	return nil
}

func (u *UserProfile) GetByUserId(ctx context.Context) error {
	err := DataBase().Model(u).Where("user_id = ?", u.UserId).First(u).WithContext(ctx).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user profile [%d] info failed : [%s]", u.ID, err.Error())
		return nil
	}
	if err != nil {
		log.Errorf("get user profile [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user profile [%d] info failed ", u.ID)
	}

	return nil
}

func (u *UserProfile) IsTokenFinished(ctx context.Context) bool {
	return u.UsedTokens >= u.Limit
}

// ... existing UserProfile struct and methods ...

// Increment methods
func (u *UserProfile) IncrementCreatedGroupNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_group_num", gorm.Expr("created_group_num + ?", 1)).Error
}

func (u *UserProfile) IncrementCreatedStoryNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_story_num", gorm.Expr("created_story_num + ?", 1)).Error
}

func (u *UserProfile) IncrementCreatedRoleNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_role_num", gorm.Expr("created_role_num + ?", 1)).Error
}

func (u *UserProfile) IncrementCreatedBoardNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_board_num", gorm.Expr("created_board_num + ?", 1)).Error
}

func (u *UserProfile) IncrementCreatedGenNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_gen_num", gorm.Expr("created_gen_num + ?", 1)).Error
}

func (u *UserProfile) IncrementWatchingStoryNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("watching_story_num", gorm.Expr("watching_story_num + ?", 1)).Error
}

func (u *UserProfile) IncrementWatchingGroupNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("watching_group_num", gorm.Expr("watching_group_num + ?", 1)).Error
}

func (u *UserProfile) IncrementWatchingStoryRoleNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("watching_story_role_num", gorm.Expr("watching_story_role_num + ?", 1)).Error
}

func (u *UserProfile) IncrementContributStoryNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("contribut_story_num", gorm.Expr("contribut_story_num + ?", 1)).Error
}

func (u *UserProfile) IncrementContributRoleNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("contribut_role_num", gorm.Expr("contribut_role_num + ?", 1)).Error
}

func (u *UserProfile) IncrementLikedStoryNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("liked_story_num", gorm.Expr("liked_story_num + ?", 1)).Error
}

func (u *UserProfile) IncrementLikedRoleNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("liked_role_num", gorm.Expr("liked_role_num + ?", 1)).Error
}

// Decrement methods
func (u *UserProfile) DecrementCreatedGroupNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_group_num", gorm.Expr("created_group_num - ?", 1)).Error
}

func (u *UserProfile) DecrementCreatedStoryNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_story_num", gorm.Expr("created_story_num - ?", 1)).Error
}

func (u *UserProfile) DecrementCreatedRoleNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_role_num", gorm.Expr("created_role_num - ?", 1)).Error
}

func (u *UserProfile) DecrementCreatedBoardNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_board_num", gorm.Expr("created_board_num - ?", 1)).Error
}

func (u *UserProfile) DecrementCreatedGenNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("created_gen_num", gorm.Expr("created_gen_num - ?", 1)).Error
}

func (u *UserProfile) DecrementWatchingStoryNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("watching_story_num", gorm.Expr("watching_story_num - ?", 1)).Error
}

func (u *UserProfile) DecrementWatchingGroupNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("watching_group_num", gorm.Expr("watching_group_num - ?", 1)).Error
}

func (u *UserProfile) DecrementWatchingStoryRoleNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("watching_story_role_num", gorm.Expr("watching_story_role_num - ?", 1)).Error
}

func (u *UserProfile) DecrementContributStoryNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("contribut_story_num", gorm.Expr("contribut_story_num - ?", 1)).Error
}

func (u *UserProfile) DecrementContributRoleNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("contribut_role_num", gorm.Expr("contribut_role_num - ?", 1)).Error
}

func (u *UserProfile) DecrementLikedStoryNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("liked_story_num", gorm.Expr("liked_story_num - ?", 1)).Error
}

func (u *UserProfile) DecrementLikedRoleNum(ctx context.Context) error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).WithContext(ctx).
		Update("liked_role_num", gorm.Expr("liked_role_num - ?", 1)).Error
}

// 新增：分页获取User列表
func GetUserList(ctx context.Context, offset, limit int) ([]*User, error) {
	var users []*User
	err := DataBase().Model(&User{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&users).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return users, nil
}

// 新增：通过Name唯一查询
func GetUserByNameUnique(ctx context.Context, name string) (*User, error) {
	user := &User{}
	err := DataBase().Model(user).
		WithContext(ctx).
		Where("name = ?", name).
		First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}
