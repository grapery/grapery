package http

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSConfig configures CORS middleware with environment-aware settings
func CORSConfig() gin.HandlerFunc {
	// Get allowed origins from environment variable
	// Format: comma-separated list of origins, e.g., "https://example.com,https://app.example.com"
	allowOriginsEnv := os.Getenv("CORS_ALLOW_ORIGINS")

	var allowOrigins []string
	if allowOriginsEnv != "" {
		// Parse comma-separated origins
		allowOrigins = strings.Split(allowOriginsEnv, ",")
		for i, origin := range allowOrigins {
			allowOrigins[i] = strings.TrimSpace(origin)
		}
	}

	// Check if we're in development mode
	isDevelopment := os.Getenv("GRAPERY_ENV") == "development" || os.Getenv("GIN_MODE") == "debug"

	// If no origins configured and not in development, use restrictive defaults
	if len(allowOrigins) == 0 {
		if isDevelopment {
			// Development: allow common local development origins
			allowOrigins = []string{
				"http://localhost:3000",
				"http://localhost:5173",
				"http://localhost:8080",
				"http://127.0.0.1:3000",
				"http://127.0.0.1:5173",
				"http://127.0.0.1:8080",
			}
		} else {
			// Production: no default - must be configured via CORS_ALLOW_ORIGINS
			allowOrigins = []string{}
		}
	}

	config := cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// In development, use AllowOriginFunc for flexibility
	// In production, use strict AllowOrigins list
	if isDevelopment && len(allowOrigins) > 0 {
		config.AllowOriginFunc = func(origin string) bool {
			// In development, allow any localhost/127.0.0.1 origin
			if strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:") {
				return true
			}
			// Also check against configured origins
			for _, allowed := range allowOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		}
	}

	return cors.New(config)
}
