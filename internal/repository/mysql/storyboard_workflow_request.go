package mysql

import (
	"context"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// Search the whole board history: a retry for turn A must not create another
// run merely because turn B became latest. Parameters stay bound SQL values.
func (r *Repository) FindStoryboardGenerationRunsByClientRequest(ctx context.Context, boardID, userID, key string) ([]*domain.StoryboardGenerationRun, error) {
	query := r.db.WithContext(ctx).Where("storyboard_id = ? AND user_id = ?", boardID, userID)
	expression := "JSON_UNQUOTE(JSON_EXTRACT(request_json, '$.clientRequestId')) = ?"
	if r.db.Dialector.Name() == "sqlite" {
		expression = "json_extract(request_json, '$.clientRequestId') = ?"
	}
	var models []StoryboardGenerationRun
	if err := query.Where(expression, strings.TrimSpace(key)).Order("created_at ASC, id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.StoryboardGenerationRun, 0, len(models))
	for _, model := range models {
		out = append(out, modelToStoryboardGenerationRun(&model))
	}
	return out, nil
}
