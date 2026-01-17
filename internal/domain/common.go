package domain

import "errors"

// Common errors
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrAlreadyLiked  = errors.New("already liked")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidInput  = errors.New("invalid input")
)

// BaseModel 基础模型（包含通用字段）
type BaseModel struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// SoftDeleteModel 软删除模型
type SoftDeleteModel struct {
	BaseModel
	DeletedAt *int64 `json:"deletedAt,omitempty"`
}
