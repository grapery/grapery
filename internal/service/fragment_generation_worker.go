package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const fragmentWorkerLeaseTTL = 2 * time.Minute

func (s *FragmentGenerationService) initFragmentWorker() {
	s.workerInit.Do(func() { s.workerSlots = make(chan struct{}, 4) })
}

// StartRecovery starts the bounded durable queue scanner. Expired leases, not
// process-local goroutines, determine which tasks can be taken over.
func (s *FragmentGenerationService) StartRecovery(ctx context.Context) {
	s.initFragmentWorker()
	s.workerMu.Lock()
	s.workerContext = ctx
	s.workerMu.Unlock()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			ids, err := s.fragmentGenRepo.ListRecoverable(ctx, 32)
			if err != nil {
				s.logger.Warn("fragment recovery scan failed", zap.Error(err))
			} else {
				for _, id := range ids {
					s.scheduleFragmentTask(id)
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *FragmentGenerationService) scheduleFragmentTask(id string) {
	s.initFragmentWorker()
	s.workerMu.RLock()
	ctx := s.workerContext
	s.workerMu.RUnlock()
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return
	}
	select {
	case s.workerSlots <- struct{}{}:
	default:
		return
	} // scanner picks up backlog
	go func() { defer func() { <-s.workerSlots }(); s.runFragmentTask(ctx, id, s.processFragmentGeneration) }()
}

func (s *FragmentGenerationService) runFragmentTask(parent context.Context, id string, process func(context.Context, string)) {
	task, err := s.fragmentGenRepo.Claim(parent, id, uuid.NewString(), fragmentWorkerLeaseTTL)
	if err != nil {
		if errors.Is(err, domain.ErrFragmentTaskLeaseLost) || parent.Err() != nil {
			return
		}
		s.logger.Warn("fragment task claim failed", zap.String("task_id", id), zap.Error(err))
		return
	}
	if task == nil {
		return
	}
	ctx, cancel := context.WithCancel(domain.WithFragmentTaskLease(parent, task))
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.fragmentGenRepo.Renew(ctx, task, fragmentWorkerLeaseTTL); err != nil {
					s.logger.Warn("fragment task lease renewal failed", zap.String("task_id", id), zap.Error(err))
					cancel()
					return
				}
			}
		}
	}()
	defer func() {
		if recovered := recover(); recovered != nil {
			s.logger.Error("fragment worker panic; task remains recoverable", zap.String("task_id", id), zap.Any("panic", recovered))
		}
		cancel()
		<-done
	}()
	process(ctx, id)
}
