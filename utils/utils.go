package utils

import "github.com/gin-gonic/gin"

type Context struct {
	*gin.Context
	UserID    uint64
	GroupID   uint64
	ProjectID uint64
}

func WrapHandler(h gin.HandlerFunc) gin.HandlerFunc {
	return nil
}
