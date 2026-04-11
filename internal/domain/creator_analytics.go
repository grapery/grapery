package domain

// CreatorAnalyticsAggregate is a single-query rollup for a user's authored content.
type CreatorAnalyticsAggregate struct {
	TotalViews           int64
	TotalStoryLikes      int64
	TotalStoryboardLikes int64
	TotalSaves           int64
	TotalComments        int64
	TotalShares          int64
	TotalForks           int64
}

// CreatorAnalyticsStoryboardRow is one row for "hot content" (top by views).
type CreatorAnalyticsStoryboardRow struct {
	ID      string
	Title   string
	Views   int
	Likes   int
	StoryID string
}
