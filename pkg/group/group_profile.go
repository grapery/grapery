package group

var profileServer GroupProfileServer

func init() {
	profileServer = NewGroupProfileService()
}

func GetGroupProfileService() GroupProfileServer {
	return profileServer
}

func NewGroupProfileService() *GroupProfileService {
	return &GroupProfileService{}
}

type GroupProfileServer interface{}

type GroupProfileService struct{}
