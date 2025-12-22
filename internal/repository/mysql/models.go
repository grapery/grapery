package mysql

import (
	"time"

	"gorm.io/gorm"
)

// ========== 核心实体表 ==========

// User database model
type User struct {
	ID                  string         `gorm:"primaryKey;size:36"`
	Username            string         `gorm:"uniqueIndex;size:50;not null"`
	Email               string         `gorm:"uniqueIndex;size:100;not null"`
	PasswordHash        string         `gorm:"size:255;not null"`
	DisplayName         string         `gorm:"size:100;not null"`
	Avatar              string         `gorm:"size:500"`
	Background          string         `gorm:"size:500"`
	Bio                 string         `gorm:"type:text"`
	Location            string         `gorm:"size:100"`
	Website             string         `gorm:"size:255"`
	AIPromptPreferences string         `gorm:"type:text"`
	DateOfBirth         int64          `gorm:"type:bigint;default:0"`
	Followers           int            `gorm:"default:0;index"`
	Following           int            `gorm:"default:0"`
	StoryboardCount     int            `gorm:"default:0;index"`                // Number of storyboards created by this user
	Status              string         `gorm:"size:20;default:'active';index"` // active, suspended, deleted
	EmailVerified       bool           `gorm:"default:false"`
	LastLoginAt         int64          `gorm:"type:bigint;default:0;index"`
	CreatedAt           int64          `gorm:"type:bigint;autoCreateTime;index"`
	UpdatedAt           int64          `gorm:"type:bigint;autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

// UserLoginRecord 用户登录记录表
type UserLoginRecord struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	UserID    string         `gorm:"size:36;not null;index"`
	User      User           `gorm:"foreignKey:UserID"`
	IPAddress string         `gorm:"size:45;index"`        // IPv4 or IPv6 address
	Location  string         `gorm:"size:200"`             // 地理位置（如：北京市、上海市）
	Device    string         `gorm:"size:100"`             // 设备类型（如：iPhone, Android, Windows, Mac）
	OS        string         `gorm:"size:50"`              // 操作系统（如：iOS 17.0, Android 13, Windows 11）
	Browser   string         `gorm:"size:100"`             // 浏览器（如：Chrome, Safari, Firefox）
	UserAgent string         `gorm:"type:text"`            // 完整的 User-Agent 字符串
	LoginAt   time.Time      `gorm:"autoCreateTime;index"` // 登录时间
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Story database model
type Story struct {
	ID                string         `gorm:"primaryKey;size:36"`
	Title             string         `gorm:"size:200;not null;index"`
	Description       string         `gorm:"type:text"`
	CoverImage        string         `gorm:"size:500"`
	AuthorID          string         `gorm:"size:36;not null;index"`
	Author            User           `gorm:"foreignKey:AuthorID"`
	GroupID           *string        `gorm:"size:36;index"` // Group ID if the story belongs to a group
	Group             *Group         `gorm:"foreignKey:GroupID"`
	Likes             int            `gorm:"default:0;index"`
	Followers         int            `gorm:"default:0"`
	Panels            int            `gorm:"default:0"`
	StoryboardCount   int            `gorm:"default:0;index"` // Number of storyboards in this story
	DefaultSceneCount int            `gorm:"default:3"`       // Default number of scenes for storyboards (2-8)
	Genre             string         `gorm:"size:50;index"`
	Style             string         `gorm:"size:50;index"`                 // Story style (e.g., action, drama, comedy, mystery)
	Status            string         `gorm:"size:20;default:'draft';index"` // draft, published, rendering
	CreatedAt         time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime;index"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// StoryContributor database model (贡献者：创建故事板或初始化故事的参与者)
type StoryContributor struct {
	ID        string    `gorm:"primaryKey;size:36"`
	StoryID   string    `gorm:"size:36;not null;index:idx_story_contributor,unique"`
	Story     Story     `gorm:"foreignKey:StoryID"`
	UserID    string    `gorm:"size:36;not null;index:idx_story_contributor,unique"`
	User      User      `gorm:"foreignKey:UserID"`
	Role      string    `gorm:"size:20;not null;default:'contributor'"` // owner, collaborator, contributor
	InvitedBy string    `gorm:"size:36"`
	Inviter   User      `gorm:"foreignKey:InvitedBy"`
	JoinedAt  time.Time `gorm:"autoCreateTime"`
}

// Panel database model
type Panel struct {
	ID        string         `gorm:"primaryKey;size:36"`
	StoryID   string         `gorm:"size:36;not null;index"`
	Story     Story          `gorm:"foreignKey:StoryID"`
	Sequence  int            `gorm:"not null"`
	Title     string         `gorm:"size:200"`
	Content   string         `gorm:"type:text"`
	Image     string         `gorm:"size:500"`
	Likes     int            `gorm:"default:0"`
	Published bool           `gorm:"default:false;index"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Storyboard database model (树状结构)
type Storyboard struct {
	ID               string         `gorm:"primaryKey;size:36"`
	StoryID          string         `gorm:"size:36;not null;index:idx_storyboard_story;index:idx_storyboard_story_parent"`
	Story            Story          `gorm:"foreignKey:StoryID"`
	ParentID         *string        `gorm:"size:36;index;index:idx_storyboard_story_parent"` // NULL or "__root__" for root storyboard
	Parent           *Storyboard    `gorm:"foreignKey:ParentID"`
	CreatorID        string         `gorm:"size:36;not null;index"`
	Creator          User           `gorm:"foreignKey:CreatorID"`
	Title            string         `gorm:"size:200;not null"`
	Content          string         `gorm:"type:text;not null"`            // AI-polished narrative
	RawInput         string         `gorm:"type:text;not null"`            // Original user input/prompt
	IsStandalone     bool           `gorm:"default:false;index"`           // Independent plot, AI won't reference parent context
	IsAIGenerated    bool           `gorm:"default:false;index"`           // Content was generated by AI
	SceneCount       int            `gorm:"default:3"`                     // Requested number of scenes to generate (2-5)
	WorkflowStatus   string         `gorm:"size:20;default:'draft';index"` // draft, content_ready, images_ready, video_ready, published
	CurrentStep      int            `gorm:"default:1"`                     // 1-5 (setup, create, images, video, publish)
	Likes            int            `gorm:"default:0;index"`
	Comments         int            `gorm:"default:0"`
	Shares           int            `gorm:"default:0"`
	ForkCount        int            `gorm:"default:0;index"`
	Views            int            `gorm:"default:0;index"`
	TokenConsumption int            `gorm:"default:0"` // Aggregated from all generation records
	CreatedAt        time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// StoryboardContentGeneration - Step 1: Raw input → AI content
type StoryboardContentGeneration struct {
	ID               string         `gorm:"primaryKey;size:36"`
	StoryboardID     string         `gorm:"size:36;not null;index"`
	Storyboard       Storyboard     `gorm:"foreignKey:StoryboardID"`
	RawInput         string         `gorm:"type:text"`                                // User's original story input
	CharacterIDsJSON string         `gorm:"type:json"`                                // JSON array of character IDs
	SceneIDsJSON     string         `gorm:"type:json"`                                // JSON array of scene IDs
	Style            string         `gorm:"size:50"`                                  // action, drama, comedy, mystery
	GeneratedContent string         `gorm:"type:text"`                                // AI-generated narrative
	Status           string         `gorm:"size:20;not null;default:'pending';index"` // pending, processing, completed, failed
	InputTokens      int            `gorm:"default:0"`
	OutputTokens     int            `gorm:"default:0"`
	TotalTokens      int            `gorm:"default:0;index"`
	ErrorMessage     string         `gorm:"type:text"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;index"`
	CompletedAt      *time.Time     `gorm:"index"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// StoryboardSceneGeneration - Step 2: Scene → detailed description
type StoryboardSceneGeneration struct {
	ID               string         `gorm:"primaryKey;size:36"`
	StoryboardID     string         `gorm:"size:36;not null;index"`
	Storyboard       Storyboard     `gorm:"foreignKey:StoryboardID"`
	SceneID          string         `gorm:"size:36;not null;index"` // Reference to StoryScene
	SceneTitle       string         `gorm:"size:200"`               // Snapshot of scene title
	SceneLocation    string         `gorm:"size:200"`               // Snapshot of scene location
	InputDescription string         `gorm:"type:text"`              // Original scene description
	GeneratedDetail  string         `gorm:"type:text"`              // AI-enhanced description
	Status           string         `gorm:"size:20;not null;default:'pending';index"`
	InputTokens      int            `gorm:"default:0"`
	OutputTokens     int            `gorm:"default:0"`
	TotalTokens      int            `gorm:"default:0;index"`
	ErrorMessage     string         `gorm:"type:text"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;index"`
	CompletedAt      *time.Time     `gorm:"index"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// StoryboardImageGeneration - Step 3: Scene + refs → image
type StoryboardImageGeneration struct {
	ID                  string         `gorm:"primaryKey;size:36"`
	StoryboardID        string         `gorm:"size:36;not null;index"`
	Storyboard          Storyboard     `gorm:"foreignKey:StoryboardID"`
	SceneID             string         `gorm:"size:36;not null;index"` // Reference to StoryScene
	SceneTitle          string         `gorm:"size:200"`               // Snapshot of scene title
	SceneDescription    string         `gorm:"type:text"`              // Snapshot of scene description
	ReferenceImagesJSON string         `gorm:"type:json"`              // JSON array of reference image URLs
	GeneratedPrompt     string         `gorm:"type:text"`              // AI-generated image prompt
	GeneratedImageURL   string         `gorm:"size:2048"`              // Final generated image URL (needs to be long for signed URLs)
	ImageWidth          int            `gorm:"default:0"`
	ImageHeight         int            `gorm:"default:0"`
	Status              string         `gorm:"size:20;not null;default:'pending';index"`
	InputTokens         int            `gorm:"default:0"`
	OutputTokens        int            `gorm:"default:0"`
	TotalTokens         int            `gorm:"default:0;index"`
	ErrorMessage        string         `gorm:"type:text"`
	CreatedAt           time.Time      `gorm:"autoCreateTime;index"`
	CompletedAt         *time.Time     `gorm:"index"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

// StoryboardVideoGeneration - Step 4: Scene → video
type StoryboardVideoGeneration struct {
	ID                string         `gorm:"primaryKey;size:36"`
	StoryboardID      string         `gorm:"size:36;not null;index"`
	Storyboard        Storyboard     `gorm:"foreignKey:StoryboardID"`
	SceneID           string         `gorm:"size:36;not null;index"` // Reference to StoryScene
	SceneTitle        string         `gorm:"size:200"`               // Snapshot of scene title
	InputDescription  string         `gorm:"type:text"`              // Scene description for video
	ReferenceImageURL string         `gorm:"size:2048"`              // Start keyframe image for video generation
	EndFrameURL       string         `gorm:"size:2048"`              // End keyframe image for video transitions
	GeneratedPrompt   string         `gorm:"type:text"`              // AI-generated video prompt
	GeneratedVideoURL string         `gorm:"size:2048"`              // Final generated video URL (needs to be long for signed URLs)
	ProviderTaskID    string         `gorm:"size:128;index"`         // Provider's task ID for async video generation (for recovery)
	ProviderName      string         `gorm:"size:50"`                // Provider name (huoshan, hailuo, etc.) for recovery
	Duration          int            `gorm:"default:0"`              // Video duration in seconds
	Status            string         `gorm:"size:20;not null;default:'pending';index"`
	InputTokens       int            `gorm:"default:0"`
	OutputTokens      int            `gorm:"default:0"`
	TotalTokens       int            `gorm:"default:0;index"`
	ErrorMessage      string         `gorm:"type:text"`
	CreatedAt         time.Time      `gorm:"autoCreateTime;index"`
	CompletedAt       *time.Time     `gorm:"index"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// Character database model
type Character struct {
	ID              string         `gorm:"primaryKey;size:36"`
	StoryID         string         `gorm:"size:36;not null;index"`
	Story           Story          `gorm:"foreignKey:StoryID"`
	Name            string         `gorm:"size:100;not null;index"`
	Description     string         `gorm:"type:text"`
	Avatar          string         `gorm:"size:500"`
	Poster          string         `gorm:"size:500"`
	AuthorID        string         `gorm:"size:36;not null;index"`
	Author          User           `gorm:"foreignKey:AuthorID"`
	Personality     string         `gorm:"type:text"`
	Background      string         `gorm:"type:text"`
	ShortTermGoal   string         `gorm:"type:text"` // Immediate objectives in current story arc
	LongTermGoal    string         `gorm:"type:text"` // Overarching ambitions
	HandlingStyle   string         `gorm:"type:text"` // Approach to handling situations
	CognitionRange  string         `gorm:"type:text"` // Knowledge and awareness of their world
	AbilityFeatures string         `gorm:"type:text"` // Special skills and capabilities
	Appearance      string         `gorm:"type:text"` // Physical appearance and features
	DressPreference string         `gorm:"type:text"` // Clothing preferences and style
	SourceType      string         `gorm:"size:20;not null;default:'manual';index"`
	SourcePrompt    string         `gorm:"type:text"`
	SourceImage     string         `gorm:"size:500"`
	CreatedBy       string         `gorm:"size:36;not null;index"`
	LastEditedBy    string         `gorm:"size:36;index"`
	Likes           int            `gorm:"default:0;index"`
	Followers       int            `gorm:"default:0"`
	Stories         int            `gorm:"default:0"`
	Traits          string         `gorm:"type:text"` // JSON array
	Skills          string         `gorm:"type:text"` // JSON array
	IsPublic        bool           `gorm:"default:true;index"`
	GroupID         *string        `gorm:"size:36;index"`
	Group           *Group         `gorm:"foreignKey:GroupID"`
	CreatedAt       time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

// StoryScene database model (story-scoped scene assets - static locations)
// These are fixed locations like "school", "airport", "playground" that belong to a story.
type StoryScene struct {
	ID           string         `gorm:"primaryKey;size:36"`
	StoryID      string         `gorm:"size:36;not null;index"`
	Story        Story          `gorm:"foreignKey:StoryID"`
	Title        string         `gorm:"size:200;not null"`
	Description  string         `gorm:"type:text"`
	Image        string         `gorm:"size:500"`
	Location     string         `gorm:"size:100"`
	TimeOfDay    string         `gorm:"size:50"`
	SourceType   string         `gorm:"size:20;not null;default:'manual';index"`
	SourcePrompt string         `gorm:"type:text"`
	SourceImage  string         `gorm:"size:500"`
	CreatedBy    string         `gorm:"size:36;not null;index"`
	LastEditedBy string         `gorm:"size:36;index"`
	IsPublic     bool           `gorm:"default:false;index"`
	CreatedAt    time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// StoryboardScene database model (AI-generated plot scenes within a storyboard)
// These are dynamic scenes generated based on user input, characters, and selected StoryScenes.
// Different from StoryScene which is a static location.
type StoryboardScene struct {
	ID            string         `gorm:"primaryKey;size:36"`
	StoryboardID  string         `gorm:"size:36;not null;index"`
	Storyboard    Storyboard     `gorm:"foreignKey:StoryboardID"`
	StorySceneID  *string        `gorm:"size:36;index"` // Optional FK to story_scenes (location where plot happens)
	StoryScene    *StoryScene    `gorm:"foreignKey:StorySceneID"`
	Sequence      int            `gorm:"default:0;index"`
	Title         string         `gorm:"size:200;not null"`
	Description   string         `gorm:"type:text"`
	Image         string         `gorm:"size:500"`
	VideoUrl      string         `gorm:"size:2048"` // Generated video URL (populated from StoryboardVideoGeneration)
	Location      string         `gorm:"size:100"`  // AI-generated or from StoryScene
	TimeOfDay     string         `gorm:"size:50"`
	Characters    string         `gorm:"type:json"` // JSON array of character names
	Mood          string         `gorm:"size:100"`
	IsAIGenerated bool           `gorm:"default:true"`
	CreatedAt     time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// StoryboardCharacterLink links storyboards to characters.
type StoryboardCharacterLink struct {
	ID           string         `gorm:"primaryKey;size:36"`
	StoryboardID string         `gorm:"size:36;not null;index"`
	Storyboard   Storyboard     `gorm:"foreignKey:StoryboardID"`
	CharacterID  *string        `gorm:"size:36;index"` // Optional: NULL for minor/background characters
	Character    *Character     `gorm:"foreignKey:CharacterID"`
	Role         string         `gorm:"size:100"` // Character's role in this storyboard
	Ordering     int            `gorm:"default:0;index"`
	Notes        string         `gorm:"type:text"`
	CreatedAt    time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// StoryboardSceneLink links storyboards to story scenes.
type StoryboardSceneLink struct {
	ID             string         `gorm:"primaryKey;size:36"`
	StoryboardID   string         `gorm:"size:36;not null;index"`
	Storyboard     Storyboard     `gorm:"foreignKey:StoryboardID"`
	StorySceneID   string         `gorm:"size:36;not null;index"`
	StoryScene     *StoryScene    `gorm:"foreignKey:StorySceneID"`
	Sequence       int            `gorm:"default:0;index"`
	IsPrimaryScene bool           `gorm:"default:false"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// Group database model
type Group struct {
	ID          string         `gorm:"primaryKey;size:36"`
	Name        string         `gorm:"size:100;not null;index"`
	Description string         `gorm:"type:text"`
	Avatar      string         `gorm:"size:500"`
	Members     int            `gorm:"default:0"`
	Stories     int            `gorm:"default:0"`
	CreatorID   string         `gorm:"size:36;not null;index"`
	Creator     User           `gorm:"foreignKey:CreatorID"`
	Public      bool           `gorm:"default:true;index"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// Comment database model (支持嵌套回复和多目标类型)
type Comment struct {
	ID         string         `gorm:"primaryKey;size:36"`
	AuthorID   string         `gorm:"size:36;not null;index"`
	Author     User           `gorm:"foreignKey:AuthorID"`
	Content    string         `gorm:"type:text;not null"`
	TargetType string         `gorm:"size:20;not null;index"` // story, storyboard, character, comment
	TargetID   string         `gorm:"size:36;not null;index:idx_target"`
	ParentID   *string        `gorm:"size:36;index"` // null for top-level comments
	Parent     *Comment       `gorm:"foreignKey:ParentID"`
	RootID     *string        `gorm:"size:36;index"` // root comment ID for nested replies
	Root       *Comment       `gorm:"foreignKey:RootID"`
	Likes      int            `gorm:"default:0;index"`
	Dislikes   int            `gorm:"default:0"`
	ReplyCount int            `gorm:"default:0"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// ChatThread database model
type ChatThread struct {
	ID                   string         `gorm:"primaryKey;size:36"`
	CharacterID          string         `gorm:"size:36;not null;index"`
	Character            Character      `gorm:"foreignKey:CharacterID"`
	UserID               string         `gorm:"size:36;not null;index"`
	User                 User           `gorm:"foreignKey:UserID"`
	StoryTitle           string         `gorm:"size:200"`
	LastMessage          string         `gorm:"type:text"`
	LastMessageTime      time.Time      `gorm:"index"`
	UnreadCount          int            `gorm:"default:0"`
	MessageCount         int            `gorm:"default:0"`
	InteractionFrequency float64        `gorm:"default:0"`
	SelectedStoryboardID *string        `gorm:"size:36;index"`
	ContextStoryboardIDs string         `gorm:"type:json"` // JSON array of storyboard IDs
	TotalTokensUsed      int64          `gorm:"default:0"`
	IsArchived           bool           `gorm:"default:false;index"`
	Summary              string         `gorm:"type:text"`
	CreatedAt            time.Time      `gorm:"autoCreateTime"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime"`
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

// ChatMessage database model
type ChatMessage struct {
	ID            string         `gorm:"primaryKey;size:36"`
	ThreadID      string         `gorm:"size:36;not null;index"`
	Thread        ChatThread     `gorm:"foreignKey:ThreadID"`
	SenderID      string         `gorm:"size:36;not null"`
	SenderName    string         `gorm:"size:100"`
	SenderAvatar  string         `gorm:"size:500"`
	Content       string         `gorm:"type:text;not null"`
	Image         string         `gorm:"size:500"`
	IsUser        bool           `gorm:"default:false"`
	IsArchived    bool           `gorm:"default:false;index"`
	ReactionCount int            `gorm:"default:0;index"`
	CreatedAt     time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// ChatThreadStoryboardBranch database model
type ChatThreadStoryboardBranch struct {
	ID           string         `gorm:"primaryKey;size:36"`
	ThreadID     string         `gorm:"size:36;not null;index"`
	Thread       ChatThread     `gorm:"foreignKey:ThreadID"`
	StoryboardID string         `gorm:"size:36;not null;index"`
	Storyboard   Storyboard     `gorm:"foreignKey:StoryboardID"`
	CharacterID  string         `gorm:"size:36;not null;index"`
	Character    Character      `gorm:"foreignKey:CharacterID"`
	SelectedAt   time.Time      `gorm:"index"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// ChatMessageReaction database model
type ChatMessageReaction struct {
	ID           string         `gorm:"primaryKey;size:36"`
	MessageID    string         `gorm:"size:36;not null;index"`
	Message      ChatMessage    `gorm:"foreignKey:MessageID"`
	UserID       string         `gorm:"size:36;not null;index"`
	User         User           `gorm:"foreignKey:UserID"`
	ReactionType string         `gorm:"size:20;not null;index"` // like, dislike, emoji
	EmojiCode    string         `gorm:"size:10"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// ChatMessageToken database model
type ChatMessageToken struct {
	ID           string      `gorm:"primaryKey;size:36"`
	MessageID    string      `gorm:"size:36;not null;uniqueIndex"`
	Message      ChatMessage `gorm:"foreignKey:MessageID"`
	InputTokens  int         `gorm:"default:0"`
	OutputTokens int         `gorm:"default:0"`
	TotalTokens  int         `gorm:"default:0;index"`
	Model        string      `gorm:"size:100"`
	CreatedAt    time.Time   `gorm:"autoCreateTime"`
	UpdatedAt    time.Time   `gorm:"autoUpdateTime"`
}

// StoryComposition database model
type StoryComposition struct {
	ID               string         `gorm:"primaryKey;size:36"`
	Title            string         `gorm:"size:200;not null"`
	CoverImage       string         `gorm:"size:500"`
	Background       string         `gorm:"type:text"`
	Theme            string         `gorm:"size:100"`
	Genre            string         `gorm:"size:50"`
	RootStoryboardID string         `gorm:"size:36"`
	TotalStoryboards int            `gorm:"default:0"`
	TotalForks       int            `gorm:"default:0"`
	CreatedAt        time.Time      `gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// StoryParticipant database model
type StoryParticipant struct {
	ID            string           `gorm:"primaryKey;size:36"`
	CompositionID string           `gorm:"size:36;not null;index"`
	Composition   StoryComposition `gorm:"foreignKey:CompositionID"`
	UserID        string           `gorm:"size:36;not null;index"`
	User          User             `gorm:"foreignKey:UserID"`
	Role          string           `gorm:"size:20;not null"` // owner, collaborator, contributor
	JoinedAt      time.Time        `gorm:"autoCreateTime"`
	DeletedAt     gorm.DeletedAt   `gorm:"index"`
}

// (Storyboard 已在上面定义)

// GroupActivity database model
type GroupActivity struct {
	ID        string         `gorm:"primaryKey;size:36"`
	GroupID   string         `gorm:"size:36;not null;index"`
	Group     Group          `gorm:"foreignKey:GroupID"`
	Type      string         `gorm:"size:50;not null"`
	UserID    string         `gorm:"size:36;not null;index"`
	User      User           `gorm:"foreignKey:UserID"`
	StoryID   *string        `gorm:"size:36;index"`
	Story     *Story         `gorm:"foreignKey:StoryID"`
	Message   string         `gorm:"size:500"`
	CreatedAt time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// ========== 关系表 ==========

// UserFollow 用户关注关系
type UserFollow struct {
	ID         string         `gorm:"primaryKey;size:36"`
	FollowerID string         `gorm:"size:36;not null;index:idx_follower_followee,unique"`
	Follower   User           `gorm:"foreignKey:FollowerID"`
	FolloweeID string         `gorm:"size:36;not null;index:idx_follower_followee,unique;index"`
	Followee   User           `gorm:"foreignKey:FolloweeID"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// StoryLike 故事点赞
type StoryLike struct {
	ID        string         `gorm:"primaryKey;size:36"`
	UserID    string         `gorm:"size:36;not null;index:idx_user_story,unique"`
	User      User           `gorm:"foreignKey:UserID"`
	StoryID   string         `gorm:"size:36;not null;index:idx_user_story,unique;index"`
	Story     Story          `gorm:"foreignKey:StoryID"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// StoryFollow 故事关注
type StoryFollow struct {
	ID        string         `gorm:"primaryKey;size:36"`
	UserID    string         `gorm:"size:36;not null;index:idx_user_story_follow,unique"`
	User      User           `gorm:"foreignKey:UserID"`
	StoryID   string         `gorm:"size:36;not null;index:idx_user_story_follow,unique;index"`
	Story     Story          `gorm:"foreignKey:StoryID"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// CharacterFollow 角色关注
type CharacterFollow struct {
	ID          string         `gorm:"primaryKey;size:36"`
	UserID      string         `gorm:"size:36;not null;index:idx_user_character,unique"`
	User        User           `gorm:"foreignKey:UserID"`
	CharacterID string         `gorm:"size:36;not null;index:idx_user_character,unique;index"`
	Character   Character      `gorm:"foreignKey:CharacterID"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// GroupRole 群组角色定义表
type GroupRole struct {
	ID          string    `gorm:"primaryKey;size:36"`
	RoleCode    string    `gorm:"size:50;not null;uniqueIndex;index"` // creator, admin, member, outsider
	Name        string    `gorm:"size:100;not null"`                  // 小组创建者、小组管理员、小组成员、小组外部人员
	Description string    `gorm:"type:text"`
	IsSystem    bool      `gorm:"default:true;index"` // 是否为系统内置角色
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// GroupMember 群组成员
type GroupMember struct {
	ID        string         `gorm:"primaryKey;size:36"`
	GroupID   string         `gorm:"size:36;not null;index:idx_group_user,unique"`
	Group     Group          `gorm:"foreignKey:GroupID"`
	UserID    string         `gorm:"size:36;not null;index:idx_group_user,unique;index"`
	User      User           `gorm:"foreignKey:UserID"`
	Role      string         `gorm:"size:20;not null"` // owner, admin, moderator, member (保留用于向后兼容)
	RoleID    string         `gorm:"size:36;index"`    // 关联到GroupRole表
	RoleRef   GroupRole      `gorm:"foreignKey:RoleID"`
	InvitedBy string         `gorm:"size:36"`
	JoinedAt  time.Time      `gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// GroupInvitation 群组邀请
type GroupInvitation struct {
	ID        string         `gorm:"primaryKey;size:36"`
	GroupID   string         `gorm:"size:36;not null;index"`
	Group     Group          `gorm:"foreignKey:GroupID"`
	InviterID string         `gorm:"size:36;not null;index"`
	Inviter   User           `gorm:"foreignKey:InviterID"`
	InviteeID string         `gorm:"size:36;not null;index"`
	Invitee   User           `gorm:"foreignKey:InviteeID"`
	Status    string         `gorm:"size:20;not null;default:'pending';index"`
	Message   string         `gorm:"type:text"`
	CreatedAt time.Time      `gorm:"autoCreateTime;index"`
	ExpiresAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// CommentLike 评论点赞
type CommentLike struct {
	ID        string         `gorm:"primaryKey;size:36"`
	UserID    string         `gorm:"size:36;not null;index:idx_user_comment,unique"`
	User      User           `gorm:"foreignKey:UserID"`
	CommentID string         `gorm:"size:36;not null;index:idx_user_comment,unique;index"`
	Comment   Comment        `gorm:"foreignKey:CommentID"`
	IsLike    bool           `gorm:"not null"` // true=like, false=dislike
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// StoryboardLike Storyboard 点赞
type StoryboardLike struct {
	ID           string         `gorm:"primaryKey;size:36"`
	UserID       string         `gorm:"size:36;not null;index:idx_user_storyboard,unique"`
	User         User           `gorm:"foreignKey:UserID"`
	StoryboardID string         `gorm:"size:36;not null;index:idx_user_storyboard,unique;index"`
	Storyboard   Storyboard     `gorm:"foreignKey:StoryboardID"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// ========== 资产管理 ==========

// Asset 用户资产（图片、音频等）
type Asset struct {
	ID         string         `gorm:"primaryKey;size:36"`
	UserID     string         `gorm:"size:36;not null;index"`
	User       User           `gorm:"foreignKey:UserID"`
	Type       string         `gorm:"size:20;not null;index"` // image, audio, video
	Name       string         `gorm:"size:200;not null"`
	URL        string         `gorm:"size:500;not null"`
	Thumbnail  string         `gorm:"size:500"`
	Size       int64          `gorm:"not null"`
	MimeType   string         `gorm:"size:100"`
	Width      int            `gorm:""`
	Height     int            `gorm:""`
	Duration   int            `gorm:""`          // 秒（音视频）
	Tags       string         `gorm:"type:json"` // JSON array
	UsageCount int            `gorm:"default:0"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// ========== 通知系统 ==========

// Notification 通知
type Notification struct {
	ID          string         `gorm:"primaryKey;size:36"`
	UserID      string         `gorm:"size:36;not null;index"`
	User        User           `gorm:"foreignKey:UserID"`
	Type        string         `gorm:"size:50;not null;index"` // like, comment, follow, mention, system
	Title       string         `gorm:"size:200;not null"`
	Content     string         `gorm:"type:text"`
	Link        string         `gorm:"size:500"`
	Read        bool           `gorm:"default:false;index"`
	ActorID     string         `gorm:"size:36;index"`
	ActorName   string         `gorm:"size:100"`
	ActorAvatar string         `gorm:"size:500"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ========== 用户设置 ==========

// UserSettings 用户设置
type UserSettings struct {
	ID                 string         `gorm:"primaryKey;size:36"`
	UserID             string         `gorm:"uniqueIndex;size:36;not null"`
	User               User           `gorm:"foreignKey:UserID"`
	Language           string         `gorm:"size:10;default:'en'"`   // en, zh, ja
	Theme              string         `gorm:"size:20;default:'auto'"` // light, dark, auto
	EmailNotifications bool           `gorm:"default:true"`
	PushNotifications  bool           `gorm:"default:true"`
	ShowAdultContent   bool           `gorm:"default:false"`
	ProfileVisibility  string         `gorm:"size:20;default:'public'"` // public, friends, private
	AllowComments      bool           `gorm:"default:true"`
	AllowMessages      bool           `gorm:"default:true"`
	ShowOnlineStatus   bool           `gorm:"default:true"`
	UpdatedAt          time.Time      `gorm:"autoUpdateTime"`
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

// ========== 会员系统 ==========

// Membership 会员信息
type Membership struct {
	ID           string         `gorm:"primaryKey;size:36"`
	UserID       string         `gorm:"uniqueIndex;size:36;not null"`
	User         User           `gorm:"foreignKey:UserID"`
	Tier         string         `gorm:"size:20;not null;index"` // free, basic, pro, enterprise
	Status       string         `gorm:"size:20;not null;index"` // active, expired, cancelled
	StartDate    time.Time      `gorm:"not null"`
	EndDate      *time.Time     `gorm:"index"`
	AutoRenew    bool           `gorm:"default:false"`
	TokenQuota   int            `gorm:"default:0"`
	TokenUsed    int            `gorm:"default:0"`
	StorageQuota int64          `gorm:"default:0"`
	StorageUsed  int64          `gorm:"default:0"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// ========== AI 生成记录 ==========

// AIGenerationRecord AI 生成记录 - 记录AI能力使用数据（任务管理、Token计费）
type AIGenerationRecord struct {
	ID       string `gorm:"primaryKey;size:36"`
	UserID   string `gorm:"size:36;not null;index"`
	User     User   `gorm:"foreignKey:UserID"`
	Type     string `gorm:"size:20;not null;index"` // text, image, video, audio
	Provider string `gorm:"size:50;not null;index"` // gemini, hailuo, huoshan, qwen
	Model    string `gorm:"size:100"`

	// 提示词信息
	OriginalPrompt string `gorm:"type:text"` // 原始提示词
	EnhancedPrompt string `gorm:"type:text"` // 增强后的提示词
	SystemPrompt   string `gorm:"type:text"` // 系统提示词
	InputParams    string `gorm:"type:json"` // 完整的输入参数（JSON）
	OutputResult   string `gorm:"type:json"` // 完整的输出结果（JSON）

	// Token 消耗统计
	InputTokens  int `gorm:"default:0"`       // 输入 token 数
	OutputTokens int `gorm:"default:0"`       // 输出 token 数
	TotalTokens  int `gorm:"default:0;index"` // 总 token 数
	ImageCount   int `gorm:"default:0"`       // 生成图片数量
	VideoCount   int `gorm:"default:0"`       // 生成视频数量

	// 任务状态
	Status       string `gorm:"size:20;not null;index"` // pending, processing, completed, failed
	Progress     int    `gorm:"default:0"`              // 0-100
	ErrorMessage string `gorm:"type:text"`              // 错误信息
	ErrorCode    string `gorm:"size:50"`                // 错误码

	// 时间统计（毫秒）
	DurationMs    int64 `gorm:"default:0"` // 总耗时
	QueueTimeMs   int64 `gorm:"default:0"` // 排队时间
	ProcessTimeMs int64 `gorm:"default:0"` // 处理时间

	// 关联的业务实体
	RelatedEntityID   string `gorm:"size:36;index"` // 关联实体ID
	RelatedEntityType string `gorm:"size:50;index"` // story, storyboard, character, poster

	// 时间戳
	CreatedAt   time.Time  `gorm:"autoCreateTime;index"` // 创建时间
	StartedAt   *time.Time `gorm:"index"`                // 开始处理时间
	CompletedAt *time.Time `gorm:"index"`                // 完成时间

	// 扩展元数据
	Metadata  string         `gorm:"type:json"` // 扩展元数据（JSON）
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// ========== 标签系统 ==========

// Tag 标签
type Tag struct {
	ID         string         `gorm:"primaryKey;size:36"`
	Name       string         `gorm:"uniqueIndex;size:100;not null"`
	Category   string         `gorm:"size:50;index"` // genre, theme, style, mood
	UsageCount int            `gorm:"default:0;index"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// StyleConfig 风格配置
type StyleConfig struct {
	ID             string `gorm:"primaryKey;size:36"`
	Style          string `gorm:"size:100;not null;index"`
	Description    string `gorm:"type:text"`
	SampleImageURL string `gorm:"column:sample_image_url;size:512"` // 示例图片URL
	GroupID        string `gorm:"column:group_id;size:36;index"`    // 所属小组ID（私有风格）
	UserID         string `gorm:"column:user_id;size:36;index"`     // 所属用户ID（私有风格）
	CreatedAt      int64  `gorm:"type:bigint;autoCreateTime;index"`
	UpdatedAt      int64  `gorm:"type:bigint;autoUpdateTime"`
}

// StoryTag 故事标签关联
type StoryTag struct {
	ID        string         `gorm:"primaryKey;size:36"`
	StoryID   string         `gorm:"size:36;not null;index:idx_story_tag,unique"`
	Story     Story          `gorm:"foreignKey:StoryID"`
	TagID     string         `gorm:"size:36;not null;index:idx_story_tag,unique;index"`
	Tag       Tag            `gorm:"foreignKey:TagID"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// CharacterTag 角色标签关联
type CharacterTag struct {
	ID          string         `gorm:"primaryKey;size:36"`
	CharacterID string         `gorm:"size:36;not null;index:idx_character_tag,unique"`
	Character   Character      `gorm:"foreignKey:CharacterID"`
	TagID       string         `gorm:"size:36;not null;index:idx_character_tag,unique;index"`
	Tag         Tag            `gorm:"foreignKey:TagID"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ========== 搜索历史 ==========

// SearchHistory 搜索历史
type SearchHistory struct {
	ID          string         `gorm:"primaryKey;size:36"`
	UserID      string         `gorm:"size:36;not null;index"`
	User        User           `gorm:"foreignKey:UserID"`
	Query       string         `gorm:"size:200;not null"`
	Type        string         `gorm:"size:50;index"` // story, character, user, group
	ResultCount int            `gorm:"default:0"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ========== 浏览历史 ==========

// ViewHistory 浏览历史
type ViewHistory struct {
	ID         string         `gorm:"primaryKey;size:36"`
	UserID     string         `gorm:"size:36;not null;index"`
	User       User           `gorm:"foreignKey:UserID"`
	EntityType string         `gorm:"size:50;not null;index"` // story, storyboard, character
	EntityID   string         `gorm:"size:36;not null;index"`
	Duration   int            `gorm:"default:0"` // 浏览时长（秒）
	ViewedAt   time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// ========== 举报系统 ==========

// Report 举报
type Report struct {
	ID          string         `gorm:"primaryKey;size:36"`
	ReporterID  string         `gorm:"size:36;not null;index"`
	Reporter    User           `gorm:"foreignKey:ReporterID"`
	EntityType  string         `gorm:"size:50;not null;index"` // story, comment, user, character
	EntityID    string         `gorm:"size:36;not null;index"`
	Reason      string         `gorm:"size:50;not null"` // spam, inappropriate, copyright, other
	Description string         `gorm:"type:text"`
	Status      string         `gorm:"size:20;not null;index"` // pending, reviewed, resolved, rejected
	ReviewerID  string         `gorm:"size:36;index"`
	Reviewer    *User          `gorm:"foreignKey:ReviewerID"`
	ReviewNote  string         `gorm:"type:text"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	ReviewedAt  *time.Time     `gorm:"index"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ========== 用户活动 ==========

// UserActivity 用户活动记录
type UserActivity struct {
	ID          string         `gorm:"primaryKey;size:36"`
	UserID      string         `gorm:"size:36;not null;index"`
	User        User           `gorm:"foreignKey:UserID"`
	Type        string         `gorm:"size:50;not null;index"` // story_created, story_updated, story_published, story_liked, character_created, character_updated, user_followed, storyboard_created, panel_added
	TargetID    string         `gorm:"size:36;index"`
	TargetType  string         `gorm:"size:20"` // story, character, user, storyboard
	TargetTitle string         `gorm:"size:255"`
	Message     string         `gorm:"size:500"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ========== 角色海报 ==========

// CharacterPoster 角色海报
type CharacterPoster struct {
	ID          string    `gorm:"primaryKey;size:36"`
	CharacterID string    `gorm:"size:36;not null;index"`
	Character   Character `gorm:"foreignKey:CharacterID"`
	AuthorID    string    `gorm:"size:36;not null;index"`
	Author      User      `gorm:"foreignKey:AuthorID"`
	Type        string    `gorm:"size:20;not null;default:'image';index"` // image, video
	Title       string    `gorm:"size:200;not null"`
	Image       string    `gorm:"size:1000"`                              // Poster image URL (for image type)
	Video       string    `gorm:"size:1000"`                              // Video URL (for video type)
	Thumbnail   string    `gorm:"size:1000"`                              // Video thumbnail URL
	Duration    int       `gorm:"default:0"`                              // Video duration in seconds
	Prompt      string    `gorm:"type:text"`                              // User's original prompt/description
	Status      string    `gorm:"size:20;not null;default:'draft';index"` // draft, generating, generated, published, failed

	// AI Generation fields
	ReferenceStoryEnabled bool   `gorm:"default:false"` // Whether to reference recent story plots
	PosterConceptJSON     string `gorm:"type:text"`     // LLM generated poster concept JSON
	FinalImagePrompt      string `gorm:"type:text"`     // Final assembled prompt for image generation
	ErrorMessage          string `gorm:"type:text"`     // Error message if generation failed
	ConceptGenerationID   string `gorm:"size:36;index"` // AI record for concept generation (Step 1)
	ImageGenerationID     string `gorm:"size:36;index"` // AI record for image generation (Step 2)

	Likes     int            `gorm:"default:0;index"`
	Shares    int            `gorm:"default:0"`
	CreatedAt time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// ========== 角色分析 ==========

// CharacterAnalytics 角色分析数据
type CharacterAnalytics struct {
	ID                   string         `gorm:"primaryKey;size:36"`
	CharacterID          string         `gorm:"uniqueIndex;size:36;not null"`
	Character            Character      `gorm:"foreignKey:CharacterID"`
	UsersWhoChattedCount int            `gorm:"default:0"`
	TotalMessagesSent    int            `gorm:"default:0"`
	TotalTokensConsumed  int64          `gorm:"default:0"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime"`
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

// ========== Agent 系统 ==========

// Agent AI Agent 数据库模型
type Agent struct {
	ID               string         `gorm:"primaryKey;size:36"`
	CharacterID      string         `gorm:"uniqueIndex;size:36;not null"`
	Character        Character      `gorm:"foreignKey:CharacterID"`
	Name             string         `gorm:"size:100;not null"`
	Description      string         `gorm:"type:text"`
	Status           string         `gorm:"size:20;not null;default:'active';index"` // active, inactive, training
	SystemPrompt     string         `gorm:"type:text;not null"`
	Temperature      float64        `gorm:"default:0.7"`
	Provider         string         `gorm:"size:50;not null"`
	Model            string         `gorm:"size:100"`
	MaxTokens        int            `gorm:"default:2048"`
	InteractionCount int            `gorm:"default:0"`
	SkillCount       int            `gorm:"default:0"`
	Config           string         `gorm:"type:json"` // 扩展配置
	CreatedAt        time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// AgentSkill Agent 技能数据库模型
type AgentSkill struct {
	ID               string         `gorm:"primaryKey;size:36"`
	AgentID          string         `gorm:"size:36;not null;index"`
	Agent            Agent          `gorm:"foreignKey:AgentID"`
	Name             string         `gorm:"size:100;not null;index"`
	DisplayName      string         `gorm:"size:100;not null"`
	Description      string         `gorm:"type:text;not null"`
	Type             string         `gorm:"size:50;not null;index"` // creative, technical, conversation, etc.
	Status           string         `gorm:"size:20;not null;default:'draft';index"`
	Instructions     string         `gorm:"type:text;not null"`
	Examples         string         `gorm:"type:json"` // JSON array
	Guidelines       string         `gorm:"type:json"` // JSON array
	Metadata         string         `gorm:"type:json"` // JSON object
	UsageCount       int            `gorm:"default:0;index"`
	SuccessCount     int            `gorm:"default:0"`
	FailureCount     int            `gorm:"default:0"`
	AvgExecutionTime int            `gorm:"default:0"` // milliseconds
	Priority         int            `gorm:"default:50;index"`
	Enabled          bool           `gorm:"default:true;index"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// AgentSkillUsage Agent 技能使用记录
type AgentSkillUsage struct {
	ID             string         `gorm:"primaryKey;size:36"`
	AgentID        string         `gorm:"size:36;not null;index"`
	Agent          Agent          `gorm:"foreignKey:AgentID"`
	SkillID        string         `gorm:"size:36;not null;index"`
	Skill          AgentSkill     `gorm:"foreignKey:SkillID"`
	UserID         string         `gorm:"size:36;index"`
	User           *User          `gorm:"foreignKey:UserID"`
	ConversationID string         `gorm:"size:36;index"`
	Scenario       string         `gorm:"type:text"`
	InputData      string         `gorm:"type:text"`
	OutputData     string         `gorm:"type:text"`
	Success        bool           `gorm:"not null;index"`
	ErrorMessage   string         `gorm:"type:text"`
	ExecutionTime  int            `gorm:"default:0"` // milliseconds
	TokensUsed     int            `gorm:"default:0"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// AgentInteraction Agent 交互记录
type AgentInteraction struct {
	ID          string         `gorm:"primaryKey;size:36"`
	AgentID     string         `gorm:"size:36;not null;index"`
	Agent       Agent          `gorm:"foreignKey:AgentID"`
	UserID      string         `gorm:"size:36;not null;index"`
	User        User           `gorm:"foreignKey:UserID"`
	CharacterID string         `gorm:"size:36;not null;index"`
	Character   Character      `gorm:"foreignKey:CharacterID"`
	MessageID   string         `gorm:"size:36;index"`
	Type        string         `gorm:"size:50;not null;index"`
	InputText   string         `gorm:"type:text"`
	OutputText  string         `gorm:"type:text"`
	TokensUsed  int            `gorm:"default:0"`
	Duration    int            `gorm:"default:0"` // milliseconds
	SkillsUsed  string         `gorm:"type:json"` // JSON array of skill IDs
	Success     bool           `gorm:"not null;index"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// AgentMemory Agent 记忆存储
type AgentMemory struct {
	ID           string         `gorm:"primaryKey;size:36"`
	AgentID      string         `gorm:"size:36;not null;index"`
	Agent        Agent          `gorm:"foreignKey:AgentID"`
	UserID       string         `gorm:"size:36;not null;index"`
	User         User           `gorm:"foreignKey:UserID"`
	MemoryType   string         `gorm:"size:50;not null;index"` // short_term, long_term, episodic
	Key          string         `gorm:"size:200;not null;index"`
	Value        string         `gorm:"type:text;not null"`
	Importance   int            `gorm:"default:50;index"` // 0-100
	AccessCount  int            `gorm:"default:0"`
	LastAccessed time.Time      `gorm:"index"`
	ExpiresAt    *time.Time     `gorm:"index"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// ========== AI 任务系统 ==========

// AITask AI 任务数据库模型
type AITask struct {
	ID                string         `gorm:"primaryKey;size:36"`
	UserID            string         `gorm:"size:36;not null;index"`
	User              User           `gorm:"foreignKey:UserID"`
	Type              string         `gorm:"size:50;not null;index"` // generate_story, enhance_prompt, generate_image, generate_video
	Status            string         `gorm:"size:20;not null;index"` // pending, processing, completed, failed, cancelled
	Provider          string         `gorm:"size:50"`
	Model             string         `gorm:"size:100"`
	Input             string         `gorm:"type:text;not null"`
	Output            string         `gorm:"type:text"`
	TokensUsed        int            `gorm:"default:0"`
	Progress          int            `gorm:"default:0"`
	ErrorMessage      string         `gorm:"type:text"`
	RelatedEntityID   string         `gorm:"size:36;index"`
	RelatedEntityType string         `gorm:"size:20"`
	QueueName         string         `gorm:"size:50;index"`
	RetryCount        int            `gorm:"default:0"`
	MaxRetries        int            `gorm:"default:3"`
	CreatedAt         time.Time      `gorm:"autoCreateTime;index"`
	StartedAt         *time.Time     `gorm:"index"`
	CompletedAt       *time.Time     `gorm:"index"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// RenderTask 渲染任务数据库模型
type RenderTask struct {
	ID           string         `gorm:"primaryKey;size:36"`
	UserID       string         `gorm:"size:36;not null;index"`
	User         User           `gorm:"foreignKey:UserID"`
	StoryID      string         `gorm:"size:36;not null;index"`
	Story        Story          `gorm:"foreignKey:StoryID"`
	Type         string         `gorm:"size:20;not null"` // video, image_set, animation
	Status       string         `gorm:"size:20;not null;index"`
	Config       string         `gorm:"type:text"`
	OutputURL    string         `gorm:"size:500"`
	ThumbnailURL string         `gorm:"size:500"`
	Progress     int            `gorm:"default:0"`
	ErrorMessage string         `gorm:"type:text"`
	FileSize     int64          `gorm:"default:0"`
	Duration     int            `gorm:"default:0"`
	Resolution   string         `gorm:"size:20"`
	QueueName    string         `gorm:"size:50;index"`
	RetryCount   int            `gorm:"default:0"`
	MaxRetries   int            `gorm:"default:3"`
	CreatedAt    time.Time      `gorm:"autoCreateTime;index"`
	StartedAt    *time.Time     `gorm:"index"`
	CompletedAt  *time.Time     `gorm:"index"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// StoryPublication 故事发布记录
type StoryPublication struct {
	ID            string         `gorm:"primaryKey;size:36"`
	StoryID       string         `gorm:"size:36;not null;uniqueIndex:idx_story_version"`
	Story         Story          `gorm:"foreignKey:StoryID"`
	Version       int            `gorm:"not null;uniqueIndex:idx_story_version"`
	Status        string         `gorm:"size:20;not null"` // published, unpublished
	RenderTaskID  string         `gorm:"size:36;index"`
	RenderTask    *RenderTask    `gorm:"foreignKey:RenderTaskID"`
	PublishedAt   time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	UnpublishedAt *time.Time     `gorm:"index"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// ========== 支付订阅系统 ==========

// SubscriptionPlan 订阅计划
type SubscriptionPlan struct {
	ID            string         `gorm:"primaryKey;size:36"`
	Name          string         `gorm:"size:100;not null;uniqueIndex"`
	Price         float64        `gorm:"not null"`
	Currency      string         `gorm:"size:10;not null;default:'USD'"`
	TokenQuota    int            `gorm:"default:0"`
	StorageQuota  int64          `gorm:"default:0"`
	MaxStories    int            `gorm:"default:0"`
	MaxCharacters int            `gorm:"default:0"`
	Features      string         `gorm:"type:json"` // JSON array
	IsActive      bool           `gorm:"default:true;index"`
	SortOrder     int            `gorm:"default:0;index"`
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// SubscriptionOrder 订阅订单
type SubscriptionOrder struct {
	ID            string           `gorm:"primaryKey;size:36"`
	UserID        string           `gorm:"size:36;not null;index"`
	User          User             `gorm:"foreignKey:UserID"`
	PlanID        string           `gorm:"size:36;not null;index"`
	Plan          SubscriptionPlan `gorm:"foreignKey:PlanID"`
	Status        string           `gorm:"size:20;not null;index"` // pending, paid, failed, refunded, cancelled
	Amount        float64          `gorm:"not null"`
	Currency      string           `gorm:"size:10;not null"`
	PaymentMethod string           `gorm:"size:50;not null"` // alipay, wechat, stripe, paypal
	PaymentID     string           `gorm:"size:200;index"`
	StartDate     time.Time        `gorm:"not null;index"`
	EndDate       time.Time        `gorm:"not null;index"`
	InvoiceURL    string           `gorm:"size:500"`
	CreatedAt     time.Time        `gorm:"autoCreateTime;index"`
	PaidAt        *time.Time       `gorm:"index"`
	UpdatedAt     time.Time        `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt   `gorm:"index"`
}

// TokenTransaction Token 交易记录
type TokenTransaction struct {
	ID          string         `gorm:"primaryKey;size:36"`
	UserID      string         `gorm:"size:36;not null;index"`
	User        User           `gorm:"foreignKey:UserID"`
	Type        string         `gorm:"size:20;not null;index"` // consume, recharge, refund, gift
	Amount      int            `gorm:"not null"`               // positive for recharge, negative for consume
	Balance     int            `gorm:"not null"`
	Source      string         `gorm:"size:50;not null;index"` // ai_generation, render, subscription, manual
	RelatedID   string         `gorm:"size:36;index"`
	Description string         `gorm:"size:500"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// InvitationCode 邀请码表
type InvitationCode struct {
	ID          string         `gorm:"primaryKey;size:36"`
	Code        string         `gorm:"size:50;not null;uniqueIndex;index"` // 邀请码（唯一）
	CreatedBy   string         `gorm:"size:36;not null;index"`
	Creator     User           `gorm:"foreignKey:CreatedBy"`
	UsedBy      string         `gorm:"size:36;index"` // 使用者用户ID（如果已使用）
	User        User           `gorm:"foreignKey:UsedBy"`
	UsedAt      time.Time      // 使用时间
	IsActive    bool           `gorm:"default:true;index"` // 是否启用
	MaxUses     int            `gorm:"default:1"`          // 最大使用次数（0表示无限制）
	CurrentUses int            `gorm:"default:0"`          // 当前使用次数
	ExpiresAt   time.Time      `gorm:"index"`              // 过期时间（零值表示永不过期）
	Description string         `gorm:"type:text"`          // 描述信息
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// migrate runs database migrations
func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// 核心实体
		&User{},
		&Story{},
		&Panel{},
		&Storyboard{},
		&Character{},
		&StoryScene{},      // Static story-level locations
		&StoryboardScene{}, // AI-generated plot scenes within storyboards
		&StoryboardCharacterLink{},
		&StoryboardSceneLink{},
		&Group{},

		// Storyboard AI 生成记录
		&StoryboardContentGeneration{},
		&StoryboardSceneGeneration{},
		&StoryboardImageGeneration{},
		&StoryboardVideoGeneration{},

		// 关系表
		&UserFollow{},
		&StoryLike{},
		&StoryFollow{},
		&StoryContributor{},
		&CharacterFollow{},
		&GroupMember{},
		&GroupRole{},
		&GroupInvitation{},
		&CommentLike{},
		&StoryboardLike{},

		// 业务功能
		&Comment{},
		&ChatThread{},
		&ChatMessage{},
		&GroupActivity{},
		&Asset{},
		&Notification{},
		&UserSettings{},
		&UserActivity{},
		&CharacterPoster{},
		&CharacterAnalytics{},

		// 标签系统
		&Tag{},
		&StoryTag{},
		&CharacterTag{},

		// 搜索和浏览
		&SearchHistory{},
		&ViewHistory{},
		&Report{},

		// Agent 系统
		&Agent{},
		&AgentSkill{},
		&AgentSkillUsage{},
		&AgentInteraction{},
		&AgentMemory{},

		// AI 和渲染任务
		&AITask{},
		&RenderTask{},
		&StoryPublication{},
		&AIGenerationRecord{},

		// 支付订阅
		&Membership{},
		&SubscriptionPlan{},
		&SubscriptionOrder{},
		&TokenTransaction{},

		// 邀请码系统
		&InvitationCode{},

		// 用户统计
		&UserStatistics{},

		// Storyboard Chat 会话
		&StoryboardChatSession{},
		&StoryboardChatMessage{},

		// 用户登录记录
		&UserLoginRecord{},

		// 第三方登录
		&ThirdPartyLogin{},
	)
}

// ThirdPartyLogin 第三方登录表（支持 Google/Apple 跨设备登录）
type ThirdPartyLogin struct {
	ID               string         `gorm:"primaryKey;size:36"`
	UserID           string         `gorm:"size:36;not null;index"`
	User             User           `gorm:"foreignKey:UserID"`
	Provider         string         `gorm:"size:32;not null;index:idx_provider_user_id,unique"` // google, apple, facebook, wechat
	ProviderUserID   string         `gorm:"size:255;not null;index:idx_provider_user_id,unique"`
	ProviderEmail    string         `gorm:"size:255;index"`
	ProviderUserName string         `gorm:"size:255"`
	ProviderUserInfo string         `gorm:"type:text"` // JSON 格式的完整用户信息
	AccessToken      string         `gorm:"type:text"`
	RefreshToken     string         `gorm:"type:text"`
	TokenExpireTime  *int64         `gorm:"type:bigint"`
	Status           int            `gorm:"default:1;index"` // 1: 正常, 2: 禁用
	CreatedAt        int64          `gorm:"type:bigint;autoCreateTime;index"`
	UpdatedAt        int64          `gorm:"type:bigint;autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}
