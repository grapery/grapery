package service

import (
	"fmt"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/config"
	models "github.com/grapery/grapery/models"
	"github.com/grapery/grapery/service/auth"
	"github.com/grapery/grapery/service/common"
	"github.com/grapery/grapery/service/group"
	"github.com/grapery/grapery/service/user"
	cache "github.com/grapery/grapery/utils/redis"
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
		log.Errorf("use redis session failed : %s", err.Error())
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
	app.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	app.Use(gin.Recovery())
	v1Route := app.Group("/api/v1")
	v1Route.POST("/login", auth.Login)
	v1Route.POST("/logout", auth.Logout)
	v1Route.POST("/register", auth.Register)
	v1Route.POST("/reset/pwd", auth.ResetPassword)
	v1Route.GET("/help", common.Help)
	v1Route.GET("/about", common.About)
	v1Route.GET("/version", common.Version)

	userRoute := v1Route.Group("/user")
	{
		userRoute.GET("/:id", user.GetUser)
		userRoute.GET("/:id/info", user.GetUserProfile)
		userRoute.GET("/:id/follower", user.GetUserProfile)
		userRoute.GET("/:id/following", user.GetUserProfile)
		userRoute.GET("/:id/groups", user.GetUserGroup)
		userRoute.GET("/:id/following/group", user.GetUserProfile)
		userRoute.DELETE("/:id", user.DeleteUser)
		userRoute.PUT("/:id", user.UpdateUser)
		userRoute.GET("/:id/active", user.GetUserActive)
	}
	v1Route.GET("/users/search", user.SearchUser)
	groupRoute := v1Route.Group("/group")
	{
		groupRoute.GET("/:id", auth.Login)
	}
	v1Route.GET("/groups/search", group.SearchGroup)
	activeGroup := v1Route.Group("/active")
	{
		activeGroup.GET("/:id", auth.Login)
	}
	eventGroup := v1Route.Group("/event")
	{
		eventGroup.GET("/:id", auth.Login)
	}

	err = app.Run(":" + cfg.Port)
	if err != nil {
		log.Errorf("start server is failed : %s", err.Error())
		return err
	}
	log.Infof("start gin server at port : %s", cfg.Port)
	return nil
}
