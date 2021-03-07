package auth

import (
	// "net/http"
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/auth"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/sessions"
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
	info, err := auth.GetAuthService().Login(context.Background(), req.GetAccount(), req.GetPassword(), req.GetLoginType())
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	ret.Code = 0
	ret.Message = "ok"
	ret.Data = api.LoginResponse{UserID: info.UserID}
	ctx.SetCookie(
		utils.CookieName,
		"",
		utils.CookieMaxAge,
		utils.CookiePath,
		utils.Domain, false, false)
	se := sessions.Default(ctx)
	seData, _ := proto.Marshal(info)
	se.Set(fmt.Sprintf("%d", info.GetUserID()), seData)
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
	err = auth.GetAuthService().Logout(context.Background(), req.GetUserID())
	if err != nil {
		ret.Code = -1
		ret.Message = err.Error()
		ctx.JSON(http.StatusOK, ret)
		return
	}
	se := sessions.Default(ctx)
	se.Delete(fmt.Sprintf("%d", req.GetUserID()))
	ret.Message = "ok"
	ret.Data = api.LoginResponse{}
	return
}

func ResetPassword(ctx *utils.Context) {
	req := %api.ResetPwd()
}
