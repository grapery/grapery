package common

import (
	// "net/http"

	"net/http"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	gin "github.com/gin-gonic/gin"

	"github.com/grapery/grapery/version"
)

func About(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "readme.html", nil)
}

func Help(ctx *gin.Context) {
	ctx.String(http.StatusOK, "please press F1")
}

func Version(ctx *gin.Context) {
	ctx.String(http.StatusOK, "version branch %s ,version %s , build time %s",
		version.GitBranch, version.GitHash, version.BuildTS)
}

func Explore(ctx *gin.Context) {

}

func Trending(ctx *gin.Context) {

}
