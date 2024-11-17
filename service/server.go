package service

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	api "github.com/grapery/common-protoc/gen"
	genconnect "github.com/grapery/common-protoc/gen/genconnect"
	"github.com/grapery/grapery/config"
	models "github.com/grapery/grapery/models"
	auth "github.com/grapery/grapery/service/auth"
	"github.com/grapery/grapery/service/common"
	"github.com/grapery/grapery/service/group"
	"github.com/grapery/grapery/service/user"
	"github.com/grapery/grapery/utils/cache"
	"github.com/grapery/grapery/utils/jwt"
	"github.com/grapery/grapery/version"
)

// TeamsService imaplement api.RegisterTeamsAPIServer interface
type TeamsService struct {
	Ctx    context.Context
	Cancel context.CancelFunc

	*auth.AuthService
	*user.UserService
	*group.GroupService
	*group.ProjectService
	*group.StoryItemService
	*group.CommentService
	*common.CommonService
	*group.StoryService
	// api.UnimplementedTeamsAPIServer
}

func (ts *TeamsService) Version(ctx context.Context, req *connect.Request[api.VersionRequest]) (*connect.Response[api.VersionResponse], error) {
	return &connect.Response[api.VersionResponse]{
		Msg: &api.VersionResponse{
			Code:    0,
			Message: "OK",
			Data: &api.VersionResponse_Data{
				Version: version.GetVersion(),
			},
		},
	}, nil
}

func (ts *TeamsService) About(ctx context.Context, req *connect.Request[api.AboutRequest]) (*connect.Response[api.AboutResponse], error) {
	return &connect.Response[api.AboutResponse]{
		Msg: &api.AboutResponse{
			Content: "Grapery",
		}}, nil
}

// NewTeamsService create a new TeamsService
func NewTeamsService() *TeamsService {
	ts := &TeamsService{}
	ts.AuthService = &auth.AuthService{
		Jwt: &jwt.JwtWrapper{
			SecretKey:       "grapery",
			ExpirationHours: 24 * 7,
		},
	}
	ts.UserService = &user.UserService{}
	ts.GroupService = &group.GroupService{}
	ts.ProjectService = &group.ProjectService{}
	ts.StoryItemService = &group.StoryItemService{}
	ts.CommentService = &group.CommentService{}
	ts.CommonService = &common.CommonService{}
	ts.StoryService = &group.StoryService{}
	ts.Ctx, ts.Cancel = context.WithCancel(context.Background())
	return ts
}

func Run(ts *TeamsService, cfg *config.Config) error {
	cache.NewRedisClient(cfg)
	err := models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Database)
	if err != nil {
		log.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	opts := []connect.HandlerOption{
		connect.WithInterceptors(
			auth.AuthInterceptorFunc{
				Handle: auth.ConnectAuthFuncfunc,
			},
		),
	}
	go func() {
		mux := http.NewServeMux()
		path, handler := genconnect.NewTeamsAPIHandler(ts, opts...)
		mux.Handle(path, handler)
		http.ListenAndServe(
			"127.0.0.1:12305",
			h2c.NewHandler(mux, &http2.Server{}),
		)
	}()
	return nil
}
