package http

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// plazaSectionDTO matches voyager PlazaService decoding (titleKey → Localizable).
type plazaSectionDTO struct {
	ID            string          `json:"id"`
	TitleKey      string          `json:"titleKey"`
	SubtitleKey   string          `json:"subtitleKey,omitempty"`
	BadgeKey      string          `json:"badgeKey,omitempty"`
	BadgeColor    string          `json:"badgeColor,omitempty"`
	CreatorUserID string          `json:"creatorUserId,omitempty"`
	AvatarURL     string          `json:"avatarUrl,omitempty"`
	// TopicTag 与 `GET /fragments?topic=` 检索串一致；客户端可优先用此字段拉碎片预览。
	TopicTag string `json:"topicTag,omitempty"`
	Stories  []*domain.Story `json:"stories"`
}

type plazaSectionDef struct {
	ID          string
	SearchTag   string
	TitleKey    string
	SubtitleKey string
	BadgeKey    string
	BadgeColor  string
}

var plazaSectionDefs = []plazaSectionDef{
	{"fantasy", "奇幻", "discover_topic_fantasy", "discover_topic_fantasy_subtitle", "discover_hot", "like"},
	{"scifi", "科幻", "discover_topic_scifi", "discover_topic_scifi_subtitle", "discover_new", "accent"},
	{"mystery", "悬疑", "discover_topic_mystery", "discover_topic_mystery_subtitle", "", "purple"},
	{"wuxia", "武侠", "discover_topic_wuxia", "discover_topic_wuxia_subtitle", "discover_trending", "orange"},
	{"romance", "青春", "discover_topic_romance", "discover_topic_romance_subtitle", "", "pink"},
	{"horror", "恐怖", "discover_topic_horror", "discover_topic_horror_subtitle", "", "muted"},
}

// GetPlaza aggregates discover-style sections in one round trip.
func (h *Handler) GetPlaza(c *gin.Context) {
	out := buildPlazaSections(c.Request.Context(), h)
	Success(c, gin.H{"sections": out})
}

func buildPlazaSections(ctx context.Context, h *Handler) []plazaSectionDTO {
	out := make([]plazaSectionDTO, 0, len(plazaSectionDefs))
	for _, def := range plazaSectionDefs {
		stories, err := h.svc.SearchStories(ctx, def.SearchTag, service.SearchTypeFuzzy, 12, 0)
		if err != nil {
			h.logger.Warn("plaza section search failed", zap.String("sectionId", def.ID), zap.Error(err))
			stories = nil
		}
		dto := plazaSectionDTO{
			ID:          def.ID,
			TitleKey:    def.TitleKey,
			SubtitleKey: def.SubtitleKey,
			BadgeKey:    def.BadgeKey,
			BadgeColor:  def.BadgeColor,
			TopicTag:    def.SearchTag,
			Stories:     stories,
		}
		if len(stories) > 0 {
			uid, avatar := unifiedCreatorFromStories(stories)
			dto.CreatorUserID = uid
			dto.AvatarURL = avatar
		}
		out = append(out, dto)
	}
	return out
}

func unifiedCreatorFromStories(stories []*domain.Story) (userID, avatarURL string) {
	if len(stories) == 0 {
		return "", ""
	}
	first := stories[0].UserID
	if first == "" {
		return "", ""
	}
	for _, s := range stories[1:] {
		if s == nil || s.UserID != first {
			return "", ""
		}
	}
	uid := first
	if stories[0].Author != nil && stories[0].Author.Avatar != "" {
		return uid, stories[0].Author.Avatar
	}
	return uid, ""
}
