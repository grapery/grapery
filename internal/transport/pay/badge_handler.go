package pay

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/grapestree/fgrapery/grapery/internal/transport/pay/middleware"
	"github.com/sirupsen/logrus"
)

// BadgeHandler 徽章处理器
type BadgeHandler struct {
	repo   *paymodels.BadgeRepository
	logger *logrus.Logger
}

// NewBadgeHandler 创建徽章处理器
func NewBadgeHandler(repo *paymodels.BadgeRepository) *BadgeHandler {
	return &BadgeHandler{
		repo:   repo,
		logger: logrus.New(),
	}
}

// GetAllBadges 获取所有徽章定义
// @Summary 获取所有徽章定义
// @Description 获取系统中所有可用的徽章定义列表
// @Tags Badge
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Router /api/vippay/badges [get]
func (h *BadgeHandler) GetAllBadges(c *gin.Context) {
	ctx := c.Request.Context()

	badges, err := h.repo.GetAllBadges(ctx)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetAllBadges",
			"error":    err.Error(),
		}).Error("Failed to get all badges")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get badges",
		})
		return
	}

	// 转换为响应格式
	badgeList := make([]gin.H, 0, len(badges))
	for _, badge := range badges {
		badgeList = append(badgeList, gin.H{
			"id":            badge.ID,
			"code":          badge.Code,
			"name":          badge.Name,
			"name_zh":       badge.NameZh,
			"description":   badge.Description,
			"desc_zh":       badge.DescZh,
			"category":      badge.Category,
			"tier":          badge.Tier,
			"icon_url":      badge.IconURL,
			"icon_emoji":    badge.IconEmoji,
			"color_hex":     badge.ColorHex,
			"threshold":     badge.Threshold,
			"points":        badge.Points,
			"display_order": badge.DisplayOrder,
		})
	}

	h.logger.WithFields(logrus.Fields{
		"endpoint":    "GetAllBadges",
		"badge_count": len(badgeList),
	}).Info("Successfully retrieved all badges")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"badges": badgeList,
			"total":  len(badgeList),
		},
	})
}

// GetBadgesByCategory 根据类别获取徽章
// @Summary 根据类别获取徽章
// @Description 根据类别获取徽章列表
// @Tags Badge
// @Accept json
// @Produce json
// @Param category path string true "徽章类别"
// @Success 200 {object} object
// @Router /api/vippay/badges/category/{category} [get]
func (h *BadgeHandler) GetBadgesByCategory(c *gin.Context) {
	ctx := c.Request.Context()
	category := c.Param("category")

	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "category is required",
		})
		return
	}

	badges, err := h.repo.GetBadgesByCategory(ctx, paymodels.BadgeCategory(category))
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetBadgesByCategory",
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to get badges by category")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get badges",
		})
		return
	}

	badgeList := make([]gin.H, 0, len(badges))
	for _, badge := range badges {
		badgeList = append(badgeList, gin.H{
			"id":            badge.ID,
			"code":          badge.Code,
			"name":          badge.Name,
			"name_zh":       badge.NameZh,
			"description":   badge.Description,
			"desc_zh":       badge.DescZh,
			"category":      badge.Category,
			"tier":          badge.Tier,
			"icon_url":      badge.IconURL,
			"icon_emoji":    badge.IconEmoji,
			"color_hex":     badge.ColorHex,
			"threshold":     badge.Threshold,
			"points":        badge.Points,
			"display_order": badge.DisplayOrder,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"category": category,
			"badges":   badgeList,
			"total":    len(badgeList),
		},
	})
}

// GetUserBadgeProfile 获取用户徽章档案
// @Summary 获取用户徽章档案
// @Description 获取当前用户的完整徽章档案，包括已获得的徽章、进度等
// @Tags Badge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object
// @Router /api/vippay/badges/profile [get]
func (h *BadgeHandler) GetUserBadgeProfile(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	profile, err := h.repo.GetUserBadgeProfile(ctx, userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetUserBadgeProfile",
			"user_id":  userID,
			"error":    err.Error(),
		}).Error("Failed to get user badge profile")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get badge profile",
		})
		return
	}

	// 转换 earned badges
	earnedBadges := make([]gin.H, 0, len(profile.EarnedBadges))
	for _, ub := range profile.EarnedBadges {
		badgeData := gin.H{
			"id":        ub.ID,
			"badge_id":  ub.BadgeID,
			"earned_at": ub.EarnedAt,
			"is_new":    ub.IsNew,
			"is_pinned": ub.IsPinned,
		}
		if ub.Badge != nil {
			badgeData["badge"] = gin.H{
				"id":          ub.Badge.ID,
				"code":        ub.Badge.Code,
				"name":        ub.Badge.Name,
				"name_zh":     ub.Badge.NameZh,
				"description": ub.Badge.Description,
				"desc_zh":     ub.Badge.DescZh,
				"category":    ub.Badge.Category,
				"tier":        ub.Badge.Tier,
				"icon_url":    ub.Badge.IconURL,
				"icon_emoji":  ub.Badge.IconEmoji,
				"color_hex":   ub.Badge.ColorHex,
				"points":      ub.Badge.Points,
			}
		}
		earnedBadges = append(earnedBadges, badgeData)
	}

	// 转换 pinned badges
	pinnedBadges := make([]gin.H, 0, len(profile.PinnedBadges))
	for _, ub := range profile.PinnedBadges {
		badgeData := gin.H{
			"id":        ub.ID,
			"badge_id":  ub.BadgeID,
			"earned_at": ub.EarnedAt,
		}
		if ub.Badge != nil {
			badgeData["badge"] = gin.H{
				"id":         ub.Badge.ID,
				"code":       ub.Badge.Code,
				"name":       ub.Badge.Name,
				"name_zh":    ub.Badge.NameZh,
				"icon_url":   ub.Badge.IconURL,
				"icon_emoji": ub.Badge.IconEmoji,
				"color_hex":  ub.Badge.ColorHex,
				"tier":       ub.Badge.Tier,
			}
		}
		pinnedBadges = append(pinnedBadges, badgeData)
	}

	// 转换 new badges
	newBadges := make([]gin.H, 0, len(profile.NewBadges))
	for _, ub := range profile.NewBadges {
		badgeData := gin.H{
			"id":        ub.ID,
			"badge_id":  ub.BadgeID,
			"earned_at": ub.EarnedAt,
		}
		if ub.Badge != nil {
			badgeData["badge"] = gin.H{
				"id":          ub.Badge.ID,
				"code":        ub.Badge.Code,
				"name":        ub.Badge.Name,
				"name_zh":     ub.Badge.NameZh,
				"description": ub.Badge.Description,
				"desc_zh":     ub.Badge.DescZh,
				"icon_url":    ub.Badge.IconURL,
				"icon_emoji":  ub.Badge.IconEmoji,
				"color_hex":   ub.Badge.ColorHex,
				"tier":        ub.Badge.Tier,
				"points":      ub.Badge.Points,
			}
		}
		newBadges = append(newBadges, badgeData)
	}

	// 转换 badge progress
	progressList := make([]gin.H, 0, len(profile.BadgeProgress))
	for _, bp := range profile.BadgeProgress {
		progressData := gin.H{
			"current":      bp.Current,
			"target":       bp.Target,
			"progress":     bp.Progress,
			"is_completed": bp.IsCompleted,
		}
		if bp.Badge != nil {
			progressData["badge"] = gin.H{
				"id":          bp.Badge.ID,
				"code":        bp.Badge.Code,
				"name":        bp.Badge.Name,
				"name_zh":     bp.Badge.NameZh,
				"description": bp.Badge.Description,
				"desc_zh":     bp.Badge.DescZh,
				"category":    bp.Badge.Category,
				"tier":        bp.Badge.Tier,
				"icon_url":    bp.Badge.IconURL,
				"icon_emoji":  bp.Badge.IconEmoji,
				"color_hex":   bp.Badge.ColorHex,
				"threshold":   bp.Badge.Threshold,
				"points":      bp.Badge.Points,
			}
		}
		progressList = append(progressList, progressData)
	}

	// 构建统计数据
	var statsData gin.H
	if profile.Stats != nil {
		statsData = gin.H{
			"story_count":      profile.Stats.StoryCount,
			"storyboard_count": profile.Stats.StoryboardCount,
			"total_likes":      profile.Stats.TotalLikes,
			"story_likes":      profile.Stats.StoryLikes,
			"storyboard_likes": profile.Stats.StoryboardLikes,
			"follower_count":   profile.Stats.FollowerCount,
			"following_count":  profile.Stats.FollowingCount,
			"total_badges":     profile.Stats.TotalBadges,
			"total_points":     profile.Stats.TotalPoints,
			"last_updated":     profile.Stats.LastUpdated,
		}
	}

	h.logger.WithFields(logrus.Fields{
		"endpoint":      "GetUserBadgeProfile",
		"user_id":       userID,
		"earned_count":  len(earnedBadges),
		"new_count":     len(newBadges),
		"total_points":  profile.TotalPoints,
	}).Info("Successfully retrieved user badge profile")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"user_id":         userID,
			"stats":           statsData,
			"earned_badges":   earnedBadges,
			"pinned_badges":   pinnedBadges,
			"new_badges":      newBadges,
			"badge_progress":  progressList,
			"total_badges":    profile.TotalBadges,
			"total_points":    profile.TotalPoints,
			"completion_rate": profile.CompletionRate,
		},
	})
}

// GetUserBadges 获取用户已获得的徽章列表
// @Summary 获取用户已获得的徽章
// @Description 获取指定用户已获得的所有徽章
// @Tags Badge
// @Accept json
// @Produce json
// @Param user_id query string false "用户ID（可选，不传则获取当前用户）"
// @Success 200 {object} object
// @Router /api/vippay/badges/user [get]
func (h *BadgeHandler) GetUserBadges(c *gin.Context) {
	ctx := c.Request.Context()
	
	// 支持查询指定用户或当前用户
	userID := c.Query("user_id")
	if userID == "" {
		userID = middleware.GetUserIDFromContext(c)
	}
	
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "user_id is required",
		})
		return
	}

	badges, err := h.repo.GetUserBadges(ctx, userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetUserBadges",
			"user_id":  userID,
			"error":    err.Error(),
		}).Error("Failed to get user badges")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get user badges",
		})
		return
	}

	badgeList := make([]gin.H, 0, len(badges))
	for _, ub := range badges {
		badgeData := gin.H{
			"id":        ub.ID,
			"badge_id":  ub.BadgeID,
			"earned_at": ub.EarnedAt,
			"is_new":    ub.IsNew,
			"is_pinned": ub.IsPinned,
		}
		if ub.Badge != nil {
			badgeData["badge"] = gin.H{
				"id":         ub.Badge.ID,
				"code":       ub.Badge.Code,
				"name":       ub.Badge.Name,
				"name_zh":    ub.Badge.NameZh,
				"icon_url":   ub.Badge.IconURL,
				"icon_emoji": ub.Badge.IconEmoji,
				"color_hex":  ub.Badge.ColorHex,
				"tier":       ub.Badge.Tier,
				"category":   ub.Badge.Category,
				"points":     ub.Badge.Points,
			}
		}
		badgeList = append(badgeList, badgeData)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"user_id": userID,
			"badges":  badgeList,
			"total":   len(badgeList),
		},
	})
}

// GetUserPinnedBadges 获取用户置顶的徽章
// @Summary 获取用户置顶的徽章
// @Description 获取用户置顶展示的徽章列表（用于主页展示）
// @Tags Badge
// @Accept json
// @Produce json
// @Param user_id query string false "用户ID（可选）"
// @Success 200 {object} object
// @Router /api/vippay/badges/pinned [get]
func (h *BadgeHandler) GetUserPinnedBadges(c *gin.Context) {
	ctx := c.Request.Context()
	
	userID := c.Query("user_id")
	if userID == "" {
		userID = middleware.GetUserIDFromContext(c)
	}
	
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "user_id is required",
		})
		return
	}

	badges, err := h.repo.GetUserPinnedBadges(ctx, userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetUserPinnedBadges",
			"user_id":  userID,
			"error":    err.Error(),
		}).Error("Failed to get user pinned badges")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get pinned badges",
		})
		return
	}

	badgeList := make([]gin.H, 0, len(badges))
	for _, ub := range badges {
		badgeData := gin.H{
			"id":        ub.ID,
			"badge_id":  ub.BadgeID,
			"earned_at": ub.EarnedAt,
		}
		if ub.Badge != nil {
			badgeData["badge"] = gin.H{
				"id":         ub.Badge.ID,
				"code":       ub.Badge.Code,
				"name":       ub.Badge.Name,
				"name_zh":    ub.Badge.NameZh,
				"icon_url":   ub.Badge.IconURL,
				"icon_emoji": ub.Badge.IconEmoji,
				"color_hex":  ub.Badge.ColorHex,
				"tier":       ub.Badge.Tier,
			}
		}
		badgeList = append(badgeList, badgeData)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"user_id": userID,
			"badges":  badgeList,
			"total":   len(badgeList),
		},
	})
}

// GetUserStats 获取用户徽章统计
// @Summary 获取用户徽章统计
// @Description 获取用户的徽章相关统计数据
// @Tags Badge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object
// @Router /api/vippay/badges/stats [get]
func (h *BadgeHandler) GetUserStats(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	stats, err := h.repo.GetUserBadgeStats(ctx, userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetUserStats",
			"user_id":  userID,
			"error":    err.Error(),
		}).Error("Failed to get user badge stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get user stats",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"user_id":          userID,
			"story_count":      stats.StoryCount,
			"storyboard_count": stats.StoryboardCount,
			"total_likes":      stats.TotalLikes,
			"story_likes":      stats.StoryLikes,
			"storyboard_likes": stats.StoryboardLikes,
			"follower_count":   stats.FollowerCount,
			"following_count":  stats.FollowingCount,
			"total_badges":     stats.TotalBadges,
			"total_points":     stats.TotalPoints,
			"last_updated":     stats.LastUpdated,
		},
	})
}

// PinBadgeRequest 置顶徽章请求
type PinBadgeRequest struct {
	BadgeID uint `json:"badge_id" binding:"required"`
}

// PinBadge 置顶徽章
// @Summary 置顶徽章
// @Description 将徽章置顶到用户主页展示
// @Tags Badge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PinBadgeRequest true "请求体"
// @Success 200 {object} object
// @Router /api/vippay/badges/pin [post]
func (h *BadgeHandler) PinBadge(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	var req PinBadgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request",
		})
		return
	}

	// 检查用户是否拥有该徽章
	has, err := h.repo.HasUserBadge(ctx, userID, req.BadgeID)
	if err != nil || !has {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "badge not found or not owned",
		})
		return
	}

	if err := h.repo.PinBadge(ctx, userID, req.BadgeID); err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "PinBadge",
			"user_id":  userID,
			"badge_id": req.BadgeID,
			"error":    err.Error(),
		}).Error("Failed to pin badge")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to pin badge",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"pinned": true,
		},
	})
}

// UnpinBadge 取消置顶徽章
// @Summary 取消置顶徽章
// @Description 取消徽章的置顶状态
// @Tags Badge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param badge_id path int true "徽章ID"
// @Success 200 {object} object
// @Router /api/vippay/badges/unpin/{badge_id} [post]
func (h *BadgeHandler) UnpinBadge(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	badgeIDStr := c.Param("badge_id")
	badgeID, err := strconv.ParseUint(badgeIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid badge_id",
		})
		return
	}

	if err := h.repo.UnpinBadge(ctx, userID, uint(badgeID)); err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "UnpinBadge",
			"user_id":  userID,
			"badge_id": badgeID,
			"error":    err.Error(),
		}).Error("Failed to unpin badge")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to unpin badge",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"unpinned": true,
		},
	})
}

// MarkBadgesViewedRequest 标记徽章已查看请求
type MarkBadgesViewedRequest struct {
	BadgeIDs []uint `json:"badge_ids"` // 可选，不传则标记所有
}

// MarkBadgesViewed 标记徽章为已查看
// @Summary 标记徽章为已查看
// @Description 标记新获得的徽章为已查看状态
// @Tags Badge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MarkBadgesViewedRequest true "请求体"
// @Success 200 {object} object
// @Router /api/vippay/badges/mark-viewed [post]
func (h *BadgeHandler) MarkBadgesViewed(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	var req MarkBadgesViewedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体，表示标记所有
		req.BadgeIDs = nil
	}

	var err error
	if len(req.BadgeIDs) == 0 {
		err = h.repo.MarkAllBadgesAsViewed(ctx, userID)
	} else {
		err = h.repo.MarkBadgeAsViewed(ctx, userID, req.BadgeIDs)
	}

	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint":  "MarkBadgesViewed",
			"user_id":   userID,
			"badge_ids": req.BadgeIDs,
			"error":     err.Error(),
		}).Error("Failed to mark badges as viewed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to mark badges as viewed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"marked": true,
		},
	})
}

// CheckAndAwardBadges 检查并授予徽章
// @Summary 检查并授予徽章
// @Description 检查用户是否满足徽章获取条件，并自动授予符合条件的徽章
// @Tags Badge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object
// @Router /api/vippay/badges/check [post]
func (h *BadgeHandler) CheckAndAwardBadges(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	awarded, err := h.repo.CheckAndAwardBadges(ctx, userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "CheckAndAwardBadges",
			"user_id":  userID,
			"error":    err.Error(),
		}).Error("Failed to check and award badges")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to check badges",
		})
		return
	}

	awardedList := make([]gin.H, 0, len(awarded))
	for _, ub := range awarded {
		badgeData := gin.H{
			"id":        ub.ID,
			"badge_id":  ub.BadgeID,
			"earned_at": ub.EarnedAt,
		}
		if ub.Badge != nil {
			badgeData["badge"] = gin.H{
				"id":          ub.Badge.ID,
				"code":        ub.Badge.Code,
				"name":        ub.Badge.Name,
				"name_zh":     ub.Badge.NameZh,
				"description": ub.Badge.Description,
				"desc_zh":     ub.Badge.DescZh,
				"icon_url":    ub.Badge.IconURL,
				"icon_emoji":  ub.Badge.IconEmoji,
				"color_hex":   ub.Badge.ColorHex,
				"tier":        ub.Badge.Tier,
				"points":      ub.Badge.Points,
			}
		}
		awardedList = append(awardedList, badgeData)
	}

	h.logger.WithFields(logrus.Fields{
		"endpoint":      "CheckAndAwardBadges",
		"user_id":       userID,
		"awarded_count": len(awardedList),
	}).Info("Badge check completed")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"awarded_badges": awardedList,
			"count":          len(awardedList),
		},
	})
}

// SyncUserStatsRequest 同步用户统计请求
type SyncUserStatsRequest struct {
	StoryCount      int `json:"story_count"`
	StoryboardCount int `json:"storyboard_count"`
	TotalLikes      int `json:"total_likes"`
	StoryLikes      int `json:"story_likes"`
	StoryboardLikes int `json:"storyboard_likes"`
	FollowerCount   int `json:"follower_count"`
	FollowingCount  int `json:"following_count"`
}

// SyncUserStats 同步用户统计数据
// @Summary 同步用户统计数据
// @Description 从主服务同步用户的故事、故事版、点赞等统计数据
// @Tags Badge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SyncUserStatsRequest true "请求体"
// @Success 200 {object} object
// @Router /api/vippay/badges/sync-stats [post]
func (h *BadgeHandler) SyncUserStats(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	var req SyncUserStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request",
		})
		return
	}

	err := h.repo.SyncUserStats(ctx, userID,
		req.StoryCount,
		req.StoryboardCount,
		req.TotalLikes,
		req.StoryLikes,
		req.StoryboardLikes,
		req.FollowerCount,
		req.FollowingCount,
	)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "SyncUserStats",
			"user_id":  userID,
			"error":    err.Error(),
		}).Error("Failed to sync user stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to sync stats",
		})
		return
	}

	// 同步后检查并授予徽章
	awarded, _ := h.repo.CheckAndAwardBadges(ctx, userID)

	awardedList := make([]gin.H, 0, len(awarded))
	for _, ub := range awarded {
		if ub.Badge != nil {
			awardedList = append(awardedList, gin.H{
				"code":       ub.Badge.Code,
				"name":       ub.Badge.Name,
				"name_zh":    ub.Badge.NameZh,
				"icon_emoji": ub.Badge.IconEmoji,
				"tier":       ub.Badge.Tier,
				"points":     ub.Badge.Points,
			})
		}
	}

	h.logger.WithFields(logrus.Fields{
		"endpoint":      "SyncUserStats",
		"user_id":       userID,
		"awarded_count": len(awardedList),
	}).Info("User stats synced and badges checked")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"synced":         true,
			"awarded_badges": awardedList,
		},
	})
}

// GetBadgeProgress 获取徽章进度
// @Summary 获取徽章进度
// @Description 获取当前用户所有徽章的获取进度
// @Tags Badge
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object
// @Router /api/vippay/badges/progress [get]
func (h *BadgeHandler) GetBadgeProgress(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	progress, err := h.repo.GetBadgeProgress(ctx, userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetBadgeProgress",
			"user_id":  userID,
			"error":    err.Error(),
		}).Error("Failed to get badge progress")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get badge progress",
		})
		return
	}

	progressList := make([]gin.H, 0, len(progress))
	for _, bp := range progress {
		progressData := gin.H{
			"current":      bp.Current,
			"target":       bp.Target,
			"progress":     bp.Progress,
			"is_completed": bp.IsCompleted,
		}
		if bp.Badge != nil {
			progressData["badge"] = gin.H{
				"id":          bp.Badge.ID,
				"code":        bp.Badge.Code,
				"name":        bp.Badge.Name,
				"name_zh":     bp.Badge.NameZh,
				"category":    bp.Badge.Category,
				"tier":        bp.Badge.Tier,
				"icon_url":    bp.Badge.IconURL,
				"icon_emoji":  bp.Badge.IconEmoji,
				"color_hex":   bp.Badge.ColorHex,
				"threshold":   bp.Badge.Threshold,
				"points":      bp.Badge.Points,
			}
		}
		progressList = append(progressList, progressData)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"progress": progressList,
			"total":    len(progressList),
		},
	})
}

