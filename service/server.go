package service

import (
	"context"
	"fmt"
	"net"
	"net/http"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	grpc_log "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/config"
	models "github.com/grapery/grapery/models"
	auth "github.com/grapery/grapery/service/auth"
	"github.com/grapery/grapery/service/common"
	"github.com/grapery/grapery/service/group"
	"github.com/grapery/grapery/service/user"
	"github.com/grapery/grapery/utils/cache"
	"github.com/grapery/grapery/utils/jwt"
	utils_log "github.com/grapery/grapery/utils/log"
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
	*common.CommonService
	// api.UnimplementedTeamsAPIServer
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
	ts.AuthService = &auth.AuthService{
		Jwt: &jwt.JwtWrapper{
			SecretKey:       "grapery",
			ExpirationHours: 24 * 7,
		},
	}
	ts.UserService = &user.UserService{}
	ts.GroupService = &group.GroupService{}
	ts.ProjectService = &group.ProjectService{}
	ts.ItemService = &group.ItemService{}
	ts.CommentService = &group.CommentService{}
	ts.CommonService = &common.CommonService{}
	ts.Ctx, ts.Cancel = context.WithCancel(context.Background())
	return ts
}

func Run(ts *TeamsService, cfg *config.Config) error {
	loggor := log.New()
	cache.NewRedisClient(cfg)
	err := models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Database)
	if err != nil {
		log.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	opt := grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_log.UnaryServerInterceptor(utils_log.InterceptorLogger(loggor)),
		grpc_auth.UnaryServerInterceptor(auth.AuthFunc),
	))
	opt1 := grpc.StreamInterceptor(
		grpc_auth.StreamServerInterceptor(auth.AuthFunc),
	)
	s := grpc.NewServer(opt, opt1)
	api.RegisterTeamsAPIServer(s, ts)
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", cfg.RpcPort))
		if err != nil {
			log.Fatal("failed to listen: ", err)
		}
		if err := s.Serve(lis); err != nil {
			log.Warn(err)
		}
	}()
	go func() {
		log.Infof("start http server port: [%s]", cfg.HttpPort)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mux := runtime.NewServeMux(runtime.WithMarshalerOption(
			runtime.MIMEWildcard, &runtime.JSONPb{}),
			runtime.WithMarshalerOption("application/smalljson", &runtime.JSONPb{}),
		)
		maxRecvMsgSize := 16 * 1024 * 1024
		log.Infof("http server max message size : [%d] Bytes", maxRecvMsgSize)
		dialOptions := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
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
		http.Handle("/v1/login", http.HandlerFunc(auth.LoginFunc))
		http.Handle("/v1/logout", http.HandlerFunc(auth.Logout))
		http.Handle("/v1/register", http.HandlerFunc(auth.Register))
		http.Handle("/v1/reset_password", http.HandlerFunc(auth.ResetPwd))
		http.Handle("/v1/about", http.HandlerFunc(auth.About))
		http.Handle("/", mux)
		httpServer := http.Server{Addr: fmt.Sprintf("localhost:%s", cfg.HttpPort)}
		if err := httpServer.ListenAndServe(); err != nil {
			log.Warn(err)
		}
	}()
	return nil
}
