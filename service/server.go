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
	common.Init()
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
	app.LoadHTMLGlob("templates/*")
	app.Use(gin.Recovery())
	v1Route := app.Group("/api/v1")
	v1Route.GET("/login", auth.Login)
	v1Route.POST("/login", auth.Login)
	v1Route.POST("/logout", auth.Logout)
	v1Route.POST("/register", auth.Register)
	v1Route.POST("/reset/pwd", auth.ResetPassword)
	v1Route.GET("/help", common.Help)
	v1Route.GET("/about", common.About)
	v1Route.GET("/version", common.Version)

	// user 除了基础的用户的信息和关注列表功能，还有一个默认的group的功能，
	// 用户在自己的空间内创建的都是default的空间
	userRoute := v1Route.Group("/user")
	{
		userRoute.GET("/:id", user.GetUser)
		userRoute.DELETE("/:id", user.DeleteUser)
		userRoute.PUT("/:id", user.UpdateUser)
		userRoute.GET("/:id/info", user.GetUserProfile)
		userRoute.GET("/:id/follower", user.GetUserProfile)
		userRoute.GET("/:id/following", user.GetUserProfile)
		userRoute.GET("/:id/groups", user.GetUserGroup)
		userRoute.GET("/:id/following/groups", user.GetUserProfile)

		userRoute.POST("/:id/follow", user.FollowUser)
		userRoute.PUT("/:id/follow", user.UnFollowUser)
		// 用户个人的active
		userRoute.GET("/:id/actives", user.GetUserActive)
	}
	v1Route.GET("/users/search", user.SearchUser)
	// group 为基础的资源持有结构，project为group中的人员建立的实际包含内容的东西
	// 类似于进程和线程的关系，每个project里面包含例如active的小项，就类似于协程一样的东西
	// 可以让组内的多个人员分别运行（或者说处理）不同的任务
	// 要方便用户参与大量的小组协作，这样多个小组就可以对抗大型组织例如公司或者非法组织
	groupRoute := v1Route.Group("/group")
	{
		groupRoute.GET("", group.CreateGroup)
		groupRoute.POST("", group.CreateGroup)
		groupRoute.GET("/:id", group.GetGroup)
		groupRoute.GET("/:id/actives", group.GetGroupActives)
		groupRoute.PUT("/:id", group.UpdateGroup)
		groupRoute.DELETE("/:id", group.DeleteGroup)
		groupRoute.GET("/:id/members", group.GetGroupMembers)
		groupRoute.POST("/:id/attention", group.AttentionProject)
		groupRoute.PUT("/:id/attention", group.UnAttentionProject)
		thingsGroup := groupRoute.Group("/:id/project")
		{

			thingsGroup.GET("", group.GetGroupProjects)
			thingsGroup.GET("/:project_id", group.GetProject)
			thingsGroup.POST("", group.CreateProject)
			thingsGroup.PUT("/:project_id", group.CreateProject)
			thingsGroup.DELETE("/:project_id", group.DeleteProject)

			thingsGroup.GET("/:project_id/profile", group.GetProject)
			thingsGroup.PUT("/:project_id/profile", group.CreateProject)
			thingsGroup.GET("/:project_id/star", group.StarProject)
			thingsGroup.PUT("/:project_id/star", group.UnStarProject)

			thingsGroup.PUT("/:project_id/watch", group.WatchProject)

			itemGroup := thingsGroup.Group("/:project_id/item")
			{
				itemGroup.GET("/:item_id", group.GetProjectItem)
				itemGroup.POST("/:item_id", group.CreateProjectItem)
				itemGroup.PUT("/:item_id", group.UpdateProjectItem)
				itemGroup.DELETE("/:item_id", group.DeleteProjectItem)
				itemGroup.PUT("/:item_id/like", group.LikeItem)
			}
			thingsGroup.GET("/:project_id/items", group.GetProjectItems)

		}
		groupRoute.GET("/:id/projects/search", group.SearchProject)
	}
	v1Route.GET("/groups/search", group.SearchGroup)

	err = app.Run(":" + cfg.Port)
	if err != nil {
		log.Errorf("start server is failed : %s", err.Error())
		return err
	}
	log.Infof("start gin server at port : %s", cfg.Port)
	return nil
}
