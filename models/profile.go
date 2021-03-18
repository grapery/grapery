package models

import (
	"fmt"

	"github.com/grapery/grapery/api"
	log "github.com/sirupsen/logrus"
)

/*
用户的profile文件
*/
type UserProfile struct {
	Base
	UserID    int64  `json:"user_id,omitempty"`
	Followers int64  `json:"followers,omitempty"`
	Following int64  `json:"following,omitempty"`
	Emotion   int    `json:"emotion,omitempty"`
	ShortDesc string `json:"short_desc,omitempty"`
}

func (p UserProfile) TableName() string {
	return "user_profile"
}

func CreateUserProfile(repo *Repository, profile *UserProfile) error {
	err := repo.DB().Model(profile).Create(profile).Error
	if err != nil {
		log.Error("create profile failed: %s", err.Error())
		return err
	}
	log.Info("create profile : ", profile.UserID)
	return nil
}

func UpdateUserProfile(repo *Repository, profile *UserProfile) error {
	err := repo.DB().Model(profile).Update(profile).
		Where("user_id = ? and deleted = ?", profile.UserID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func GetUserProfile(repo *Repository, profileID uint64) (*UserProfile, error) {
	profile := new(UserProfile)
	err := repo.DB().Model(profile).First(profile).
		Where("id = ?", profileID).Error
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func GetUserProfileByUserID(repo *Repository, userID uint64) (*UserProfile, error) {
	profile := new(UserProfile)
	err := repo.DB().Model(profile).First(profile).
		Where("user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func DeleteUserProfile(repo *Repository, profileID uint64) error {
	err := repo.DB().Model(&UserProfile{}).Update("delete = ? ", true).
		Where("id = ?", profileID).Error
	if err != nil {
		log.Error("update profile failed: ", err)
		return err
	}
	return nil
}

/*
组织的profile文件
*/
type GroupProfile struct {
	IDBase
	GroupID     uint64          `json:"group_id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Avatar      string          `json:"avatar,omitempty"`
	Visable     api.VisibleType `json:"visable,omitempty"`
}

func (g GroupProfile) TableName() string {
	return "group_profile"
}

func (g *GroupProfile) Create() error {
	if err := database.Where("group_id = ? and deleted = ?", g.GroupID, 0).Find(g).Error; err != nil {
		log.Errorf("create group [%d] profile failed : %s", g.GroupID, err.Error())
		return err
	}
	var err error
	if g.ID != 0 {
		err = database.Create(g).Error
	} else {
		log.Errorf("group [%s] profile is exist : ", g.ID)
		return fmt.Errorf("group [%d] profile is exist : ", g.ID)
	}
	if err != nil {
		log.Errorf("create group [%d] profile failed : [%s]", g.GroupID, err.Error())
		return fmt.Errorf("create group [%d] profile failed : %s", g.GroupID, err.Error())
	}
	return nil
}

func (g *GroupProfile) Update() error {
	database.Model(g).Update("emotion", g.Name)
	return nil
}

func (g *GroupProfile) Get() error {
	database.First(g)
	return nil
}

func (g *GroupProfile) Delete() error {
	database.Delete(g)
	return nil
}

/*
项目中的组织的profile文件
*/
type ProjectFile struct {
	IDBase
	GroupID     uint64          `json:"group_id,omitempty"`
	ProjectID   uint64          `json:"project_id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Avatar      string          `json:"avatar,omitempty"`
	Visable     api.VisibleType `json:"visable,omitempty"`
}

func (p ProjectFile) TableName() string {
	return "project_profile"
}
