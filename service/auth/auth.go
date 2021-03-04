package auth

import (
	// "net/http"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/auth"
	"github.com/grapery/grapery/utils"
)

func Register(ctx *utils.Context) {
	req := &api.RegisterRequest{}
	err := ctx.C.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	err = auth.GetAuthService().Register(ctx.C, req.GetAccount(), req.GetPassword(), req.GetLoginType())
	if err != nil {
		ctx.Err = err
		return
	}
}

func Login(ctx *utils.Context) {

}

// just for wechat,not support weibo and QQ
func LoginWithThirdPart(ctx *utils.Context) {

}

func Logout(ctx *utils.Context) {

}

func ResetPassword(ctx *utils.Context) {
}

func CheckSession(ctx *utils.Context) {}
