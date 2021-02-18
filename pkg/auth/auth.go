package auth

import (
	// "net/http"
	"net/http"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	gin "github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/api"
	models "github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils"
	//cache "github.com/grapery/grapery/pkg/redis"
)

func ParseSession(ctx *gin.Context) {
	sessionID := ctx.Request.Header.Get("session_id")
	if sessionID == "" {
		log.Info("session is empty")
	}
	//cache.RedisCache
}

var AuthSrv = new(AuthService)

// auth service
type AuthService struct {
}

func (auth *AuthService) Register(ctx *gin.Context) {
	var ret = new(utils.Result)
	login := new(api.LoginRequest)
	err := ctx.BindJSON(login)
	if err != nil {
		ret.Message = err.Error()
		ctx.JSON(http.StatusBadRequest, ret)
		return
	}
	var authRecord = new(models.Auth)
	err = authRecord.Create()
	if err != nil {
		log.Errorf("create new user failed : ", err.Error())
		ctx.Abort()
		return
	}
	ret.Code = 0
	ret.Message = "ok"
	ret.Data = api.LoginResponse{UserID: authRecord.UserID}
	ctx.JSON(http.StatusOK, ret)
}

func (auth *AuthService) Login(ctx *gin.Context) {
	sessionID := ctx.Request.Header.Get("session_id")
	if sessionID == "" {
		log.Info("session is empty")
	}
	var uAccount, uPassword string
	uAccount = ctx.Request.FormValue("account")
	uPassword = ctx.Request.FormValue("password")
	//TODO : more detail info for login error
	if uAccount == "" || uPassword == "" {
		log.Errorf("invalied input params")
		ctx.Abort()
	}
	userAuth := &models.Auth{
		Phone:    uAccount,
		Password: uPassword,
	}
	err := userAuth.GetByPhone()
	if err != nil {
		log.Errorf("get user info failed : [%s]", err)
		ctx.Abort()
	}

}

func (auth *AuthService) Logout(ctx *gin.Context) {
	sessionID := ctx.Request.Header.Get("session_id")
	if sessionID == "" {
		log.Errorf("session is empty")
		ctx.Abort()
	}
	var uAccount string
	uAccount = ctx.Request.FormValue("account")
	//TODO : more detail info for login error
	if uAccount == "" {
		log.Errorf("invalied input params")
		ctx.Abort()
	}
	log.Info("session_id ", sessionID)
	ctx.Writer.WriteString(sessionID)
}

func (auth *AuthService) ResetPassword(ctx *gin.Context) {
	sessionID := ctx.Request.Header.Get("session_id")
	if sessionID == "" {
		log.Errorf("session is empty")
		ctx.Abort()
	}
	ctx.Writer.WriteString(sessionID)
}
