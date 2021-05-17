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
	GetTeamMember()
	GetTeamProfile()
	// GetGroupTeams()
	// GetUserTeams()
	CreateTeam()
	UpdateTeam()
	DeleteTeam()
	ExpiredTeam()
}

type TeamService struct {
}

func (c *TeamService) GetTeamMember()  {}
func (c *TeamService) GetTeamProfile() {}
func (c *TeamService) GetGroupTeams()  {}
func (c *TeamService) GetUserTeams()   {}
func (c *TeamService) CreateTeam()     {}
func (c *TeamService) UpdateTeam()     {}
func (c *TeamService) DeleteTeam()     {}
func (c *TeamService) ExpiredTeam()    {}
