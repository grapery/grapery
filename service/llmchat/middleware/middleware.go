package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4" // 需先 go get github.com/golang-jwt/jwt/v4
	"github.com/grapery/grapery/models"
	"github.com/ulule/limiter/v3"
	limiterGin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	memoryStore "github.com/ulule/limiter/v3/drivers/store/memory"
)

var jwtSecret = []byte("your_jwt_secret") // TODO: 替换为实际密钥

// AuthMiddleware JWT鉴权中间件，解析userId并查库
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" || !strings.HasPrefix(tokenStr, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in token"})
			return
		}
		userID := int64(userIDFloat)
		user := &models.User{}
		user.ID = uint(userID)
		if err := user.GetById(); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		c.Set("userID", userID)
		c.Next()
	}
}

// RateLimitMiddleware 用户级限流（每分钟60次）
func RateLimitMiddleware() gin.HandlerFunc {
	// 60 req/min
	rate, _ := limiter.NewRateFromFormatted("60-M")
	store := memoryStore.NewStore()
	middleware := limiterGin.NewMiddleware(limiter.New(store, rate))
	return middleware
}
