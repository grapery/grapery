package domain

import (
	"context"
	"errors"
	"time"
)

var ErrCharacterTaskLeaseLost = errors.New("character task execution lease lost")

// CharacterTaskExecutionRepository is deliberately separate from the broad Repository.
// Claims and all task-owned writes are fenced in the same database transaction.
type CharacterTaskExecutionRepository interface {
	ClaimCharacterTask(context.Context, string, string, time.Duration) (*CharacterGenerationTask, error)
	RenewCharacterTask(context.Context, *CharacterGenerationTask, time.Duration) error
	RecoverableCharacterTasks(context.Context, int) ([]string, error)
	QueueCharacterTaskRetry(context.Context, string, string) (bool, error)
	CreateCharacterForTask(context.Context, *CharacterGenerationTask, *Character) error
	UpdateCharacterForTask(context.Context, *CharacterGenerationTask, *Character) error
}

// CharacterVisualRepository patches only generated image fields so a slow model
// response cannot overwrite text edited by the owner while it was in flight.
type CharacterVisualRepository interface {
	UpdateCharacterVisualAssets(context.Context, *CharacterGenerationTask, *Character, map[string]any) error
}
