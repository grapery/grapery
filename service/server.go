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

func (ts *TeamsService) Version(ctx context.Context, req *connect.Request[api.VersionRequest]) (*connect.Response[api.VersionResponse], error) {
	return &connect.Response[api.VersionResponse]{
		Msg: &api.VersionResponse{
			Version: "0.0.1",
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
	ts.ItemService = &group.ItemService{}
	ts.CommentService = &group.CommentService{}
	ts.CommonService = &common.CommonService{}
	ts.Ctx, ts.Cancel = context.WithCancel(context.Background())
	return ts
}

func Run(ts *TeamsService, cfg *config.Config) error {
	//loggor := log.New()
	cache.NewRedisClient(cfg)
	err := models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Database)
	if err != nil {
		log.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	// opt := grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
	// 	grpc_log.UnaryServerInterceptor(utils_log.InterceptorLogger(loggor)),
	// 	auth.AuthInterceptor(auth.AuthFunc),
	// ))
	// s := grpc.NewServer(opt)
	// api.RegisterTeamsAPIServer(s, ts)
	// go func() {
	// 	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", cfg.RpcPort))
	// 	if err != nil {
	// 		log.Fatal("failed to listen: ", err)
	// 	}
	// 	if err := s.Serve(lis); err != nil {
	// 		log.Warn(err)
	// 	}
	// }()
	opts := []connect.HandlerOption{
		connect.WithInterceptors(
			auth.AuthInterceptorFunc{
				Handle: auth.ConnectAuthFuncfunc,
			},
		),
		connect.WithRecover(nil),
	}
	go func() {
		mux := http.NewServeMux()
		path, handler := genconnect.NewTeamsAPIHandler(ts, opts...)
		mux.Handle(path, handler)
		http.ListenAndServe(
			"localhost:12307",
			h2c.NewHandler(mux, &http2.Server{}),
		)
	}()
	// go func() {
	// 	log.Infof("start http server port: [%s]", cfg.HttpPort)
	// 	ctx, cancel := context.WithCancel(context.Background())
	// 	defer cancel()
	// 	mux := runtime.NewServeMux(runtime.WithMarshalerOption(
	// 		runtime.MIMEWildcard, &runtime.JSONPb{}),
	// 		runtime.WithMarshalerOption("application/smalljson", &runtime.JSONPb{}),
	// 	)
	// 	maxRecvMsgSize := 16 * 1024 * 1024
	// 	log.Infof("http server max message size : [%d] Bytes", maxRecvMsgSize)
	// 	dialOptions := []grpc.DialOption{
	// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
	// 		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvMsgSize)),
	// 	}
	// 	err := api.RegisterTeamsAPIHandlerFromEndpoint(
	// 		ctx,
	// 		mux,
	// 		fmt.Sprintf("localhost:%s", cfg.RpcPort),
	// 		dialOptions,
	// 	)
	// 	if err != nil {
	// 		log.Fatal("failed to register: ", err)
	// 	}
	// 	http.Handle("/", mux)
	// 	httpServer := http.Server{Addr: fmt.Sprintf("localhost:%s", cfg.HttpPort)}
	// 	if err := httpServer.ListenAndServe(); err != nil {
	// 		log.Warn(err)
	// 	}
	// }()
	return nil
}
