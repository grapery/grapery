package http

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// IssueShareLink POST /api/v1/share/issue
func (h *Handler) IssueShareLink(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if h.shareSigner == nil || !h.shareSigner.IsConfigured() {
		InternalError(c, "share signing is not configured")
		return
	}

	var req struct {
		Kind string `json:"kind" binding:"required"`
		ID   string `json:"id" binding:"required"`
	}
	if !BindJSON(c, &req) {
		return
	}

	kind := service.ShareKind(strings.TrimSpace(strings.ToLower(req.Kind)))
	id := strings.TrimSpace(req.ID)
	if id == "" {
		InvalidParams(c, "id is required")
		return
	}

	if err := h.ensureShareIssuerCanAccess(c, userID, kind, id); err != nil {
		HandleError(c, err)
		return
	}

	issue, err := h.shareSigner.Issue(kind, id, 0)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, issue)
}

// GetPublicSharePreview GET /api/v1/public/share/preview?kind=fragment&id=...&t=...&exp=...
func (h *Handler) GetPublicSharePreview(c *gin.Context) {
	kind := service.ShareKind(strings.TrimSpace(strings.ToLower(c.Query("kind"))))
	id := strings.TrimSpace(c.Query("id"))
	if id == "" || !validShareKindQuery(kind) {
		InvalidParams(c, "kind and id are required")
		return
	}

	token, exp, hasGrant := service.ParseShareGrantFromQuery(c.Query("t"), c.Query("exp"))
	shareGrant := false
	if hasGrant && h.shareSigner != nil {
		shareGrant = h.shareSigner.Verify(kind, id, token, exp)
	}

	preview, err := h.svc.BuildSharePreview(c.Request.Context(), kind, id)
	if err != nil {
		HandleError(c, err)
		return
	}

	if !h.canPublicPreview(c, kind, id, shareGrant) {
		Forbidden(c, "content is not publicly shareable")
		return
	}

	Success(c, preview)
}

func (h *Handler) ensureShareIssuerCanAccess(c *gin.Context, userID string, kind service.ShareKind, id string) error {
	ctx := c.Request.Context()
	switch kind {
	case service.ShareKindFragment:
		f, err := h.svc.GetFragmentByID(ctx, id)
		if err != nil {
			return domain.ErrNotFound
		}
		if !h.svc.CanViewerSeeFragment(ctx, userID, f, false) {
			return domain.ErrForbidden
		}
	case service.ShareKindStory:
		st, err := h.svc.GetStory(ctx, id)
		if err != nil {
			return err
		}
		if !h.svc.CanViewerSeeStory(ctx, userID, st) {
			return domain.ErrForbidden
		}
	case service.ShareKindStoryboard:
		sb, err := h.svc.GetStoryboard(ctx, id)
		if err != nil {
			return err
		}
		if sb.StoryID != "" {
			st, err := h.svc.GetStory(ctx, sb.StoryID)
			if err != nil {
				return err
			}
			if !h.svc.CanViewerSeeStory(ctx, userID, st) {
				return domain.ErrForbidden
			}
		}
	case service.ShareKindCharacter:
		if _, err := h.svc.GetCharacter(ctx, id); err != nil {
			return err
		}
	default:
		return domain.ErrNotFound
	}
	return nil
}

func (h *Handler) canPublicPreview(c *gin.Context, kind service.ShareKind, id string, shareGrant bool) bool {
	ctx := c.Request.Context()
	viewerID := GetUserID(c)
	switch kind {
	case service.ShareKindFragment:
		f, err := h.svc.GetFragmentByID(ctx, id)
		if err != nil {
			return false
		}
		return h.svc.CanViewerSeeFragment(ctx, viewerID, f, shareGrant)
	case service.ShareKindStory:
		st, err := h.svc.GetStory(ctx, id)
		if err != nil {
			return false
		}
		if shareGrant {
			return true
		}
		return h.svc.CanViewerSeeStory(ctx, viewerID, st)
	case service.ShareKindStoryboard:
		sb, err := h.svc.GetStoryboard(ctx, id)
		if err != nil {
			return false
		}
		if shareGrant {
			return true
		}
		if sb.StoryID == "" {
			return true
		}
		st, err := h.svc.GetStory(ctx, sb.StoryID)
		if err != nil {
			return false
		}
		return h.svc.CanViewerSeeStory(ctx, viewerID, st)
	case service.ShareKindCharacter:
		return true
	default:
		return false
	}
}

func validShareKindQuery(kind service.ShareKind) bool {
	switch kind {
	case service.ShareKindFragment, service.ShareKindStoryboard, service.ShareKindStory, service.ShareKindCharacter:
		return true
	default:
		return false
	}
}

// ShareGrantFromRequest returns whether the request carries a valid share token for kind/id.
func (h *Handler) ShareGrantFromRequest(c *gin.Context, kind service.ShareKind, id string) bool {
	if h.shareSigner == nil {
		return false
	}
	token, exp, ok := service.ParseShareGrantFromQuery(c.Query("t"), c.Query("exp"))
	if !ok {
		return false
	}
	return h.shareSigner.Verify(kind, id, token, exp)
}
