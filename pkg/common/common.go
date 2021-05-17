package common

var server CommonServer

func init() {
	server = NewCommonService()
}

func GetCommonService() CommonServer {
	return server
}

func NewCommonService() *CommonService {
	return &CommonService{}
}

type CommonServer interface {
	Version()
	About()
}

type CommonService struct {
}

func (c CommonService) Version() {

}
func (c CommonService) About() {

}
