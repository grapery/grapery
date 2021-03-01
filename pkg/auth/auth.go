package auth

import (
	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
)

var servicer AuthServicer

func init() {
	servicer = NewAuthService()
}

func GetAuthService() AuthServicer {
	return servicer
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

type AuthServicer interface {
	Register(uid uint64) error
	Login(uid uint64) error
	Logout(uid uint64) error
	ResetPassword(uid uint64, newPwd, oldPwd string) error
}

// auth service
type AuthService struct {
}

func (auth *AuthService) Register(uid uint64) error {
	return nil
}

func (auth *AuthService) Login(uid uint64) error {
	return nil
}

func (auth *AuthService) Logout(uid uint64) error {
	return nil
}

func (auth *AuthService) ResetPassword(uid uint64, newPwd, oldPwd string) error {
	return nil
}
