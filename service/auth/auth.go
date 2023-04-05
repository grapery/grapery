package auth

import (
	// "net/http"
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/auth"
	"github.com/grapery/grapery/utils/jwt"
)

func AuthFunc(ctx context.Context) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "token")
	if err != nil {
		return nil, err
	}
	jwtInfo := jwt.JwtWrapper{}
	tokenInfo, err := jwtInfo.ValidateToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}
	grpc_ctxtags.Extract(ctx).Set("auth.sub", jwtInfo.SecretKey)
	newCtx := context.WithValue(ctx, "user_id", tokenInfo.Id)
	return newCtx, nil
}

type AuthService struct {
	Jwt jwt.JwtWrapper
}

func (ts *AuthService) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	info, err := auth.GetAuthService().Login(ctx, req.GetAccount(), req.GetPassword(), req.GetLoginType())
	if err != nil {

		return nil, err
	}
	ret := &api.LoginResponse{UserId: info.GetUserId()}
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
	hashpwd := jwt.HashPassword(req.GetPassword())
	err := auth.GetAuthService().Register(
		context.Background(),
		req.GetAccount(),
		hashpwd,
		req.GetLoginType(),
	)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (ts *AuthService) ResetPwd(ctx context.Context, req *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error) {
	req.OldPwd = jwt.HashPassword(req.GetOldPwd())
	req.NewPwd = jwt.HashPassword(req.GetNewPwd())
	_, err := auth.GetAuthService().ResetPassword(ctx, req)
	if err != nil {
		return nil, err
	}
	return &api.ResetPasswordResponse{}, nil
}
