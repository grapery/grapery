package auth

import (
	gin "github.com/gin-gonic/gin"
	models "github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

func ParseSession(ctx *gin.Context) {
	sessionID := ctx.Request.Header.Get("session_id")
	if sessionID == "" {
		log.Info("session is empty")
	}
}

const (
	RegisterWithPhone    = "phone"
	RegisterWithEmail    = "email"
	RegisterWithNickname = "nickname"
)

// auth service
type AuthService struct {
}

func (auth *AuthService) Register(ctx *gin.Context) {
	sessionID := ctx.Request.Header.Get("session_id")
	if sessionID == "" {
		log.Info("session is empty")
	}
	registerType := ctx.Request.FormValue("account_type")
	var uAccount, uPassword string
	if registerType == RegisterWithPhone {
		// TODO :
		log.Infof("register type : [%s]", registerType)
	} else if registerType == RegisterWithEmail {
		log.Infof("register type : [%s]", registerType)
	} else if registerType == RegisterWithNickname {
		log.Infof("register type : [%s]", registerType)
	} else {
		log.Errorf("error register type")
		ctx.Abort()
	}
	uAccount = ctx.Request.FormValue("account")
	uPassword = ctx.Request.FormValue("password")
	if uAccount == "" || uPassword == "" {
		log.Errorf("invalied input params")
		ctx.Abort()
	}
	authRecord := &models.Auth{
		Email:    uAccount,
		Password: uPassword,
		AuthType: registerType,
	}
	err := authRecord.Create()
	if err != nil {
		log.Errorf("create new user failed : ", err.Error())
		ctx.Abort()
	}
	log.Infof("user [%s] register success ", uAccount)
	ctx.Redirect(http.StatusPermanentRedirect, "/v1/login")
}

func (auth *AuthService) Login(ctx *gin.Context) {
	sessionID := ctx.Request.Header.Get("session_id")
	if sessionID == "" {
		log.Info("session is empty")
	}
	uAccount = ctx.Request.FormValue("account")
	uPassword = ctx.Request.FormValue("password")
	//TODO : more detail info for login error
	if uAccount == "" || uPassword == "" {
		log.Errorf("invalied input params")
		ctx.Abort()
	}
	userAuth :=&models.Auth(
		Email:    uAccount, 
		Password: uPassword,
	)
	err :=userAuth.Get()
}

func (auth *AuthService) Logout(ctx *gin.Context) {

}

func (auth *AuthService) ResetPassword(ctx *gin.Context) {

}
