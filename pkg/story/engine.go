package story

// 用来渲染故事，渲染场景、渲染图片、混合角色
var storyEngine EngineServer

func init() {
	storyEngine = NewStoryEngine()
}

func GetStoryEngine() EngineServer {
	return storyEngine
}

func NewStoryEngine() *EngineService {
	return &EngineService{
		NewStoryService(),
		NewStoryboardService(),
		NewStoryboardSenceService(),
		NewStoryroleService(),
		NewStoryHelper(),
	}
}

type EngineService struct {
	*StoryService
	*StoryboardService
	*StoryboardSenceService
	*StoryroleService
	*HelperService
}

type EngineServer interface {
	StoryServer
	StoryboardServer
	StoryboardSenceServer
	StoryroleServer
	HelperServer
}

func NewEngineService() *EngineService {
	return &EngineService{
		StoryService:           NewStoryService(),
		StoryboardService:      NewStoryboardService(),
		StoryboardSenceService: NewStoryboardSenceService(),
		StoryroleService:       NewStoryroleService(),
		HelperService:          NewStoryHelper(),
	}
}
