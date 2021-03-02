package project

var projectServicer ProjectServicer

func init() {
	projectServicer = NewProjectService()
}

func GetProjectServicer() ProjectServicer {
	return projectServicer
}

func NewProjectService() *ProjectService {
	return &ProjectService{}
}

type ProjectServicer interface{}

type ProjectService struct{}
