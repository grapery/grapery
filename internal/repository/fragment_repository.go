package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/recommendation"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type FragmentRepository struct {
	db      *gorm.DB
	recoCfg config.RecommendationConfig
	cache   cache.Cache
	logger  *zap.Logger
}

type FragmentEngagement struct {
	Likes    int
	Comments int
	IsLiked  bool
}

func NewFragmentRepository(db *gorm.DB, recoCfg config.RecommendationConfig, c cache.Cache, logger *zap.Logger) *FragmentRepository {
	return &FragmentRepository{
		db:      db,
		recoCfg: recoCfg,
		cache:   c,
		logger:  logger,
	}
}

// DB exposes the underlying GORM handle for cross-repository transactions.
func (r *FragmentRepository) DB() *gorm.DB {
	return r.db
}

// Create creates a new fragment and increments user's fragments count
func (r *FragmentRepository) Create(ctx context.Context, fragment *domain.Fragment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.CreateWithTx(ctx, tx, fragment)
	})
}

// CreateWithTx inserts a fragment inside an existing transaction (no nested transaction).
func (r *FragmentRepository) CreateWithTx(ctx context.Context, tx *gorm.DB, fragment *domain.Fragment) error {
	if tx == nil {
		return fmt.Errorf("nil transaction")
	}
	dbFragment := mysql.DomainToFragmentDB(fragment)
	if dbFragment == nil {
		return domain.ErrInvalidInput
	}

	userID := fragment.UserID
	if userID == "" {
		userID = fragment.CreatorID
	}

	tx = tx.WithContext(ctx)
	if err := tx.Create(dbFragment).Error; err != nil {
		return err
	}

	fragment.ID = dbFragment.ID
	fragment.CreatedAt = dbFragment.CreatedAt
	fragment.UpdatedAt = dbFragment.UpdatedAt

	if !fragment.IsDraft {
		if err := tx.Model(&domain.User{}).
			Where("id = ?", userID).
			UpdateColumn("fragments_count", gorm.Expr("fragments_count + 1")).
			Error; err != nil {
			return err
		}
	}

	return nil
}

// GetByID retrieves a fragment by ID
func (r *FragmentRepository) GetByID(ctx context.Context, id string) (*domain.Fragment, error) {
	var fragment mysql.FragmentDB
	err := r.db.WithContext(ctx).Preload("Creator").Where("id = ?", id).First(&fragment).Error
	if err != nil {
		return nil, err
	}
	return mysql.FragmentDBToDomain(&fragment), nil
}

// GetBySource returns the newest fragment row for the given source_type + source_id (e.g. AI 任务草稿).
func (r *FragmentRepository) GetBySource(ctx context.Context, sourceType, sourceID string) (*domain.Fragment, error) {
	var fragment mysql.FragmentDB
	err := r.db.WithContext(ctx).Preload("Creator").
		Where("source_type = ? AND source_id = ?", sourceType, sourceID).
		Order("created_at DESC").
		First(&fragment).Error
	if err != nil {
		return nil, err
	}
	return mysql.FragmentDBToDomain(&fragment), nil
}

// List retrieves fragments with pagination
func (r *FragmentRepository) List(ctx context.Context, limit, offset int, visibility string) ([]*domain.Fragment, int64, error) {
	var dbFragments []*mysql.FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("COALESCE(is_draft, 0) = ?", 0)

	if visibility != "" {
		query = query.Where("visibility = ?", visibility)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

func fragmentDBListToDomain(list []*mysql.FragmentDB) []*domain.Fragment {
	if list == nil {
		return nil
	}
	result := make([]*domain.Fragment, len(list))
	for i, f := range list {
		result[i] = mysql.FragmentDBToDomain(f)
	}
	return result
}

// ListByCreatorID retrieves published fragments by creator ID (excludes drafts).
func (r *FragmentRepository) ListByCreatorID(ctx context.Context, creatorID string, limit, offset int) ([]*domain.Fragment, int64, error) {
	var dbFragments []*mysql.FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("creator_id = ? AND COALESCE(is_draft, 0) = ?", creatorID, 0)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

// ListDraftsByCreatorID returns draft fragments for a user (is_draft = 1).
func (r *FragmentRepository) ListDraftsByCreatorID(ctx context.Context, creatorID string, limit, offset int) ([]*domain.Fragment, int64, error) {
	var dbFragments []*mysql.FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("creator_id = ? AND is_draft = ?", creatorID, true)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

// ListByTopic retrieves fragments by topic with optional converted filter
// convertedOnly: nil = all, true = only converted, false = only not converted
func (r *FragmentRepository) ListByTopic(ctx context.Context, topic string, limit, offset int, convertedOnly *bool) ([]*domain.Fragment, int64, error) {
	var dbFragments []*mysql.FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("topic = ? AND visibility = ? AND COALESCE(is_draft, 0) = ?", topic, domain.FragmentVisibilityPublic, 0)

	if convertedOnly != nil {
		if *convertedOnly {
			query = query.Where("is_converted = ?", true)
		} else {
			query = query.Where("is_converted = ?", false)
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

// ListFollowing lists fragments newest first: the viewer’s own (any visibility), plus public fragments
// from users the viewer follows (user_follows). Self is always included like an implicit self-follow.
func (r *FragmentRepository) ListFollowing(ctx context.Context, userID string, limit, offset int) ([]*domain.Fragment, int64, error) {
	if userID == "" {
		return []*domain.Fragment{}, 0, nil
	}

	followeeSubq := func() *gorm.DB {
		return r.db.WithContext(ctx).Table("user_follows").
			Select("followee_id").
			Where("follower_id = ? AND deleted_at IS NULL", userID)
	}

	followingScope := func(q *gorm.DB) *gorm.DB {
		return q.Where("COALESCE(is_draft, 0) = ?", 0).
			Where("(creator_id = ?) OR (visibility = ? AND creator_id IN (?))",
				userID, domain.FragmentVisibilityPublic, followeeSubq())
	}

	var total int64
	if err := followingScope(r.db.WithContext(ctx).Model(&mysql.FragmentDB{})).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var dbFragments []*mysql.FragmentDB
	if err := followingScope(r.db.WithContext(ctx).Model(&mysql.FragmentDB{})).
		Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error; err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

// ListDiscoverFragmentsForUser returns the discover feed: public fragments by time for all users
// (no onboarding genre filter; userID is ignored for listing).
func (r *FragmentRepository) ListDiscoverFragmentsForUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Fragment, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	_ = userID
	return r.List(ctx, limit, offset, domain.FragmentVisibilityPublic)
}

type fragmentRecoCachePayload struct {
	Fragments []*domain.Fragment `json:"fragments"`
	Total     int64              `json:"total"`
}

// RecordFragmentForYouSeen records a fragment detail view for for_you exclusion (no-op if cache nil).
func (r *FragmentRepository) RecordFragmentForYouSeen(ctx context.Context, userID, fragmentID string) {
	recommendation.RecordFragmentSeen(ctx, r.cache, r.recoCfg, userID, fragmentID)
}

func (r *FragmentRepository) ListRecommendedForUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Fragment, int64, error) {
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
		return r.List(ctx, limit, offset, domain.FragmentVisibilityPublic)
	}

	page := (offset / limit) + 1
	cacheTTL := time.Duration(r.recoCfg.CacheTTLSeconds) * time.Second
	if cacheTTL <= 0 {
		cacheTTL = 3 * time.Minute
	}
	cacheKey := recommendation.FeedCacheKey(recommendation.FeedTypeFragments, userID, page, limit)
	if r.cache != nil {
		var cached fragmentRecoCachePayload
		if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
			if r.logger != nil {
				r.logger.Debug("fragment for_you feed cache hit",
					zap.String("userID", userID),
					zap.Int("count", len(cached.Fragments)))
			}
			return cached.Fragments, cached.Total, nil
		}
		if r.logger != nil {
			r.logger.Debug("fragment for_you feed cache miss",
				zap.String("userID", userID),
				zap.String("cacheKey", cacheKey))
		}
	}

	preferredGenres, err := r.preferredGenres(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	var exclude map[string]struct{}
	if r.cache != nil {
		seen, errSeen := recommendation.ListSeenFragmentIDs(ctx, r.cache, r.recoCfg, userID)
		if errSeen != nil {
			if r.logger != nil {
				r.logger.Warn("list seen fragments for feed", zap.Error(errSeen))
			}
		} else {
			exclude = recommendation.SeenIDsToSet(seen)
		}
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
		mergedIDs, pg, pf, errMerge := r.buildFragmentPreferenceMerged(ctx, preferredGenres, mergeNeeded)
		if errMerge != nil {
			return nil, 0, errMerge
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

	pagedIDs := paginateIDs(filtered, offset, limit)
	fragments, err := r.fragmentsByIDsInOrder(ctx, pagedIDs)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.publicFragmentCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	if r.cache != nil {
		payload := fragmentRecoCachePayload{Fragments: fragments, Total: total}
		_ = r.cache.Set(ctx, cacheKey, payload, cacheTTL)
		recommendation.TrackFeedCacheKey(ctx, r.cache, recommendation.FeedTypeFragments, userID, cacheKey, cacheTTL)
	}
	if r.logger != nil {
		r.logger.Info("fragment for_you recommendation generated",
			zap.String("userID", userID),
			zap.Int("offset", offset),
			zap.Int("limit", limit),
			zap.Int("preferred_genre_count", len(preferredGenres)),
			zap.Int("seen_set_size", mapLen(exclude)),
			zap.Int("oversample_multiplier", lastMult),
			zap.Int("merged_raw_count", lastMergedLen),
			zap.Int("pool_genre_match", lastPoolGenre),
			zap.Int("pool_fallback", lastPoolFallback),
			zap.Int("seen_excluded_count", lastSeenExcluded),
			zap.Int("after_filters_count", len(filtered)),
			zap.Int("returned_count", len(fragments)),
			zap.Int64("total_public_fragments", total))
	}
	return fragments, total, nil
}

// buildFragmentPreferenceMerged merges onboarding genre preferences with a popularity fallback pool only.
func (r *FragmentRepository) buildFragmentPreferenceMerged(ctx context.Context, preferredGenres []string, mergeNeeded int) ([]string, int, int, error) {
	multiplier := r.recoCfg.CandidateMultiplier
	if multiplier < 2 {
		multiplier = 4
	}
	candidateLimit := mergeNeeded * multiplier
	if candidateLimit < 40 {
		candidateLimit = 40
	}
	poolGenre, err := r.fragmentIDsFromPreferredGenres(ctx, preferredGenres, candidateLimit)
	if err != nil {
		return nil, 0, 0, err
	}
	poolFallback, err := r.fragmentFallbackIDs(ctx, candidateLimit)
	if err != nil {
		return nil, 0, 0, err
	}
	if len(preferredGenres) == 0 {
		merged := poolFallback
		if len(merged) > mergeNeeded {
			merged = merged[:mergeNeeded]
		}
		return merged, 0, len(poolFallback), nil
	}
	genreRatio := r.recoCfg.FragmentGenreRatio
	fallbackRatio := r.recoCfg.FragmentFallbackRatio
	if genreRatio <= 0 || fallbackRatio <= 0 {
		genreRatio, fallbackRatio = 7, 3
	}
	var empty []string
	mergedIDs := interleaveByRatio(empty, poolGenre, poolFallback, 0, genreRatio, fallbackRatio, mergeNeeded)
	if len(mergedIDs) < mergeNeeded {
		mergedIDs = interleaveByRatio(empty, poolGenre, poolFallback, 0, genreRatio, fallbackRatio, mergeNeeded*2)
	}
	if len(mergedIDs) < mergeNeeded {
		mergedIDs = appendUniqueIDs(mergedIDs, poolGenre, mergeNeeded)
		mergedIDs = appendUniqueIDs(mergedIDs, poolFallback, mergeNeeded)
	}
	return mergedIDs, len(poolGenre), len(poolFallback), nil
}

func (r *FragmentRepository) preferredGenres(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return []string{}, nil
	}
	var ns sql.NullString
	err := r.db.WithContext(ctx).Table("user_settings").Select("preferred_genres_json").Where("user_id = ?", userID).Scan(&ns).Error
	if err != nil {
		return nil, err
	}
	raw := ""
	if ns.Valid {
		raw = ns.String
	}
	var genres []string
	_ = jsonUnmarshalStringArray(raw, &genres)
	return genres, nil
}

func (r *FragmentRepository) fragmentIDsFromPreferredGenres(ctx context.Context, genres []string, limit int) ([]string, error) {
	if len(genres) == 0 {
		return []string{}, nil
	}
	var ids []string
	err := r.db.WithContext(ctx).
		Table("fragments f").
		Select("f.id").
		Joins("LEFT JOIN storyboards sb ON f.source_type = ? AND f.source_id = sb.id", domain.FragmentSourceStoryboardNode).
		Joins("LEFT JOIN stories st ON ((f.source_type = ? AND f.source_id = st.id) OR (f.source_type = ? AND sb.story_id = st.id))",
			domain.FragmentSourceStoryExcerpt, domain.FragmentSourceStoryboardNode).
		Where("f.visibility = ?", domain.FragmentVisibilityPublic).
		Where("COALESCE(f.is_draft, 0) = ?", 0).
		Where("st.genre IN ?", genres).
		Order("f.created_at DESC").
		Limit(limit).
		Pluck("f.id", &ids).Error
	return ids, err
}

func (r *FragmentRepository) fragmentFallbackIDs(ctx context.Context, limit int) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Table("fragments").
		Where("visibility = ?", domain.FragmentVisibilityPublic).
		Where("COALESCE(is_draft, 0) = ?", 0).
		Order("likes DESC, comments DESC, created_at DESC").
		Limit(limit).
		Pluck("id", &ids).Error
	return ids, err
}

func (r *FragmentRepository) fragmentsByIDsInOrder(ctx context.Context, ids []string) ([]*domain.Fragment, error) {
	if len(ids) == 0 {
		return []*domain.Fragment{}, nil
	}
	var rows []*mysql.FragmentDB
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := make(map[string]*mysql.FragmentDB, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	out := make([]*domain.Fragment, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			out = append(out, mysql.FragmentDBToDomain(row))
		}
	}
	return out, nil
}

func (r *FragmentRepository) publicFragmentCount(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("visibility = ?", domain.FragmentVisibilityPublic).
		Where("COALESCE(is_draft, 0) = ?", 0).
		Count(&total).Error
	return total, err
}

func jsonUnmarshalStringArray(raw string, out *[]string) error {
	if raw == "" {
		*out = []string{}
		return nil
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		*out = []string{}
		return err
	}
	return nil
}

func interleaveByRatio(poolA, poolB, poolC []string, ra, rb, rc, needed int) []string {
	if needed <= 0 {
		return []string{}
	}
	out := make([]string, 0, needed)
	seen := map[string]struct{}{}
	ia, ib, ic := 0, 0, 0
	appendFrom := func(pool []string, idx *int, count int) {
		for i := 0; i < count && *idx < len(pool) && len(out) < needed; i++ {
			id := pool[*idx]
			*idx = *idx + 1
			if _, ok := seen[id]; ok {
				i--
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	for len(out) < needed && (ia < len(poolA) || ib < len(poolB) || ic < len(poolC)) {
		appendFrom(poolA, &ia, ra)
		appendFrom(poolB, &ib, rb)
		appendFrom(poolC, &ic, rc)
	}
	return out
}

func appendUniqueIDs(base []string, pool []string, needed int) []string {
	if len(base) >= needed {
		return base
	}
	seen := make(map[string]struct{}, len(base))
	for _, id := range base {
		seen[id] = struct{}{}
	}
	for _, id := range pool {
		if len(base) >= needed {
			break
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		base = append(base, id)
	}
	return base
}

func paginateIDs(ids []string, offset, limit int) []string {
	if offset >= len(ids) {
		return []string{}
	}
	end := offset + limit
	if end > len(ids) {
		end = len(ids)
	}
	return ids[offset:end]
}

func mapLen(m map[string]struct{}) int {
	if m == nil {
		return 0
	}
	return len(m)
}

// IncrementUserFragmentsCount bumps users.fragments_count by 1 (e.g. draft → published).
func (r *FragmentRepository) IncrementUserFragmentsCount(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", userID).
		UpdateColumn("fragments_count", gorm.Expr("fragments_count + 1")).Error
}

// DecrementUserFragmentsCount decreases users.fragments_count by 1 (published → draft).
func (r *FragmentRepository) DecrementUserFragmentsCount(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", userID).
		UpdateColumn("fragments_count", gorm.Expr("GREATEST(fragments_count - 1, 0)")).Error
}

// Update updates a fragment
func (r *FragmentRepository) Update(ctx context.Context, fragment *domain.Fragment) error {
	dbFragment := mysql.DomainToFragmentDB(fragment)
	if dbFragment == nil {
		return domain.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Save(dbFragment).Error
}

// Delete deletes a fragment and decrements user's fragments count
func (r *FragmentRepository) Delete(ctx context.Context, id string) error {
	// First get the fragment to know the creator ID
	var fragment mysql.FragmentDB
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&fragment).Error
	if err != nil {
		return err
	}

	userID := fragment.UserID

	// Start a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete the fragment
		if err := tx.Where("id = ?", id).Delete(&mysql.FragmentDB{}).Error; err != nil {
			return err
		}

		if !fragment.IsDraft {
			if err := tx.Model(&domain.User{}).
				Where("id = ?", userID).
				UpdateColumn("fragments_count", gorm.Expr("fragments_count - 1")).
				Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// IncrementLikes increments the likes count
func (r *FragmentRepository) IncrementLikes(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("likes + 1")).
		Error
}

// DecrementLikes decrements the likes count
func (r *FragmentRepository) DecrementLikes(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("GREATEST(likes - 1, 0)")).
		Error
}

// IncrementComments increments the comments count
func (r *FragmentRepository) IncrementComments(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", id).
		UpdateColumn("comments", gorm.Expr("comments + 1")).
		Error
}

// IncrementViews increments the view count for a fragment
func (r *FragmentRepository) IncrementViews(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).
		Error
}

// CreateLike creates a like record
func (r *FragmentRepository) CreateLike(ctx context.Context, like *domain.FragmentLike) error {
	return r.db.WithContext(ctx).Create(like).Error
}

// DeleteLike deletes a like record
func (r *FragmentRepository) DeleteLike(ctx context.Context, fragmentID, userID string) error {
	return r.db.WithContext(ctx).
		Where("fragment_id = ? AND user_id = ?", fragmentID, userID).
		Delete(&domain.FragmentLike{}).Error
}

// IsLiked checks if a user has liked a fragment
func (r *FragmentRepository) IsLiked(ctx context.Context, fragmentID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.FragmentLike{}).
		Where("fragment_id = ? AND user_id = ?", fragmentID, userID).
		Count(&count).Error
	return count > 0, err
}


// ToggleLike atomically toggles like status for a fragment using delete-first strategy.
// Returns (true, nil) if liked, (false, nil) if unliked.
func (r *FragmentRepository) ToggleLike(ctx context.Context, fragmentID, userID string) (bool, error) {
	// Try to delete an existing like first
	result := r.db.WithContext(ctx).
		Where("fragment_id = ? AND user_id = ?", fragmentID, userID).
		Delete(&domain.FragmentLike{})

	if result.Error != nil {
		return false, result.Error
	}

	if result.RowsAffected > 0 {
		// Successfully deleted — this was an unlike
		_ = r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
			Where("id = ?", fragmentID).
			UpdateColumn("likes", gorm.Expr("GREATEST(likes - 1, 0)")).Error
		return false, nil
	}

	// No row deleted — try to insert (like)
	like := &domain.FragmentLike{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		FragmentID: fragmentID,
		UserID:     userID,
	}
	if err := r.db.WithContext(ctx).Create(like).Error; err != nil {
		if isDuplicateKeyError(err) {
			return true, nil // idempotent: already liked by concurrent request
		}
		return false, err
	}

	// Successfully liked — increment counter
	_ = r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", fragmentID).
		UpdateColumn("likes", gorm.Expr("likes + 1")).Error
	return true, nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Error 1062") || strings.Contains(msg, "Duplicate entry")
}
// GetEngagementStats retrieves likes/comments counts and current user's like status.
// Comments are aggregated from both legacy comments table and fragment_comments table.
func (r *FragmentRepository) GetEngagementStats(ctx context.Context, fragmentID, userID string) (FragmentEngagement, error) {
	statsMap, err := r.BatchGetEngagementStats(ctx, []string{fragmentID}, userID)
	if err != nil {
		return FragmentEngagement{}, err
	}
	if stats, ok := statsMap[fragmentID]; ok {
		return stats, nil
	}
	return FragmentEngagement{}, nil
}

// BatchGetEngagementStats retrieves likes/comments counts and current user's like status for multiple fragments.
func (r *FragmentRepository) BatchGetEngagementStats(ctx context.Context, fragmentIDs []string, userID string) (map[string]FragmentEngagement, error) {
	result := make(map[string]FragmentEngagement, len(fragmentIDs))
	if len(fragmentIDs) == 0 {
		return result, nil
	}

	for _, id := range fragmentIDs {
		result[id] = FragmentEngagement{}
	}

	type groupedCount struct {
		FragmentID string
		Count      int64
	}

	var likeCounts []groupedCount
	if err := r.db.WithContext(ctx).
		Model(&domain.FragmentLike{}).
		Select("fragment_id as fragment_id, count(*) as count").
		Where("fragment_id IN ?", fragmentIDs).
		Group("fragment_id").
		Find(&likeCounts).Error; err != nil {
		return nil, err
	}
	for _, row := range likeCounts {
		stats := result[row.FragmentID]
		stats.Likes = int(row.Count)
		result[row.FragmentID] = stats
	}

	var commentCounts []groupedCount
	if err := r.db.WithContext(ctx).
		Table("comments").
		Select("target_id as fragment_id, count(*) as count").
		Where("target_type = ? AND target_id IN ?", "fragment", fragmentIDs).
		Group("target_id").
		Find(&commentCounts).Error; err != nil {
		return nil, err
	}
	for _, row := range commentCounts {
		stats := result[row.FragmentID]
		stats.Comments += int(row.Count)
		result[row.FragmentID] = stats
	}

	var fragmentCommentCounts []groupedCount
	if err := r.db.WithContext(ctx).
		Table("fragment_comments").
		Select("fragment_id as fragment_id, count(*) as count").
		Where("fragment_id IN ?", fragmentIDs).
		Group("fragment_id").
		Find(&fragmentCommentCounts).Error; err != nil {
		return nil, err
	}
	for _, row := range fragmentCommentCounts {
		stats := result[row.FragmentID]
		stats.Comments += int(row.Count)
		result[row.FragmentID] = stats
	}

	if userID != "" {
		var likedRows []struct {
			FragmentID string
		}
		if err := r.db.WithContext(ctx).
			Model(&domain.FragmentLike{}).
			Select("fragment_id").
			Where("user_id = ? AND fragment_id IN ?", userID, fragmentIDs).
			Find(&likedRows).Error; err != nil {
			return nil, err
		}
		for _, row := range likedRows {
			stats := result[row.FragmentID]
			stats.IsLiked = true
			result[row.FragmentID] = stats
		}
	}

	return result, nil
}
