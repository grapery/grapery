package models

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	CommentTypeUnknown = iota
	CommentTypeComment
	CommentTypeReply
)

// pic/wold/emoji
// 可以是普通的评论，或者是讨论中的回复
type Comment struct {
	IDBase
	UserID       int64  `json:"user_id,omitempty"`
	StoryID      int64  `json:"story_id,omitempty"`
	GroupID      int64  `json:"group_id,omitempty"`
	RoleID       int64  `json:"role_id,omitempty"`
	TimelineID   int64  `json:"timeline_id,omitempty"`
	StoryBoardID int64  `json:"storyboard_id,omitempty"`
	ItemType     int64  `json:"item_type,omitempty"`
	PreID        int64  `json:"pre_id,omitempty"`
	Content      []byte `json:"content,omitempty"`
	RefID        int64  `json:"ref_id,omitempty"`
	LikeCount    int64  `json:"like_count,omitempty"`
	DislikeCount int64  `json:"dislike_count,omitempty"`
	CommentType  int64  `json:"comment_type,omitempty"`
	Status       int64  `json:"status,omitempty"`
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
		Where("id = ? ", c.ID).Error; err != nil {
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
		Where("id = ? ", c.ID); err != nil {
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

func GetCommentByStory(storyID uint64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("story_id = ?", storyID).
		Scan(ret).Error; err != nil {
		log.Errorf("get story [%d] comment failed: %s ", storyID, err.Error())
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

func GetCommentListByStoryBoard(storyBoardID uint64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Where("storyboard_id = ? and delete = 0",
		storyBoardID).
		Scan(ret).Error; err != nil {
		log.Errorf("get storyboard [%d] comment failed ", storyBoardID)
		return nil, err
	}
	return ret, nil
}
