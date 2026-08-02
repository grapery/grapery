package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/jwt"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

// AuthMiddleware JWT鉴权中间件，解析userId并查库
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" || !strings.HasPrefix(tokenStr, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing or invalid token"})
			return
		}
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
		log.Log().Info("AuthMiddleware: tokenStr", zap.String("tokenStr", tokenStr))
		jwtInfo := jwt.NewJwtWrapper(utils.SecretKey, utils.ExpirationHours)
		tokenInfo, err := jwtInfo.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid token"})
			log.Log().Error("AuthMiddleware: invalid token", zap.Error(err))
			return
		}
		userID := tokenInfo.UID
		user := &models.User{}
		user.ID = uint(userID)
		if err := user.GetById(c.Request.Context()); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "user not found"})
			log.Log().Error("AuthMiddleware: user not found", zap.Error(err))
			return
		}
		c.Set(utils.UserIdKey, userID)
		log.Log().Info("AuthMiddleware: userID", zap.Int64("userID", userID))
		c.Set(utils.UserInfoKey, user)
		log.Log().Info("AuthMiddleware: userInfo", zap.Any("userInfo", user))
		newCtx := context.WithValue(c.Request.Context(), utils.UserIdKey, tokenInfo.UID)
		c.Request = c.Request.WithContext(newCtx)
		c.Next()
	}
}

// GetUserIDFromContext 从上下文中获取用户ID
func GetUserIDFromContext(c *gin.Context) uint64 {
	userID, exists := c.Get(utils.UserIdKey)
	if !exists {
		return 0
	}
	return uint64(userID.(int64))
}

// GetUserInfoFromContext 从上下文中获取用户信息
func GetUserInfoFromContext(c *gin.Context) *models.User {
	userInfo, exists := c.Get(utils.UserInfoKey)
	if !exists {
		return nil
	}
	return userInfo.(*models.User)
}
