package domain

import (
	"context"
	"time"
)

// GenerationExecution is the durable, cross-pipeline envelope for work executed
// by grapery-agent. Pipeline-specific task tables remain the source of their
// business payloads; this record owns execution lifecycle and recovery data.
type GenerationExecution struct {
	ID              string           `json:"id"`
	UserID          string           `json:"userId,omitempty"`
	Kind            string           `json:"kind"`
	Status          string           `json:"status"`
	Phase           string           `json:"phase,omitempty"`
	Progress        int              `json:"progress,omitempty"`
	AgentVersion    string           `json:"agentVersion,omitempty"`
	UserIntent      string           `json:"userIntent,omitempty"`
	Input           map[string]any   `json:"input,omitempty"`
	Output          map[string]any   `json:"output,omitempty"`
	ParentRunID     string           `json:"parentRunId,omitempty"`
	BranchIndex     int              `json:"branchIndex,omitempty"`
	Strategy        string           `json:"strategy,omitempty"`
	ContentIDs      map[string]any   `json:"contentIds,omitempty"`
	ToolCalls       []map[string]any `json:"toolCalls,omitempty"`
	Error           string           `json:"error,omitempty"`
	ErrorCode       string           `json:"errorCode,omitempty"`
	TokensUsed      int              `json:"tokensUsed,omitempty"`
	ModelProvider   string           `json:"modelProvider,omitempty"`
	ModelName       string           `json:"modelName,omitempty"`
	CheckpointID    string           `json:"checkpointId,omitempty"`
	ClientRequestID string           `json:"clientRequestId,omitempty"`
	SourceTaskID    string           `json:"sourceTaskId,omitempty"`
	Sequence        int64            `json:"sequence"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
	CompletedAt     *time.Time       `json:"completedAt,omitempty"`
}

// GenerationEvent is a replayable semantic execution event. Redis Streams are
// the fast replay path; this database row is the durable fallback.
type GenerationEvent struct {
	ID        string         `json:"id"`
	RunID     string         `json:"runId"`
	Sequence  int64          `json:"sequence"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

type GenerationCheckpoint struct {
	ID        string    `json:"id"`
	State     []byte    `json:"-"`
	UpdatedAt time.Time `json:"updatedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type GenerationRuntimeRepository interface {
	SaveGenerationExecution(ctx context.Context, run *GenerationExecution, eventType string) (*GenerationEvent, error)
	GetGenerationExecution(ctx context.Context, id string) (*GenerationExecution, error)
	ListGenerationExecutions(ctx context.Context, kind string, limit int) ([]*GenerationExecution, error)
	ListGenerationEvents(ctx context.Context, runID string, afterSequence int64, limit int) ([]*GenerationEvent, error)
	ListUnpublishedGenerationEvents(ctx context.Context, limit int) ([]*GenerationEvent, error)
	MarkGenerationEventPublished(ctx context.Context, eventID string, publishedAt time.Time) error
	SaveGenerationCheckpoint(ctx context.Context, checkpoint *GenerationCheckpoint) error
	GetGenerationCheckpoint(ctx context.Context, id string) (*GenerationCheckpoint, error)
}
