package service

import (
	"context"
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
	"github.com/grapery/grapery/utils/cache"
)

// TeamsService imaplement api.RegisterTeamsAPIServer interface
type TeamsService struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

func (ts *TeamsService) Login(context.Context, *api.LoginRequest) (*api.LoginResponse, error) {
	return nil, nil
}
func (ts *TeamsService) Logout(context.Context, *api.LogoutRequest) (*api.LogoutResponse, error) {
	return nil, nil
}
func (ts *TeamsService) Register(context.Context, *api.RegisterRequest) (*api.RegisterResponse, error) {
	return nil, nil
}
func (ts *TeamsService) ResetPwd(context.Context, *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UserInfo(context.Context, *api.UserInfoRequest) (*api.UserInfoResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UpdateUserAvator(context.Context, *api.UpdateUserAvatorRequest) (*api.UpdateUserAvatorResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UserWatching(context.Context, *api.UserWatchingRequest) (*api.UserWatchingResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UserGroup(context.Context, *api.UserGroupRequest) (*api.UserGroupResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UserFollowingGroup(context.Context, *api.UserFollowingGroupRequest) (*api.UserFollowingGroupResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UserUpdate(context.Context, *api.UserUpdateRequest) (*api.UserUpdateResponse, error) {
	return nil, nil
}
func (ts *TeamsService) FetchUserActives(context.Context, *api.FetchUserActivesRequest) (*api.FetchUserActivesResponse, error) {
	return nil, nil
}
func (ts *TeamsService) SearchUser(context.Context, *api.SearchUserRequest) (*api.SearchUserResponse, error) {
	return nil, nil
}
func (ts *TeamsService) CreateGroup(context.Context, *api.CreateGroupReqeust) (*api.CreateGroupResponse, error) {
	return nil, nil
}
func (ts *TeamsService) GetGroup(context.Context, *api.GetGroupReqeust) (*api.GetGroupResponse, error) {
	return nil, nil
}
func (ts *TeamsService) GetGroupActives(context.Context, *api.GetGroupActivesRequest) (*api.GetGroupActivesResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UpdateGroupInfo(context.Context, *api.UpdateGroupInfoRequest) (*api.UpdateGroupInfoResponse, error) {
	return nil, nil
}
func (ts *TeamsService) DeleteGroup(context.Context, *api.DeleteGroupRequest) (*api.DeleteGroupResponse, error) {
	return nil, nil
}
func (ts *TeamsService) FetchGroupMembers(context.Context, *api.FetchGroupMembersRequest) (*api.FetchGroupMembersResponse, error) {
	return nil, nil
}
func (ts *TeamsService) SearchGroup(context.Context, *api.SearchGroupReqeust) (*api.SearchGroupResponse, error) {
	return nil, nil
}
func (ts *TeamsService) FetchGroupProjects(context.Context, *api.FetchGroupProjectsReqeust) (*api.FetchGroupProjectsResponse, error) {
	return nil, nil
}
func (ts *TeamsService) JoinGroup(context.Context, *api.JoinGroupRequest) (*api.JoinGroupResponse, error) {
	return nil, nil
}
func (ts *TeamsService) LeaveGroup(context.Context, *api.LeaveGroupRequest) (*api.LeaveGroupResponse, error) {
	return nil, nil
}
func (ts *TeamsService) GetProject(context.Context, *api.GetProjectRequest) (*api.GetProjectResponse, error) {
	return nil, nil
}
func (ts *TeamsService) CreateProject(context.Context, *api.CreateProjectRequest) (*api.CreateProjectResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UpdateProject(context.Context, *api.UpdateProjectRequest) (*api.UpdateProjectResponse, error) {
	return nil, nil
}
func (ts *TeamsService) DeleteProject(context.Context, *api.DeleteProjectRequest) (*api.DeleteProjectResponse, error) {
	return nil, nil
}
func (ts *TeamsService) GetProjectProfile(context.Context, *api.GetProjectProfileRequest) (*api.GetProjectProfileResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UpdateProjectProfile(context.Context, *api.UpdateProjectProfileRequest) (*api.UpdateProjectProfileResponse, error) {
	return nil, nil
}
func (ts *TeamsService) WatchProject(context.Context, *api.WatchProjectReqeust) (*api.WatchProjectResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UnWatchProject(context.Context, *api.UnWatchProjectReqeust) (*api.UnWatchProjectResponse, error) {
	return nil, nil
}
func (ts *TeamsService) SearchGroupProject(context.Context, *api.SearchProjectRequest) (*api.SearchProjectResponse, error) {
	return nil, nil
}
func (ts *TeamsService) SearchProject(context.Context, *api.SearchAllProjectRequest) (*api.SearchAllProjectResponse, error) {
	return nil, nil
}
func (ts *TeamsService) ExploreProject(context.Context, *api.ExploreProjectsRequest) (*api.ExploreProjectsResponse, error) {
	return nil, nil
}
func (ts *TeamsService) GetProjectItems(context.Context, *api.GetProjectItemsRequest) (*api.GetProjectItemsResponse, error) {
	return nil, nil
}
func (ts *TeamsService) GetGroupItems(context.Context, *api.GetGroupItemsRequest) (*api.GetGroupItemsResponse, error) {
	return nil, nil
}
func (ts *TeamsService) GetUserItems(context.Context, *api.GetUserItemsRequest) (*api.GetUserItemsResponse, error) {
	return nil, nil
}
func (ts *TeamsService) GetItem(context.Context, *api.GetItemRequest) (*api.GetItemResponse, error) {
	return nil, nil
}
func (ts *TeamsService) CreateItem(context.Context, *api.CreateItemRequest) (*api.CreateItemResponse, error) {
	return nil, nil
}
func (ts *TeamsService) UpdateItem(context.Context, *api.UpdateItemRequest) (*api.UpdateItemResponse, error) {
	return nil, nil
}
func (ts *TeamsService) DeleteItem(context.Context, *api.DeleteItemRequest) (*api.DeleteItemResponse, error) {
	return nil, nil
}
func (ts *TeamsService) LikeItem(context.Context, *api.LikeItemRequest) (*api.LikeItemResponse, error) {
	return nil, nil
}
func (ts *TeamsService) CreateComment(context.Context, *api.CreateCommentReq) (*api.CreateCommentResp, error) {
	return nil, nil
}
func (ts *TeamsService) GetItemComment(context.Context, *api.GetItemCommentReq) (*api.GetItemCommentResp, error) {
	return nil, nil
}
func (ts *TeamsService) GetGroupItemComment(context.Context, *api.GetUserProjectCommentReq) (*api.GetUserProjectCommentResp, error) {
	return nil, nil
}

// NewTeamsService create a new TeamsService
func NewTeamsService() *TeamsService {
	ts := &TeamsService{}
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
		lis, err := net.Listen("tcp", cfg.RpcPort)
		if err != nil {
			log.Fatal("failed to listen: ", err)
		}
		if err := s.Serve(lis); err != nil {
			log.Warn(err)
		}
	}()
	// listen http
	go func() {
		log.Infof("start http server : [%s]", cfg.HttpPort)
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
			grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(auth.ExampleAuthFunc)),
			grpc.UnaryInterceptor(grpc_auth.UnaryServerInterceptor(auth.ExampleAuthFunc)),
		}
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
		httpServer := http.Server{Addr: cfg.HttpPort}
		if err := httpServer.ListenAndServe(); err != nil {
			log.Warn(err)
		}
	}()
	log.Infoln("server is going stop")
	return nil
}
