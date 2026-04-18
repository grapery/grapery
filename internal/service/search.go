package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// SearchType 搜索类型
type SearchType string

const (
	SearchTypeFuzzy   SearchType = "fuzzy"         // 模糊搜索
	SearchTypeExact   SearchType = "exact"         // 精确搜索
	SearchTypeDefault SearchType = SearchTypeFuzzy // 默认模糊搜索
)

// 搜索缓存过期时间
const (
	searchCacheTTL = 30 * time.Minute // 搜索结果缓存30分钟
	searchIndexTTL = 24 * time.Hour   // 搜索索引缓存24小时
)

// generateCacheKey 生成缓存键（使用MD5避免键过长）
func generateCacheKey(prefix, query, searchType string, limit, offset int) string {
	// 使用MD5哈希查询字符串，避免键过长
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s:%d:%d", query, searchType, limit, offset)))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("%s%s", prefix, hashStr)
}

// SearchStories 搜索故事（支持模糊搜索和精确搜索，带缓存）
func (s *Service) SearchStories(ctx context.Context, query string, searchType SearchType, limit, offset int) ([]*domain.Story, error) {
	s.logger.Info("searching stories",
		zap.String("query", query),
		zap.String("searchType", string(searchType)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// 参数验证和默认值设置
	if limit <= 0 {
		limit = 20
		s.logger.Debug("using default limit", zap.Int("limit", limit))
	}
	if limit > 100 {
		limit = 100
		s.logger.Debug("limit capped to maximum", zap.Int("limit", limit))
	}
	if searchType == "" {
		searchType = SearchTypeDefault
		s.logger.Debug("using default search type", zap.String("searchType", string(searchType)))
	}

	// 规范化查询字符串
	query = strings.TrimSpace(query)
	if query == "" {
		s.logger.Warn("empty query string")
		return []*domain.Story{}, nil
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.SearchStoriesKey(query, string(searchType), limit, offset)
		s.logger.Debug("checking cache for stories search",
			zap.String("cacheKey", cacheKey),
			zap.String("query", query))

		var cachedResults []*domain.Story
		if err := c.Get(ctx, cacheKey, &cachedResults); err == nil {
			s.logger.Info("stories search cache hit",
				zap.String("query", query),
				zap.Int("resultCount", len(cachedResults)))
			return cachedResults, nil
		} else {
			s.logger.Debug("stories search cache miss",
				zap.String("query", query),
				zap.Error(err))
		}
	}

	// 执行搜索
	var results []*domain.Story
	var err error

	if searchType == SearchTypeFuzzy {
		// 模糊搜索：可以使用 Redis 辅助
		s.logger.Debug("performing fuzzy search for stories",
			zap.String("query", query))
		results, err = s.fuzzySearchStories(ctx, query, limit, offset, c)
	} else {
		// 精确搜索：直接查询数据库
		s.logger.Debug("performing exact search for stories",
			zap.String("query", query))
		results, err = s.repo.SearchStories(ctx, query, limit, offset)
	}

	if err != nil {
		s.logger.Error("failed to search stories",
			zap.String("query", query),
			zap.String("searchType", string(searchType)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to search stories: %w", err)
	}

	// 缓存结果
	if c != nil && len(results) > 0 {
		cacheKey := cache.SearchStoriesKey(query, string(searchType), limit, offset)
		if err := c.Set(ctx, cacheKey, results, searchCacheTTL); err != nil {
			s.logger.Warn("failed to cache search results",
				zap.String("query", query),
				zap.String("cacheKey", cacheKey),
				zap.Error(err))
		} else {
			s.logger.Debug("stories search results cached",
				zap.String("query", query),
				zap.Int("resultCount", len(results)))
		}
	}

	s.logger.Info("stories search completed",
		zap.String("query", query),
		zap.String("searchType", string(searchType)),
		zap.Int("resultCount", len(results)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	return results, nil
}

// fuzzySearchStories 模糊搜索故事（使用 Redis 辅助）
func (s *Service) fuzzySearchStories(ctx context.Context, query string, limit, offset int, c cache.Cache) ([]*domain.Story, error) {
	// 如果 Redis 可用，尝试使用 Redis 的集合进行模糊匹配
	if c != nil {
		// 将查询字符串拆分为关键词
		keywords := s.extractKeywords(query)
		s.logger.Debug("extracted keywords for fuzzy search",
			zap.String("query", query),
			zap.Strings("keywords", keywords))

		// 尝试从 Redis 搜索索引中查找匹配的故事ID
		var matchedStoryIDs []string
		for _, keyword := range keywords {
			indexKey := cache.SearchIndexStoriesKey(keyword)
			members, err := c.SMembers(ctx, indexKey)
			if err == nil && len(members) > 0 {
				matchedStoryIDs = append(matchedStoryIDs, members...)
				s.logger.Debug("found story IDs from search index",
					zap.String("keyword", keyword),
					zap.Int("matchCount", len(members)))
			}
		}

		// 如果从索引中找到结果，直接返回（简化实现，实际应该去重并排序）
		if len(matchedStoryIDs) > 0 {
			// 去重
			uniqueIDs := s.deduplicateStrings(matchedStoryIDs)
			// 限制数量
			if offset < len(uniqueIDs) {
				end := offset + limit
				if end > len(uniqueIDs) {
					end = len(uniqueIDs)
				}
				uniqueIDs = uniqueIDs[offset:end]
			} else {
				uniqueIDs = []string{}
			}

			// 从数据库获取完整故事信息
			if len(uniqueIDs) > 0 {
				var stories []*domain.Story
				for _, id := range uniqueIDs {
					story, err := s.repo.StoryByID(ctx, id)
					if err == nil && story != nil {
						stories = append(stories, story)
					}
				}
				if len(stories) > 0 {
					s.logger.Info("fuzzy search completed using Redis index",
						zap.String("query", query),
						zap.Int("resultCount", len(stories)))
					return stories, nil
				}
			}
		}
	}

	// 降级到数据库搜索
	s.logger.Debug("falling back to database search",
		zap.String("query", query))
	return s.repo.SearchStories(ctx, query, limit, offset)
}

// SearchCharacters 搜索角色（支持模糊搜索和精确搜索，带缓存）
func (s *Service) SearchCharacters(ctx context.Context, query string, searchType SearchType, limit, offset int) ([]*domain.Character, error) {
	s.logger.Info("searching characters",
		zap.String("query", query),
		zap.String("searchType", string(searchType)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// 参数验证和默认值设置
	if limit <= 0 {
		limit = 20
		s.logger.Debug("using default limit", zap.Int("limit", limit))
	}
	if limit > 100 {
		limit = 100
		s.logger.Debug("limit capped to maximum", zap.Int("limit", limit))
	}
	if searchType == "" {
		searchType = SearchTypeDefault
		s.logger.Debug("using default search type", zap.String("searchType", string(searchType)))
	}

	// 规范化查询字符串
	query = strings.TrimSpace(query)
	if query == "" {
		s.logger.Warn("empty query string")
		return []*domain.Character{}, nil
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.SearchCharactersKey(query, string(searchType), limit, offset)
		s.logger.Debug("checking cache for characters search",
			zap.String("cacheKey", cacheKey),
			zap.String("query", query))

		var cachedResults []*domain.Character
		if err := c.Get(ctx, cacheKey, &cachedResults); err == nil {
			s.logger.Info("characters search cache hit",
				zap.String("query", query),
				zap.Int("resultCount", len(cachedResults)))
			return cachedResults, nil
		} else {
			s.logger.Debug("characters search cache miss",
				zap.String("query", query),
				zap.Error(err))
		}
	}

	// 执行搜索
	var results []*domain.Character
	var err error

	if searchType == SearchTypeFuzzy {
		// 模糊搜索：可以使用 Redis 辅助
		s.logger.Debug("performing fuzzy search for characters",
			zap.String("query", query))
		results, err = s.fuzzySearchCharacters(ctx, query, limit, offset, c)
	} else {
		// 精确搜索：直接查询数据库
		s.logger.Debug("performing exact search for characters",
			zap.String("query", query))
		results, err = s.repo.SearchCharacters(ctx, query, limit, offset)
	}

	if err != nil {
		s.logger.Error("failed to search characters",
			zap.String("query", query),
			zap.String("searchType", string(searchType)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to search characters: %w", err)
	}

	// 缓存结果
	if c != nil && len(results) > 0 {
		cacheKey := cache.SearchCharactersKey(query, string(searchType), limit, offset)
		if err := c.Set(ctx, cacheKey, results, searchCacheTTL); err != nil {
			s.logger.Warn("failed to cache search results",
				zap.String("query", query),
				zap.String("cacheKey", cacheKey),
				zap.Error(err))
		} else {
			s.logger.Debug("characters search results cached",
				zap.String("query", query),
				zap.Int("resultCount", len(results)))
		}
	}

	s.logger.Info("characters search completed",
		zap.String("query", query),
		zap.String("searchType", string(searchType)),
		zap.Int("resultCount", len(results)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	return results, nil
}

// fuzzySearchCharacters 模糊搜索角色（使用 Redis 辅助）
func (s *Service) fuzzySearchCharacters(ctx context.Context, query string, limit, offset int, c cache.Cache) ([]*domain.Character, error) {
	// 如果 Redis 可用，尝试使用 Redis 的集合进行模糊匹配
	if c != nil {
		// 将查询字符串拆分为关键词
		keywords := s.extractKeywords(query)
		s.logger.Debug("extracted keywords for fuzzy search",
			zap.String("query", query),
			zap.Strings("keywords", keywords))

		// 尝试从 Redis 搜索索引中查找匹配的角色ID
		var matchedCharacterIDs []string
		for _, keyword := range keywords {
			indexKey := cache.SearchIndexCharactersKey(keyword)
			members, err := c.SMembers(ctx, indexKey)
			if err == nil && len(members) > 0 {
				matchedCharacterIDs = append(matchedCharacterIDs, members...)
				s.logger.Debug("found character IDs from search index",
					zap.String("keyword", keyword),
					zap.Int("matchCount", len(members)))
			}
		}

		// 如果从索引中找到结果，直接返回
		if len(matchedCharacterIDs) > 0 {
			// 去重
			uniqueIDs := s.deduplicateStrings(matchedCharacterIDs)
			// 限制数量
			if offset < len(uniqueIDs) {
				end := offset + limit
				if end > len(uniqueIDs) {
					end = len(uniqueIDs)
				}
				uniqueIDs = uniqueIDs[offset:end]
			} else {
				uniqueIDs = []string{}
			}

			// 从数据库获取完整角色信息
			if len(uniqueIDs) > 0 {
				var characters []*domain.Character
				for _, id := range uniqueIDs {
					character, err := s.repo.CharacterByID(ctx, id)
					if err == nil && character != nil {
						characters = append(characters, character)
					}
				}
				if len(characters) > 0 {
					s.logger.Info("fuzzy search completed using Redis index",
						zap.String("query", query),
						zap.Int("resultCount", len(characters)))
					return characters, nil
				}
			}
		}
	}

	// 降级到数据库搜索
	s.logger.Debug("falling back to database search",
		zap.String("query", query))
	return s.repo.SearchCharacters(ctx, query, limit, offset)
}

// SearchUsers 搜索用户（支持模糊搜索和精确搜索，带缓存）
func (s *Service) SearchUsers(ctx context.Context, query string, searchType SearchType, limit, offset int) ([]*domain.User, error) {
	s.logger.Info("searching users",
		zap.String("query", query),
		zap.String("searchType", string(searchType)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// 参数验证和默认值设置
	if limit <= 0 {
		limit = 20
		s.logger.Debug("using default limit", zap.Int("limit", limit))
	}
	if limit > 100 {
		limit = 100
		s.logger.Debug("limit capped to maximum", zap.Int("limit", limit))
	}
	if searchType == "" {
		searchType = SearchTypeDefault
		s.logger.Debug("using default search type", zap.String("searchType", string(searchType)))
	}

	// 规范化查询字符串
	query = strings.TrimSpace(query)
	if query == "" {
		s.logger.Warn("empty query string")
		return []*domain.User{}, nil
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.SearchUsersKey(query, string(searchType), limit, offset)
		s.logger.Debug("checking cache for users search",
			zap.String("cacheKey", cacheKey),
			zap.String("query", query))

		var cachedResults []*domain.User
		if err := c.Get(ctx, cacheKey, &cachedResults); err == nil {
			s.logger.Info("users search cache hit",
				zap.String("query", query),
				zap.Int("resultCount", len(cachedResults)))
			return cachedResults, nil
		} else {
			s.logger.Debug("users search cache miss",
				zap.String("query", query),
				zap.Error(err))
		}
	}

	// 执行搜索
	var results []*domain.User
	var err error

	if searchType == SearchTypeFuzzy {
		// 模糊搜索：可以使用 Redis 辅助
		s.logger.Debug("performing fuzzy search for users",
			zap.String("query", query))
		results, err = s.fuzzySearchUsers(ctx, query, limit, offset, c)
	} else {
		// 精确搜索：直接查询数据库
		s.logger.Debug("performing exact search for users",
			zap.String("query", query))
		results, err = s.repo.SearchUsers(ctx, query, limit, offset)
	}

	if err != nil {
		s.logger.Error("failed to search users",
			zap.String("query", query),
			zap.String("searchType", string(searchType)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	// 缓存结果
	if c != nil && len(results) > 0 {
		cacheKey := cache.SearchUsersKey(query, string(searchType), limit, offset)
		if err := c.Set(ctx, cacheKey, results, searchCacheTTL); err != nil {
			s.logger.Warn("failed to cache search results",
				zap.String("query", query),
				zap.String("cacheKey", cacheKey),
				zap.Error(err))
		} else {
			s.logger.Debug("users search results cached",
				zap.String("query", query),
				zap.Int("resultCount", len(results)))
		}
	}

	s.logger.Info("users search completed",
		zap.String("query", query),
		zap.String("searchType", string(searchType)),
		zap.Int("resultCount", len(results)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	return results, nil
}

// fuzzySearchUsers 模糊搜索用户（使用 Redis 辅助）
func (s *Service) fuzzySearchUsers(ctx context.Context, query string, limit, offset int, c cache.Cache) ([]*domain.User, error) {
	// 如果 Redis 可用，尝试使用 Redis 的集合进行模糊匹配
	if c != nil {
		// 将查询字符串拆分为关键词
		keywords := s.extractKeywords(query)
		s.logger.Debug("extracted keywords for fuzzy search",
			zap.String("query", query),
			zap.Strings("keywords", keywords))

		// 尝试从 Redis 搜索索引中查找匹配的用户ID
		var matchedUserIDs []string
		for _, keyword := range keywords {
			indexKey := cache.SearchIndexUsersKey(keyword)
			members, err := c.SMembers(ctx, indexKey)
			if err == nil && len(members) > 0 {
				matchedUserIDs = append(matchedUserIDs, members...)
				s.logger.Debug("found user IDs from search index",
					zap.String("keyword", keyword),
					zap.Int("matchCount", len(members)))
			}
		}

		// 如果从索引中找到结果，直接返回
		if len(matchedUserIDs) > 0 {
			// 去重
			uniqueIDs := s.deduplicateStrings(matchedUserIDs)
			// 限制数量
			if offset < len(uniqueIDs) {
				end := offset + limit
				if end > len(uniqueIDs) {
					end = len(uniqueIDs)
				}
				uniqueIDs = uniqueIDs[offset:end]
			} else {
				uniqueIDs = []string{}
			}

			// 从数据库获取完整用户信息
			if len(uniqueIDs) > 0 {
				var users []*domain.User
				for _, id := range uniqueIDs {
					user, err := s.repo.UserByID(ctx, id)
					if err == nil && user != nil {
						users = append(users, user)
					}
				}
				if len(users) > 0 {
					s.logger.Info("fuzzy search completed using Redis index",
						zap.String("query", query),
						zap.Int("resultCount", len(users)))
					return users, nil
				}
			}
		}
	}

	// 降级到数据库搜索
	s.logger.Debug("falling back to database search",
		zap.String("query", query))
	return s.repo.SearchUsers(ctx, query, limit, offset)
}

// SearchAll 搜索所有类型（支持模糊搜索和精确搜索，带缓存）
func (s *Service) SearchAll(ctx context.Context, query string, searchType SearchType, limit int) (map[string]interface{}, error) {
	s.logger.Info("searching all types",
		zap.String("query", query),
		zap.String("searchType", string(searchType)),
		zap.Int("limit", limit))

	// 参数验证和默认值设置
	if limit <= 0 {
		limit = 10
		s.logger.Debug("using default limit", zap.Int("limit", limit))
	}
	if searchType == "" {
		searchType = SearchTypeDefault
		s.logger.Debug("using default search type", zap.String("searchType", string(searchType)))
	}

	// 规范化查询字符串
	query = strings.TrimSpace(query)
	if query == "" {
		s.logger.Warn("empty query string")
		return map[string]interface{}{
			"stories":    []*domain.Story{},
			"characters": []*domain.Character{},
			"users":      []*domain.User{},
		}, nil
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.SearchAllKey(query, limit)
		s.logger.Debug("checking cache for all search",
			zap.String("cacheKey", cacheKey),
			zap.String("query", query))

		var cachedResults map[string]interface{}
		if err := c.Get(ctx, cacheKey, &cachedResults); err == nil {
			s.logger.Info("all search cache hit",
				zap.String("query", query))
			return cachedResults, nil
		} else {
			s.logger.Debug("all search cache miss",
				zap.String("query", query),
				zap.Error(err))
		}
	}

	// 并行搜索所有类型
	var stories []*domain.Story
	var characters []*domain.Character
	var users []*domain.User

	// 使用 goroutine 并行搜索（简化实现，实际应该使用 errgroup）
	stories, _ = s.SearchStories(ctx, query, searchType, limit, 0)
	characters, _ = s.SearchCharacters(ctx, query, searchType, limit, 0)
	users, _ = s.SearchUsers(ctx, query, searchType, limit, 0)

	result := map[string]interface{}{
		"stories":    stories,
		"characters": characters,
		"users":      users,
	}

	// 缓存结果
	if c != nil {
		cacheKey := cache.SearchAllKey(query, limit)
		if err := c.Set(ctx, cacheKey, result, searchCacheTTL); err != nil {
			s.logger.Warn("failed to cache all search results",
				zap.String("query", query),
				zap.String("cacheKey", cacheKey),
				zap.Error(err))
		} else {
			s.logger.Debug("all search results cached",
				zap.String("query", query))
		}
	}

	s.logger.Info("all search completed",
		zap.String("query", query),
		zap.String("searchType", string(searchType)),
		zap.Int("storiesCount", len(stories)),
		zap.Int("charactersCount", len(characters)),
		zap.Int("usersCount", len(users)))

	return result, nil
}

// extractKeywords 从查询字符串中提取关键词
func (s *Service) extractKeywords(query string) []string {
	// 移除多余空格并转换为小写
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []string{}
	}

	// 按空格分割
	parts := strings.Fields(query)
	keywords := make([]string, 0, len(parts))

	// 过滤空字符串并添加完整查询字符串
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && len(part) >= 2 { // 至少2个字符
			keywords = append(keywords, part)
		}
	}

	// 如果查询字符串本身长度>=2，也添加
	if len(query) >= 2 {
		keywords = append(keywords, query)
	}

	// 去重
	return s.deduplicateStrings(keywords)
}

// deduplicateStrings 字符串去重
func (s *Service) deduplicateStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(strs))

	for _, str := range strs {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// BuildSearchIndex 构建搜索索引（将实体添加到 Redis 搜索索引中）
// 这个方法应该在创建/更新实体时调用，以维护搜索索引
func (s *Service) BuildSearchIndex(ctx context.Context, entityType string, entityID string, searchableText string) error {
	c := s.getCache()
	if c == nil {
		s.logger.Debug("cache not available, skipping search index build",
			zap.String("entityType", entityType),
			zap.String("entityID", entityID))
		return nil
	}

	s.logger.Debug("building search index",
		zap.String("entityType", entityType),
		zap.String("entityID", entityID),
		zap.String("searchableText", truncateForLog(searchableText, 100)))

	// 提取关键词
	keywords := s.extractKeywords(searchableText)
	if len(keywords) == 0 {
		s.logger.Debug("no keywords extracted, skipping index",
			zap.String("entityType", entityType),
			zap.String("entityID", entityID))
		return nil
	}

	// 根据实体类型选择索引键前缀
	var indexKeyFunc func(string) string
	switch entityType {
	case "story":
		indexKeyFunc = cache.SearchIndexStoriesKey
	case "character":
		indexKeyFunc = cache.SearchIndexCharactersKey
	case "user":
		indexKeyFunc = cache.SearchIndexUsersKey
	default:
		s.logger.Warn("unknown entity type for search index",
			zap.String("entityType", entityType))
		return nil
	}

	// 将实体ID添加到每个关键词的索引集合中
	for _, keyword := range keywords {
		indexKey := indexKeyFunc(keyword)
		if err := c.SAdd(ctx, indexKey, entityID); err != nil {
			s.logger.Warn("failed to add entity to search index",
				zap.String("entityType", entityType),
				zap.String("entityID", entityID),
				zap.String("keyword", keyword),
				zap.String("indexKey", indexKey),
				zap.Error(err))
		} else {
			// 设置索引过期时间
			if err := c.Expire(ctx, indexKey, searchIndexTTL); err != nil {
				s.logger.Debug("failed to set search index expiration",
					zap.String("indexKey", indexKey),
					zap.Error(err))
			}
		}
	}

	s.logger.Debug("search index built successfully",
		zap.String("entityType", entityType),
		zap.String("entityID", entityID),
		zap.Int("keywordCount", len(keywords)))

	return nil
}

// RemoveFromSearchIndex 从搜索索引中移除实体
func (s *Service) RemoveFromSearchIndex(ctx context.Context, entityType string, entityID string, searchableText string) error {
	c := s.getCache()
	if c == nil {
		return nil
	}

	s.logger.Debug("removing from search index",
		zap.String("entityType", entityType),
		zap.String("entityID", entityID))

	// 提取关键词
	keywords := s.extractKeywords(searchableText)
	if len(keywords) == 0 {
		return nil
	}

	// 根据实体类型选择索引键前缀
	var indexKeyFunc func(string) string
	switch entityType {
	case "story":
		indexKeyFunc = cache.SearchIndexStoriesKey
	case "character":
		indexKeyFunc = cache.SearchIndexCharactersKey
	case "user":
		indexKeyFunc = cache.SearchIndexUsersKey
	default:
		return nil
	}

	// 从每个关键词的索引集合中移除实体ID
	for _, keyword := range keywords {
		indexKey := indexKeyFunc(keyword)
		if err := c.SRem(ctx, indexKey, entityID); err != nil {
			s.logger.Warn("failed to remove entity from search index",
				zap.String("entityType", entityType),
				zap.String("entityID", entityID),
				zap.String("keyword", keyword),
				zap.Error(err))
		}
	}

	s.logger.Debug("removed from search index",
		zap.String("entityType", entityType),
		zap.String("entityID", entityID))

	return nil
}

// InvalidateSearchCache 使搜索缓存失效
func (s *Service) InvalidateSearchCache(ctx context.Context, query string) error {
	c := s.getCache()
	if c == nil {
		return nil
	}

	s.logger.Debug("invalidating search cache",
		zap.String("query", query))

	// 提取关键词，用于清除相关索引
	keywords := s.extractKeywords(query)

	// 清除所有搜索类型的缓存（简化实现，实际应该更精确）
	keysToDelete := []string{
		cache.SearchStoriesKey(query, string(SearchTypeFuzzy), 20, 0),
		cache.SearchStoriesKey(query, string(SearchTypeExact), 20, 0),
		cache.SearchCharactersKey(query, string(SearchTypeFuzzy), 20, 0),
		cache.SearchCharactersKey(query, string(SearchTypeExact), 20, 0),
		cache.SearchUsersKey(query, string(SearchTypeFuzzy), 20, 0),
		cache.SearchUsersKey(query, string(SearchTypeExact), 20, 0),
		cache.SearchAllKey(query, 10),
	}

	// 添加基于关键词的缓存键（简化实现）
	for _, keyword := range keywords {
		keysToDelete = append(keysToDelete,
			cache.SearchStoriesKey(keyword, string(SearchTypeFuzzy), 20, 0),
			cache.SearchCharactersKey(keyword, string(SearchTypeFuzzy), 20, 0),
			cache.SearchUsersKey(keyword, string(SearchTypeFuzzy), 20, 0),
		)
	}

	if err := c.Delete(ctx, keysToDelete...); err != nil {
		s.logger.Warn("failed to invalidate search cache",
			zap.String("query", query),
			zap.Error(err))
		return err
	}

	s.logger.Info("search cache invalidated",
		zap.String("query", query),
		zap.Int("keysDeleted", len(keysToDelete)))

	return nil
}
