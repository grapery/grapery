package domain

// GroupShowcaseRelationType 小组展示内容关系类型
type GroupShowcaseRelationType string

const (
	// GroupShowcaseTypeFragment 展示碎片
	GroupShowcaseTypeFragment GroupShowcaseRelationType = "fragment"
	// GroupShowcaseTypeStory 展示故事
	GroupShowcaseTypeStory GroupShowcaseRelationType = "story"
)

// GroupShowcaseStatus 小组展示状态
type GroupShowcaseStatus string

const (
	// GroupShowcaseStatusActive 激活状态
	GroupShowcaseStatusActive GroupShowcaseStatus = "active"
	// GroupShowcaseStatusRemoved 已移除
	GroupShowcaseStatusRemoved GroupShowcaseStatus = "removed"
)

// GroupShowcase 小组展示内容
// 用于记录小组展示的碎片和外部故事
type GroupShowcase struct {
	ID         string                    `json:"id"`
	GroupID    string                    `json:"groupId"`    // 所属小组ID
	ContentID  string                    `json:"contentId"`  // 内容ID（碎片ID或故事ID）
	ContentType GroupShowcaseRelationType `json:"contentType"` // 内容类型: fragment, story
	AddedBy    string                    `json:"addedBy"`    // 添加者ID（小组管理员或内容作者）
	Status     GroupShowcaseStatus       `json:"status"`     // 状态: active, removed
	SortOrder  int                       `json:"sortOrder"`  // 排序顺序（数值越大越靠前）
	CreatedAt  int64                     `json:"createdAt"`  // 创建时间
	UpdatedAt  int64                     `json:"updatedAt"`  // 更新时间

	// 非持久化字段
	Group       *Group       `json:"group,omitempty" gorm:"-"`       // 小组信息
	Fragment    *Fragment    `json:"fragment,omitempty" gorm:"-"`    // 碎片详情（当ContentType=fragment时）
	Story       *Story       `json:"story,omitempty" gorm:"-"`       // 故事详情（当ContentType=story时）
}

// TableName 指定表名
func (GroupShowcase) TableName() string {
	return "group_showcases"
}

// IsFragment 检查是否是碎片类型
func (gs *GroupShowcase) IsFragment() bool {
	return gs.ContentType == GroupShowcaseTypeFragment
}

// IsStory 检查是否是故事类型
func (gs *GroupShowcase) IsStory() bool {
	return gs.ContentType == GroupShowcaseTypeStory
}

// IsActive 检查是否处于激活状态
func (gs *GroupShowcase) IsActive() bool {
	return gs.Status == GroupShowcaseStatusActive
}

// AddGroupShowcaseRequest 添加展示内容请求
type AddGroupShowcaseRequest struct {
	ContentID   string                    `json:"contentId" binding:"required"`   // 内容ID
	ContentType GroupShowcaseRelationType `json:"contentType" binding:"required"` // 内容类型
	SortOrder   int                       `json:"sortOrder"`                      // 排序顺序
}

// RemoveGroupShowcaseRequest 移除展示内容请求
type RemoveGroupShowcaseRequest struct {
	ShowcaseID string `json:"showcaseId" binding:"required"` // 展示记录ID
}

// ListGroupShowcasesResponse 列表展示内容响应
type ListGroupShowcasesResponse struct {
	Showcases []*GroupShowcase `json:"showcases"` // 展示列表
	Total     int              `json:"total"`     // 总数
}
