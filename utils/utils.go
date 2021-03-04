package utils

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/sessions"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

var (
	CookieName   = "grapery"
	Domain       = ""
	CookieMaxAge = 86400
	CookiePath   = ""
)

type Context struct {
	C      *gin.Context
	Ctx    context.Context
	UserID int64
	Err    error
	Resp   interface{}
}

type HandlerFunc func(c *Context)

func WrapHandler(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ret = new(Result)
		cookie, err := c.Cookie(CookieName)
		if err != nil {
			c.Redirect(http.StatusMovedPermanently, "/api/v1/login")
			return
		}
		log.Info("cookie :", cookie)
		infoData := sessions.Default(c).Get(cookie).([]byte)
		var info = new(api.UserInfo)
		err = proto.Unmarshal(infoData, info)
		ctx := &Context{
			C:      c,
			Ctx:    context.Background(),
			UserID: 0,
		}
		if err != nil {
			ctx.Err = err
			ctx.Resp = nil
		} else {
			h(ctx)
		}

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
