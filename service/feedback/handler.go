package feedback

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/utils"
	log "github.com/sirupsen/logrus"
)

// Handler 反馈处理器
type Handler struct {
	service *FeedbackService
}

// NewHandler 创建反馈处理器
func NewHandler() *Handler {
	return &Handler{
		service: NewFeedbackService(),
	}
}

// Response 统一响应结构体
type Response struct {
	Code    int         `json:"code"`    // 状态码
	Message string      `json:"message"` // 消息
	Data    interface{} `json:"data"`    // 数据
}

// SuccessResponse 成功响应
func SuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// ErrorResponse 错误响应
func ErrorResponse(code int, message string) *Response {
	return &Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// CreateFeedback 创建用户反馈
// @Summary 创建用户反馈
// @Description 用户提交反馈信息
// @Tags 用户反馈
// @Accept json
// @Produce json
// @Param request body CreateFeedbackRequest true "反馈信息"
// @Success 200 {object} Response{data=CreateFeedbackResponse}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/feedback [post]
func (h *Handler) CreateFeedback(c *gin.Context) {
	// 获取用户ID（从中间件中获取）
	userID, exists := c.Get(utils.UserIdKey)
	if !exists {
		log.Error("用户ID不存在")
		c.JSON(http.StatusUnauthorized, ErrorResponse(401, "用户未认证"))
		return
	}

	userIDInt64, ok := userID.(int64)
	if !ok {
		log.Error("用户ID类型错误")
		c.JSON(http.StatusInternalServerError, ErrorResponse(500, "用户ID类型错误"))
		return
	}

	// 解析请求参数
	var req CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("解析请求参数失败: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse(400, "请求参数错误: "+err.Error()))
		return
	}

	// 调用服务层创建反馈
	response, err := h.service.CreateFeedback(c.Request.Context(), userIDInt64, &req)
	if err != nil {
		log.Errorf("创建反馈失败: %v", err)
		if IsBadRequestError(err) {
			c.JSON(http.StatusBadRequest, ErrorResponse(400, "请求参数错误"))
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse(500, "创建反馈失败"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(response))
}

// GetUserFeedbackList 获取用户反馈列表
// @Summary 获取用户反馈列表
// @Description 获取当前用户的反馈列表
// @Tags 用户反馈
// @Produce json
// @Param offset query int false "偏移量" default(0)
// @Param limit query int false "限制数量" default(20)
// @Success 200 {object} Response{data=FeedbackListResponse}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/feedback [get]
func (h *Handler) GetUserFeedbackList(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get(utils.UserIdKey)
	if !exists {
		log.Error("用户ID不存在")
		c.JSON(http.StatusUnauthorized, ErrorResponse(401, "用户未认证"))
		return
	}

	userIDInt64, ok := userID.(int64)
	if !ok {
		log.Error("用户ID类型错误")
		c.JSON(http.StatusInternalServerError, ErrorResponse(500, "用户ID类型错误"))
		return
	}

	// 解析分页参数
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		log.Errorf("解析offset参数失败: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse(400, "offset参数错误"))
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		log.Errorf("解析limit参数失败: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse(400, "limit参数错误"))
		return
	}

	// 调用服务层获取反馈列表
	response, err := h.service.GetUserFeedbackList(c.Request.Context(), userIDInt64, offset, limit)
	if err != nil {
		log.Errorf("获取反馈列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse(500, "获取反馈列表失败"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(response))
}

// GetFeedbackDetail 获取反馈详情
// @Summary 获取反馈详情
// @Description 获取指定反馈的详细信息
// @Tags 用户反馈
// @Produce json
// @Param id path int true "反馈ID"
// @Success 200 {object} Response{data=FeedbackItem}
// @Failure 400 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/feedback/{id} [get]
func (h *Handler) GetFeedbackDetail(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get(utils.UserIdKey)
	if !exists {
		log.Error("用户ID不存在")
		c.JSON(http.StatusUnauthorized, ErrorResponse(401, "用户未认证"))
		return
	}

	userIDInt64, ok := userID.(int64)
	if !ok {
		log.Error("用户ID类型错误")
		c.JSON(http.StatusInternalServerError, ErrorResponse(500, "用户ID类型错误"))
		return
	}

	// 解析反馈ID
	idStr := c.Param("id")
	feedbackID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Errorf("解析反馈ID失败: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse(400, "反馈ID格式错误"))
		return
	}

	// 调用服务层获取反馈详情
	response, err := h.service.GetFeedbackDetail(c.Request.Context(), uint(feedbackID), userIDInt64)
	if err != nil {
		log.Errorf("获取反馈详情失败: %v", err)
		if IsNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse(404, "反馈不存在"))
			return
		}
		if IsForbiddenError(err) {
			c.JSON(http.StatusForbidden, ErrorResponse(403, "无权限查看此反馈"))
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse(500, "获取反馈详情失败"))
		return
	}

	c.JSON(http.StatusOK, SuccessResponse(response))
}

// GetFeedbackTypes 获取反馈类型列表
// @Summary 获取反馈类型列表
// @Description 获取所有可用的反馈类型
// @Tags 用户反馈
// @Produce json
// @Success 200 {object} Response{data=map[int]string}
// @Router /api/feedback/types [get]
func (h *Handler) GetFeedbackTypes(c *gin.Context) {
	types := h.service.GetFeedbackTypes()
	c.JSON(http.StatusOK, SuccessResponse(types))
}

// GetFeedbackStatuses 获取反馈状态列表
// @Summary 获取反馈状态列表
// @Description 获取所有可用的反馈状态
// @Tags 用户反馈
// @Produce json
// @Success 200 {object} Response{data=map[int]string}
// @Router /api/feedback/statuses [get]
func (h *Handler) GetFeedbackStatuses(c *gin.Context) {
	statuses := h.service.GetFeedbackStatuses()
	c.JSON(http.StatusOK, SuccessResponse(statuses))
}

// HealthCheck 健康检查
// @Summary 健康检查
// @Description 检查反馈服务是否正常
// @Tags 系统
// @Produce json
// @Success 200 {object} Response
// @Router /api/feedback/health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, SuccessResponse(map[string]string{
		"status":  "ok",
		"service": "feedback",
	}))
}
