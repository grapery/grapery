package group

import (
	// "net/http"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/group"
	"github.com/grapery/grapery/utils"
)

func GetGroupActives(ctx *utils.Context) {
	req := &api.GetGroupActivesRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := group.GetGroupServer().GetGroupActives(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}
