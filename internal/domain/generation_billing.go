package domain

import "context"

// GenerationChargeRepository keeps each completed generation debit atomic and replay-safe.
// operationID identifies one provider result; relatedID identifies its parent creation task.
type GenerationChargeRepository interface {
	ApplyGenerationCharge(context.Context, string, string, string, string, int) (*TokenTransaction, error)
	GenerationChargedTokens(context.Context, string, string) (int, error)
}

type GenerationBillingError struct{ Err error }

func (e *GenerationBillingError) Error() string {
	return "generation billing could not be settled: " + e.Err.Error()
}
func (e *GenerationBillingError) Unwrap() error { return e.Err }

type sponsoredGenerationKey struct{}

// WithSponsoredGeneration is reserved for server-initiated corrective renders.
func WithSponsoredGeneration(ctx context.Context) context.Context {
	return context.WithValue(ctx, sponsoredGenerationKey{}, true)
}
func IsSponsoredGeneration(ctx context.Context) bool {
	value, _ := ctx.Value(sponsoredGenerationKey{}).(bool)
	return value
}

// Only known failed calls may be retried. An in-flight call with unknown provider
// outcome must not be blindly repeated after a process interruption.
type GenerationImageExecutionRepository interface {
	ClaimFailedGenerationImage(context.Context, string) (bool, error)
}
