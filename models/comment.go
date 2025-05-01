package models

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
	UserID        int64  `json:"user_id,omitempty"`
	StoryID       int64  `json:"story_id,omitempty"`
	GroupID       int64  `json:"group_id,omitempty"`
	RoleID        int64  `json:"role_id,omitempty"`
	TimelineID    int64  `json:"timeline_id,omitempty"`
	StoryboardID  int64  `json:"storyboard_id,omitempty"`
	ItemType      int64  `json:"item_type,omitempty"`
	PreID         int64  `json:"pre_id,omitempty"`
	Content       []byte `json:"content,omitempty"`
	RootCommentID int64  `json:"root_comment_id,omitempty"`
	LikeCount     int64  `json:"like_count,omitempty"`
	DislikeCount  int64  `json:"dislike_count,omitempty"`
	CommentType   int64  `json:"comment_type,omitempty"`
	Status        int64  `json:"status,omitempty"`
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

func GetCommentByUserID(userID uint64, page int64, pageSize int64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(int(pageSize)).
		Offset(int((page - 1) * pageSize)).
		Scan(&ret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.Errorf("get user [%d] comment failed: %s ", userID, err.Error())
		return nil, err
	}

	return ret, nil
}

func GetCommentByStory(storyID uint64, commentType int64, page int64, pageSize int64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("story_id = ?", storyID).
		Where("root_comment_id = 0").
		Where("status = ?", 1).
		Order("created_at desc").
		Order("like_count desc").
		Limit(int(pageSize)).
		Offset(int((page - 1) * pageSize)).
		Scan(&ret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.Errorf("get story [%d] comment failed: %s ", storyID, err.Error())
		return nil, err
	}

	return ret, nil
}

func GetCommentListByTimeRange(start time.Time, end time.Time, commentType int64, page int64, pageSize int64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("created_at < ? and created_at > ? and deleted = 0 and comment_type = ?",
			end,
			start,
			commentType).
		Order("created_at desc").
		Order("like_count desc").
		Limit(int(pageSize)).
		Offset(int((page - 1) * pageSize)).
		Scan(&ret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.Errorf("get comment in range [%s--%s] failed ", start.String(), end.String())
		return nil, err
	}
	return ret, nil
}

func GetCommentListByStoryBoard(storyBoardID uint64, page int64, pageSize int64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("storyboard_id = ? and deleted = 0", storyBoardID).
		Where("root_comment_id = 0").
		Order("created_at desc").
		Order("like_count desc").
		Limit(int(pageSize)).
		Offset(int((page - 1) * pageSize)).
		Scan(&ret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.Errorf("get storyboard [%d] comment failed %s", storyBoardID, err.Error())
		return nil, err
	}
	return ret, nil
}

func GetStoryCommentReplies(commentID uint64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).Where("root_comment_id = ? and deleted = 0",
		commentID).
		Order("created_at desc").
		Order("like_count desc").
		Scan(&ret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.Errorf("get comment [%d] reply failed %s", commentID, err.Error())
		return nil, err
	}
	return ret, nil
}

func GetStoryBoardCommentReplies(commentID uint64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).Where("root_comment_id = ? and deleted = 0",
		commentID).
		Order("created_at desc").
		Order("like_count desc").
		Scan(ret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.Errorf("get comment [%d] reply failed %s", commentID, err.Error())
		return nil, err
	}
	return ret, nil
}

func DeleteComment(commentID uint64) error {
	if err := DataBase().Model(&Comment{}).
		Where("id = ?", commentID).
		Update("deleted", 1).Error; err != nil {
		log.Errorf("delete comment [%d] failed %s", commentID, err.Error())
		return err
	}
	return nil
}

func DeleteStoryCommentReply(commentID uint64) error {
	if err := DataBase().Model(&Comment{}).
		Where("id = ?", commentID).
		Update("deleted", 1).Error; err != nil {
		log.Errorf("delete comment [%d] reply failed ", commentID)
		return err
	}
	return nil
}

func DeleteStoryBoardCommentReply(commentID uint64) error {
	if err := DataBase().Model(&Comment{}).
		Where("id = ?", commentID).
		Update("deleted", 1).Error; err != nil {
		log.Errorf("delete comment [%d] reply failed %s", commentID, err.Error())
		return err
	}
	return nil
}

func LikeComment(commentID uint64, userId uint64) error {
	newCommentLike := &CommentLike{
		UserID:    int64(userId),
		CommentID: int64(commentID),
	}
	err := newCommentLike.Create()
	if err != nil {
		return nil
	}
	if err := DataBase().Model(&Comment{}).
		Where("id = ?", commentID).
		Update("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
		log.Errorf("like comment [%d] failed %s", commentID, err.Error())
		return err
	}
	return nil
}

func DislikeComment(commentID uint64, userId uint64) error {
	isLiked, err := GetCommentLike(commentID, userId)
	if err != nil {
		log.Errorf("get comment like [%d] failed %s", commentID, err.Error())
		return err
	}
	if isLiked == nil {
		return nil
	}
	err = isLiked.Delete()
	if err != nil {
		log.Errorf("delete comment like [%d] failed %s", commentID, err.Error())
		return err
	}
	if err := DataBase().Model(&Comment{}).
		Where("id = ?", commentID).
		Update("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
		log.Errorf("dislike comment [%d] failed %s", commentID, err.Error())
		return err
	}
	return nil
}

type CommentLike struct {
	IDBase
	UserID    int64 `json:"user_id,omitempty"`
	CommentID int64 `json:"comment_id,omitempty"`
}

func (c CommentLike) TableName() string {
	return "comment_like"
}

func (c *CommentLike) Create() error {
	if err := DataBase().Model(c).Create(c).Error; err != nil {
		log.Errorf("create new comment like [%d] failed : [%s]", c.ID, err.Error())
		return fmt.Errorf("create new comment like [%d] failed ", c.ID)
	}
	return nil
}

func (c *CommentLike) Delete() error {
	if err := DataBase().Model(c).Delete(c).Error; err != nil {
		log.Errorf("delete comment like [%d] failed ", c.ID)
		return err
	}
	return nil
}

func GetCommentLike(commentID uint64, userId uint64) (*CommentLike, error) {
	var ret = new(CommentLike)
	if err := DataBase().Model(&CommentLike{}).
		Where("comment_id = ? and user_id = ?", commentID, userId).
		First(ret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		log.Errorf("get comment like [%d] failed ", commentID)
		return nil, err
	}
	return ret, nil
}
