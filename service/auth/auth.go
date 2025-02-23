package auth

import (
	// "net/http"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/pkg/auth"
	utils "github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/jwt"
)

func AuthInterceptor(authFunc grpc_auth.AuthFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		println("AuthInterceptor method: ", info.FullMethod)
		if info.FullMethod == "/common.TeamsAPI/Login" ||
			info.FullMethod == "/common.TeamsAPI/About" ||
			info.FullMethod == "/common.TeamsAPI/Register" ||
			info.FullMethod == "/common.TeamsAPI/Reset_password" {
			return handler(ctx, req)
		}
		var newCtx context.Context
		var err error
		if overrideSrv, ok := info.Server.(grpc_auth.ServiceAuthFuncOverride); ok {
			newCtx, err = overrideSrv.AuthFuncOverride(ctx, info.FullMethod)
		} else {
			newCtx, err = authFunc(ctx)
		}
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}

type AuthInterceptorFunc struct {
	Handle func(context.Context, connect.Spec, http.Header, any) error
}

func (f AuthInterceptorFunc) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Spec().Procedure == "/common.TeamsAPI/Login" ||
			req.Spec().Procedure == "/common.TeamsAPI/About" ||
			req.Spec().Procedure == "/common.TeamsAPI/Register" ||
			req.Spec().Procedure == "/common.TeamsAPI/Reset_password" {
			return next(ctx, req)
		}
		err := f.Handle(ctx, req.Spec(), req.Header(), req)
		if err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (f AuthInterceptorFunc) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
func (f AuthInterceptorFunc) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func ConnectAuthFuncfunc(ctx context.Context, spec connect.Spec, header http.Header, a any) error {
	cookieInfo := header.Get(utils.GrpcGateWayCookie)
	if len(cookieInfo) == 0 {
		return status.Errorf(codes.Unauthenticated, "empty auth from md: %s", utils.GrpcGateWayCookie)
	}
	token := cookieInfo
	jwtInfo := jwt.NewJwtWrapper(utils.SecretKey, utils.ExpirationHours)
	tokenInfo, err := jwtInfo.ValidateToken(token)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}
	header.Set("auth.sub", jwtInfo.SecretKey)
	header.Set(utils.UserIdKey, fmt.Sprintf("%d", tokenInfo.UID))
	// ------------------------------
	aData, _ := json.Marshal(a)
	println("WithRequestLogInterceptor method: ", spec.Procedure, " params: ", string(aData))
	// ------------------------------
	return nil
}

func WithRequestLogInterceptor(ctx context.Context, spec connect.Spec, header http.Header, a any) error {
	aData := a.([]byte)
	println("WithRequestLogInterceptor method: ", spec.Procedure, " params: ", string(aData))
	return nil
}

func AuthFunc(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "metadata.FromIncomingContext: %v", codes.FailedPrecondition)
	}
	tokenList := md[utils.GrpcGateWayCookie]

	if len(tokenList) <= 0 {
		return nil, fmt.Errorf("empty auth from md: %s", utils.GrpcGateWayCookie)
	}
	tokenListTemp := strings.Split(tokenList[0], "=")
	token := tokenListTemp[1]
	jwtInfo := jwt.NewJwtWrapper(utils.SecretKey, utils.ExpirationHours)
	tokenInfo, err := jwtInfo.ValidateToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}
	grpc_ctxtags.Extract(ctx).Set("auth.sub", jwtInfo.SecretKey)
	newCtx := context.WithValue(ctx, utils.UserIdKey, tokenInfo.UID)
	return newCtx, nil
}

type Result struct {
	Code  int    `json:"code,omitempty"`
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

func LoginFunc(w http.ResponseWriter, r *http.Request) {
	auth := NewAuthService(utils.SecretKey, utils.ExpirationHours)
	reqBody, err := io.ReadAll(r.Body)
	ret := new(Result)
	if err != nil {
		ret.Code = -1
		ret.Error = "params error"
		resultData, _ := json.Marshal(ret)
		_, _ = w.Write(resultData)
		return
	}
	info := &api.LoginRequest{}
	err = json.Unmarshal(reqBody, info)
	if err != nil {
		ret.Code = -1
		ret.Error = "params error"
		resultData, _ := json.Marshal(ret)
		_, _ = w.Write(resultData)
		return
	}
	if info.Account == "" || info.Password == "" {
		ret.Code = -1
		ret.Error = "account or password params error"
		resultData, _ := json.Marshal(ret)
		_, _ = w.Write(resultData)
		return
	}
	req := &connect.Request[api.LoginRequest]{
		Msg: info,
	}
	resp, err := auth.Login(r.Context(), req)
	if err != nil {
		ret.Code = -1
		ret.Error = err.Error()
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}
	ret.Code = 1
	ret.Token = resp.Msg.GetData().GetToken()
	resultData, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Cookie", "token="+ret.Token)
	_, _ = w.Write(resultData)
	return
}

func ParseInt(s string) int {
	if len(s) == 0 {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

type AuthService struct {
	Jwt *jwt.JwtWrapper
}

func NewAuthService(key string, expiration int) *AuthService {
	return &AuthService{
		Jwt: jwt.NewJwtWrapper(key, expiration),
	}
}

func (ts *AuthService) Login(ctx context.Context, req *connect.Request[api.LoginRequest]) (*connect.Response[api.LoginResponse], error) {
	info, err := auth.GetAuthService().
		Login(ctx, req.Msg.GetAccount(), req.Msg.GetPassword())
	if err != nil {
		return nil, err
	}
	token, err := ts.Jwt.GenerateToken(info)
	if err != nil {
		return nil, err
	}
	ret := &api.LoginResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.LoginResponse_Data{
			UserId: info.GetUserId(),
			Token:  token,
		},
	}
	return &connect.Response[api.LoginResponse]{Msg: ret}, nil
}

func (ts *AuthService) Logout(ctx context.Context, req *connect.Request[api.LogoutRequest]) (*connect.Response[api.LogoutResponse], error) {
	_, err := auth.GetAuthService().Logout(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.LogoutResponse]{}, nil
}

func (ts *AuthService) Register(ctx context.Context, req *connect.Request[api.RegisterRequest]) (*connect.Response[api.RegisterResponse], error) {
	err := auth.GetAuthService().Register(
		context.Background(),
		req.Msg.GetName(),
		req.Msg.GetAccount(),
		req.Msg.GetPassword(),
	)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.RegisterResponse]{
		Msg: &api.RegisterResponse{
			Code: 1,
			Msg:  "register success",
		},
	}, nil
}

func (ts *AuthService) ResetPwd(ctx context.Context, req *connect.Request[api.ResetPasswordRequest]) (*connect.Response[api.ResetPasswordResponse], error) {
	_, err := auth.GetAuthService().ResetPassword(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.ResetPasswordResponse]{}, nil
}

func (ts *AuthService) RefreshToken(ctx context.Context, req *connect.Request[api.RefreshTokenRequest]) (*connect.Response[api.RefreshTokenResponse], error) {
	token, err := ts.Jwt.ValidateToken(req.Msg.GetToken())
	if err != nil {
		return nil, err
	}
	fmt.Println("RefreshToken : ", req.Msg.String())
	info, err := auth.GetAuthService().GetUserInfo(ctx, token.UID, token.Email)
	if err != nil {
		return nil, err
	}
	infoData, _ := json.Marshal(info)
	fmt.Println("RefreshToken : ", string(infoData))
	if info == nil {
		return nil, fmt.Errorf("user not found")
	}
	if info.GetUserId() != token.UID {
		return nil, fmt.Errorf("user id not match")
	}
	newToken, err := ts.Jwt.GenerateToken(info)
	if err != nil {
		return nil, err
	}
	fmt.Println("newToken :", newToken)
	ret := &api.RefreshTokenResponse{
		UserId: info.GetUserId(),
		Token:  newToken,
	}
	return &connect.Response[api.RefreshTokenResponse]{Msg: ret}, nil
}
