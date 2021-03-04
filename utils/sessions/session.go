package sessions

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/config"
	log "github.com/sirupsen/logrus"
)

var sessionStore redis.Store

func InitSession(cfg *config.Config) {
	var err error
	sessionStore, err = redis.NewStore(10, "tcp", cfg.Redis.Address, cfg.Redis.Password, nil)
	if err != nil {
		log.Errorf("use redis session failed : %s", err.Error())
		return
	}
}

func UseSession(sessionName string) gin.HandlerFunc {
	return sessions.Sessions(sessionName, sessionStore)
}

func Default(ctx *gin.Context) sessions.Session {
	return sessions.Default(ctx)
}
