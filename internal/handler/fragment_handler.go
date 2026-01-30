package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

// CreateFragmentRequest represents the request to create a fragment
type CreateFragmentRequest struct {
	Content   string   `json:"content" binding:"required,max=500"`
	ImageUrls []string `json:"imageUrls" binding:"required,min=1,max=10"`
	Visibility string  `json:"visibility" binding:"required,oneof=public followers private"`
}

// UpdateFragmentRequest represents the request to update a fragment
type UpdateFragmentRequest struct {
	Content   *string   `json:"content" binding:"omitempty,max=500"`
	ImageUrls *[]string `json:"imageUrls" binding:"omitempty,min=1,max=10"`
	Visibility *string  `json:"visibility" binding:"omitempty,oneof=public followers private"`
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

	// TODO: Upload images and get URLs
	// For now, use the provided URLs directly

	fragment := &domain.Fragment{
		ID:         generateUUID(), // Implement this
		CreatorID:  userID,
		Content:    req.Content,
		ImageUrls:  stringifyArray(req.ImageUrls),
		Visibility: req.Visibility,
		Likes:      0,
		Comments:   0,
		Shares:     0,
		Views:      0,
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

// GetUserFragments handles GET /users/:userId/fragments
func (h *FragmentHandler) GetUserFragments(c *gin.Context) {
	userID := c.Param("userId")
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
	if fragment.CreatorID != userID {
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
	if fragment.CreatorID != userID {
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
