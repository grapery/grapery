package utils

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/cache"
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
		Ctx := context.Background()
		cookie, err := c.Cookie(CookieName)
		if err != nil {
			c.Redirect(http.StatusMovedPermanently, "/api/v1/login")
			return
		}

		// parse cookie
		infoData, err := cache.GetString(Ctx, cookie)
		if err != nil {
			c.Redirect(http.StatusMovedPermanently, "/api/v1/login")
			return
		}
		ctx := &Context{
			C:      c,
			Ctx:    Ctx,
			UserID: 0,
		}
		var info = new(api.UserInfo)
		err = json.Unmarshal([]byte(infoData), info)

		if err != nil {
			ctx.Err = err
			ctx.Resp = nil
		} else {
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
