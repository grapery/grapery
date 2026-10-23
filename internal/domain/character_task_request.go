package domain

import (
	"context"
	"errors"
)

var ErrCharacterTaskRequestConflict = errors.New("character creation request key was already used for a different intent")

// RequestKey is a server-derived per-user identity; the immutable intent hash
// protects reuse across retries, multiple API replicas and agent recovery.
type CharacterTaskRequestRepository interface {
	CreateCharacterTaskIdempotent(context.Context, *CharacterGenerationTask) (*CharacterGenerationTask, bool, error)
}
