package http

import (
	"context"
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
		Kind     string `json:"kind" binding:"required"`
		ID       string `json:"id" binding:"required"`
		Platform string `json:"platform"`
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

	h.recordShareEvent(c.Request.Context(), domain.ShareEventIssue, kind, id, userID,
		service.NormalizeSharePlatform(req.Platform, service.SharePlatformApp), service.ShareSourceAPIIssue)

	Success(c, issue)
}

// TrackShareOpen POST /api/v1/public/share/open
// Used by iOS Universal Link handlers (and future clients) to report a share open.
func (h *Handler) TrackShareOpen(c *gin.Context) {
	var req struct {
		Kind     string `json:"kind" binding:"required"`
		ID       string `json:"id" binding:"required"`
		Token    string `json:"t"`
		Exp      string `json:"exp"`
		Platform string `json:"platform"`
		Source   string `json:"source"`
	}
	if !BindJSON(c, &req) {
		return
	}

	kind := service.ShareKind(strings.TrimSpace(strings.ToLower(req.Kind)))
	id := strings.TrimSpace(req.ID)
	if id == "" || !validShareKindQuery(kind) {
		InvalidParams(c, "kind and id are required")
		return
	}

	token, exp, hasGrant := service.ParseShareGrantFromQuery(req.Token, req.Exp)
	shareGrant := false
	if hasGrant && h.shareSigner != nil {
		shareGrant = h.shareSigner.Verify(kind, id, token, exp)
	}

	if !h.canPublicPreview(c, kind, id, shareGrant) {
		Forbidden(c, "content is not publicly shareable")
		return
	}

	h.recordShareEvent(c.Request.Context(), domain.ShareEventOpen, kind, id, GetUserID(c),
		service.NormalizeSharePlatform(req.Platform, service.SharePlatformApp),
		service.NormalizeShareSource(req.Source, service.ShareSourceUniversalLink))
	Success(c, gin.H{"ok": true})
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

	// No share-open event here: preview is fetched by link crawlers (WeChat, IM cards)
	// rather than by a human opening the content, and counting it inflates the funnel.
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
		if !h.svc.CanViewerSeeStoryboard(ctx, userID, sb, false) {
			return domain.ErrForbidden
		}
	case service.ShareKindCharacter:
		ch, err := h.svc.GetCharacter(ctx, id)
		if err != nil {
			return err
		}
		if !h.svc.CanViewerSeeCharacter(ctx, userID, ch, false) {
			return domain.ErrForbidden
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
		return h.svc.CanViewerSeeStoryWithGrant(ctx, viewerID, st, shareGrant)
	case service.ShareKindStoryboard:
		sb, err := h.svc.GetStoryboard(ctx, id)
		if err != nil {
			return false
		}
		return h.svc.CanViewerSeeStoryboard(ctx, viewerID, sb, shareGrant)
	case service.ShareKindCharacter:
		ch, err := h.svc.GetCharacter(ctx, id)
		if err != nil {
			return false
		}
		return h.svc.CanViewerSeeCharacter(ctx, viewerID, ch, shareGrant)
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

func (h *Handler) recordShareEvent(
	ctx context.Context,
	eventType domain.ShareEventType,
	kind service.ShareKind,
	contentID, userID, platform, source string,
) {
	if h.svc == nil {
		return
	}
	h.svc.RecordShareEvent(ctx, eventType, kind, contentID, userID, platform, source)
}
