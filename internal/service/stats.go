package service

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// UserStats 用户统计数据
type UserStats struct {
	TotalStories     int `json:"totalStories"`
	TotalCharacters  int `json:"totalCharacters"`
	TotalViews       int `json:"totalViews"`
	TotalLikes       int `json:"totalLikes"`
	TotalFollowers   int `json:"totalFollowers"`
	TotalFollowing   int `json:"totalFollowing"`
	TotalStoryboards int `json:"totalStoryboards"`
}

// StoryStats 故事统计数据
type StoryStats struct {
	Views     int `json:"views"`
	Likes     int `json:"likes"`
	Followers int `json:"followers"`
	Comments  int `json:"comments"`
	Panels    int `json:"panels"`
}

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	TotalStories    int    `json:"totalStories"`
	TotalViews      string `json:"totalViews"`
	TotalFollowers  int    `json:"totalFollowers"`
	AvgRating       string `json:"avgRating"`
	TrendingStories int    `json:"trendingStories"`
	RecentActivity  int    `json:"recentActivity"`
	MonthlyGrowth   string `json:"monthlyGrowth"`
}

// GetUserStats 获取用户统计数据
func (s *Service) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取用户故事
	stories, err := s.repo.StoriesByUser(ctx, userID, 0, 0)
	if err != nil {
		stories = []*domain.Story{}
	}

	// 获取用户角色
	characters, err := s.repo.CharactersByUser(ctx, userID, 0, 0)
	if err != nil {
		characters = []*domain.Character{}
	}

	// 计算总浏览量和总点赞数
	totalViews := 0
	totalLikes := 0
	for _, story := range stories {
		totalLikes += story.Likes

		// 获取该故事的所有故事板，累加真实的浏览量
		storyboards, err := s.repo.StoryboardsByStory(ctx, story.ID, 0, 0)
		if err == nil {
			for _, sb := range storyboards {
				totalViews += sb.Views
			}
		}
	}

	// 获取用户创建的分镜数量
	storyboardCount, err := s.repo.CountStoryboardsByCreator(ctx, userID)
	if err != nil {
		storyboardCount = 0
	}

	stats := &UserStats{
		TotalStories:     len(stories),
		TotalCharacters:  len(characters),
		TotalViews:       totalViews,
		TotalLikes:       totalLikes,
		TotalFollowers:   user.Followers,
		TotalFollowing:   user.Following,
		TotalStoryboards: int(storyboardCount),
	}

	return stats, nil
}

// GetStoryStats 获取故事统计数据
func (s *Service) GetStoryStats(ctx context.Context, storyID string) (*StoryStats, error) {
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		return nil, err
	}

	// 获取评论数量
	comments, _, err := s.repo.CommentsByTarget(ctx, "story", storyID, 0, 0)
	if err != nil {
		comments = []*domain.Comment{}
	}

	// 计算故事的实际浏览量（累加所有故事板的浏览量）
	totalViews := 0
	storyboards, err := s.repo.StoryboardsByStory(ctx, storyID, 0, 0)
	if err == nil {
		for _, sb := range storyboards {
			totalViews += sb.Views
		}
	}

	stats := &StoryStats{
		Views:     totalViews,
		Likes:     story.Likes,
		Followers: story.Followers,
		Comments:  len(comments),
		Panels:    story.PanelCount,
	}

	return stats, nil
}

// GetDashboardStats 获取仪表盘统计数据
func (s *Service) GetDashboardStats(ctx context.Context, userID string) (*DashboardStats, error) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取用户故事
	stories, err := s.repo.StoriesByUser(ctx, userID, 0, 0)
	if err != nil {
		stories = []*domain.Story{}
	}

	// 计算总浏览量
	totalViews := 0
	for _, story := range stories {
		// 获取该故事的所有故事板，累加真实的浏览量
		storyboards, err := s.repo.StoryboardsByStory(ctx, story.ID, 0, 0)
		if err == nil {
			for _, sb := range storyboards {
				totalViews += sb.Views
			}
		}
	}

	stats := &DashboardStats{
		TotalStories:    len(stories),
		TotalViews:      formatNumber(totalViews),
		TotalFollowers:  user.Followers,
		AvgRating:       "4.8",
		TrendingStories: 3,
		RecentActivity:  15,
		MonthlyGrowth:   "+12%",
	}

	return stats, nil
}

// formatNumber 格式化数字（例如：24500 -> "24.5K"）
func formatNumber(n int) string {
	if n >= 1000000 {
		m := float64(n) / 1000000.0
		return fmt.Sprintf("%.1fM", m)
	}
	if n >= 1000 {
		k := float64(n) / 1000.0
		return fmt.Sprintf("%.1fK", k)
	}
	return fmt.Sprintf("%d", n)
}
