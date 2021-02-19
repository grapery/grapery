package common

import (
	// "net/http"

	"io/ioutil"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	gin "github.com/gin-gonic/gin"
	"github.com/grapery/grapery/version"
)

func About(ctx *gin.Context) {
	data, err := ioutil.ReadFile("README.md")
	if err != nil {
		ctx.String(503, err.Error())
		ctx.Abort()
		return
	}
	ctx.String(200, string(data))
}

func Help(ctx *gin.Context) {
	ctx.String(200, "please press F1")
}

func Version(ctx *gin.Context) {
	ctx.String(200, "version branch %s ,version %s , build time %s",
		version.GitBranch, version.GitHash, version.BuildTS)
}
