package auth

import (
	// "net/http"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/auth"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/cache"
)

func parseToken(token string) (struct{}, error) {
	return struct{}{}, nil
}

func userClaimFromToken(struct{}) string {
	return "foobar"
}

// exampleAuthFunc is used by a middleware to authenticate requests
func ExampleAuthFunc(ctx context.Context) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}

	tokenInfo, err := parseToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	grpc_ctxtags.Extract(ctx).Set("auth.sub", userClaimFromToken(tokenInfo))

	// WARNING: in production define your own type to avoid context collisions
	newCtx := context.WithValue(ctx, "tokenInfo", tokenInfo)

	return newCtx, nil
}

type AuthService struct {
}

func (a *AuthService) Register(ctx *gin.Context) {
	req := &api.RegisterRequest{}
	err := ctx.ShouldBindJSON(req)
	ret := utils.NewResult()
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	err = auth.GetAuthService().Register(
		context.Background(),
		req.GetAccount(),
		req.GetPassword(),
		req.GetLoginType(),
	)
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	ret.Code = 0
	ret.Message = "ok"
	ret.Data = &api.RegisterResponse{}
	ctx.JSON(http.StatusOK, ret)
}

func (a *AuthService) Confirm(ctx *gin.Context) {
	req := &api.RegisterRequest{}
	err := ctx.ShouldBindJSON(req)
	ret := utils.NewResult()
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	err = auth.GetAuthService().Register(context.Background(), req.GetAccount(), req.GetPassword(), req.GetLoginType())
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	ret.Code = 0
	ret.Message = "ok"
	ret.Data = &api.RegisterResponse{}
	ctx.JSON(http.StatusOK, ret)
}

func (a *AuthService) Login(ctx *gin.Context) {
	req := &api.LoginRequest{}
	err := ctx.ShouldBindJSON(req)
	ret := utils.NewResult()
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	c := ctx.Request.Context()
	info, err := auth.GetAuthService().Login(c, req.GetAccount(), req.GetPassword(), req.GetLoginType())
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	ret.Code = 0
	ret.Message = "ok"
	ret.Data = api.LoginResponse{UserId: info.GetUserId()}
	infoData, _ := json.Marshal(info)

	cookieKey := fmt.Sprintf("grapery_%d_%d", info.GetUserId(), time.Now().Unix())
	err = cache.SetBytes(c, cookieKey, infoData, 86400)
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ret.Data = nil
		ctx.JSON(http.StatusOK, ret)
		return
	}
	b64Info := base64.StdEncoding.EncodeToString([]byte(cookieKey))
	ctx.SetCookie(
		utils.CookieName,
		b64Info,
		utils.CookieMaxAge,
		utils.CookiePath,
		utils.Domain, false, false)

	ctx.JSON(http.StatusOK, ret)
}

func (a *AuthService) Logout(ctx *gin.Context) {
	req := &api.LogoutRequest{}
	err := ctx.ShouldBindJSON(req)
	ret := utils.NewResult()
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	c := context.Background()
	_, err = auth.GetAuthService().Logout(c, req)
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	cookie, _ := ctx.Cookie(utils.CookieName)
	_ = cache.DelCache(c, cookie)
	ret.Message = "ok"
	ret.Data = api.LoginResponse{}
}

func (a *AuthService) ResetPassword(ctx *utils.Context) {
	req := &api.ResetPasswordRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	ret := utils.NewResult()
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.GinC.JSON(http.StatusOK, ret)
		return
	}
	resp, err := auth.GetAuthService().ResetPassword(ctx.Ctx, req)
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.GinC.JSON(http.StatusOK, ret)
		return
	}
	cookie, _ := ctx.GinC.Cookie(utils.CookieName)
	cache.DelCache(ctx.Ctx, cookie)
	ret.Message = "ok"
	ret.Data = api.ResetPasswordResponse{
		Account: req.GetAccount(),
		Status:  resp.GetStatus(),
	}
	return
}
