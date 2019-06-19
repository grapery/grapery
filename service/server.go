package service

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/config"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	Stop chan struct{}
}

func NewService() *Service {
	return &Service{
		Stop: make(chan struct{}),
	}
}

func (s *Service) Run(cfg *config.Config) error {
	sessionStore, err := redis.Newstore()
	app := gin.Default()
	v1Route := app.Group("/v1")
	v1Route.POST("/login", nil)
	v1Route.POST("/logout", nil)
	v1Route.POST("/register", nil)
	userRoute := v1Route.Group("/user")
	userRoute.Any("", func(ctx *gin.Context) {
		ctx.Writer.WriteString("not useable")
	})
	groupRoute := v1Route.Group("/group")
	groupRoute.Any("", func(ctx *gin.Context) {
		ctx.Writer.WriteString("not useable")
	})
	activeGroup := v1Route.Group("/active")
	activeGroup.Any("", func(ctx *gin.Context) {
		ctx.Writer.WriteString("not useable")
	})
	projectGroup := v1Route.Group("/project")
	projectGroup.Any("", func(ctx *gin.Context) {
		ctx.Writer.WriteString("not useable")
	})
	eventGroup := v1Route.Group("/event")
	eventGroup.Any("", func(ctx *gin.Context) {
		ctx.Writer.WriteString("not useable")
	})
	v2Route := app.Group("/v2")
	{
		v2Route.Any("", func(ctx *gin.Context) {
			ctx.Writer.WriteString("not useable")
		})
	}
	log.Info("start gin server")
	app.Run(":" + cfg.Port)
	return nil
}
