package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	generationSnapshotTTL    = 72 * time.Hour
	generationEventTTL       = 72 * time.Hour
	generationCheckpointTTL  = 7 * 24 * time.Hour
	generationIdempotencyTTL = 7 * 24 * time.Hour
	generationStreamMaxLen   = 2000
	generationLeaseDefault   = 5 * time.Minute
	generationLeaseMax       = 30 * time.Minute
)

type GenerationRuntimeService struct {
	repo   domain.GenerationRuntimeRepository
	redis  *redis.Client
	logger *zap.Logger
}

type GenerationLease struct {
	RunID string `json:"runId"`
	Owner string `json:"owner"`
	Token int64  `json:"token"`
	Value string `json:"value"`
}

func NewGenerationRuntimeService(repo domain.GenerationRuntimeRepository, redisClient *redis.Client, logger *zap.Logger) *GenerationRuntimeService {
	return &GenerationRuntimeService{repo: repo, redis: redisClient, logger: logger}
}

func (s *GenerationRuntimeService) SaveExecution(ctx context.Context, run *domain.GenerationExecution, eventType string) (*domain.GenerationExecution, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("generation runtime service unavailable")
	}
	if err := validateGenerationExecution(run); err != nil {
		return nil, err
	}
	if key := generationIdempotencyKey(run); s.redis != nil && key != "" {
		if canonicalID, err := s.redis.Get(ctx, key).Result(); err == nil && canonicalID != "" && canonicalID != run.ID {
			canonical, getErr := s.GetExecution(ctx, canonicalID)
			if getErr == nil {
				return canonical, nil
			}
		}
	}
	event, err := s.repo.SaveGenerationExecution(ctx, run, eventType)
	if err != nil {
		return nil, err
	}
	if key := generationIdempotencyKey(run); s.redis != nil && key != "" {
		_ = s.redis.Set(ctx, key, run.ID, generationIdempotencyTTL).Err()
	}
	if err := s.publish(ctx, run, event); err != nil {
		if s.logger != nil {
			s.logger.Warn("generation event persisted but redis publish failed", zap.String("runId", run.ID), zap.Int64("sequence", run.Sequence), zap.Error(err))
		}
	} else if event != nil {
		if err := s.repo.MarkGenerationEventPublished(ctx, event.ID, time.Now().UTC()); err != nil && s.logger != nil {
			s.logger.Warn("failed to mark generation event published", zap.String("eventId", event.ID), zap.Error(err))
		}
		s.replayPendingEvents(ctx, 100)
	}
	return run, nil
}

func (s *GenerationRuntimeService) replayPendingEvents(ctx context.Context, limit int) {
	if s.redis == nil {
		return
	}
	events, err := s.repo.ListUnpublishedGenerationEvents(ctx, limit)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("failed to load generation outbox", zap.Error(err))
		}
		return
	}
	for _, event := range events {
		raw, err := json.Marshal(event.Payload)
		if err != nil {
			continue
		}
		pipe := s.redis.TxPipeline()
		pipe.XAdd(ctx, &redis.XAddArgs{Stream: generationEventKey(event.RunID), MaxLen: generationStreamMaxLen, Approx: true, Values: map[string]any{
			"eventId": event.ID, "sequence": event.Sequence, "type": event.Type, "payload": string(raw), "createdAt": event.CreatedAt.UnixMilli(),
		}})
		pipe.Expire(ctx, generationEventKey(event.RunID), generationEventTTL)
		if _, err := pipe.Exec(ctx); err != nil {
			return
		}
		_ = s.repo.MarkGenerationEventPublished(ctx, event.ID, time.Now().UTC())
	}
}

func (s *GenerationRuntimeService) GetExecution(ctx context.Context, id string) (*domain.GenerationExecution, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("generation id is required")
	}
	if s.redis != nil {
		var run domain.GenerationExecution
		if raw, err := s.redis.Get(ctx, generationSnapshotKey(id)).Bytes(); err == nil && json.Unmarshal(raw, &run) == nil {
			return &run, nil
		}
	}
	run, err := s.repo.GetGenerationExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	s.cacheSnapshot(ctx, run)
	return run, nil
}

func (s *GenerationRuntimeService) ListExecutions(ctx context.Context, kind string, limit int) ([]*domain.GenerationExecution, error) {
	return s.repo.ListGenerationExecutions(ctx, kind, limit)
}

func (s *GenerationRuntimeService) FindLatestExecution(ctx context.Context, userID, kind, contentID string) (*domain.GenerationExecution, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("generation runtime service unavailable")
	}
	return s.repo.FindLatestGenerationExecution(ctx, strings.TrimSpace(userID), strings.TrimSpace(kind), strings.TrimSpace(contentID))
}

func (s *GenerationRuntimeService) ListEvents(ctx context.Context, runID string, afterSequence int64, limit int) ([]*domain.GenerationEvent, error) {
	return s.repo.ListGenerationEvents(ctx, runID, afterSequence, limit)
}

func (s *GenerationRuntimeService) SaveCheckpoint(ctx context.Context, id string, state []byte) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("generation runtime service unavailable")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("checkpoint id is required")
	}
	cp := &domain.GenerationCheckpoint{ID: id, State: append([]byte(nil), state...), ExpiresAt: time.Now().UTC().Add(generationCheckpointTTL)}
	if err := s.repo.SaveGenerationCheckpoint(ctx, cp); err != nil {
		return err
	}
	if s.redis != nil {
		if err := s.redis.Set(ctx, generationCheckpointKey(id), state, generationCheckpointTTL).Err(); err != nil && s.logger != nil {
			s.logger.Warn("checkpoint persisted but redis cache failed", zap.String("checkpointId", id), zap.Error(err))
		}
	}
	return nil
}

func (s *GenerationRuntimeService) GetCheckpoint(ctx context.Context, id string) ([]byte, bool, error) {
	if s == nil || s.repo == nil {
		return nil, false, fmt.Errorf("generation runtime service unavailable")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false, fmt.Errorf("checkpoint id is required")
	}
	if s.redis != nil {
		if state, err := s.redis.Get(ctx, generationCheckpointKey(id)).Bytes(); err == nil {
			return state, true, nil
		} else if err != nil && !errors.Is(err, redis.Nil) && s.logger != nil {
			s.logger.Warn("checkpoint redis read failed; falling back to database", zap.String("checkpointId", id), zap.Error(err))
		}
	}
	cp, err := s.repo.GetGenerationCheckpoint(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if s.redis != nil {
		_ = s.redis.Set(ctx, generationCheckpointKey(id), cp.State, time.Until(cp.ExpiresAt)).Err()
	}
	return append([]byte(nil), cp.State...), true, nil
}

func (s *GenerationRuntimeService) AcquireLease(ctx context.Context, runID, owner string, ttl time.Duration) (*GenerationLease, bool, error) {
	if s.redis == nil {
		return nil, false, fmt.Errorf("redis is required for generation lease")
	}
	runID, owner = strings.TrimSpace(runID), strings.TrimSpace(owner)
	if runID == "" || owner == "" {
		return nil, false, fmt.Errorf("runId and owner are required")
	}
	if ttl < time.Minute || ttl > generationLeaseMax {
		ttl = generationLeaseDefault
	}
	const script = `
local token = redis.call('INCR', KEYS[2])
local value = ARGV[1] .. ':' .. tostring(token)
local ok = redis.call('SET', KEYS[1], value, 'NX', 'PX', ARGV[2])
if ok then return {1, tostring(token), value} end
return {0, '0', redis.call('GET', KEYS[1]) or ''}`
	result, err := s.redis.Eval(ctx, script, []string{generationLeaseKey(runID), generationFenceKey(runID)}, owner, ttl.Milliseconds()).Result()
	if err != nil {
		return nil, false, err
	}
	parts, ok := result.([]interface{})
	if !ok || len(parts) != 3 {
		return nil, false, fmt.Errorf("unexpected generation lease response")
	}
	acquired := fmt.Sprint(parts[0]) == "1"
	if !acquired {
		return nil, false, nil
	}
	token, _ := strconv.ParseInt(fmt.Sprint(parts[1]), 10, 64)
	return &GenerationLease{RunID: runID, Owner: owner, Token: token, Value: fmt.Sprint(parts[2])}, true, nil
}

func (s *GenerationRuntimeService) RenewLease(ctx context.Context, runID, value string, ttl time.Duration) (bool, error) {
	if s.redis == nil {
		return false, fmt.Errorf("redis is required for generation lease")
	}
	if ttl < time.Minute || ttl > generationLeaseMax {
		ttl = generationLeaseDefault
	}
	const script = `if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('PEXPIRE', KEYS[1], ARGV[2]) end return 0`
	result, err := s.redis.Eval(ctx, script, []string{generationLeaseKey(runID)}, value, ttl.Milliseconds()).Int()
	return result == 1, err
}

func (s *GenerationRuntimeService) VerifyLease(ctx context.Context, runID, value string) (bool, error) {
	if s.redis == nil {
		return false, fmt.Errorf("redis is required for generation lease")
	}
	current, err := s.redis.Get(ctx, generationLeaseKey(strings.TrimSpace(runID))).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return current == strings.TrimSpace(value) && current != "", nil
}

func (s *GenerationRuntimeService) ReleaseLease(ctx context.Context, runID, value string) error {
	if s.redis == nil {
		return nil
	}
	const script = `if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) end return 0`
	return s.redis.Eval(ctx, script, []string{generationLeaseKey(runID)}, value).Err()
}

func (s *GenerationRuntimeService) publish(ctx context.Context, run *domain.GenerationExecution, event *domain.GenerationEvent) error {
	if s.redis == nil || event == nil {
		return nil
	}
	raw, err := json.Marshal(run)
	if err != nil {
		return err
	}
	pipe := s.redis.TxPipeline()
	pipe.Set(ctx, generationSnapshotKey(run.ID), raw, generationSnapshotTTL)
	pipe.XAdd(ctx, &redis.XAddArgs{Stream: generationEventKey(run.ID), MaxLen: generationStreamMaxLen, Approx: true, Values: map[string]any{
		"eventId": event.ID, "sequence": event.Sequence, "type": event.Type, "payload": string(raw), "createdAt": event.CreatedAt.UnixMilli(),
	}})
	pipe.Expire(ctx, generationEventKey(run.ID), generationEventTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *GenerationRuntimeService) cacheSnapshot(ctx context.Context, run *domain.GenerationExecution) {
	if s.redis == nil || run == nil {
		return
	}
	if raw, err := json.Marshal(run); err == nil {
		_ = s.redis.Set(ctx, generationSnapshotKey(run.ID), raw, generationSnapshotTTL).Err()
	}
}

func validateGenerationExecution(run *domain.GenerationExecution) error {
	if run == nil {
		return fmt.Errorf("generation execution is required")
	}
	if strings.TrimSpace(run.ID) == "" {
		return fmt.Errorf("generation id is required")
	}
	if strings.TrimSpace(run.Kind) == "" {
		return fmt.Errorf("generation kind is required")
	}
	if strings.TrimSpace(run.Status) == "" {
		return fmt.Errorf("generation status is required")
	}
	if run.Progress < 0 || run.Progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100")
	}
	return nil
}

func generationSnapshotKey(id string) string   { return "generation:snapshot:" + id }
func generationEventKey(id string) string      { return "generation:events:" + id }
func generationCheckpointKey(id string) string { return "generation:checkpoint:" + id }
func generationLeaseKey(id string) string      { return "generation:lease:" + id }
func generationFenceKey(id string) string      { return "generation:fence:" + id }
func generationIdempotencyKey(run *domain.GenerationExecution) string {
	if run == nil || strings.TrimSpace(run.UserID) == "" || strings.TrimSpace(run.Kind) == "" || strings.TrimSpace(run.ClientRequestID) == "" {
		return ""
	}
	return "generation:idempotency:" + run.UserID + ":" + run.Kind + ":" + run.ClientRequestID
}
