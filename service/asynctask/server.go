package asynctask

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/models"
	asynctaskpkg "github.com/grapery/grapery/pkg/asynctask"
)

// Server exposes HTTP endpoints for managing and monitoring async video generation tasks.
type Server struct {
	config      *config.Config
	taskManager *asynctaskpkg.TaskManager
}

// New creates a new async task HTTP server.
func New(cfg *config.Config, manager *asynctaskpkg.TaskManager) *Server {
	return &Server{
		config:      cfg,
		taskManager: manager,
	}
}

// RegisterRoutes registers HTTP routes on the provided gin engine.
func (s *Server) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", s.health)

	api := r.Group("/api/v1")
	{
		task := api.Group("/task")
		{
			task.POST("/video", s.createVideoTask)
			task.GET("/video", s.listVideoTasks)
			task.GET("/video/:task_id", s.getVideoTask)
		}

		worker := api.Group("/worker")
		{
			worker.GET("/queues", s.listQueues)
			worker.GET("/queues/:name", s.getQueueInfo)
		}

		system := api.Group("/system")
		{
			system.GET("/info", s.systemInfo)
		}
	}
}

type createVideoTaskRequest struct {
	StoryID       int64                  `json:"story_id"`
	BoardID       int64                  `json:"board_id"`
	SceneID       int64                  `json:"scene_id"`
	RoleID        int64                  `json:"role_id"`
	UserID        int64                  `json:"user_id"`
	Platform      string                 `json:"platform" binding:"required"`
	Prompt        string                 `json:"prompt" binding:"required"`
	AspectRatio   string                 `json:"aspect_ratio"`
	Duration      int                    `json:"duration"`
	Quality       string                 `json:"quality"`
	Style         string                 `json:"style"`
	Model         string                 `json:"model"`
	CallbackURL   string                 `json:"callback_url"`
	StartRefImage string                 `json:"start_ref_image"`
	EndRefImage   string                 `json:"end_ref_image"`
	Metadata      map[string]interface{} `json:"metadata"`
	RefImages     []string               `json:"ref_images"`
}

func (s *Server) createVideoTask(c *gin.Context) {
	var req createVideoTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	platform := strings.ToLower(req.Platform)
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform is required"})
		return
	}

	taskID := uuid.NewString()
	now := time.Now()

	video := &models.VideoGen{
		StoryId:    req.StoryID,
		BoardId:    req.BoardID,
		SceneId:    req.SceneID,
		RoleId:     req.RoleID,
		TaskId:     taskID,
		Prompt:     req.Prompt,
		Timelength: req.Duration,
		RefImages:  strings.Join(req.RefImages, ","),
		FisrtFrame: req.StartRefImage,
		EndFrame:   req.EndRefImage,
		UserID:     req.UserID,
		GenStatus:  gen.StoryGenStatus_STORY_GEN_STATUS_INIT,
		Provider:   platform,
		StartTime:  now.Unix(),
		EndTime:    0,
		Code:       "",
		Message:    "任务已提交，正在异步处理",
		Deleted:    0,
		Tokens:     0,
		Seed:       int64(rand.Intn(10000000)),
	}

	videoID, err := models.CreateVideoGen(c.Request.Context(), video)
	if err != nil {
		log.Errorf("create video task record failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create video task record failed"})
		return
	}

	payload := &asynctaskpkg.VideoGeneratePayload{
		VideoGenID:  videoID,
		TaskID:      taskID,
		UserID:      req.UserID,
		Platform:    platform,
		Prompt:      req.Prompt,
		AspectRatio: req.AspectRatio,
		Duration:    req.Duration,
		Quality:     req.Quality,
		Style:       req.Style,
		Model:       req.Model,
		CallbackURL: req.CallbackURL,
		Metadata:    req.Metadata,
	}

	info, err := s.taskManager.EnqueueVideoTask(c.Request.Context(), payload)
	if err != nil {
		log.Errorf("enqueue video task failed: %v", err)
		_ = models.UpdateVideoGenFields(context.Background(), videoID, map[string]interface{}{
			"gen_status": gen.StoryGenStatus_STORY_GEN_STATUS_ERROR,
			"message":    err.Error(),
			"end_time":   time.Now().Unix(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "enqueue video task failed"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"task_id":         taskID,
		"video_gen_id":    videoID,
		"queue":           info.Queue,
		"state":           info.State,
		"max_retry":       info.MaxRetry,
		"next_process_at": info.NextProcessAt,
	})
}

func (s *Server) getVideoTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	video, err := models.GetVideoGenByTaskID(c.Request.Context(), taskID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "request canceled"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toVideoResponse(video)})
}

func (s *Server) listVideoTasks(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	videos, err := models.GetVideoGenListPage(c.Request.Context(), offset, limit)
	if err != nil {
		log.Errorf("list video tasks failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list video tasks failed"})
		return
	}

	resp := make([]map[string]interface{}, 0, len(videos))
	for _, v := range videos {
		resp = append(resp, toVideoResponse(v))
	}

	c.JSON(http.StatusOK, gin.H{"data": resp, "count": len(resp)})
}

func (s *Server) listQueues(c *gin.Context) {
	queues, err := s.taskManager.Inspector().Queues()
	if err != nil {
		log.Errorf("list queues failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list queues failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queues": queues})
}

func (s *Server) getQueueInfo(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue name required"})
		return
	}

	info, err := s.taskManager.Inspector().GetQueueInfo(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"queue": info.Queue, "size": info.Size, "pending": info.Pending, "active": info.Active, "scheduled": info.Scheduled, "retry": info.Retry, "archived": info.Archived, "completed": info.Completed, "processed": info.Processed, "failed": info.Failed, "latency": info.Latency.String(), "timestamp": info.Timestamp})
}

func (s *Server) systemInfo(c *gin.Context) {
	port := ""
	if s.config.Asynctask != nil {
		port = s.config.Asynctask.HttpPort
	}
	if port == "" {
		port = s.config.HttpPort
	}
	redisAddr := ""
	redisDB := ""
	if s.config.Redis != nil {
		redisAddr = s.config.Redis.Address
		redisDB = s.config.Redis.Database
	}
	c.JSON(http.StatusOK, gin.H{
		"http_port":     port,
		"redis_address": redisAddr,
		"redis_db":      redisDB,
		"time":          time.Now().Format(time.RFC3339),
	})
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Unix()})
}

func toVideoResponse(v *models.VideoGen) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"id":         v.ID,
		"task_id":    v.TaskId,
		"uuid":       v.UUID,
		"prompt":     v.Prompt,
		"video_url":  v.VideoUrl,
		"thumbnail":  v.FisrtFrame,
		"status":     v.GenStatus,
		"message":    v.Message,
		"code":       v.Code,
		"provider":   v.Provider,
		"user_id":    v.UserID,
		"story_id":   v.StoryId,
		"board_id":   v.BoardId,
		"scene_id":   v.SceneId,
		"role_id":    v.RoleId,
		"duration":   v.Timelength,
		"start_time": v.StartTime,
		"end_time":   v.EndTime,
		"ref_images": v.RefImages,
	}
}
