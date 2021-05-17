package teams

var server TeamServer

func init() {
	server = NewTeamService()
}

func GetTeamService() TeamServer {
	return server
}

func NewTeamService() *TeamService {
	return &TeamService{}
}

type TeamServer interface {
	Version()
	About()
}

type TeamService struct {
}

func (c TeamService) Version() {

}

func (c TeamService) About() {

}
