package auth

import (
	// "net/http"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	gin "github.com/gin-gonic/gin"
)

func Register(ctx *gin.Context) {

}

func Login(ctx *gin.Context) {

}

// just for wechat,not support weibo and QQ
func LoginWithThirdPart(ctx *gin.Context) {

}

func Logout(ctx *gin.Context) {

}

func ResetPassword(ctx *gin.Context) {
}
