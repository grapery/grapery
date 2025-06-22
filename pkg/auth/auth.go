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
	info := new(models.Auth)

	// TODO: Password should be hashed before storing.
	// Example: hashedPassword, err := HashPassword(pwd); if err != nil { return err }
	// info.Password = hashedPassword
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
		log.Log().WithOptions(logFieldModels).Error("create user failed", zap.Error(err))
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
		log.Log().WithOptions(logFieldModels).Error("create auth failed", zap.Error(err))
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
	err = profile.Create()
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error("create profile failed", zap.Error(err))
		return err // Return the error
	}
	return nil
}

// Login handles user authentication.
// It retrieves user auth info by account (email or phone) and verifies the password.
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
	// TODO: CRITICAL SECURITY: Passwords must be compared using a secure hash comparison.
	// Example: if !CheckPasswordHash(pwd, info.Password) { return nil, errors.ErrAuthPasswordIsWrong }
	if info.Password != pwd { // This is insecure
		return nil, errors.ErrAuthPasswordIsWrong
	}
	return &api.UserInfo{
		UserId: int64(info.ID),
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

	// TODO: CRITICAL SECURITY: Old password must be compared using a secure hash comparison.
	// Example: if !CheckPasswordHash(req.GetOldPwd(), info.Password) { return nil, errors.ErrAuthPasswordIsWrong }
	if info.Password == req.GetOldPwd() { // This is insecure for comparison
		// TODO: New password should be hashed before storing.
		// Example: hashedPassword, err := HashPassword(req.GetNewPwd()); if err != nil { return appropriate error response }
		// info.Password = hashedPassword
		info.Password = req.GetNewPwd()
	} else {
		return nil, errors.ErrAuthPasswordIsWrong
	}
	err = models.UpdatePwd(ctx, info)
	if err != nil {
		return &api.ResetPasswordResponse{
			Account:   req.GetAccount(),
			Status:    -1, // Indicate failure
			Timestamp: time.Now().Unix(),
		}, err
	}
	return &api.ResetPasswordResponse{
		Account:   req.GetAccount(),
		Status:    0,
		Timestamp: time.Now().Unix(),
	}, nil
}

// Confirm handles account confirmation, typically using a token.
// Currently, this is a stub and needs implementation for token validation and account activation.
func (auth *AuthService) Confirm(ctx context.Context, req *api.ConfirmRequest) (*api.ConfirmResponse, error) {
	if req.GetToken() == "" {
		return nil, fmt.Errorf("token is empty")
	}
	// TODO: Implement token validation and account confirmation logic.
	// For now, returning a placeholder success response or an error if not implemented.
	return nil, fmt.Errorf("confirmation feature not implemented") // Or a specific error
}

// GetUserInfo retrieves user information based on account (email or phone).
// Note: The 'uid' parameter is passed but not currently used in the lookup logic.
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
