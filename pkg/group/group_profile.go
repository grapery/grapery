package group

var profileServicer GroupProfileServicer

func init() {
	profileServicer = NewGroupProfileService()
}

func GetGroupProfileService() GroupProfileServicer {
	return profileServicer
}

func NewGroupProfileService() *GroupProfileService {
	return &GroupProfileService{}
}

type GroupProfileServicer interface{}

type GroupProfileService struct{}
