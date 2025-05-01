package comment

import (
	"context"
	"encoding/json"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/log"
)

var commentServer CommentServer

func init() {
	commentServer = NewCommentService()
}

func GetCommentServer() CommentServer {
	return commentServer
}

type CommentServer interface {
	CreateStoryComment(ctx context.Context, req *api.CreateStoryCommentRequest) (*api.CreateStoryCommentResponse, error)
	GetStoryComments(ctx context.Context, req *api.GetStoryCommentsRequest) (*api.GetStoryCommentsResponse, error)
	DeleteStoryComment(ctx context.Context, req *api.DeleteStoryCommentRequest) (*api.DeleteStoryCommentResponse, error)
	GetStoryCommentReplies(ctx context.Context, req *api.GetStoryCommentRepliesRequest) (*api.GetStoryCommentRepliesResponse, error)
	CreateStoryCommentReply(ctx context.Context, req *api.CreateStoryCommentReplyRequest) (*api.CreateStoryCommentReplyResponse, error)
	DeleteStoryCommentReply(ctx context.Context, req *api.DeleteStoryCommentReplyRequest) (*api.DeleteStoryCommentReplyResponse, error)
	CreateStoryBoardComment(ctx context.Context, req *api.CreateStoryBoardCommentRequest) (*api.CreateStoryBoardCommentResponse, error)
	DeleteStoryBoardComment(ctx context.Context, req *api.DeleteStoryBoardCommentRequest) (*api.DeleteStoryBoardCommentResponse, error)
	GetStoryBoardComments(ctx context.Context, req *api.GetStoryBoardCommentsRequest) (*api.GetStoryBoardCommentsResponse, error)
	LikeComment(ctx context.Context, req *api.LikeCommentRequest) (*api.LikeCommentResponse, error)
	DislikeComment(ctx context.Context, req *api.DislikeCommentRequest) (*api.DislikeCommentResponse, error)
}

type CommentService struct {
}

func NewCommentService() *CommentService {
	return &CommentService{}
}

func (s *CommentService) CreateStoryComment(ctx context.Context, req *api.CreateStoryCommentRequest) (*api.CreateStoryCommentResponse, error) {
	comment := &models.Comment{
		UserID:       req.GetUserId(),
		StoryID:      req.GetStoryId(),
		Content:      []byte(req.GetContent()),
		CommentType:  models.CommentTypeComment,
		Status:       1,
		LikeCount:    0,
		DislikeCount: 0,
	}
	err := comment.Create()
	if err != nil {
		return nil, err
	}
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		return &api.CreateStoryCommentResponse{
			Code:    -1,
			Message: "get story failed",
		}, nil
	}
	story.CommentCount++
	err = models.UpdateStory(ctx, story)
	if err != nil {
		return &api.CreateStoryCommentResponse{
			Code:    -1,
			Message: "update story comment count failed",
		}, nil
	}
	return &api.CreateStoryCommentResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryComments(ctx context.Context, req *api.GetStoryCommentsRequest) (*api.GetStoryCommentsResponse, error) {
	comments, err := models.GetCommentByStory(uint64(req.GetStoryId()),
		models.CommentTypeComment, int64(req.GetOffset()), int64(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	if len(*comments) == 0 {
		return &api.GetStoryCommentsResponse{
			Code:     0,
			Message:  "success",
			Total:    0,
			Comments: []*api.StoryComment{},
		}, nil
	}
	apiComments := make([]*api.StoryComment, 0)
	for _, comment := range *comments {
		apiComments = append(apiComments, &api.StoryComment{
			CommentId: int64(comment.ID),
			Content:   string(comment.Content),
			CreatedAt: comment.CreateAt.Unix(),
			UpdatedAt: comment.UpdateAt.Unix(),
			UserId:    comment.UserID,
			StoryId:   comment.StoryID,
			LikeCount: comment.LikeCount,
		})
	}
	return &api.GetStoryCommentsResponse{
		Code:     0,
		Message:  "success",
		Total:    int64(len(*comments)),
		Comments: apiComments,
	}, nil
}

func (s *CommentService) DeleteStoryComment(ctx context.Context, req *api.DeleteStoryCommentRequest) (*api.DeleteStoryCommentResponse, error) {
	err := models.DeleteComment(uint64(req.GetCommentId()))
	if err != nil {
		return nil, err
	}

	return &api.DeleteStoryCommentResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryCommentReplies(ctx context.Context, req *api.GetStoryCommentRepliesRequest) (*api.GetStoryCommentRepliesResponse, error) {
	replies, err := models.GetStoryCommentReplies(uint64(req.GetCommentId()))
	if err != nil {
		return nil, err
	}
	if len(*replies) == 0 {
		return &api.GetStoryCommentRepliesResponse{
			Code:    0,
			Message: "success",
			Total:   0,
			Replies: []*api.StoryComment{},
		}, nil
	}
	createrIds := make([]int64, 0)
	for _, comment := range *replies {
		createrIds = append(createrIds, comment.UserID)
	}
	createrMap, err := models.GetUsersByIdsMap(createrIds)
	if err != nil {
		log.Log().Sugar().Info("get user by ids map error: %s", err.Error())
		return nil, err
	}
	apiReplies := make([]*api.StoryComment, 0)
	for _, reply := range *replies {
		apiReplies = append(apiReplies, &api.StoryComment{
			CommentId:  int64(reply.ID),
			Content:    string(reply.Content),
			CreatedAt:  reply.CreateAt.Unix(),
			UpdatedAt:  reply.UpdateAt.Unix(),
			UserId:     reply.UserID,
			StoryId:    reply.StoryID,
			LikeCount:  reply.LikeCount,
			ReplyCount: reply.ReplyCount,
			Creator: &api.UserInfo{
				UserId: reply.UserID,
				Name:   createrMap[int(reply.UserID)].Name,
				Avatar: createrMap[int(reply.UserID)].Avatar,
			},
		})
	}
	return &api.GetStoryCommentRepliesResponse{
		Code:    0,
		Message: "success",
		Total:   int64(len(*replies)),
		Replies: apiReplies,
	}, nil
}

func (s *CommentService) CreateStoryCommentReply(ctx context.Context, req *api.CreateStoryCommentReplyRequest) (*api.CreateStoryCommentReplyResponse, error) {
	rootComment := &models.Comment{
		IDBase: models.IDBase{
			ID: uint(req.GetCommentId()),
		},
	}
	err := rootComment.GetComment()
	if err != nil {
		return nil, err
	}
	comment := &models.Comment{
		UserID:        req.GetUserId(),
		StoryID:       rootComment.StoryID,
		StoryboardID:  rootComment.StoryboardID,
		PreID:         req.GetCommentId(),
		RootCommentID: int64(rootComment.RootCommentID),
		Content:       []byte(req.GetContent()),
		CommentType:   models.CommentTypeReply,
		Status:        1,
		ReplyCount:    0,
	}
	err = comment.Create()
	if err != nil {
		return nil, err
	}
	err = models.IncreaseReplyCount(uint64(rootComment.ID))
	if err != nil {
		log.Log().Sugar().Info("increase story comment reply count failed: %s", err.Error())
	}
	return &api.CreateStoryCommentReplyResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (s *CommentService) DeleteStoryCommentReply(ctx context.Context, req *api.DeleteStoryCommentReplyRequest) (*api.DeleteStoryCommentReplyResponse, error) {
	err := models.DeleteStoryCommentReply(uint64(req.GetReplyId()))
	if err != nil {
		return nil, err
	}
	targetComment := &models.Comment{
		IDBase: models.IDBase{
			ID: uint(req.GetReplyId()),
		},
	}
	err = targetComment.GetComment()
	if err != nil {
		return nil, err
	}
	err = models.DecreaseReplyCount(uint64(targetComment.RootCommentID))
	if err != nil {
		log.Log().Sugar().Info("decrease story comment reply count failed: %s", err.Error())
	}
	return &api.DeleteStoryCommentReplyResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (s *CommentService) CreateStoryBoardComment(ctx context.Context, req *api.CreateStoryBoardCommentRequest) (*api.CreateStoryBoardCommentResponse, error) {
	comment := &models.Comment{
		UserID:        req.GetUserId(),
		StoryboardID:  req.GetBoardId(),
		Content:       []byte(req.GetContent()),
		CommentType:   models.CommentTypeComment,
		RootCommentID: 0,
		PreID:         0,
		Status:        1,
		LikeCount:     0,
	}
	err := comment.Create()
	if err != nil {
		return nil, err
	}
	storyBoard, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return &api.CreateStoryBoardCommentResponse{
			Code:    -1,
			Message: "get storyboard failed",
		}, nil
	}
	storyBoard.CommentNum++
	err = models.UpdateStoryboard(ctx, storyBoard)
	if err != nil {
		return &api.CreateStoryBoardCommentResponse{
			Code:    -1,
			Message: "update storyboard comment count failed",
		}, nil
	}
	return &api.CreateStoryBoardCommentResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (s *CommentService) DeleteStoryBoardComment(ctx context.Context, req *api.DeleteStoryBoardCommentRequest) (*api.DeleteStoryBoardCommentResponse, error) {
	err := models.DeleteComment(uint64(req.GetCommentId()))
	if err != nil {
		return nil, err
	}
	storyBoard, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return &api.DeleteStoryBoardCommentResponse{
			Code:    -1,
			Message: "get storyboard failed",
		}, nil
	}
	storyBoard.CommentNum--
	return &api.DeleteStoryBoardCommentResponse{
		Code:    0,
		Message: "success",
	}, nil
}
func (s *CommentService) GetStoryBoardComments(ctx context.Context, req *api.GetStoryBoardCommentsRequest) (*api.GetStoryBoardCommentsResponse, error) {
	comments, err := models.GetCommentListByStoryBoard(
		uint64(req.GetBoardId()), int64(req.GetOffset()), int64(req.GetPageSize()))
	if err != nil {
		return &api.GetStoryBoardCommentsResponse{
			Code:     -1,
			Message:  "get comments error",
			Total:    0,
			Comments: []*api.StoryComment{},
		}, nil
	}
	if len(*comments) == 0 {
		log.Log().Info("get comment list by story board empty")
		return &api.GetStoryBoardCommentsResponse{
			Code:     0,
			Message:  "success",
			Total:    0,
			Comments: []*api.StoryComment{},
		}, nil
	}
	createrIds := make([]int64, 0)
	for _, comment := range *comments {
		createrIds = append(createrIds, comment.UserID)
	}
	createrMap, err := models.GetUsersByIdsMap(createrIds)
	if err != nil {
		log.Log().Sugar().Info("get user by ids map error: %s", err.Error())
		return nil, err
	}
	createrMapData, _ := json.Marshal(createrMap)
	log.Log().Info("get user by ids map success: " + string(createrMapData))
	apiComments := make([]*api.StoryComment, 0)
	for _, comment := range *comments {
		apiComments = append(apiComments, &api.StoryComment{
			CommentId:  int64(comment.ID),
			Content:    string(comment.Content),
			CreatedAt:  comment.CreateAt.Unix(),
			UpdatedAt:  comment.UpdateAt.Unix(),
			UserId:     comment.UserID,
			BoardId:    comment.StoryboardID,
			LikeCount:  comment.LikeCount,
			ReplyCount: comment.ReplyCount,
			Creator: &api.UserInfo{
				UserId: comment.UserID,
				Name:   createrMap[int(comment.UserID)].Name,
				Avatar: createrMap[int(comment.UserID)].Avatar,
			},
		})
	}
	log.Log().Info("get comment list by story board success")
	return &api.GetStoryBoardCommentsResponse{
		Code:     0,
		Message:  "success",
		Total:    int64(len(*comments)),
		Comments: apiComments,
	}, nil
}

func (s *CommentService) LikeComment(ctx context.Context, req *api.LikeCommentRequest) (*api.LikeCommentResponse, error) {
	err := models.LikeComment(uint64(req.GetCommentId()), uint64(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	return &api.LikeCommentResponse{
		Code:    0,
		Message: "success",
	}, nil
}
func (s *CommentService) DislikeComment(ctx context.Context, req *api.DislikeCommentRequest) (*api.DislikeCommentResponse, error) {
	err := models.DislikeComment(uint64(req.GetCommentId()), uint64(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	return &api.DislikeCommentResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryBoardCommentReplies(ctx context.Context, req *api.GetStoryBoardCommentRepliesRequest) (*api.GetStoryBoardCommentRepliesResponse, error) {
	replies, err := models.GetStoryBoardCommentReplies(uint64(req.GetCommentId()))
	if err != nil {
		return nil, err
	}
	if len(*replies) == 0 {
		return &api.GetStoryBoardCommentRepliesResponse{
			Code:    0,
			Message: "success",
			Total:   0,
			Replies: []*api.StoryComment{},
		}, nil
	}
	createrIds := make([]int64, 0)
	for _, reply := range *replies {
		createrIds = append(createrIds, reply.UserID)
	}
	createrMap, err := models.GetUsersByIdsMap(createrIds)
	if err != nil {
		log.Log().Sugar().Info("get user by ids map error: %s", err.Error())
		return nil, err
	}
	createrMapData, _ := json.Marshal(createrMap)
	log.Log().Info("get user by ids map success: " + string(createrMapData))
	apiReplies := make([]*api.StoryComment, 0)
	for _, reply := range *replies {
		apiReplies = append(apiReplies, &api.StoryComment{
			CommentId:  int64(reply.ID),
			Content:    string(reply.Content),
			CreatedAt:  reply.CreateAt.Unix(),
			UpdatedAt:  reply.UpdateAt.Unix(),
			UserId:     reply.UserID,
			BoardId:    reply.StoryboardID,
			LikeCount:  reply.LikeCount,
			ReplyCount: reply.ReplyCount,
			Creator: &api.UserInfo{
				UserId: reply.UserID,
				Name:   createrMap[int(reply.UserID)].Name,
				Avatar: createrMap[int(reply.UserID)].Avatar,
			},
		})
	}
	return &api.GetStoryBoardCommentRepliesResponse{
		Code:    0,
		Message: "success",
		Total:   int64(len(*replies)),
		Replies: apiReplies,
	}, nil
}
