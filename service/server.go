package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/config"
	models "github.com/grapery/grapery/models"
	"github.com/grapery/grapery/service/auth"
	"github.com/grapery/grapery/service/common"
	"github.com/grapery/grapery/service/group"
	"github.com/grapery/grapery/service/user"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/cache"
)

type Service struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

func NewService() *Service {
	s := &Service{}
	s.Ctx, s.Cancel = context.WithCancel(context.Background())
	return s
}

func (s *Service) Run(cfg *config.Config) error {
	cache.NewRedisClient(cfg)
	err := models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Database)
	if err != nil {
		log.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	common.Init()
	app := gin.Default()
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
	v1Route.GET("/login", auth.Login)
	v1Route.POST("/login", auth.Login)
	v1Route.POST("/logout", auth.Logout)
	v1Route.POST("/register", auth.Register)
	v1Route.POST("/reset/pwd", utils.WrapHandler(auth.ResetPassword))
	v1Route.GET("/help", common.Help)
	v1Route.GET("/about", common.About)
	v1Route.GET("/version", common.Version)
	userRoute := v1Route.Group("/user")
	{
		userRoute.GET("/:id", utils.WrapHandler(user.GetUser))
		userRoute.DELETE("/:id", utils.WrapHandler(user.DeleteUser))
		userRoute.GET("/:id/info", utils.WrapHandler(user.GetUserProfile))
		userRoute.GET("/:id/groups", utils.WrapHandler(user.GetUserGroup))
		// 用户个人的active
		userRoute.GET("/:id/actives", utils.WrapHandler(user.GetUserActive))
		userRoute.GET("/:id/setting", utils.WrapHandler(user.GetUserSetting))
		userRoute.PUT("/:id/setting", utils.WrapHandler(user.UpdateUserSetting))
		//新增加临时会话，可以不用立即添加好友，会话可以设置为多长时间过期，或者会话转为邮件组的形式
	}
	v1Route.GET("/users/search", utils.WrapHandler(user.SearchUser))
	groupRoute := v1Route.Group("/group")
	{
		groupRoute.POST("", utils.WrapHandler(group.CreateGroup))
		groupRoute.GET("/:id", utils.WrapHandler(group.GetGroup))
		groupRoute.GET("/:id/actives", utils.WrapHandler(group.GetGroupActives))
		groupRoute.PUT("/:id", utils.WrapHandler(group.UpdateGroup))
		groupRoute.DELETE("/:id", utils.WrapHandler(group.DeleteGroup))
		groupRoute.GET("/:id/members", utils.WrapHandler(group.GetGroupMembers))
		groupRoute.POST("/:id/join", utils.WrapHandler(group.JoinGroup))
		groupRoute.PUT("/:id/leave", utils.WrapHandler(group.LeaveGroup))
		groupRoute.GET("/:id/projects", utils.WrapHandler(group.GetGroupProjects))
		thingsGroup := groupRoute.Group("/:id/project")
		{
			thingsGroup.GET("/:project_id", utils.WrapHandler(group.GetProject))
			thingsGroup.POST("", utils.WrapHandler(group.CreateProject))
			thingsGroup.PUT("/:project_id", utils.WrapHandler(group.UpdateGroup))
			thingsGroup.DELETE("/:project_id", utils.WrapHandler(group.DeleteProject))
			thingsGroup.GET("/:project_id/profile", utils.WrapHandler(group.GetProject))
			thingsGroup.PUT("/:project_id/profile", utils.WrapHandler(group.CreateProject))

		}
		groupRoute.GET("/:id/projects/search", utils.WrapHandler(group.SearchProject))
	}
	v1Route.GET("/groups/search", group.SearchGroup)
	v1Route.GET("/explore", common.Explore)
	v1Route.GET("/trending", common.Trending)

	err = app.Run(":" + cfg.Port)
	if err != nil {
		log.Errorf("start server is failed : %s", err.Error())
		return err
	}
	log.Infof("start gin server at port : %s", cfg.Port)
	return nil
}
