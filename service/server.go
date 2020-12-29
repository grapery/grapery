package service

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/config"
	models "github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/auth"
	cache "github.com/grapery/grapery/utils/redis"
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
	sessionStore, err := redis.NewStore(10, "tcp", "localhost:6379", "", nil)
	if err != nil {
		log.Errorf("use redis session failed : ", err.Error())
		return err
	}
	cache.RedisCache = cache.NewRedisClient(cfg)
	err = models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Database)
	if err != nil {
		log.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	app := gin.Default()
	app.Use(sessions.Sessions("grapestree", sessionStore))
	v1Route := app.Group("/v1")
	// login about and something
	v1Route.POST("/login", auth.AuthSrv.Login)
	v1Route.POST("/logout", auth.AuthSrv.Logout)
	v1Route.POST("/register", auth.AuthSrv.Register)
	v1Route.POST("/reset/pwd", auth.AuthSrv.ResetPassword)
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
	err = app.Run(":" + cfg.Port)
	if err != nil {
		log.Errorf("start server is failed : %s", err.Error())
		return err
	}
	log.Infof("start gin server at port : %s", cfg.Port)
	return nil
}
