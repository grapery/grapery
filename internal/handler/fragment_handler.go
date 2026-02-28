package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
)

type FragmentHandler struct {
	fragmentRepo *repository.FragmentRepository
}

func NewFragmentHandler(fragmentRepo *repository.FragmentRepository) *FragmentHandler {
	return &FragmentHandler{
		fragmentRepo: fragmentRepo,
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

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// CreateFragmentRequest represents the request to create a fragment
type CreateFragmentRequest struct {
	Content       string   `json:"content" binding:"required,max=500"`
	ImageUrls     []string `json:"imageUrls" binding:"required,min=1,max=10"`
	Style         *string  `json:"style" binding:"omitempty"`
	FragmentCount *int     `json:"fragmentCount" binding:"omitempty,min=1,max=16"`
	Visibility    string   `json:"visibility" binding:"required,oneof=public followers private"`
}

// UpdateFragmentRequest represents the request to update a fragment
type UpdateFragmentRequest struct {
	Content    *string   `json:"content" binding:"omitempty,max=500"`
	ImageUrls  *[]string `json:"imageUrls" binding:"omitempty,min=1,max=10"`
	Visibility *string   `json:"visibility" binding:"omitempty,oneof=public followers private"`
}

// CreateFragment handles POST /fragments
func (h *FragmentHandler) CreateFragment(c *gin.Context) {
	userID := c.GetString("user_id")
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

	// Set default fragment count to 1 if not provided
	fragmentCount := 1
	if req.FragmentCount != nil {
		fragmentCount = *req.FragmentCount
	}

	// TODO: Upload images and get URLs
	// For now, use the provided URLs directly

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
		ImageUrls:     stringifyArray(req.ImageUrls),
		Style:         req.Style,
		FragmentCount: &fragmentCount,
		Visibility:    req.Visibility,
		SourceType:    string(domain.FragmentSourceOriginal), // 用户手动创建的碎片为原创
		SourceID:      "",                                    // 原创碎片无来源ID
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

	// Check if current user has liked this fragment
	userID := c.GetString("user_id")
	if userID != "" {
		isLiked, _ := h.fragmentRepo.IsLiked(c.Request.Context(), id, userID)
		fragment.IsLiked = &isLiked
	}

	// Increment view count
	go h.fragmentRepo.IncrementLikes(c.Request.Context(), id) // Reuse for views, or create separate

	c.JSON(http.StatusOK, fragment)
}

// ListFragments handles GET /fragments
func (h *FragmentHandler) ListFragments(c *gin.Context) {
	userID := c.GetString("user_id")

	// Parse query parameters
	tab := c.DefaultQuery("tab", "for_you") // for_you, following
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	if limit > 100 {
		limit = 100
	}

	var fragments []*domain.Fragment
	var total int64
	var err error

	switch tab {
	case "following":
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		fragments, total, err = h.fragmentRepo.ListFollowing(c.Request.Context(), userID, limit, offset)
	default: // for_you
		fragments, total, err = h.fragmentRepo.List(c.Request.Context(), limit, offset, domain.FragmentVisibilityPublic)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fragments"})
		return
	}

	// Check likes for authenticated user
	if userID != "" {
		for _, fragment := range fragments {
			isLiked, _ := h.fragmentRepo.IsLiked(c.Request.Context(), fragment.ID, userID)
			fragment.IsLiked = &isLiked
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"fragments": fragments,
		"total":     total,
		"page":      page,
		"limit":     limit,
		"hasMore":   int64(offset+len(fragments)) < total,
	})
}

// GetUserFragments handles GET /users/:id/fragments
func (h *FragmentHandler) GetUserFragments(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := c.GetString("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	if limit > 100 {
		limit = 100
	}

	// Check permission - only show own private fragments or public/followers-only
	fragments, total, err := h.fragmentRepo.ListByCreatorID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fragments"})
		return
	}

	// Filter fragments based on visibility and user relationship
	filteredFragments := make([]*domain.Fragment, 0)
	for _, fragment := range fragments {
		// Show if: public, or (followers and current user is following), or own fragments
		if fragment.Visibility == domain.FragmentVisibilityPublic ||
			(currentUserID == userID) ||
			(fragment.Visibility == domain.FragmentVisibilityFollowers && currentUserID != "") {
			filteredFragments = append(filteredFragments, fragment)
		}
	}

	// Check likes
	if currentUserID != "" {
		for _, fragment := range filteredFragments {
			isLiked, _ := h.fragmentRepo.IsLiked(c.Request.Context(), fragment.ID, currentUserID)
			fragment.IsLiked = &isLiked
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"fragments": filteredFragments,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// UpdateFragment handles PUT /fragments/:id
func (h *FragmentHandler) UpdateFragment(c *gin.Context) {
	userID := c.GetString("user_id")
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
		fragment.ImageUrls = stringifyArray(*req.ImageUrls)
	}
	if req.Visibility != nil {
		if !domain.ValidFragmentVisibility(*req.Visibility) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visibility"})
			return
		}
		fragment.Visibility = *req.Visibility
	}

	if err := h.fragmentRepo.Update(c.Request.Context(), fragment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update fragment"})
		return
	}

	c.JSON(http.StatusOK, fragment)
}

// DeleteFragment handles DELETE /fragments/:id
func (h *FragmentHandler) DeleteFragment(c *gin.Context) {
	userID := c.GetString("user_id")
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
func (h *FragmentHandler) ToggleLike(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Check if already liked
	isLiked, err := h.fragmentRepo.IsLiked(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check like status"})
		return
	}

	if isLiked {
		// Unlike
		if err := h.fragmentRepo.DeleteLike(c.Request.Context(), id, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlike"})
			return
		}
		if err := h.fragmentRepo.DecrementLikes(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update likes"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"liked": false})
	} else {
		// Like
		like := &domain.FragmentLike{
			ID:         generateUUID(),
			FragmentID: id,
			UserID:     userID,
		}
		if err := h.fragmentRepo.CreateLike(c.Request.Context(), like); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like"})
			return
		}
		if err := h.fragmentRepo.IncrementLikes(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update likes"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"liked": true})
	}
}

// Helper functions
func generateUUID() string {
	// Implement UUID generation
	return "fragment-" + strconv.FormatInt(int64(float64(999999)), 10) // Placeholder
}

func stringifyArray(arr []string) string {
	// Convert array to JSON string for storage
	// In production, use json.Marshal
	result := "["
	for i, s := range arr {
		if i > 0 {
			result += ","
		}
		result += `"` + s + `"`
	}
	result += "]"
	return result
}
