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
		cookie, err := c.Cookie(CookieName)
		if err != nil {

		}
		log.Info("cookie name: ", cookie)
		h(c)
		if c.Keys[ErrKey] != nil {
			c.JSON(http.StatusOK, c.Keys[ErrKey])
			return
		}
		c.JSON(http.StatusOK, c.Keys[RespKey])
	}
}
