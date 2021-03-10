package models

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

/* Comment
用户的评论，如果用户的活动是评论的话，评论会被加载
评论可以针对：
视频，图片，短说，长文，音乐，项目，问题（？暂时可以不做）
*/
type Comment struct {
	IDBase
	UserID    uint64 `json:"user_id,omitempty"`
	GroupID   uint64 `json:"group_id,omitempty"`
	ProjectID uint64 `json:"project_id,omitempty"`
	ItemID    int    `json:"item_id,omitempty"`
	PreID     uint64 `json:"pre_id,omitempty"`
	Content   []byte `json:"content,omitempty"`
	Tags      string `json:"tags,omitempty"`
}

func (c Comment) TableName() string {
	return "comment"
}

func (c *Comment) Create() error {
	if err := database.Model(c).Create(c).Error; err != nil {
		log.Errorf("create new active [%d] failed : [%s]", c.ID, err.Error())
		return fmt.Errorf("create new active [%d] failed ", c.ID)
	}
	return nil
}

func (c *Comment) UpdateContent() error {
	if err := database.Model(c).Update("content", c.Content).
		Where("group_id = ? and project_id = ? and item_id = ? and pre_id = ? and id = ? ",
			c.GroupID, c.ProjectID, c.ItemID, c.PreID, c.ID).Error; err != nil {
		log.Errorf("update active [%d] failed : [%s]", c.ID, err.Error())
		return fmt.Errorf("update active failed [%s]", err.Error())
	}
	return nil
}

func (c *Comment) GetComment() error {
	if err := database.Model(c).First(c).Error; err != nil {
		return err
	}
	return nil
}

func (c *Comment) Delete() error {
	if err := database.Model(c).Update("deleted", 1).
		Where("group_id = ? and project_id = ? and item_id = ? and pre_id = ? and id = ? ",
			c.GroupID, c.ProjectID, c.ItemID, c.PreID, c.ID); err != nil {
		log.Errorf("update active [%d] deleted failed ", c.IDBase.ID)
		return fmt.Errorf("deleted active [%d] failed ", c.IDBase.ID)
	}
	return nil
}

func GetCommentByUserID(userID uint64) (*[]Comment, error) {
	var ret = new([]Comment)
	if err := database.Model(&Comment{}).Where("user_id = ? and delete = 0", userID).Find(ret).Error; err != nil {
		log.Errorf("get user [%d] active failed: %s ", userID, err.Error())
		return nil, err
	}

	return ret, nil
}

func GetCommentListByTimeRange(start time.Time, end time.Time) (*[]Comment, error) {
	var ret = new([]Comment)
	if err := database.Where("created_at < ? and  created_at > ? and delete = 0", end, start).Find(ret).Error; err != nil {
		log.Errorf("get active in range [%s--%s] failed ", start.String(), end.String())
		return nil, err
	}
	return ret, nil
}

func GetCommentListByItem(userID uint64, activeType uint) (*[]Comment, error) {
	var ret = new([]Comment)
	if err := database.Where("user_id = ? and active_type = ? and delete = 0", userID, activeType).Find(ret).Error; err != nil {
		log.Errorf("get user [%d] active type [%d] failed ", userID, activeType)
		return nil, err
	}
	return ret, nil
}
