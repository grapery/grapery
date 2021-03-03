package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	CookieName = "grapery"

	UserKey    = "_userid"
	GroupKey   = "_groupid"
	ProjectKey = "_projectid"
	ErrKey     = "_err"
	RespKey    = "_resp"
)

func WrapHandler(h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ret = new(Result)
		cookie, err := c.Cookie(CookieName)
		if err != nil {
			c.Redirect(http.StatusMovedPermanently, "/api/v1/login")
			return
		}
		log.Info("cookie val: ", cookie)
		h(c)
		if c.Keys[ErrKey] != nil {
			ret.Code = http.StatusOK
			ret.Message = c.Keys[ErrKey].(error).Error()
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		ret.Message = "ok"
		ret.Code = http.StatusOK
		ret.Data = c.Keys[RespKey]
		c.JSON(http.StatusOK, ret)
	}
}
