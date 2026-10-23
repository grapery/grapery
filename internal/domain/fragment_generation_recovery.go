package domain

type FragmentGenerationRecovery struct {
	CanRetry               bool   `json:"canRetry"`
	RequiresReconciliation bool   `json:"requiresReconciliation"`
	Reason                 string `json:"reason"`
}
