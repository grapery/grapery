package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// Plaza rail kinds (client `PlazaRailKind` must stay in sync).
const (
	plazaKindStoriesTrending = "stories_trending"
	plazaKindFragmentsTopic  = "fragments_topic"
)

// plazaSectionDTO matches voyager PlazaService: titleKey → Localizable, optional title/subtitle for dynamic rails.
type plazaSectionDTO struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	TitleKey      string `json:"titleKey,omitempty"`
	Title         string `json:"title,omitempty"`
	SubtitleKey   string `json:"subtitleKey,omitempty"`
	Subtitle      string `json:"subtitle,omitempty"`
	BadgeKey      string `json:"badgeKey,omitempty"`
	BadgeColor    string `json:"badgeColor,omitempty"`
	CreatorUserID string `json:"creatorUserId,omitempty"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
	// TopicTag used when kind == fragments_topic; same semantics as GET /fragments?topic=
	TopicTag string `json:"topicTag,omitempty"`
	// Stories embedded preview for story rails (trending / latest published).
	Stories []*domain.Story `json:"stories,omitempty"`
	// Fragments embedded preview for fragment rails.
	Fragments []*domain.Fragment `json:"fragments,omitempty"`
}

const plazaPreviewLimit = 12

var plazaTopicRailBadgeColors = []string{"like", "accent", "purple", "orange", "pink", "muted"}

// GetPlaza returns algorithm/config-driven rails in one round trip with embedded previews.
func (h *Handler) GetPlaza(c *gin.Context) {
	out := buildPlazaSections(c.Request.Context(), h)
	Success(c, gin.H{"sections": out})
}

func buildPlazaSections(ctx context.Context, h *Handler) []plazaSectionDTO {
	out := make([]plazaSectionDTO, 0, 1+8)

	// 1) Trending stories (unchanged keys for localization).
	trending := plazaSectionDTO{
		ID:          "stories_trending",
		Kind:        plazaKindStoriesTrending,
		TitleKey:    "plaza_rail_trending_title",
		SubtitleKey: "plaza_rail_trending_subtitle",
		BadgeKey:    "discover_hot",
		BadgeColor:  "like",
	}
	stories, err := h.svc.GetTrendingStories24h(ctx, plazaPreviewLimit)
	if err != nil {
		h.logger.Warn("plaza rail failed", zap.String("kind", plazaKindStoriesTrending), zap.Error(err))
		stories = nil
	}
	trending.Stories = stories
	trending.AvatarURL = firstStoryAuthorAvatar(stories)
	out = append(out, trending)

	// 2) Dynamic topic rails (MySQL top topics + Redis read-through in service).
	topics, err := h.svc.TopPublicFragmentTopicLabelsForPlaza(ctx)
	if err != nil {
		h.logger.Warn("plaza top fragment topics failed", zap.Error(err))
		topics = nil
	}
	for i, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		sum := sha256.Sum256([]byte(topic))
		secID := "fragments_topic_" + hex.EncodeToString(sum[:8])
		dto := plazaSectionDTO{
			ID:          secID,
			Kind:        plazaKindFragmentsTopic,
			Title:       plazaTopicDisplayTitle(topic),
			SubtitleKey: "plaza_rail_topic_generic_subtitle",
			BadgeColor:  plazaTopicRailBadgeColors[i%len(plazaTopicRailBadgeColors)],
			TopicTag:    topic,
		}
		frags, ferr := h.svc.ListPublicFragmentsByTopicPreview(ctx, topic, plazaPreviewLimit)
		if ferr != nil {
			h.logger.Warn("plaza rail failed", zap.String("kind", plazaKindFragmentsTopic), zap.String("topic", topic), zap.Error(ferr))
			frags = nil
		}
		dto.Fragments = frags
		dto.AvatarURL = firstFragmentAuthorAvatar(frags)
		out = append(out, dto)
	}

	return out
}

func plazaTopicDisplayTitle(topic string) string {
	t := strings.TrimSpace(topic)
	if t == "" {
		return ""
	}
	if strings.HasPrefix(t, "#") {
		return t
	}
	return "#" + t
}

func firstStoryAuthorAvatar(stories []*domain.Story) string {
	if len(stories) == 0 || stories[0] == nil {
		return ""
	}
	if stories[0].Author != nil && stories[0].Author.Avatar != "" {
		return stories[0].Author.Avatar
	}
	return ""
}

func firstFragmentAuthorAvatar(fragments []*domain.Fragment) string {
	if len(fragments) == 0 || fragments[0] == nil {
		return ""
	}
	if fragments[0].Author != nil && fragments[0].Author.Avatar != "" {
		return fragments[0].Author.Avatar
	}
	return ""
}
