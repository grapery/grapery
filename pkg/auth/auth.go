package auth

import (
	"context"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/errors"
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
	Register(ctx context.Context, account string, pwd string, authType api.AuthType) error
	Login(ctx context.Context, account string, pwd string, authType api.AuthType) (*api.UserInfo, error)
	Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error)
	ResetPassword(ctx context.Context, req *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error)
	Confirm(ctx context.Context, req *api.ConfirmRequest) (*api.ConfirmResponse, error)
}

// auth service
type AuthService struct {
}

func (auth *AuthService) Register(ctx context.Context, account string, pwd string, authType api.AuthType) (err error) {
	info := new(models.Auth)

	info.Password = pwd
	if models.IsUserAuthExist(account) {
		return errors.ErrAuthIsExist
	}
	user := new(models.User)
	if authType == api.AuthType_WithEmail {
		user.Email = account
	} else {
		user.Phone = account
	}
	err = user.Create()
	if err != nil {
		return nil
	}
	info.UID = uint64(user.ID)
	if authType == api.AuthType_WithEmail {
		info.AuthType = api.AuthType_WithEmail
		info.Email = account
		err = info.CreateWithEmail()
	} else if authType == api.AuthType_WithPhone {
		info.AuthType = api.AuthType_WithPhone
		info.Phone = account
		err = info.CreateWithPhone()
	}
	if err != nil {
		return err
	}
	return
}

func (auth *AuthService) Login(ctx context.Context, account string, pwd string, authType api.AuthType) (*api.UserInfo, error) {
	info := new(models.Auth)
	var err error
	if authType == api.AuthType_WithEmail {
		info.Email = account
		info.Password = pwd
		info.AuthType = authType
		err = info.GetByEmail()
	} else if authType == api.AuthType_WithPhone {
		info.Phone = account
		info.Password = pwd
		info.AuthType = authType
		err = info.GetByPhone()
	}
	if err != nil {
		return nil, err
	}
	return &api.UserInfo{
		UserId: uint64(info.ID),
	}, nil
}

func (auth *AuthService) Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	log.Printf("user %d is logout", req.GetUserId())
	return &api.LogoutResponse{}, nil
}

func (auth *AuthService) ResetPassword(ctx context.Context, req *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error) {
	info := new(models.Auth)
	info.ID = uint(req.GetUserId())
	err := info.GetByUID()
	if err != nil {
		return nil, err
	}
	if info.Password == req.GetOldPwd() {
		info.Password = req.GetNewPwd()
	} else {
		return nil, errors.ErrAuthPasswordIsWrong
	}
	err = info.UpdatePwd()
	if err != nil {
		return nil, err
	}
	return &api.ResetPasswordResponse{
		UserId: req.GetUserId(),
	}, nil
}

func (auth *AuthService) Confirm(ctx context.Context, req *api.ConfirmRequest) (*api.ConfirmResponse, error) {
	return nil, nil
}
