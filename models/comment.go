package models

import (
	"context"
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
// Comment 评论/讨论内容
type Comment struct {
	IDBase
	UserID        int64  `gorm:"column:user_id" json:"user_id,omitempty"`                 // 用户ID
	StoryID       int64  `gorm:"column:story_id" json:"story_id,omitempty"`               // 故事ID
	GroupID       int64  `gorm:"column:group_id" json:"group_id,omitempty"`               // 群组ID
	RoleID        int64  `gorm:"column:role_id" json:"role_id,omitempty"`                 // 角色ID
	TimelineID    int64  `gorm:"column:timeline_id" json:"timeline_id,omitempty"`         // 时间线ID
	StoryboardID  int64  `gorm:"column:storyboard_id" json:"storyboard_id,omitempty"`     // 故事板ID
	ItemType      int64  `gorm:"column:item_type" json:"item_type,omitempty"`             // 评论类型
	PreID         int64  `gorm:"column:pre_id" json:"pre_id,omitempty"`                   // 上一条评论ID
	Content       []byte `gorm:"column:content" json:"content,omitempty"`                 // 评论内容
	RootCommentID int64  `gorm:"column:root_comment_id" json:"root_comment_id,omitempty"` // 根评论ID
	LikeCount     int64  `gorm:"column:like_count" json:"like_count,omitempty"`           // 点赞数
	DislikeCount  int64  `gorm:"column:dislike_count" json:"dislike_count,omitempty"`     // 点踩数
	CommentType   int64  `gorm:"column:comment_type" json:"comment_type,omitempty"`       // 评论类型
	ReplyCount    int64  `gorm:"column:reply_count" json:"reply_count,omitempty"`         // 回复数
	Status        int64  `gorm:"column:status" json:"status,omitempty"`                   // 状态
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
	if err := DataBase().Model(c).
		Update("deleted", 1).
		Where("id = ? ", c.ID).Error; err != nil {
		log.Errorf("update comment [%d] deleted failed ", c.ID)
		return fmt.Errorf("deleted comment [%d] failed ", c.ID)
	}
	return nil
}

func GetCommentByUserID(userID uint64, page int64, pageSize int64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("user_id = ?", userID).
		Order("create_at desc").
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

func IncreaseReplyCount(commentID uint64) error {
	if err := DataBase().Model(&Comment{}).
		Where("id = ?", commentID).
		Update("reply_count", gorm.Expr("reply_count + 1")).Error; err != nil {
		return err
	}
	return nil
}

func DecreaseReplyCount(commentID uint64) error {
	if err := DataBase().Model(&Comment{}).
		Where("id = ?", commentID).
		Update("reply_count", gorm.Expr("reply_count - 1")).Error; err != nil {
		return err
	}
	return nil
}

func GetCommentByStory(storyID uint64, commentType int64, page int64, pageSize int64) (*[]*Comment, error) {
	var ret = new([]*Comment)
	if err := DataBase().Model(&Comment{}).
		Where("story_id = ?", storyID).
		Where("root_comment_id = 0").
		Where("status = ?", 1).
		Order("create_at desc").
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
		Where("create_at < ? and create_at > ? and deleted = 0 and comment_type = ?",
			end,
			start,
			commentType).
		Order("create_at desc").
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
		Order("create_at desc").
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
		Order("create_at").
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
		Order("create_at desc").
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

// 新增：分页获取Comment列表
func GetCommentList(ctx context.Context, offset, limit int) ([]*Comment, error) {
	var comments []*Comment
	err := DataBase().Model(&Comment{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&comments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return comments, nil
}

// 新增：通过主键唯一查询
func GetCommentByID(ctx context.Context, id int64) (*Comment, error) {
	comment := &Comment{}
	err := DataBase().Model(comment).
		WithContext(ctx).
		Where("id = ?", id).
		First(comment).Error
	if err != nil {
		return nil, err
	}
	return comment, nil
}
