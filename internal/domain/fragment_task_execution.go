package domain

import (
	"context"
	"errors"
)

var ErrFragmentTaskLeaseLost = errors.New("fragment task execution lease lost")

type fragmentTaskLeaseContextKey struct{}

// FragmentTaskLease contains only immutable fencing identity, never mutable
// progress or the heartbeat's local estimate of expiry.
type FragmentTaskLease struct {
	TaskID string
	Owner  string
	Fence  int64
}

func WithFragmentTaskLease(ctx context.Context, task *FragmentGenerationTask) context.Context {
	if task == nil {
		return ctx
	}
	return context.WithValue(ctx, fragmentTaskLeaseContextKey{}, FragmentTaskLease{TaskID: task.ID, Owner: task.LeaseOwner, Fence: task.LeaseFence})
}

func FragmentTaskLeaseFromContext(ctx context.Context) (FragmentTaskLease, bool) {
	lease, ok := ctx.Value(fragmentTaskLeaseContextKey{}).(FragmentTaskLease)
	return lease, ok
}
