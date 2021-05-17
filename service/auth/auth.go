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

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/auth"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/cache"
)

func Register(ctx *gin.Context) {
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

func Confirm(ctx *gin.Context) {
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

func Login(ctx *gin.Context) {
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

func Logout(ctx *gin.Context) {
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

func ResetPassword(ctx *utils.Context) {
	req := &api.ResetPasswordRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	ret := utils.NewResult()
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.GinC.JSON(http.StatusOK, ret)
		return
	}
	_, err = auth.GetAuthService().ResetPassword(ctx.Ctx, req)
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.GinC.JSON(http.StatusOK, ret)
		return
	}
	cookie, _ := ctx.GinC.Cookie(utils.CookieName)
	cache.DelCache(ctx.Ctx, cookie)
	ret.Message = "ok"
	ret.Data = api.LoginResponse{
		UserId: req.GetUserId(),
	}
	return
}
