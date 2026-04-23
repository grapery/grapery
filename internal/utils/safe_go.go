package utils

import (
	"context"
	"fmt"
	"runtime/debug"

	"go.uber.org/zap"
)

// SafeGo launches fn in a new goroutine with panic recovery.
// If fn panics, the panic value and stack trace are logged via the provided logger.
func SafeGo(fn func(), logger *zap.Logger) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("goroutine panic recovered",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
			}
		}()
		fn()
	}()
}

// SafeGoWithContext launches fn in a new goroutine with panic recovery,
// passing context.Background() to avoid using a request-scoped context
// that may be cancelled after the HTTP response is sent.
func SafeGoWithContext(fn func(ctx context.Context), logger *zap.Logger) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("goroutine panic recovered",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
			}
		}()
		fn(context.Background())
	}()
}

// RecoverPanic is a helper for adding panic recovery to existing goroutines
// that have their own lifecycle (e.g., loops with WaitGroup).
// Call `defer RecoverPanic(logger)` at the start of the goroutine function.
func RecoverPanic(logger *zap.Logger) {
	if r := recover(); r != nil {
		logger.Error("goroutine panic recovered",
			zap.Any("panic", r),
			zap.String("stack", string(debug.Stack())),
		)
	}
}

// FormatPanicValue formats a recovered panic value for logging.
func FormatPanicValue(r interface{}) string {
	return fmt.Sprintf("%v\n%s", r, debug.Stack())
}
