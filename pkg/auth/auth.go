package auth

import (
	"context"
	"strings"
	"time"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	models "github.com/grapery/grapery/models"
	emailutils "github.com/grapery/grapery/utils/email"
	"github.com/grapery/grapery/utils/errors"
	"github.com/grapery/grapery/utils/jwt"
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

// GetAuthService returns the singleton instance of the AuthServer.
func GetAuthService() AuthServer {
	return server
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService() *AuthService {
	return &AuthService{}
}

// AuthServer defines the interface for authentication operations.
type AuthServer interface {
	// Register creates a new user account.
	Register(ctx context.Context, name string, account string, pwd string) error
	// Login authenticates a user and returns user information upon success.
	Login(ctx context.Context, account string, pwd string) (*api.UserInfo, error)
	// Logout handles user logout.
	// Note: Current implementation is a stub.
	Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error)
	// ResetPassword allows a user to reset their password.
	ResetPassword(ctx context.Context, req *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error)
	// Confirm handles account confirmation, typically via a token.
	// Note: Current implementation is a stub.
	Confirm(ctx context.Context, req *api.ConfirmRequest) (*api.ConfirmResponse, error)
	// GetUserInfo retrieves user information.
	// Note: The 'uid' parameter is currently unused in the implementation.
	GetUserInfo(ctx context.Context, uid int64, account string) (*api.UserInfo, error)
}

// AuthService implements the AuthServer interface.
type AuthService struct {
}

// Register handles new user registration.
// It creates a user, an authentication record, and a user profile.
func (auth *AuthService) Register(ctx context.Context, name string, account string, pwd string) (err error) {
	info := new(models.UserAuth)

	// 使用 bcrypt 对密码进行哈希存储，提升安全性
	hashedPassword := jwt.HashPassword(pwd)
	info.Password = hashedPassword
	if models.IsUserAuthExist(ctx, account) {
		log.Log().WithOptions(logFieldModels).Warn("register failed: account already exists", zap.String("account", account))
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
		log.Log().WithOptions(logFieldModels).Error("create user failed", zap.Error(err), zap.String("account", account))
		return err // Return the error
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
		log.Log().WithOptions(logFieldModels).Error("create auth failed", zap.Error(err), zap.String("account", account))
		return err
	}
	profile := new(models.UserProfile)
	profile.IDBase = models.IDBase{
		Base: models.Base{
			CreateAt: time.Now(),
			UpdateAt: time.Now(),
		},
	}
	profile.UserId = info.UID
	profile.Status = 1
	profile.Background = ""
	profile.NumGroup = 0
	profile.DefaultGroupID = 0
	profile.MinSameGroup = 0
	profile.Limit = 0
	profile.UsedTokens = 0
	profile.CreatedGroupNum = 0
	profile.CreatedStoryNum = 0
	profile.CreatedRoleNum = 0
	profile.CreatedBoardNum = 0
	profile.CreatedGenNum = 0
	profile.WatchingStoryNum = 0
	profile.WatchingGroupNum = 0
	profile.WatchingStoryRoleNum = 0
	err = profile.Create(ctx)
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error("create profile failed", zap.Error(err), zap.String("account", account))
		return err // Return the error
	}
	log.Log().WithOptions(logFieldModels).Info("register success", zap.String("account", account), zap.Int64("uid", info.UID))
	// 注册成功邮件通知（仅当账号为邮箱时发送）
	if strings.Contains(account, "@") {
		subject := "欢迎注册 Grapery"
		body := "您好，" + name + "，欢迎加入 Grapery！您的账号（" + account + "）已成功注册。"
		if err := emailutils.SendSystemEmails([]string{account}, subject, body); err != nil {
			log.Log().WithOptions(logFieldModels).Warn("send register email failed", zap.Error(err), zap.String("to", account))
		}
	}
	return nil
}

// Login handles user authentication.
// It retrieves user auth info by account (email or phone) and verifies the password.
func (auth *AuthService) Login(ctx context.Context, account string, pwd string) (*api.UserInfo, error) {
	info := new(models.UserAuth)
	var err error
	if strings.Contains(account, "@") {
		info, err = models.GetByEmail(ctx, account)
	} else {
		info, err = models.GetByPhone(ctx, account)
	}
	if err != nil {
		log.Log().WithOptions(logFieldModels).Warn("login failed: account not found", zap.String("account", account), zap.Error(err))
		return nil, errors.ErrAuthNotFound
	}
	// 使用 bcrypt 进行安全的密码哈希比对
	if !jwt.CheckPasswordHash(pwd, info.Password) {
		log.Log().WithOptions(logFieldModels).Warn("login failed: wrong password", zap.String("account", account))
		return nil, errors.ErrAuthPasswordIsWrong
	}
	log.Log().WithOptions(logFieldModels).Info("login success", zap.String("account", account), zap.Int64("uid", info.UID))
	return &api.UserInfo{
		UserId: info.UID,
		Email:  info.Email,
	}, nil
}

// Logout handles user logout.
// Currently, this is a stub and does not perform any server-side session invalidation.
func (auth *AuthService) Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	return &api.LogoutResponse{}, nil
}

// ResetPassword allows a user to change their password after verifying the old one.
func (auth *AuthService) ResetPassword(ctx context.Context, req *api.ResetPasswordRequest) (*api.ResetPasswordResponse, error) {
	info := new(models.UserAuth)
	var err error
	if strings.Contains(req.GetAccount(), "@") {
		info, err = models.GetByEmail(ctx, req.GetAccount())
	} else {
		info, err = models.GetByPhone(ctx, req.GetAccount())
	}
	if err != nil {
		log.Log().WithOptions(logFieldModels).Warn("reset password failed: account not found", zap.String("account", req.GetAccount()), zap.Error(err))
		return &api.ResetPasswordResponse{
			Account:   req.GetAccount(),
			Status:    int64(api.ResponseCode_ACCOUNT_NOT_FOUND),
			Timestamp: time.Now().Unix(),
		}, nil
	}

	// 使用 bcrypt 进行安全的密码哈希比对
	if jwt.CheckPasswordHash(req.GetOldPwd(), info.Password) {
		// 新密码也要哈希存储
		hashedPassword := jwt.HashPassword(req.GetNewPwd())
		info.Password = hashedPassword
	} else {
		log.Log().WithOptions(logFieldModels).Warn("reset password failed: wrong old password", zap.String("account", req.GetAccount()))
		return &api.ResetPasswordResponse{
			Account:   req.GetAccount(),
			Status:    int64(api.ResponseCode_WRONG_PASSWORD),
			Timestamp: time.Now().Unix(),
		}, nil
	}
	err = models.UpdatePwd(ctx, info)
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error("reset password failed: update error", zap.String("account", req.GetAccount()), zap.Error(err))
		return &api.ResetPasswordResponse{
			Account:   req.GetAccount(),
			Status:    int64(api.ResponseCode_OPERATION_FAILED), // Indicate failure
			Timestamp: time.Now().Unix(),
		}, err
	}
	log.Log().WithOptions(logFieldModels).Info("reset password success", zap.String("account", req.GetAccount()), zap.Int64("uid", info.UID))
	// 重置密码成功邮件通知（仅当账号为邮箱时发送）
	if strings.Contains(req.GetAccount(), "@") {
		subject := "您的 Grapery 密码已重置"
		body := "您好，您的账号（" + req.GetAccount() + "）密码已于当前时间完成重置。如非本人操作，请尽快联系我们或再次修改密码。"
		if err := emailutils.SendSystemEmails([]string{req.GetAccount()}, subject, body); err != nil {
			log.Log().WithOptions(logFieldModels).Warn("send reset password email failed", zap.Error(err), zap.String("to", req.GetAccount()))
		}
	}
	return &api.ResetPasswordResponse{
		Account:   req.GetAccount(),
		Status:    int64(api.ResponseCode_OK),
		Timestamp: time.Now().Unix(),
	}, nil
}

// Confirm handles account confirmation, typically using a token.
// Currently, this is a stub and needs implementation for token validation and account activation.
func (auth *AuthService) Confirm(ctx context.Context, req *api.ConfirmRequest) (*api.ConfirmResponse, error) {
	if req.GetToken() == "" {
		return nil, errors.ErrTokenIsEmpty
	}
	// TODO: Implement token validation and account confirmation logic.
	// For now, returning a placeholder success response or an error if not implemented.
	return nil, errors.ErrFeatureNotImplemented // Or a specific error
}

// GetUserInfo retrieves user information based on account (email or phone).
// Note: The 'uid' parameter is passed but not currently used in the lookup logic.
func (auth *AuthService) GetUserInfo(ctx context.Context, uid int64, account string) (*api.UserInfo, error) {
	info := new(models.UserAuth)
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
		UserId: info.UID,
		Email:  info.Email,
	}, nil
}
