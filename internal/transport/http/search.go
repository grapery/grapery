package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

func (h *Handler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		InvalidParams(c, "search query is required")
		return
	}

	searchType := c.DefaultQuery("type", "all")
	mode := c.DefaultQuery("mode", "fuzzy") // 搜索模式：fuzzy（模糊）或 exact（精确）
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 解析搜索模式
	var searchMode service.SearchType
	switch mode {
	case "exact":
		searchMode = service.SearchTypeExact
	case "fuzzy":
		fallthrough
	default:
		searchMode = service.SearchTypeFuzzy
	}

	switch searchType {
	case "story", "stories":
		stories, err := h.svc.SearchStories(c.Request.Context(), query, searchMode, limit, offset)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{
			"stories":    stories,
			"total":      len(stories),
			"searchMode": mode,
		})
	case "character", "characters":
		characters, err := h.svc.SearchCharacters(c.Request.Context(), query, searchMode, limit, offset)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{
			"characters": characters,
			"total":      len(characters),
			"searchMode": mode,
		})
	case "user", "users":
		users, err := h.svc.SearchUsers(c.Request.Context(), query, searchMode, limit, offset)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{
			"users":      users,
			"total":      len(users),
			"searchMode": mode,
		})
	case "group", "groups":
		groups, err := h.svc.SearchGroups(c.Request.Context(), query, searchMode, limit, offset)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{
			"groups":     groups,
			"total":      len(groups),
			"searchMode": mode,
		})
	case "all":
		results, err := h.svc.SearchAll(c.Request.Context(), query, searchMode, limit)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		results["searchMode"] = mode
		Success(c, results)
	default:
		InvalidParams(c, "invalid search type")
	}
}
