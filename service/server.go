package service

import (
	"context"
	"fmt"
	"net"
	"net/http"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/config"
	models "github.com/grapery/grapery/models"
	auth "github.com/grapery/grapery/service/auth"
	"github.com/grapery/grapery/service/group"
	"github.com/grapery/grapery/service/user"
	"github.com/grapery/grapery/utils/cache"
)

// TeamsService imaplement api.RegisterTeamsAPIServer interface
type TeamsService struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	*auth.AuthService
	*user.UserService
	*group.GroupService
	*group.ProjectService
	*group.ItemService
	*group.CommentService
}

func (ts *TeamsService) Version(ctx context.Context, req *api.VersionRequest) (*api.VersionResponse, error) {
	return &api.VersionResponse{Version: "0.0.1"}, nil
}

func (ts *TeamsService) About(ctx context.Context, req *api.AboutRequest) (*api.AboutResponse, error) {
	return &api.AboutResponse{
		Content: "Grapery is a project management tool for teams. It is a web application that allows you to manage your projects, tasks, and team members. It is a web application that allows you to manage your projects, tasks, and team members.",
	}, nil
}

// NewTeamsService create a new TeamsService
func NewTeamsService() *TeamsService {
	ts := &TeamsService{}
	ts.AuthService = &auth.AuthService{}
	ts.UserService = &user.UserService{}
	ts.GroupService = &group.GroupService{}
	ts.ProjectService = &group.ProjectService{}
	ts.ItemService = &group.ItemService{}
	ts.CommentService = &group.CommentService{}
	ts.Ctx, ts.Cancel = context.WithCancel(context.Background())
	return ts
}

func Run(ts *TeamsService, cfg *config.Config) error {
	grpcLog := log.WithField("server", "api")
	cache.NewRedisClient(cfg)
	err := models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Database)
	if err != nil {
		log.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	opt := grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_logrus.UnaryServerInterceptor(grpcLog),
		grpc_auth.UnaryServerInterceptor(auth.ExampleAuthFunc),
	))
	opt1 := grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(auth.ExampleAuthFunc))
	s := grpc.NewServer(opt, opt1)
	api.RegisterTeamsAPIServer(s, ts)
	// listen grpc
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", cfg.RpcPort))
		if err != nil {
			log.Fatal("failed to listen: ", err)
		}
		if err := s.Serve(lis); err != nil {
			log.Warn(err)
		}
	}()
	// listen http
	go func() {
		log.Infof("start http server port: [%s]", cfg.HttpPort)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mux := runtime.NewServeMux(runtime.WithMarshalerOption(
			runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}),
			runtime.WithMarshalerOption("application/smalljson", &runtime.JSONPb{OrigName: true, EmitDefaults: false}),
		)
		maxRecvMsgSize := 16 * 1024 * 1024
		log.Infof("http server max message size : [%d] Bytes", maxRecvMsgSize)
		dialOptions := []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvMsgSize)),
		}
		err := api.RegisterTeamsAPIHandlerFromEndpoint(
			ctx,
			mux,
			fmt.Sprintf("localhost:%s", cfg.RpcPort),
			dialOptions,
		)
		if err != nil {
			log.Fatal("failed to register: ", err)
		}
		http.Handle("/", mux)
		httpServer := http.Server{Addr: fmt.Sprintf("localhost:%s", cfg.HttpPort)}
		if err := httpServer.ListenAndServe(); err != nil {
			log.Warn(err)
		}
	}()
	log.Infoln("server is going stop")
	return nil
}
