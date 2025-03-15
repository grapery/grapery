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
type User struct {
	IDBase
	Name      string `json:"name,omitempty" gorm:"index"`
	Email     string `json:"email,omitempty" gorm:"index"`
	Phone     string `json:"phone,omitempty" gorm:"index"`
	Gender    int
	BioID     string
	Status    api.UserStatus
	Location  string
	Avatar    string
	ShortDesc string
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

func (u *User) UpdateName() error {
	err := DataBase().Model(u).Update("name", u.Name).Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] name failed ", u.ID)
		return fmt.Errorf("update user [%d] name failed ", u.ID)
	}
	return nil
}

func (u *User) UpdateBio() error {
	err := DataBase().Model(u).Update("bio", u.BioID).Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] bio [%s] failed ", u.ID, u.BioID)
		return fmt.Errorf("update user [%d] bio failed ", u.ID)
	}
	return nil
}

func (u *User) UpdateAvatar() error {
	err := DataBase().Model(u).Update("avatar", u.Avatar).Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] avatar [%s] failed ", u.ID, u.Avatar)
		return fmt.Errorf("update user [%d] avatar failed ", u.ID)
	}
	return nil
}

func (u *User) UpdateAll() error {
	err := DataBase().Model(u).
		Update("avatar", u.Avatar).
		Update("short_desc", u.ShortDesc).
		Update("name", u.Name).
		Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] all [%s] failed ", u.ID, u.Avatar)
		return fmt.Errorf("update user [%d] all failed ", u.ID)
	}
	return nil
}

func UpdateUserInfo(ctx context.Context, userId int64, needUpdates map[string]interface{}) error {
	err := DataBase().Model(User{}).
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

func (u *User) GetById() error {
	err := DataBase().Model(u).Where("id = ? and deleted = ?", u.ID, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%d] info failed ", u.ID)
	}
	log.Infof("get user [%d] info success ", u.ID)
	return nil
}

func (u *User) GetByName() error {
	err := DataBase().Model(u).Where("name = ? and deleted = ? ", u.Name, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%s] info failed : [%s]", u.Name, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Name)
	}
	return nil
}

func (u *User) GetByPhone() error {
	err := DataBase().Model(u).Where("phone = ? and deleted = ?", u.Phone, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Phone)
	}
	return nil
}

func (u *User) GetByEmail() error {
	err := DataBase().Model(u).Where("email = ? and deleted = ?", u.Email, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Email)
	}
	return nil
}

func (u *User) Delete() error {
	err := DataBase().Model(u).Update("deleted", 1).Where("id = ? ", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] deleted failed ", u.ID)
		return fmt.Errorf("deleted user [%d] failed ", u.ID)
	}
	return nil
}

func GetUsersByIds(ids []int64) (users []*User, err error) {
	if len(ids) == 0 {
		return nil, nil
	}
	err = DataBase().Model(User{}).Where("id in (?)", ids).Scan(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func GetUserById(ctx context.Context, id int64) (*User, error) {
	user := &User{}
	err := DataBase().
		Model(user).
		Where("id = ? and deleted = ?", id, 0).First(user).Error
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
	if err != nil {
		log.Errorf("get user [%s] info failed : [%s]", name, err.Error())
		return nil, fmt.Errorf("get user [%s] info failed ", name)
	}
	return user, nil
}

type UserProfile struct {
	IDBase
	UserId         int64  `json:"user_id,omitempty"`
	Background     string `json:"background,omitempty"`
	NumGroup       int    `json:"num_group,omitempty"`
	DefaultGroupID int64  `json:"default_group_id,omitempty"`
	MinSameGroup   int    `json:"min_same_group,omitempty"`

	Limit      int `json:"limit,omitempty"`
	UsedTokens int `json:"used_tokens,omitempty"`
	Status     int `json:"status,omitempty"`

	CreatedGroupNum      int `json:"created_group_num,omitempty"`
	CreatedStoryNum      int `json:"created_story_num,omitempty"`
	CreatedRoleNum       int `json:"created_role_num,omitempty"`
	CreatedBoardNum      int `json:"created_board_num,omitempty"`
	CreatedGenNum        int `json:"created_gen_num,omitempty"`
	WatchingStoryNum     int `json:"watching_story_num,omitempty"`
	WatchingGroupNum     int `json:"watching_group_num,omitempty"`
	WatchingStoryRoleNum int `json:"watching_story_role_num,omitempty"`

	ContributStoryNum int `json:"contribut_story_num,omitempty"`
	ContributRoleNum  int `json:"contribut_role_num,omitempty"`

	LikedStoryNum int `json:"liked_story_num,omitempty"`
	LikedRoleNum  int `json:"liked_role_num,omitempty"`
}

func (u *UserProfile) TableName() string {
	return "user_profile"
}

func (u *UserProfile) Create() error {
	err := DataBase().Model(u).Create(u).First(u).Error
	if err != nil {
		log.Errorf("create user profile [%d] failed [%s] ", u.UserId, err.Error())
		return fmt.Errorf("create user profile failed")
	}
	return nil
}

func (u *UserProfile) Update() error {
	err := DataBase().Model(u).Updates(u).Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user profile [%d] failed ", u.ID)
		return fmt.Errorf("update user profile [%d] failed ", u.ID)
	}
	return nil
}

func (u *UserProfile) GetById() error {
	err := DataBase().Model(u).Where("id = ?", u.ID).First(u).Error
	if err != nil {
		log.Errorf("get user profile [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user profile [%d] info failed ", u.ID)
	}
	return nil
}

func (u *UserProfile) GetByUserId() error {
	err := DataBase().Model(u).Where("user_id = ?", u.UserId).First(u).Error
	if err != nil {
		log.Errorf("get user profile [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user profile [%d] info failed ", u.ID)
	}
	return nil
}

func (u *UserProfile) IsTokenFinished() bool {
	return u.UsedTokens >= u.Limit
}

// ... existing UserProfile struct and methods ...

// Increment methods
func (u *UserProfile) IncrementCreatedGroupNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_group_num", gorm.Expr("created_group_num + ?", 1)).Error
}

func (u *UserProfile) IncrementCreatedStoryNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_story_num", gorm.Expr("created_story_num + ?", 1)).Error
}

func (u *UserProfile) IncrementCreatedRoleNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_role_num", gorm.Expr("created_role_num + ?", 1)).Error
}

func (u *UserProfile) IncrementCreatedBoardNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_board_num", gorm.Expr("created_board_num + ?", 1)).Error
}

func (u *UserProfile) IncrementCreatedGenNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_gen_num", gorm.Expr("created_gen_num + ?", 1)).Error
}

func (u *UserProfile) IncrementWatchingStoryNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("watching_story_num", gorm.Expr("watching_story_num + ?", 1)).Error
}

func (u *UserProfile) IncrementWatchingGroupNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("watching_group_num", gorm.Expr("watching_group_num + ?", 1)).Error
}

func (u *UserProfile) IncrementWatchingStoryRoleNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("watching_story_role_num", gorm.Expr("watching_story_role_num + ?", 1)).Error
}

func (u *UserProfile) IncrementContributStoryNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("contribut_story_num", gorm.Expr("contribut_story_num + ?", 1)).Error
}

func (u *UserProfile) IncrementContributRoleNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("contribut_role_num", gorm.Expr("contribut_role_num + ?", 1)).Error
}

func (u *UserProfile) IncrementLikedStoryNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("liked_story_num", gorm.Expr("liked_story_num + ?", 1)).Error
}

func (u *UserProfile) IncrementLikedRoleNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("liked_role_num", gorm.Expr("liked_role_num + ?", 1)).Error
}

// Decrement methods
func (u *UserProfile) DecrementCreatedGroupNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_group_num", gorm.Expr("created_group_num - ?", 1)).Error
}

func (u *UserProfile) DecrementCreatedStoryNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_story_num", gorm.Expr("created_story_num - ?", 1)).Error
}

func (u *UserProfile) DecrementCreatedRoleNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_role_num", gorm.Expr("created_role_num - ?", 1)).Error
}

func (u *UserProfile) DecrementCreatedBoardNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_board_num", gorm.Expr("created_board_num - ?", 1)).Error
}

func (u *UserProfile) DecrementCreatedGenNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("created_gen_num", gorm.Expr("created_gen_num - ?", 1)).Error
}

func (u *UserProfile) DecrementWatchingStoryNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("watching_story_num", gorm.Expr("watching_story_num - ?", 1)).Error
}

func (u *UserProfile) DecrementWatchingGroupNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("watching_group_num", gorm.Expr("watching_group_num - ?", 1)).Error
}

func (u *UserProfile) DecrementWatchingStoryRoleNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("watching_story_role_num", gorm.Expr("watching_story_role_num - ?", 1)).Error
}

func (u *UserProfile) DecrementContributStoryNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("contribut_story_num", gorm.Expr("contribut_story_num - ?", 1)).Error
}

func (u *UserProfile) DecrementContributRoleNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("contribut_role_num", gorm.Expr("contribut_role_num - ?", 1)).Error
}

func (u *UserProfile) DecrementLikedStoryNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("liked_story_num", gorm.Expr("liked_story_num - ?", 1)).Error
}

func (u *UserProfile) DecrementLikedRoleNum() error {
	return DataBase().Model(u).Where("user_id = ?", u.UserId).
		Update("liked_role_num", gorm.Expr("liked_role_num - ?", 1)).Error
}
