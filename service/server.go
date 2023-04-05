package service

import (
	"context"
	"net"
	"net/http"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/config"
	models "github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/cache"
)

// TeamsService imaplement api.RegisterTeamsAPIServer interface
type TeamsService struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

// imaplement interface api.RegisterTeamsAPIServer
func (s *TeamsService) GetTeam(ctx context.Context, req *api.GetTeamRequest) (*api.GetTeamResponse, error) {
	return &api.GetTeamResponse{}, nil
}

// NewTeamsService create a new TeamsService
func NewTeamsService() *TeamsService {
	ts := &TeamsService{}
	ts.Ctx, ts.Cancel = context.WithCancel(context.Background())
	return ts
}
func Run(cfg *config.Config) {
	grpcLog := log.WithField("server", "api")
	cache.NewRedisClient(cfg)
	err := models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Database)
	if err != nil {
		log.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	opt := grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_logrus.UnaryServerInterceptor(grpcLog),
	))
	s := grpc.NewServer(opt)
	api.RegisterTeamsAPIServer(s, NewTeamsService())
	// listen grpc
	go func() {
		lis, err := net.Listen("tcp", ":12345")
		if err != nil {
			log.Fatal("failed to listen: ", err)
		}
		if err := s.Serve(lis); err != nil {
			log.Warn(err)
		}
	}()
	// listen http
	go func() {
		log.Infof("start http server : [%s]", "12346")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mux := runtime.NewServeMux(runtime.WithMarshalerOption(
			runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}),
			runtime.WithMarshalerOption("application/smalljson", &runtime.JSONPb{OrigName: true, EmitDefaults: false}),
		)
		maxRecvMsgSize := 16 * 1024 * 1024
		log.Infof("http server max message size : [%d] Bytes", maxRecvMsgSize)
		dialOptions := []grpc.DialOption{grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvMsgSize))}
		err := api.RegisterTeamsAPIHandlerFromEndpoint(
			ctx,
			mux,
			"12346",
			dialOptions,
		)
		if err != nil {
			log.Fatal("failed to register: ", err)
		}
		http.Handle("/", mux)
		httpServer := http.Server{Addr: "12346"}
		if err := httpServer.ListenAndServe(); err != nil {
			log.Warn(err)
		}
	}()
}
