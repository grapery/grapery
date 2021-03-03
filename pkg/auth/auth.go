package auth

import (
	"context"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
)

//https://blog.gokit.info/post/understand-golang-with-pic/
var server AuthServer

func init() {
	server = NewAuthService()
}

func GetAuthService() AuthServer {
	return server
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

type AuthServer interface {
	Register(ctx context.Context, uid uint64) error
	Login(ctx context.Context, uid uint64) error
	Logout(ctx context.Context, uid uint64) error
	ResetPassword(ctx context.Context, uid uint64, newPwd, oldPwd string) error
}

// auth service
type AuthService struct {
}

func (auth *AuthService) Register(ctx context.Context, uid uint64) error {
	return nil
}

func (auth *AuthService) Login(ctx context.Context, uid uint64) error {
	return nil
}

func (auth *AuthService) Logout(ctx context.Context, uid uint64) error {
	return nil
}

func (auth *AuthService) ResetPassword(ctx context.Context, uid uint64, newPwd, oldPwd string) error {
	return nil
}
