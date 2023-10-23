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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/pkg/auth"
	"github.com/grapery/grapery/utils/jwt"
)

const (
	GrpcGateWayCookie = "grpcgateway-cookie"
	SecretKey         = "grapery"
	ExpirationHours   = 24 * 7
)

func AuthFunc(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "metadata.FromIncomingContext: %v", codes.FailedPrecondition)
	}
	tokenList := md[GrpcGateWayCookie]

	if len(tokenList) <= 0 {
		return nil, fmt.Errorf("empty auth from md: %s", GrpcGateWayCookie)
	}
	tokenListTemp := strings.Split(tokenList[0], "=")
	token := tokenListTemp[1]
	jwtInfo := jwt.NewJwtWrapper(SecretKey, ExpirationHours)
	tokenInfo, err := jwtInfo.ValidateToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}
	grpc_ctxtags.Extract(ctx).Set("auth.sub", jwtInfo.SecretKey)
	newCtx := context.WithValue(ctx, "user_id", tokenInfo.UID)
	return newCtx, nil
}

type Result struct {
	Code  int    `json:"code,omitempty"`
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

func LoginFunc(w http.ResponseWriter, r *http.Request) {
	auth := NewAuthService(SecretKey, ExpirationHours)
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

func Logout(w http.ResponseWriter, r *http.Request) {
	auth := NewAuthService(SecretKey, ExpirationHours)
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
	auth := NewAuthService(SecretKey, ExpirationHours)
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
	return
}

func ResetPwd(w http.ResponseWriter, r *http.Request) {
	auth := NewAuthService(SecretKey, ExpirationHours)
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
	json.Unmarshal(reqBody, req)
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
	return
}

func About(w http.ResponseWriter, r *http.Request) {
	ret := new(Result)
	ret.Code = 1
	resultData, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	w.Write(resultData)
	return
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
