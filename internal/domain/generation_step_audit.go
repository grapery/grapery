package domain

// GenerationStepAuditRecord 是 agent 侧 GenerationStepAudit 的持久化形态（grapery 权威存储）。
type GenerationStepAuditRecord struct {
	ID           string         `json:"id"`
	Sequence     int            `json:"sequence"`
	RunID        string         `json:"runId"`
	TaskID       string         `json:"taskId,omitempty"`
	BusinessType string         `json:"businessType,omitempty"`
	BusinessID   string         `json:"businessId,omitempty"`
	AgentVersion string         `json:"agentVersion,omitempty"`
	StepName     string         `json:"stepName"`
	Attempt      int            `json:"attempt"`
	Status       string         `json:"status"`
	Provider     string         `json:"provider,omitempty"`
	Model        string         `json:"model,omitempty"`
	Prompt       string         `json:"prompt,omitempty"`
	InputRefs    []string       `json:"inputRefs,omitempty"`
	RawOutput    string         `json:"rawOutput,omitempty"`
	ParsedOutput map[string]any `json:"parsedOutput,omitempty"`
	ErrorCode    string         `json:"errorCode,omitempty"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	InputTokens  int            `json:"inputTokens,omitempty"`
	OutputTokens int            `json:"outputTokens,omitempty"`
	TotalTokens  int            `json:"totalTokens,omitempty"`
	DurationMs   int64          `json:"durationMs,omitempty"`
	StartedAt    int64          `json:"startedAt,omitempty"`
	EndedAt      int64          `json:"endedAt,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    int64          `json:"createdAt"`
	UserID       string         `json:"userId,omitempty"`
}
