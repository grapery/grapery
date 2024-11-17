package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/errors"
	"github.com/grapery/grapery/utils/log"
)

// https://blog.gokit.info/post/understand-golang-with-pic/
var (
	server         AuthServer
	logFieldModels = zap.Fields(
		zap.String("module", "models"))
)

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
	Register(ctx context.Context, name string, account string, pwd string) error
	Login(ctx context.Context, account string, pwd string) (*api.UserInfo, error)
	Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error)
	ResetPassword(ctx context.Context, req *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error)
	Confirm(ctx context.Context, req *api.ConfirmRequest) (*api.ConfirmResponse, error)
	GetUserInfo(ctx context.Context, uid int64, account string) (*api.UserInfo, error)
}

// auth service
type AuthService struct {
}

func (auth *AuthService) Register(ctx context.Context, name string, account string, pwd string) (err error) {
	info := new(models.Auth)

	info.Password = pwd
	if models.IsUserAuthExist(ctx, account) {
		return errors.ErrAuthIsExist
	}

	user := new(models.User)
	user.Name = name
	if strings.Contains(account, "@") {
		user.Email = account
	} else {
		user.Phone = account
	}
	user.CreateAt = time.Now()
	user.UpdateAt = time.Now()
	err = user.Create()
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error("create auth failed", zap.Error(err))
		return nil
	}
	info.UID = int64(user.ID)
	info.CreateAt = time.Now()
	info.UpdateAt = time.Now()
	if strings.Contains(account, "@") {
		info.Email = account
		err = models.CreateWithEmail(ctx, info)
	} else {
		info.Phone = account
		err = models.CreateWithPhone(ctx, info)
	}
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error("create auth failed", zap.Error(err))
		return err
	}
	return
}

func (auth *AuthService) Login(ctx context.Context, account string, pwd string) (*api.UserInfo, error) {
	info := new(models.Auth)
	var err error
	if strings.Contains(account, "@") {
		info, err = models.GetByEmail(ctx, account)
	} else {
		info, err = models.GetByPhone(ctx, account)
	}
	if err != nil {
		return nil, err
	}
	if info.Password != pwd {
		return nil, errors.ErrAuthPasswordIsWrong
	}
	return &api.UserInfo{
		UserId: int64(info.ID),
		Email:  info.Email,
	}, nil
}

func (auth *AuthService) Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	return &api.LogoutResponse{}, nil
}

func (auth *AuthService) ResetPassword(ctx context.Context, req *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error) {
	info := new(models.Auth)
	var err error
	if strings.Contains(req.GetAccount(), "@") {
		info, err = models.GetByEmail(ctx, req.GetAccount())
	} else {
		info, err = models.GetByPhone(ctx, req.GetAccount())

	}
	if err != nil {
		return nil, err
	}

	if info.Password == req.GetOldPwd() {
		info.Password = req.GetNewPwd()
	} else {
		return nil, errors.ErrAuthPasswordIsWrong
	}
	err = models.UpdatePwd(ctx, info)
	if err != nil {
		if err != nil {
			return &api.ResetPasswordResponse{
				Account:   req.GetAccount(),
				Status:    -1,
				Timestamp: time.Now().Unix(),
			}, err
		}
	}
	return &api.ResetPasswordResponse{
		Account:   req.GetAccount(),
		Status:    0,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (auth *AuthService) Confirm(ctx context.Context, req *api.ConfirmRequest) (*api.ConfirmResponse, error) {
	if req.GetToken() == "" {
		return nil, fmt.Errorf("token is empty")
	}

	return nil, nil
}

func (auth *AuthService) GetUserInfo(ctx context.Context, uid int64, account string) (*api.UserInfo, error) {
	info := new(models.Auth)
	var err error
	if strings.Contains(account, "@") {
		info, err = models.GetByEmail(ctx, account)
	} else {
		info, err = models.GetByPhone(ctx, account)
	}
	if err != nil {
		return nil, err
	}
	return &api.UserInfo{
		UserId: int64(info.ID),
		Email:  info.Email,
	}, nil
}
