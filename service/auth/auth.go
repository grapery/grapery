package auth

import (
	// "net/http"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	ret.Data = api.LoginResponse{UserID: info.UserID}
	infoData, _ := json.Marshal(info)
	cache.SetBytes(c, fmt.Sprintf("%d", info.GetUserID()), infoData, 86400)
	ctx.SetCookie(
		utils.CookieName,
		fmt.Sprintf("%d", info.GetUserID()),
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
	err = auth.GetAuthService().Logout(c, req.GetUserID())
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	cookie, _ := ctx.Cookie(utils.CookieName)
	cache.DelCache(c, cookie)
	ret.Message = "ok"
	ret.Data = api.LoginResponse{}
	return
}

func ResetPassword(ctx *utils.Context) {
	_ = &api.ResetPasswordRequest{}
}
