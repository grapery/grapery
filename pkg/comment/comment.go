package comment

var server CommentServer

func init() {
	server = NewCommentService()
}

func GetCommentService() CommentServer {
	return server
}

func NewCommentService() *CommentService {
	return &CommentService{}
}

type CommentServer interface {
}

type CommentService struct {
}

func (cs *CommentService) GetGroupComment() {

}

func (cs *CommentService) GetProjectComment() {

}

func (cs *CommentService) GetItemComment() {

}
