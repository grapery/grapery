package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// GetGroupActivities 获取群组活动流
// GET /api/groups/:id/activities
// Query params:
//   - limit: int (default 20, max 100)
//   - offset: int (default 0)
//   - time_range: string (today, week, month) - default: week
//   - date: string (YYYY-MM-DD) - filter by specific date
func (h *Handler) GetGroupActivities(c *gin.Context) {
	groupID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	timeRangeStr := c.DefaultQuery("time_range", "week")
	date := c.Query("date")

	// Convert time range string to enum
	var timeRange domain.ActivityTimeRange
	switch timeRangeStr {
	case "today":
		timeRange = domain.TimeRangeToday
	case "week":
		timeRange = domain.TimeRangeWeek
	case "month":
		timeRange = domain.TimeRangeMonth
	default:
		timeRange = domain.TimeRangeWeek
	}

	req := &domain.ActivityListRequest{
		GroupID:   groupID,
		TimeRange: timeRange,
		Date:      date,
		Limit:     limit,
		Offset:    offset,
	}

	activities, count, err := h.svc.GetGroupActivitiesWithFilter(c.Request.Context(), req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"activities": activities,
		"count":      count,
	})
}

// GetGroupActivityHeatmap 获取群组活动热力图数据
// GET /api/groups/:id/activities/heatmap
// Query params:
//   - time_range: string (today, week, month) - default: week
func (h *Handler) GetGroupActivityHeatmap(c *gin.Context) {
	groupID := c.Param("id")
	timeRangeStr := c.DefaultQuery("time_range", "week")

	// Convert time range string to enum
	var timeRange domain.ActivityTimeRange
	switch timeRangeStr {
	case "today":
		timeRange = domain.TimeRangeToday
	case "week":
		timeRange = domain.TimeRangeWeek
	case "month":
		timeRange = domain.TimeRangeMonth
	default:
		timeRange = domain.TimeRangeWeek
	}

	heatmapResponse, err := h.svc.GetGroupActivityHeatmap(c.Request.Context(), groupID, timeRange)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, heatmapResponse)
}

// GetUserActivities 获取用户活动流
// GET /api/activities
// GET /api/users/:id/activities
// Query params:
//   - limit: int (default 50, max 100)
//   - offset: int (default 0)
//   - time_range: string (today, week, month) - default: week
//   - date: string (YYYY-MM-DD) - filter by specific date
func (h *Handler) GetUserActivities(c *gin.Context) {
	// Support both /api/activities (current user) and /api/users/:id/activities (specific user)
	userID := c.Param("id")
	if userID == "" {
		userID = authPkg.GetUserID(c)
	}
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	timeRangeStr := c.DefaultQuery("time_range", "week")
	date := c.Query("date")

	// Convert time range string to enum
	var timeRange domain.ActivityTimeRange
	switch timeRangeStr {
	case "today":
		timeRange = domain.TimeRangeToday
	case "week":
		timeRange = domain.TimeRangeWeek
	case "month":
		timeRange = domain.TimeRangeMonth
	default:
		timeRange = domain.TimeRangeWeek
	}

	activities, count, err := h.svc.GetUserActivitiesWithFilter(c.Request.Context(), userID, timeRange, date, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"activities": activities,
		"count":      count,
	})
}

// GetUserActivityHeatmap 获取用户活动热力图数据
// GET /api/users/:id/activities/heatmap
// Query params:
//   - time_range: string (today, week, month) - default: week
func (h *Handler) GetUserActivityHeatmap(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		userID = authPkg.GetUserID(c)
	}
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	timeRangeStr := c.DefaultQuery("time_range", "week")

	// Convert time range string to enum
	var timeRange domain.ActivityTimeRange
	switch timeRangeStr {
	case "today":
		timeRange = domain.TimeRangeToday
	case "week":
		timeRange = domain.TimeRangeWeek
	case "month":
		timeRange = domain.TimeRangeMonth
	default:
		timeRange = domain.TimeRangeWeek
	}

	heatmapResponse, err := h.svc.GetUserActivityHeatmap(c.Request.Context(), userID, timeRange)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, heatmapResponse)
}

// GetGlobalActivities 获取全局活动流
// GET /api/activities/global
func (h *Handler) GetGlobalActivities(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	activities, err := h.svc.GetGlobalActivities(c.Request.Context(), limit)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"activities": activities,
		"count":      len(activities),
	})
}
