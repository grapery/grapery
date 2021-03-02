package common

import (
	// "net/http"

	"io/ioutil"
	"net/http"
	"os"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	gin "github.com/gin-gonic/gin"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/grapery/grapery/version"
)

var p *parser.Parser
var r *html.Renderer

func Init() {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p = parser.NewWithExtensions(extensions)
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	r = html.NewRenderer(opts)
	data, err := ioutil.ReadFile("README.md")
	if err != nil {
		panic(err)
	}
	output := markdown.ToHTML(data, p, r)
	fi, err := os.Create("templates/readme.html")
	if err != nil {
		panic(err)
	}
	_, err = fi.Write(output)
	if err != nil {
		panic(err)
	}
}

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
