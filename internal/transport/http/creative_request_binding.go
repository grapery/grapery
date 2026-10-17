package http

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func bindCreativeRequest(c *gin.Context, request any) error {
	if err := c.ShouldBindBodyWith(request, binding.JSON); err != nil {
		return err
	}
	if raw, ok := c.Get(gin.BodyBytesKey); ok {
		if body, ok := raw.([]byte); ok {
			return domain.ValidateCreativeControlsEnvelope(body)
		}
	}
	return nil
}
