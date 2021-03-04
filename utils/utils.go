package utils

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
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
		ctx := &Context{
			C:      c,
			Ctx:    context.Background(),
			UserID: 0,
		}
		h(ctx)
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
