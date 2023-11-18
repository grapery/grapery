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
		println("method: ", info.FullMethod)
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

func AuthFunc(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	fmt.Println("metadata", md)
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
		w.Write(resultData)
		return
	}
	req := &api.LoginRequest{}
	err = json.Unmarshal(reqBody, req)
	fmt.Println(req.String())
	if req.Account == "" || req.Password == "" {
		ret.Code = -1
		ret.Error = "account or password params error"
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
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
	ret.Token = resp.GetToken()
	resultData, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Cookie", "token="+ret.Token)
	w.Write(resultData)
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

func Logout(w http.ResponseWriter, r *http.Request) {
	auth := NewAuthService(utils.SecretKey, utils.ExpirationHours)
	query := r.URL.Query()
	req := &api.LogoutRequest{
		Token: query.Get("token"),
	}
	val, err := strconv.Atoi(query.Get("user_id"))
	ret := new(Result)
	if err != nil {
		ret.Code = -1
		ret.Error = "parse params error"
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}
	req.UserId = uint64(val)
	if req.Token == "" || req.UserId == 0 {
		ret.Code = -1
		ret.Error = "check params error"
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}

	_, err = auth.Logout(r.Context(), req)
	if err != nil {
		ret.Code = -1
		ret.Error = err.Error()
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}
	ret.Code = 1
	resultData, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Cookie", "token="+ret.Token)
	w.Write(resultData)
	return
}

func Register(w http.ResponseWriter, r *http.Request) {
	auth := NewAuthService(utils.SecretKey, utils.ExpirationHours)
	reqBody, err := io.ReadAll(r.Body)
	ret := new(Result)
	if err != nil {
		ret.Code = -1
		ret.Error = "params error"
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}
	req := &api.RegisterRequest{}
	json.Unmarshal(reqBody, req)
	if req.Account == "" || req.Password == "" {
		ret.Code = -1
		ret.Error = "params error"
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}

	resp, err := auth.Register(r.Context(), req)
	if err != nil {
		ret.Code = -1
		ret.Error = err.Error()
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}
	ret.Code = int(resp.GetStatus())
	resultData, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	w.Write(resultData)
}

func ResetPwd(w http.ResponseWriter, r *http.Request) {
	auth := NewAuthService(utils.SecretKey, utils.ExpirationHours)
	reqBody, err := io.ReadAll(r.Body)
	ret := new(Result)
	if err != nil {
		ret.Code = -1
		ret.Error = "params error"
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}
	req := &api.ResetPasswordRequest{}
	err = json.Unmarshal(reqBody, req)
	if err != nil {
		ret.Code = -1
		ret.Error = "params error"
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}
	if req.Account == "" || req.OldPwd == "" || req.NewPwd == "" {
		ret.Code = -1
		ret.Error = "params error"
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}

	resp, err := auth.ResetPwd(r.Context(), req)
	if err != nil {
		ret.Code = -1
		ret.Error = err.Error()
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}
	ret.Code = int(resp.GetStatus())
	resultData, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	w.Write(resultData)
}

func About(w http.ResponseWriter, r *http.Request) {
	ret := new(Result)
	ret.Code = 1
	resultData, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	w.Write(resultData)
}

type AuthService struct {
	Jwt *jwt.JwtWrapper
}

func NewAuthService(key string, expiration int) *AuthService {
	return &AuthService{
		Jwt: jwt.NewJwtWrapper(key, expiration),
	}
}

func (ts *AuthService) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	info, err := auth.GetAuthService().
		Login(ctx, req.GetAccount(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	token, err := ts.Jwt.GenerateToken(info)
	if err != nil {
		return nil, err
	}
	ret := &api.LoginResponse{
		UserId: info.GetUserId(),
		Token:  token,
	}
	return ret, nil
}
func (ts *AuthService) Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	_, err := auth.GetAuthService().Logout(ctx, req)
	if err != nil {
		return nil, err
	}
	return &api.LogoutResponse{}, nil
}

func (ts *AuthService) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	err := auth.GetAuthService().Register(
		context.Background(),
		req.GetName(),
		req.GetAccount(),
		req.GetPassword(),
	)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (ts *AuthService) ResetPwd(ctx context.Context, req *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error) {
	req.OldPwd = req.GetOldPwd()
	req.NewPwd = req.GetNewPwd()
	_, err := auth.GetAuthService().ResetPassword(ctx, req)
	if err != nil {
		return nil, err
	}
	return &api.ResetPasswordResponse{}, nil
}
