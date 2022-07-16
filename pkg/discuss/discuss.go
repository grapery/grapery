package discuss

var server DiscussServer

func init() {
	server = NewDisscussService()
}

func GetDiscussService() DiscussServer {
	return server
}

func NewDisscussService() DiscussServer {
	return &DiscussService{}
}

type DiscussServer interface {
}

// Discuss service
type DiscussService struct {
}
