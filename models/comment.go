package models

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type Comment struct {
	IDBase
	UserID  int64  `json:"user_id,omitempty"`
	ItemID  int    `json:"item_id,omitempty"`
	PreID   int64  `json:"pre_id,omitempty"`
	Content []byte `json:"content,omitempty"`
}

func (c Comment) TableName() string {
	return "comment"
}

func (c *Comment) Create() error {
	if err := DataBase().Model(c).Create(c).Error; err != nil {
		log.Errorf("create new comment [%d] failed : [%s]", c.ID, err.Error())
		return fmt.Errorf("create new comment [%d] failed ", c.ID)
	}
	return nil
}

func (c *Comment) UpdateContent() error {
	if err := DataBase().Model(c).Update("content", c.Content).
		Where("and item_id = ? and pre_id = ? and id = ? ",
			c.ItemID, c.PreID, c.ID).Error; err != nil {
		log.Errorf("update comment [%d] failed : [%s]", c.ID, err.Error())
		return fmt.Errorf("update comment failed [%s]", err.Error())
	}
	return nil
}

func (c *Comment) GetComment() error {
	if err := DataBase().Model(c).First(c).Error; err != nil {
		return err
	}
	return nil
}

func (c *Comment) Delete() error {
	if err := DataBase().Model(c).Update("deleted", 1).
		Where("item_id = ? and pre_id = ? and id = ? ",
			c.ItemID,
			c.PreID,
			c.ID); err != nil {
		log.Errorf("update comment [%d] deleted failed ", c.ID)
		return fmt.Errorf("deleted comment [%d] failed ", c.ID)
	}
	return nil
}

func GetCommentByUserID(userID uint64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("user_id = ?", userID).
		Scan(ret).Error; err != nil {
		log.Errorf("get user [%d] comment failed: %s ", userID, err.Error())
		return nil, err
	}

	return ret, nil
}

func GetCommentByProject(projectID uint64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("project_id = ?", projectID).
		Scan(ret).Error; err != nil {
		log.Errorf("get project [%d] comment failed: %s ", projectID, err.Error())
		return nil, err
	}

	return ret, nil
}

func GetCommentListByTimeRange(start time.Time, end time.Time) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().
		Where("created_at < ? and created_at > ? and delete = 0",
			end,
			start).
		Scan(ret).Error; err != nil {
		log.Errorf("get comment in range [%s--%s] failed ", start.String(), end.String())
		return nil, err
	}
	return ret, nil
}

func GetCommentListByItem(userID uint64, commentType uint) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Where("user_id = ? and comment_type = ? and delete = 0",
		userID,
		commentType).
		Scan(ret).Error; err != nil {
		log.Errorf("get user [%d] comment type [%d] failed ", userID, commentType)
		return nil, err
	}
	return ret, nil
}
