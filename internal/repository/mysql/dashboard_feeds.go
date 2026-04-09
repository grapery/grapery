package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/recommendation"
	"go.uber.org/zap"
)

// REMOVED: DashboardStoryboards - not in StoryCreationAppUI design
// REMOVED: DashboardCharacterStoryboards - not in StoryCreationAppUI design
// REMOVED: TrendingStoryboards (authenticated) - not in StoryCreationAppUI design

// GetPublicTrendingStoryboards returns published trending storyboards accessible to all users.
// Returns ALL published storyboards ordered by popularity metrics (likes, storyboard_count, followers, updated_at).
// If userID is provided (authenticated), user contributions are prioritized in the ranking.
func (r *Repository) GetPublicTrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Build base query for ALL published storyboards (no time or count restrictions)
	base := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Joins("JOIN stories ON stories.id = storyboards.story_id").
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished)

	// If user is authenticated, add personalization with story_contributors
	if userID != "" {
		base = base.Joins(`LEFT JOIN story_contributors sc ON sc.story_id = stories.id AND sc.user_id = ?`, userID)
	}

	// Count total distinct storyboards
	var total int64
	if err := base.
		Select("storyboards.id").
		Distinct().
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("public trending storyboards count: %w", err)
	}

	// Get page of storyboard IDs
	type idRow struct {
		ID string `gorm:"column:id"`
	}
	var idRows []idRow

	// Order by popularity metrics - prioritized for authenticated users
	var orderClause string
	if userID != "" {
		// Authenticated: prioritize user contributions, then by story metrics
		orderClause = `
			MAX(CASE WHEN sc.user_id IS NOT NULL THEN 1 ELSE 0 END) DESC,
			MAX(stories.likes) DESC,
			MAX(stories.storyboard_count) DESC,
			MAX(stories.followers) DESC,
			MAX(storyboards.updated_at) DESC
		`
	} else {
		// Guest: global ranking by popularity (no time restrictions)
		orderClause = `
			MAX(stories.likes) DESC,
			MAX(stories.storyboard_count) DESC,
			MAX(stories.followers) DESC,
			MAX(storyboards.updated_at) DESC
		`
	}

	if err := base.
		Select("storyboards.id").
		Group("storyboards.id").
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Scan(&idRows).Error; err != nil {
		return nil, 0, fmt.Errorf("public trending storyboards ids: %w", err)
	}

	ids := make([]string, 0, len(idRows))
	for _, row := range idRows {
		ids = append(ids, row.ID)
	}

	if len(ids) == 0 {
		return []*domain.Storyboard{}, total, nil
	}

	// Load storyboards with relations (do not re-order by updated_at — preserve ranking from id query above)
	var rows []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Preload("Story.Author").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("public trending storyboards load: %w", err)
	}

	byID := make(map[string]Storyboard, len(rows))
	for _, sb := range rows {
		byID[sb.ID] = sb
	}
	result := make([]*domain.Storyboard, 0, len(idRows))
	for _, id := range ids {
		sb, ok := byID[id]
		if !ok {
			continue
		}
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, 0, err
		}
		copySb := domainSb
		result = append(result, &copySb)
	}
	return result, total, nil
}

// StoryboardFeedRecommended is the “for you” tab: onboarding preferred genres + public engagement fallback; guests get trending.
func (r *Repository) StoryboardFeedRecommended(ctx context.Context, userID string, limit, offset int, exclude map[string]struct{}) ([]*domain.Storyboard, int64, error) {
	if userID == "" {
		return r.GetPublicTrendingStoryboards(ctx, userID, limit, offset)
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	genres, err := r.preferredGenres(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("recommended preferred genres: %w", err)
	}

	target := offset + limit
	var filtered []string
	var lastMult int
	var lastMergedLen, lastPoolGenre, lastPoolFallback int
	var lastSeenExcluded int
	for mult := 1; mult <= 32; mult *= 2 {
		mergeNeeded := target * mult
		if mergeNeeded < 40 {
			mergeNeeded = 40
		}
		mergedIDs, pg, pf, err := r.buildStoryboardPreferenceMerged(ctx, genres, mergeNeeded)
		if err != nil {
			return nil, 0, err
		}
		afterSeen := recommendation.FilterExcludedOrderedIDs(mergedIDs, exclude)
		filtered = afterSeen
		lastMult = mult
		lastMergedLen = len(mergedIDs)
		lastPoolGenre, lastPoolFallback = pg, pf
		lastSeenExcluded = len(mergedIDs) - len(afterSeen)
		if len(filtered) >= target || mult >= 32 {
			break
		}
	}

	pagedIDs := storyboardPageIDs(filtered, offset, limit)
	rows, err := r.storyboardsByIDsInOrder(ctx, pagedIDs)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.totalPublishedStoryboards(ctx)
	if err != nil {
		return nil, 0, err
	}
	seenSetSize := 0
	if exclude != nil {
		seenSetSize = len(exclude)
	}
	r.log.Info("storyboard for_you recommendation generated",
		zap.String("userID", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.Int("preferred_genre_count", len(genres)),
		zap.Int("seen_set_size", seenSetSize),
		zap.Int("oversample_multiplier", lastMult),
		zap.Int("merged_raw_count", lastMergedLen),
		zap.Int("pool_genre_match", lastPoolGenre),
		zap.Int("pool_fallback", lastPoolFallback),
		zap.Int("seen_excluded_count", lastSeenExcluded),
		zap.Int("after_exclude_count", len(filtered)),
		zap.Int("returned_count", len(rows)),
		zap.Int64("total_published_storyboards", total))
	return rows, total, nil
}

// StoryboardFeedDiscover is the discover tab: only onboarding genres (no fallback, no seen filter); guests get trending.
func (r *Repository) StoryboardFeedDiscover(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	if userID == "" {
		return r.GetPublicTrendingStoryboards(ctx, userID, limit, offset)
	}
	genres, err := r.preferredGenres(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	if len(genres) == 0 {
		return []*domain.Storyboard{}, 0, nil
	}
	publicVis := string(domain.StoryVisibilityPublic)
	q := r.db.WithContext(ctx).Table("storyboards").
		Joins("JOIN stories ON stories.id = storyboards.story_id AND stories.deleted_at IS NULL").
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished).
		Where("stories.visibility = ?", publicVis).
		Where("stories.genre IN ?", genres)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ids []string
	if err := r.db.WithContext(ctx).Table("storyboards").
		Select("storyboards.id").
		Joins("JOIN stories ON stories.id = storyboards.story_id AND stories.deleted_at IS NULL").
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished).
		Where("stories.visibility = ?", publicVis).
		Where("stories.genre IN ?", genres).
		Order("storyboards.updated_at DESC").
		Limit(limit).
		Offset(offset).
		Pluck("storyboards.id", &ids).Error; err != nil {
		return nil, 0, err
	}
	rows, err := r.storyboardsByIDsInOrder(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// buildStoryboardPreferenceMerged merges onboarding genre matches with a popularity fallback pool only.
func (r *Repository) buildStoryboardPreferenceMerged(ctx context.Context, genres []string, mergeNeeded int) ([]string, int, int, error) {
	multiplier := r.recoCfg.CandidateMultiplier
	if multiplier < 2 {
		multiplier = 4
	}
	candidateLimit := mergeNeeded * multiplier
	if candidateLimit < 40 {
		candidateLimit = 40
	}
	poolGenre, err := r.storyboardIDsByGenres(ctx, genres, candidateLimit)
	if err != nil {
		return nil, 0, 0, err
	}
	poolFallback, err := r.storyboardFallbackIDs(ctx, candidateLimit)
	if err != nil {
		return nil, 0, 0, err
	}
	if len(genres) == 0 {
		merged := poolFallback
		if len(merged) > mergeNeeded {
			merged = merged[:mergeNeeded]
		}
		return merged, 0, len(poolFallback), nil
	}
	genreRatio := r.recoCfg.StoryboardGenreRatio
	fallbackRatio := r.recoCfg.StoryboardFallbackRatio
	if genreRatio <= 0 || fallbackRatio <= 0 {
		genreRatio, fallbackRatio = 7, 3
	}
	var empty []string
	mergedIDs := mergeStoryboardIDs(empty, poolGenre, poolFallback, 0, genreRatio, fallbackRatio, mergeNeeded)
	if len(mergedIDs) < mergeNeeded {
		mergedIDs = mergeStoryboardIDs(empty, poolGenre, poolFallback, 0, genreRatio, fallbackRatio, mergeNeeded*2)
	}
	if len(mergedIDs) < mergeNeeded {
		mergedIDs = appendStoryboardUnique(mergedIDs, poolGenre, mergeNeeded)
		mergedIDs = appendStoryboardUnique(mergedIDs, poolFallback, mergeNeeded)
	}
	return mergedIDs, len(poolGenre), len(poolFallback), nil
}

func (r *Repository) storyboardIDsByGenres(ctx context.Context, genres []string, limit int) ([]string, error) {
	if len(genres) == 0 {
		return []string{}, nil
	}
	publicVis := string(domain.StoryVisibilityPublic)
	var ids []string
	err := r.db.WithContext(ctx).
		Table("storyboards").
		Select("storyboards.id").
		Joins("JOIN stories ON stories.id = storyboards.story_id AND stories.deleted_at IS NULL").
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished).
		Where("stories.visibility = ?", publicVis).
		Where("stories.genre IN ?", genres).
		Order("storyboards.updated_at DESC").
		Limit(limit).
		Pluck("storyboards.id", &ids).Error
	return ids, err
}

func (r *Repository) storyboardFallbackIDs(ctx context.Context, limit int) ([]string, error) {
	publicVis := string(domain.StoryVisibilityPublic)
	var ids []string
	err := r.db.WithContext(ctx).
		Table("storyboards").
		Select("storyboards.id").
		Joins("JOIN stories ON stories.id = storyboards.story_id AND stories.deleted_at IS NULL").
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished).
		Where("stories.visibility = ?", publicVis).
		Order("storyboards.likes DESC, storyboards.views DESC, storyboards.fork_count DESC, storyboards.updated_at DESC").
		Limit(limit).
		Pluck("storyboards.id", &ids).Error
	return ids, err
}

func (r *Repository) storyboardsByIDsInOrder(ctx context.Context, ids []string) ([]*domain.Storyboard, error) {
	if len(ids) == 0 {
		return []*domain.Storyboard{}, nil
	}
	var rows []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Preload("Story.Author").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := make(map[string]Storyboard, len(rows))
	for _, sb := range rows {
		byID[sb.ID] = sb
	}
	out := make([]*domain.Storyboard, 0, len(ids))
	for _, id := range ids {
		sb, ok := byID[id]
		if !ok {
			continue
		}
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, err
		}
		copySb := domainSb
		out = append(out, &copySb)
	}
	return out, nil
}

func (r *Repository) totalPublishedStoryboards(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&Storyboard{}).Where("workflow_status = ?", domain.WorkflowStatusPublished).Count(&total).Error
	return total, err
}

func mergeStoryboardIDs(poolA, poolB, poolC []string, ra, rb, rc, needed int) []string {
	if needed <= 0 {
		return []string{}
	}
	out := make([]string, 0, needed)
	seen := map[string]struct{}{}
	ia, ib, ic := 0, 0, 0
	appendPool := func(pool []string, idx *int, n int) {
		for i := 0; i < n && *idx < len(pool) && len(out) < needed; i++ {
			id := strings.TrimSpace(pool[*idx])
			*idx = *idx + 1
			if id == "" {
				i--
				continue
			}
			if _, ok := seen[id]; ok {
				i--
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	for len(out) < needed && (ia < len(poolA) || ib < len(poolB) || ic < len(poolC)) {
		appendPool(poolA, &ia, ra)
		appendPool(poolB, &ib, rb)
		appendPool(poolC, &ic, rc)
	}
	return out
}

func appendStoryboardUnique(base, pool []string, needed int) []string {
	if len(base) >= needed {
		return base
	}
	seen := map[string]struct{}{}
	for _, id := range base {
		seen[id] = struct{}{}
	}
	for _, id := range pool {
		if len(base) >= needed {
			break
		}
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		base = append(base, id)
	}
	return base
}

func storyboardPageIDs(ids []string, offset, limit int) []string {
	if offset >= len(ids) {
		return []string{}
	}
	end := offset + limit
	if end > len(ids) {
		end = len(ids)
	}
	return ids[offset:end]
}

func (r *Repository) preferredGenres(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return []string{}, nil
	}
	var ns sql.NullString
	err := r.db.WithContext(ctx).Table("user_settings").Select("preferred_genres_json").Where("user_id = ?", userID).Scan(&ns).Error
	if err != nil {
		return nil, err
	}
	var genres []string
	if !ns.Valid || strings.TrimSpace(ns.String) == "" {
		return []string{}, nil
	}
	if err := json.Unmarshal([]byte(ns.String), &genres); err != nil {
		return []string{}, nil
	}
	return genres, nil
}
