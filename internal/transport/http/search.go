package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		InvalidParams(c, "search query is required")
		return
	}

	searchType := c.DefaultQuery("type", "all")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	switch searchType {
	case "story", "stories":
		stories, err := h.svc.SearchStories(c.Request.Context(), query, limit, offset)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{"stories": stories, "total": len(stories)})
	case "character", "characters":
		characters, err := h.svc.SearchCharacters(c.Request.Context(), query, limit, offset)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{"characters": characters, "total": len(characters)})
	case "user", "users":
		users, err := h.svc.SearchUsers(c.Request.Context(), query, limit, offset)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{"users": users, "total": len(users)})
	case "group", "groups":
		groups, err := h.svc.SearchGroups(c.Request.Context(), query, limit, offset)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, gin.H{"groups": groups, "total": len(groups)})
	case "all":
		results, err := h.svc.SearchAll(c.Request.Context(), query, 10)
		if err != nil {
			InternalError(c, err.Error())
			return
		}
		Success(c, results)
	default:
		InvalidParams(c, "invalid search type")
	}
}
