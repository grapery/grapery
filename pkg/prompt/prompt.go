package prompt

import "context"

// prompt engine

type PromptInfo struct {
	Prompt    string
	Author    string
	Category  string
	Tag       string
	Negative  string
	Positive  string
	CreatedAt int64
	UpdatedAt int64
}

type GetPromptByIdParams struct {
	Id string
}

type GetPromptByIdResult struct {
	Info *PromptInfo
	Err  error
	Code int
}

type GetPromptByTagParams struct {
	Tag string
}

type GetPromptByTagResult struct {
	List []*PromptInfo
	Err  error
	Code int
}

type GetPromptsByAuthorParams struct {
	Author string
}

type GetPromptsByAuthorResult struct {
	List []*PromptInfo
	Err  error
	Code int
}
type GetPromptsByCategoryParams struct {
	Category string
}

type GetPromptsByCategoryResult struct {
	List []*PromptInfo
	Err  error
	Code int
}

type CreatePromptParams struct {
	Prompt   string
	Author   string
	Category string
	Tag      string
	Negative string
	Positive string
}
type CreatePromptResult struct {
	Info *PromptInfo
	Err  error
	Code int
}

type UpdatePromptParams struct {
	Id       string
	Prompt   string
	Author   string
	Category string
	Tag      string
	Negative string
	Positive string
}
type UpdatePromptResult struct {
	Info *PromptInfo
	Err  error
	Code int
}

type DeletePromptParams struct {
	Id       string
	Author   string
	Category string
}
type DeletePromptResult struct {
	Err  error
	Code int
}

type SearchPromptParams struct {
	Prompt   string
	Author   string
	Category string
	Tag      string
	Negative string
	Positive string

	Page int
	Size int
}
type SearchPromptResult struct {
	List []*PromptInfo
	Page int
	Size int
	Err  error
	Code int
}

type PromptServer interface {
	GetPromptById(ctx context.Context, req *GetPromptByIdParams) (resp *GetPromptByIdResult, err error)
	GetPromptByTag(ctx context.Context, req *GetPromptByTagParams) (resp *GetPromptByTagResult, err error)
	GetPromptsByAuthor(ctx context.Context, req *GetPromptsByAuthorParams) (resp *GetPromptsByAuthorResult, err error)
	GetPromptsByCategory(ctx context.Context, req *GetPromptsByCategoryParams) (resp *GetPromptsByCategoryResult, err error)
	CreatePrompt(ctx context.Context, req *CreatePromptParams) (resp *CreatePromptResult, err error)
	UpdatePrompt(ctx context.Context, req *UpdatePromptParams) (resp *UpdatePromptResult, err error)
	DeletePrompt(ctx context.Context, req *DeletePromptParams) (resp *DeletePromptResult, err error)
	SearchPrompt(ctx context.Context, req *SearchPromptParams) (resp *SearchPromptResult, err error)
}

func NewPromptServer() PromptServer {
	return &PromptService{}
}

var promptServer PromptServer

func init() {
	promptServer = NewPromptServer()
}

func GetNewPromptServer() PromptServer {
	return promptServer
}

func NewPromptService() *PromptService {
	return &PromptService{}
}

type CloudClient struct {
	Name      string
	Address   string
	AppId     string
	AppSecret string
	Limit     int
	NumReq    int
}

type PromptService struct {
	RemoteHosts map[string]*CloudClient
}

func (p *PromptService) GetPromptById(ctx context.Context, req *GetPromptByIdParams) (resp *GetPromptByIdResult, err error) {
	return &GetPromptByIdResult{}, nil
}
func (p *PromptService) GetPromptByTag(ctx context.Context, req *GetPromptByTagParams) (resp *GetPromptByTagResult, err error) {
	return &GetPromptByTagResult{}, nil
}
func (p *PromptService) GetPromptsByAuthor(ctx context.Context, req *GetPromptsByAuthorParams) (resp *GetPromptsByAuthorResult, err error) {
	return &GetPromptsByAuthorResult{}, nil
}
func (p *PromptService) GetPromptsByCategory(ctx context.Context, req *GetPromptsByCategoryParams) (resp *GetPromptsByCategoryResult, err error) {
	return &GetPromptsByCategoryResult{}, nil
}
func (p *PromptService) CreatePrompt(ctx context.Context, req *CreatePromptParams) (resp *CreatePromptResult, err error) {
	return &CreatePromptResult{}, nil
}
func (p *PromptService) UpdatePrompt(ctx context.Context, req *UpdatePromptParams) (resp *UpdatePromptResult, err error) {
	return &UpdatePromptResult{}, nil
}
func (p *PromptService) DeletePrompt(ctx context.Context, req *DeletePromptParams) (resp *DeletePromptResult, err error) {
	return &DeletePromptResult{}, nil
}
func (p *PromptService) SearchPrompt(ctx context.Context, req *SearchPromptParams) (resp *SearchPromptResult, err error) {
	return &SearchPromptResult{}, nil
}
