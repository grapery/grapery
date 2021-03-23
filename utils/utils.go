package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/cache"
)

var (
	EmailRegexp       = regexp.MustCompile(`^(\w+\.?)*\w+@(?:\w+\.)\w+$`)
	PhoneNumberRegexp = regexp.MustCompile(`^[0-9\-\+]+$`)
)

var (
	CookieName   = "grapery"
	Domain       = ""
	CookieMaxAge = 86400
	CookiePath   = ""
)

type Context struct {
	GinC   *gin.Context
	Ctx    context.Context
	UserID uint64
	Err    error
	Resp   interface{}
}

type HandlerFunc func(c *Context)

func WrapHandler(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ret = new(Result)
		Ctx, _ := context.WithCancel(c.Request.Context())
		cookie, err := c.Cookie(CookieName)
		if err != nil {
			ret.Code = http.StatusOK
			ret.Message = "need login"
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		log.Info("cookie : ", cookie)
		cookieKey, err := base64.StdEncoding.DecodeString(cookie)
		if err != nil {
			ret.Code = http.StatusOK
			ret.Message = "need login"
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		// parse cookie
		infoData, err := cache.GetBytes(Ctx, string(cookieKey))
		if err != nil {
			ret.Code = http.StatusOK
			ret.Message = "need login"
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		log.Info(string(infoData))
		ctx := &Context{
			GinC:   c,
			Ctx:    Ctx,
			UserID: 0,
		}
		var info = new(api.UserInfo)
		err = json.Unmarshal(infoData, info)
		if err != nil {
			ctx.Err = err
			ctx.Resp = nil
		} else {
			ctx.UserID = info.GetUserId()
			h(ctx)
		}
		// err handle
		if ctx.Err != nil {
			ret.Code = http.StatusOK
			ret.Message = ctx.Err.Error()
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		ret.Message = "ok"
		ret.Code = http.StatusOK
		ret.Data = ctx.Resp
		c.JSON(http.StatusOK, ret)
	}
}
