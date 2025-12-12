package domain

import "encoding/json"

// SearchHistory 搜索历史
type SearchHistory struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Query       string `json:"query"`
	Type        string `json:"type"` // story, character, user, group
	ResultCount int    `json:"resultCount"`
	CreatedAt   int64  `json:"createdAt"`

	// Relations
	User *User `json:"user,omitempty"`
}

// ViewHistory 浏览历史
type ViewHistory struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	EntityType string `json:"entityType"` // story, storyboard, character
	EntityID   string `json:"entityId"`
	Duration   int    `json:"duration"` // 浏览时长（秒）
	ViewedAt   int64  `json:"viewedAt"`

	// Relations
	User *User `json:"user,omitempty"`
}

// SearchFilter 搜索过滤器（请求参数，不存储到数据库）
type SearchFilter struct {
	Query      string   `json:"query"`
	Type       string   `json:"type"`                 // story, character, user, group, all
	Categories []string `json:"categories,omitempty"` // 分类
	Tags       []string `json:"tags,omitempty"`       // 标签
	AuthorID   string   `json:"authorId,omitempty"`
	MinViews   int      `json:"minViews,omitempty"`
	MaxViews   int      `json:"maxViews,omitempty"`
	MinLikes   int      `json:"minLikes,omitempty"`
	Status     string   `json:"status,omitempty"` // published, draft
	SortBy     string   `json:"sortBy"`           // relevance, views, likes, date
	SortOrder  string   `json:"sortOrder"`        // asc, desc
	DateFrom   *int64   `json:"dateFrom,omitempty"`
	DateTo     *int64   `json:"dateTo,omitempty"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
}

// SearchResult 搜索结果（返回结构，不存储到数据库）
type SearchResult struct {
	Type        string  `json:"type"` // story, character, user, group
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Cover       string  `json:"cover,omitempty"`
	Author      string  `json:"author,omitempty"`
	AuthorID    string  `json:"authorId,omitempty"`
	Views       int     `json:"views,omitempty"`
	Likes       int     `json:"likes,omitempty"`
	TagsJSON    string  `json:"-"`
	CreatedAt   int64   `json:"createdAt"`
	Relevance   float64 `json:"relevance"` // 相关度评分

	// Transient fields
	Tags []string `json:"tags,omitempty"`
}

// MarshalJSON custom JSON marshaling
func (sr *SearchResult) MarshalJSON() ([]byte, error) {
	type Alias SearchResult
	if sr.TagsJSON != "" && sr.Tags == nil {
		_ = json.Unmarshal([]byte(sr.TagsJSON), &sr.Tags)
	}
	return json.Marshal(&struct {
		*Alias
		TagsJSON string `json:"-"`
	}{
		Alias: (*Alias)(sr),
	})
}

// Report 举报
type Report struct {
	ID          string `json:"id"`
	ReporterID  string `json:"reporterId"`
	EntityType  string `json:"entityType"` // story, comment, user, character
	EntityID    string `json:"entityId"`
	Reason      string `json:"reason"` // spam, inappropriate, copyright, other
	Description string `json:"description"`
	Status      string `json:"status"` // pending, reviewed, resolved, rejected
	ReviewerID  string `json:"reviewerId,omitempty"`
	ReviewNote  string `json:"reviewNote,omitempty"`
	CreatedAt   int64  `json:"createdAt"`
	ReviewedAt  *int64 `json:"reviewedAt,omitempty"`

	// Relations
	Reporter *User `json:"reporter,omitempty"`
	Reviewer *User `json:"reviewer,omitempty"`
}
