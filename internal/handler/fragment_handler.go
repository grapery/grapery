package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
	coreservice "github.com/grapestree/fgrapery/grapery/internal/service"
)

type FragmentHandler struct {
	fragmentRepo     *repository.FragmentRepository
	userSettingsRepo domain.UserSettingsRepository
	repo             domain.Repository
	comicStyleSvc    *coreservice.FragmentComicStyleService
}

func NewFragmentHandler(fragmentRepo *repository.FragmentRepository, userSettingsRepo domain.UserSettingsRepository, repo domain.Repository, comicStyleSvc *coreservice.FragmentComicStyleService) *FragmentHandler {
	return &FragmentHandler{
		fragmentRepo:     fragmentRepo,
		userSettingsRepo: userSettingsRepo,
		repo:             repo,
		comicStyleSvc:    comicStyleSvc,
	}
}

// FragmentStyle represents a fragment image style
type FragmentStyle struct {
	ID       string  `json:"id"`
	Value    string  `json:"value"`
	Name     string  `json:"name"`
	Icon     string  `json:"icon"`
	Category *string `json:"category,omitempty"`
}

// FragmentStyleListResponse represents the response for styles list
type FragmentStyleListResponse struct {
	Styles []FragmentStyle `json:"styles"`
}

// GetFragmentStyles handles GET /fragments/styles
func (h *FragmentHandler) GetFragmentStyles(c *gin.Context) {
	// TODO: In the future, these could be loaded from database or configuration
	// For now, return hardcoded styles
	styles := []FragmentStyle{
		{ID: "style_001", Value: "photorealistic", Name: "超写实", Icon: "camera.metering.matrix", Category: stringPtr("photography")},
		{ID: "style_002", Value: "cinematic", Name: "电影感", Icon: "film", Category: stringPtr("cinematic")},
		{ID: "style_003", Value: "documentary", Name: "纪实摄影", Icon: "camera.viewfinder", Category: stringPtr("photography")},
		{ID: "style_004", Value: "anime", Name: "日系动漫", Icon: "sparkles", Category: stringPtr("illustration")},
		{ID: "style_005", Value: "cel_shading", Name: "赛璐璐", Icon: "paintbrush.pointed", Category: stringPtr("illustration")},
		{ID: "style_006", Value: "chibi", Name: "Q版", Icon: "face.smiling", Category: stringPtr("illustration")},
		{ID: "style_007", Value: "manga", Name: "漫画", Icon: "book.closed", Category: stringPtr("illustration")},
		{ID: "style_008", Value: "digital_painting", Name: "数字绘画", Icon: "paintbrush", Category: stringPtr("digital")},
		{ID: "style_009", Value: "concept_art", Name: "概念艺术", Icon: "lightbulb", Category: stringPtr("digital")},
		{ID: "style_010", Value: "3d_render", Name: "3D渲染", Icon: "cube", Category: stringPtr("3d")},
		{ID: "style_011", Value: "synthetic_impressionism", Name: "合成印象派", Icon: "scribble", Category: stringPtr("artistic")},
		{ID: "style_012", Value: "cyberpunk", Name: "赛博朋克", Icon: "brain.head.profile", Category: stringPtr("scifi")},
		{ID: "style_013", Value: "eco_futurism", Name: "生态未来主义", Icon: "leaf.fill", Category: stringPtr("scifi")},
		{ID: "style_014", Value: "quantum_expressionism", Name: "量子表现主义", Icon: "atom", Category: stringPtr("abstract")},
		{ID: "style_015", Value: "sci_fi_architecture", Name: "科幻建筑", Icon: "building.2", Category: stringPtr("architectural")},
		{ID: "style_016", Value: "digital_renaissance", Name: "数字文艺复兴", Icon: "building.columns", Category: stringPtr("artistic")},
		{ID: "style_017", Value: "neo_classical", Name: "新古典主义", Icon: "building", Category: stringPtr("architectural")},
		{ID: "style_018", Value: "pop_surrealism", Name: "流行超现实主义", Icon: "theatermasks", Category: stringPtr("surreal")},
		{ID: "style_019", Value: "abstract_cinematic", Name: "抽象电影叙事", Icon: "circle.circle", Category: stringPtr("cinematic")},
		{ID: "style_020", Value: "ink_punk", Name: "水墨朋克", Icon: "drop", Category: stringPtr("artistic")},
		{ID: "style_021", Value: "product_shot", Name: "产品静物", Icon: "photo.badge.plus", Category: stringPtr("commercial")},
		{ID: "style_022", Value: "brand_style", Name: "品牌视觉", Icon: "tag", Category: stringPtr("commercial")},
		{ID: "style_023", Value: "lifestyle_mockup", Name: "生活场景植入", Icon: "house", Category: stringPtr("commercial")},
		{ID: "style_024", Value: "augmented_surrealism", Name: "增强超现实", Icon: "eyebrow", Category: stringPtr("surreal")},
		{ID: "style_025", Value: "neural_abstract", Name: "神经抽象", Icon: "network", Category: stringPtr("abstract")},
		{ID: "style_026", Value: "vaporwave", Name: "蒸汽波美学", Icon: "waveform.path", Category: stringPtr("aesthetic")},
	}

	c.JSON(http.StatusOK, FragmentStyleListResponse{
		Styles: styles,
	})
}

// PostFragmentStylesNext handles POST /fragments/styles/next.
// Query allow_ai=false: first 8 rows by id ASC (cheap default catalog).
// allow_ai=true: best-effort insert one AI row, then return newest 8 by id DESC (fresh styles visible); cold empty DB still AI-fills.
func (h *FragmentHandler) PostFragmentStylesNext(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.comicStyleSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "comic styles service unavailable"})
		return
	}
	allowAI := c.DefaultQuery("allow_ai", "true") != "false"
	var items []coreservice.FragmentComicStyleItem
	var err error
	if allowAI {
		items, err = h.comicStyleSvc.NextBatch(c.Request.Context(), userID)
	} else {
		items, err = h.comicStyleSvc.NextBatchDBOnly(c.Request.Context(), userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]FragmentStyle, 0, len(items))
	for _, it := range items {
		out = append(out, FragmentStyle{
			ID:       it.ID,
			Value:    it.Value,
			Name:     it.Name,
			Icon:     it.Icon,
			Category: it.Category,
		})
	}
	c.JSON(http.StatusOK, FragmentStyleListResponse{Styles: out})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// stringPtrValue returns empty string if nil
func stringPtrValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// CreateFragmentRequest represents the request to create a fragment
type CreateFragmentRequest struct {
	Content       string   `json:"content" binding:"required,max=500"`
	ImageUrls     []string `json:"imageUrls" binding:"omitempty,max=10"`
	Style         *string  `json:"style" binding:"omitempty"`
	FragmentCount *int     `json:"fragmentCount" binding:"omitempty,min=1,max=16"`
	Visibility    string   `json:"visibility" binding:"required,oneof=public followers followers_only private"`
	Topic         *string  `json:"topic" binding:"omitempty,max=200"`
	Caption       *string  `json:"caption" binding:"omitempty,max=500"`
	AspectRatio   string   `json:"aspectRatio" binding:"omitempty,oneof=1:1 16:9 9:16 3:4 4:3"`
}

// UpdateFragmentRequest represents the request to update a fragment
type UpdateFragmentRequest struct {
	Content       *string   `json:"content" binding:"omitempty,max=500"`
	ImageUrls     *[]string `json:"imageUrls" binding:"omitempty,max=10"`
	Style         *string   `json:"style" binding:"omitempty"`
	FragmentCount *int      `json:"fragmentCount" binding:"omitempty,min=1,max=16"`
	Topic         *string   `json:"topic" binding:"omitempty,max=200"`
	Caption       *string   `json:"caption" binding:"omitempty,max=500"`
	Visibility    *string   `json:"visibility" binding:"omitempty,oneof=public followers followers_only private"`
	IsDraft       *bool     `json:"isDraft,omitempty"`
	AspectRatio   *string   `json:"aspectRatio" binding:"omitempty,oneof=1:1 16:9 9:16 3:4 4:3"`
}

// CreateFragment handles POST /fragments
func (h *FragmentHandler) CreateFragment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateFragmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate visibility
	if !domain.ValidFragmentVisibility(req.Visibility) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visibility"})
		return
	}
	normalizedVisibility := domain.NormalizeFragmentVisibility(req.Visibility)

	// Set default fragment count to 1 if not provided
	fragmentCount := 1
	if req.FragmentCount != nil {
		fragmentCount = *req.FragmentCount
	}

	// Process image URLs - validate and normalize
	// Images should already be uploaded via a separate upload endpoint
	// Here we just validate that they are valid URLs
	processedImageUrls := make([]string, 0, len(req.ImageUrls))
	for _, url := range req.ImageUrls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		// Basic URL validation - must start with http:// or https://
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			processedImageUrls = append(processedImageUrls, url)
		}
		// Note: Base64 images should be uploaded via a separate endpoint first
	}

	fragment := &domain.Fragment{
		BaseModel: common.BaseModel{
			ID:        generateUUID(),
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		},
		EngagementStats: common.EngagementStats{
			Likes:    0,
			Comments: 0,
			Shares:   0,
			Views:    0,
		},
		UserID:        userID,
		CreatorID:     userID, // 兼容旧代码
		Content:       req.Content,
		ImageUrls:     stringifyArray(processedImageUrls),
		Style:         req.Style,
		FragmentCount: &fragmentCount,
		Visibility:    normalizedVisibility,
		Topic:         strings.TrimSpace(stringPtrValue(req.Topic)),
		Caption:       strings.TrimSpace(stringPtrValue(req.Caption)),
		SourceType:    string(domain.FragmentSourceOriginal), // 用户手动创建的碎片为原创
		SourceID:      "",                                    // 原创碎片无来源ID
	}
	if ar := domain.NormalizeFragmentAspectRatio(strings.TrimSpace(req.AspectRatio)); ar != "" {
		fragment.AspectRatio = ar
	}

	if err := h.fragmentRepo.Create(c.Request.Context(), fragment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create fragment"})
		return
	}

	c.JSON(http.StatusCreated, fragment)
}

// GetFragment handles GET /fragments/:id
func (h *FragmentHandler) GetFragment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment id is required"})
		return
	}

	fragment, err := h.fragmentRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fragment not found"})
		return
	}

	userID := c.GetString("userID")
	if stats, statsErr := h.fragmentRepo.GetEngagementStats(c.Request.Context(), id, userID); statsErr == nil {
		fragment.Likes = stats.Likes
		fragment.Comments = stats.Comments
		liked := stats.IsLiked
		fragment.IsLiked = &liked
	}

	// Increment view count (fire-and-forget with panic recovery, context.Background to avoid cancel)
	go func(fragmentID string) {
		defer func() { _ = recover() }()
		h.fragmentRepo.IncrementViews(context.Background(), fragmentID)
	}(id)
	if userID != "" {
		go func(uid, fid string) {
			defer func() { _ = recover() }()
			h.fragmentRepo.RecordFragmentForYouSeen(context.Background(), uid, fid)
		}(userID, id)
	}

	c.JSON(http.StatusOK, fragment)
}

// ListFragments handles GET /fragments
//
// Query (topic mode — when topic is non-empty, tab is ignored):
//   - topic: exact topic label to list (same as stored on fragments, no leading #)
//   - converted / convertedOnly: optional filter — "true" (hatched to story only),
//     "false" (not hatched only), empty or other (all). Aliases are equivalent.
//   - public_feed=1: when tab is discover / for_you / recommended (and no topic), return all public non-draft
//     fragments by time — same catalog as GET /plaza embedded previews. Ignores onboarding genre filter so
//     plaza “查看全部” matches the rail.
//   - page, limit, offset: pagination (offset overrides page-derived offset when valid)
func (h *FragmentHandler) ListFragments(c *gin.Context) {
	userID := c.GetString("userID")

	// Parse query parameters
	tab := strings.TrimSpace(strings.ToLower(c.DefaultQuery("tab", "discover"))) // discover (default), for_you / recommended (alias), following
	if tab == "" {
		tab = "discover"
	}
	publicFeed := false
	switch strings.ToLower(strings.TrimSpace(c.Query("public_feed"))) {
	case "1", "true", "yes":
		publicFeed = true
	}
	topic := c.Query("topic") // optional: filter by topic
	converted := c.Query("converted")
	if converted == "" {
		converted = c.Query("convertedOnly") // Voyager / OpenAPI-friendly alias
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if rawOffset := c.Query("offset"); rawOffset != "" {
		if parsedOffset, err := strconv.Atoi(rawOffset); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
			page = (offset / limit) + 1
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
	} else {
		offset = (page - 1) * limit
	}

	if limit > 100 {
		limit = 100
	}

	var fragments []*domain.Fragment
	var total int64
	var err error

	// topic takes precedence: when topic is set, list by topic
	if topic != "" {
		var convertedOnly *bool
		switch converted {
		case "true":
			b := true
			convertedOnly = &b
		case "false":
			b := false
			convertedOnly = &b
		}
		fragments, total, err = h.fragmentRepo.ListByTopic(c.Request.Context(), topic, limit, offset, convertedOnly)
	} else {
		switch tab {
		case "following":
			if userID == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			fragments, total, err = h.fragmentRepo.ListFollowing(c.Request.Context(), userID, limit, offset)
		default: // discover, for_you, recommended
			if publicFeed {
				// Align with plaza rail: full public timeline (not genre-scoped discover).
				fragments, total, err = h.fragmentRepo.List(c.Request.Context(), limit, offset, domain.FragmentVisibilityPublic)
			} else {
				fragments, total, err = h.fragmentRepo.ListDiscoverFragmentsForUser(c.Request.Context(), userID, limit, offset)
			}
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fragments"})
		return
	}

	if len(fragments) > 0 {
		fragmentIDs := make([]string, 0, len(fragments))
		for _, fragment := range fragments {
			fragmentIDs = append(fragmentIDs, fragment.ID)
		}

		if statsMap, statsErr := h.fragmentRepo.BatchGetEngagementStats(c.Request.Context(), fragmentIDs, userID); statsErr == nil {
			for _, fragment := range fragments {
				stats := statsMap[fragment.ID]
				fragment.Likes = stats.Likes
				fragment.Comments = stats.Comments
				liked := stats.IsLiked
				fragment.IsLiked = &liked
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"fragments": fragments,
		"total":     total,
		"page":      page,
		"limit":     limit,
		"offset":    offset,
		"hasMore":   int64(offset+len(fragments)) < total,
	})
}

// GetUserFragments handles GET /users/:id/fragments
func (h *FragmentHandler) GetUserFragments(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := c.GetString("userID")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if rawOffset := c.Query("offset"); rawOffset != "" {
		if parsedOffset, err := strconv.Atoi(rawOffset); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
			page = (offset / limit) + 1
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
	} else {
		offset = (page - 1) * limit
	}

	if limit > 100 {
		limit = 100
	}

	draftsOnly := c.Query("drafts_only") == "1" || c.Query("draftsOnly") == "true"
	if draftsOnly {
		if currentUserID == "" || currentUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		fragments, total, err := h.fragmentRepo.ListDraftsByCreatorID(c.Request.Context(), userID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch drafts"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"fragments": fragments,
			"total":     total,
			"page":      page,
			"limit":     limit,
			"offset":    offset,
			"hasMore":   int64(offset+len(fragments)) < total,
		})
		return
	}

	// Respect owner's profile visibility configuration.
	if currentUserID != userID && h.userSettingsRepo != nil {
		if settings, err := h.userSettingsRepo.GetUserSettings(userID); err == nil && settings != nil && !settings.ShowPublicFragments {
			c.JSON(http.StatusOK, gin.H{
				"fragments": []*domain.Fragment{},
				"total":     0,
				"page":      page,
				"limit":     limit,
				"offset":    offset,
				"hasMore":   false,
			})
			return
		}
	}

	// Check permission - only show own private fragments or public/followers-only
	fragments, total, err := h.fragmentRepo.ListByCreatorID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fragments"})
		return
	}

	// Filter fragments based on visibility and user relationship.
	isFollower := false
	if currentUserID != "" && currentUserID != userID && h.repo != nil {
		if following, followErr := h.repo.IsFollowing(c.Request.Context(), currentUserID, userID); followErr == nil {
			isFollower = following
		}
	}
	filteredFragments := make([]*domain.Fragment, 0)
	for _, fragment := range fragments {
		// Show if: own fragment, public fragment, or followers-only fragment for followers.
		if currentUserID == userID || fragment.Visibility == domain.FragmentVisibilityPublic {
			filteredFragments = append(filteredFragments, fragment)
			continue
		}
		if (fragment.Visibility == domain.FragmentVisibilityFollowers ||
			fragment.Visibility == domain.FragmentVisibilityFollowersLegacy) && isFollower {
			filteredFragments = append(filteredFragments, fragment)
		}
	}

	if len(filteredFragments) > 0 {
		fragmentIDs := make([]string, 0, len(filteredFragments))
		for _, fragment := range filteredFragments {
			fragmentIDs = append(fragmentIDs, fragment.ID)
		}

		if statsMap, statsErr := h.fragmentRepo.BatchGetEngagementStats(c.Request.Context(), fragmentIDs, currentUserID); statsErr == nil {
			for _, fragment := range filteredFragments {
				stats := statsMap[fragment.ID]
				fragment.Likes = stats.Likes
				fragment.Comments = stats.Comments
				liked := stats.IsLiked
				fragment.IsLiked = &liked
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"fragments": filteredFragments,
		"total":     total,
		"page":      page,
		"limit":     limit,
		"offset":    offset,
		"hasMore":   int64(offset+len(filteredFragments)) < total,
	})
}

// UpdateFragment handles PUT /fragments/:id
func (h *FragmentHandler) UpdateFragment(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fragment, err := h.fragmentRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fragment not found"})
		return
	}

	// Check ownership
	if fragment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req UpdateFragmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Content != nil {
		fragment.Content = *req.Content
	}
	if req.ImageUrls != nil {
		processed := make([]string, 0, len(*req.ImageUrls))
		for _, u := range *req.ImageUrls {
			u = strings.TrimSpace(u)
			if u == "" {
				continue
			}
			if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
				processed = append(processed, u)
			}
		}
		fragment.ImageUrls = stringifyArray(processed)
		fragment.MediaURLs = append([]string(nil), processed...)
	}
	if req.Style != nil {
		fragment.Style = req.Style
	}
	if req.FragmentCount != nil {
		fragment.FragmentCount = req.FragmentCount
	}
	if req.Topic != nil {
		fragment.Topic = strings.TrimSpace(*req.Topic)
	}
	if req.Caption != nil {
		fragment.Caption = strings.TrimSpace(*req.Caption)
	}
	if req.Visibility != nil {
		if !domain.ValidFragmentVisibility(*req.Visibility) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visibility"})
			return
		}
		fragment.Visibility = domain.NormalizeFragmentVisibility(*req.Visibility)
	}

	wasDraft := fragment.IsDraft
	if req.IsDraft != nil {
		fragment.IsDraft = *req.IsDraft
	}
	if req.AspectRatio != nil {
		if ar := domain.NormalizeFragmentAspectRatio(strings.TrimSpace(*req.AspectRatio)); ar != "" {
			fragment.AspectRatio = ar
		}
	}

	if err := h.fragmentRepo.Update(c.Request.Context(), fragment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update fragment"})
		return
	}

	// 草稿 ↔ 已发布 时同步用户 fragments_count（创建草稿时未计入）
	if req.IsDraft != nil {
		if wasDraft && !fragment.IsDraft {
			if err := h.fragmentRepo.IncrementUserFragmentsCount(c.Request.Context(), fragment.UserID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user stats"})
				return
			}
		}
		if !wasDraft && fragment.IsDraft {
			if err := h.fragmentRepo.DecrementUserFragmentsCount(c.Request.Context(), fragment.UserID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user stats"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, fragment)
}

// DeleteFragment handles DELETE /fragments/:id
func (h *FragmentHandler) DeleteFragment(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fragment, err := h.fragmentRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fragment not found"})
		return
	}

	// Check ownership
	if fragment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := h.fragmentRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete fragment"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ToggleLike handles POST /fragments/:id/like
// Uses atomic delete-first strategy to prevent race conditions under concurrent requests.
func (h *FragmentHandler) ToggleLike(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	liked, err := h.fragmentRepo.ToggleLike(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle like"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"liked": liked})
}

// Helper functions
func generateUUID() string {
	return uuid.New().String()
}

func stringifyArray(arr []string) string {
	// Convert array to JSON string for storage
	if len(arr) == 0 {
		return "[]"
	}
	bytes, err := json.Marshal(arr)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

// RegisterRoutes registers the fragment CRUD routes
func (h *FragmentHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Fragment CRUD routes
	router.GET("", h.ListFragments)                         // GET /fragments
	router.GET("/:id", h.GetFragment)                       // GET /fragments/:id
	router.POST("", authMiddleware, h.CreateFragment)       // POST /fragments
	router.PUT("/:id", authMiddleware, h.UpdateFragment)    // PUT /fragments/:id
	router.DELETE("/:id", authMiddleware, h.DeleteFragment) // DELETE /fragments/:id
	router.GET("/styles", h.GetFragmentStyles)              // GET /fragments/styles
}
