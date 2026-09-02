package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const (
	workflowArtifactCacheTTL      = 72 * time.Hour
	maxPublishedPromptTemplateLen = 256 * 1024
)

type WorkflowRegistryService struct {
	repo   domain.WorkflowRegistryRepository
	redis  *redis.Client
	logger *zap.Logger
}

func NewWorkflowRegistryService(repo domain.WorkflowRegistryRepository, redisClient *redis.Client, logger *zap.Logger) *WorkflowRegistryService {
	return &WorkflowRegistryService{repo: repo, redis: redisClient, logger: logger}
}

func (s *WorkflowRegistryService) PublishPromptVersion(ctx context.Context, prompt *domain.PromptTemplateVersion) (*domain.PromptTemplateVersion, error) {
	if prompt == nil || strings.TrimSpace(prompt.ID) == "" || strings.TrimSpace(prompt.Key) == "" || prompt.Version <= 0 {
		return nil, errors.New("prompt id, key and positive version are required")
	}
	prompt.Type = strings.TrimSpace(prompt.Type)
	if prompt.Type != "text" && prompt.Type != "chat" && prompt.Type != "image" {
		return nil, errors.New("prompt type must be text, chat or image")
	}
	if strings.TrimSpace(prompt.SystemTemplate) == "" && strings.TrimSpace(prompt.UserTemplate) == "" {
		return nil, errors.New("prompt must contain a system or user template")
	}
	if len(prompt.SystemTemplate)+len(prompt.UserTemplate) > maxPublishedPromptTemplateLen {
		return nil, fmt.Errorf("prompt templates exceed %d bytes", maxPublishedPromptTemplateLen)
	}
	for name, source := range map[string]string{"system": prompt.SystemTemplate, "user": prompt.UserTemplate} {
		if strings.TrimSpace(source) == "" {
			continue
		}
		if _, err := template.New(prompt.Key + "." + name).Option("missingkey=error").Parse(source); err != nil {
			return nil, fmt.Errorf("invalid %s prompt template: %w", name, err)
		}
	}
	providedChecksum := prompt.Checksum
	calculated, err := checksumValue(struct {
		Key             string         `json:"key"`
		Version         int            `json:"version"`
		Type            string         `json:"type"`
		SystemTemplate  string         `json:"systemTemplate,omitempty"`
		UserTemplate    string         `json:"userTemplate,omitempty"`
		VariablesSchema map[string]any `json:"variablesSchema,omitempty"`
		OutputSchema    map[string]any `json:"outputSchema,omitempty"`
		ModelConfig     map[string]any `json:"modelConfig,omitempty"`
	}{prompt.Key, prompt.Version, prompt.Type, prompt.SystemTemplate, prompt.UserTemplate, prompt.VariablesSchema, prompt.OutputSchema, prompt.ModelConfig})
	if err != nil {
		return nil, err
	}
	if providedChecksum != "" && providedChecksum != calculated {
		return nil, errors.New("prompt checksum mismatch")
	}
	prompt.Checksum = calculated
	if prompt.CreatedAt.IsZero() {
		prompt.CreatedAt = time.Now().UTC()
	}
	if err := s.repo.SavePromptVersion(ctx, prompt); err != nil {
		return nil, err
	}
	stored, err := s.repo.GetPromptVersion(ctx, prompt.ID)
	if err != nil {
		return nil, err
	}
	s.cache(ctx, "workflow:prompt:"+stored.ID, stored)
	return stored, nil
}

func (s *WorkflowRegistryService) GetPromptVersion(ctx context.Context, id string) (*domain.PromptTemplateVersion, error) {
	var cached domain.PromptTemplateVersion
	if s.readCache(ctx, "workflow:prompt:"+strings.TrimSpace(id), &cached) {
		return &cached, nil
	}
	prompt, err := s.repo.GetPromptVersion(ctx, id)
	if err == nil {
		s.cache(ctx, "workflow:prompt:"+prompt.ID, prompt)
	}
	return prompt, err
}

func (s *WorkflowRegistryService) PublishRelease(ctx context.Context, release *domain.WorkflowRelease) (*domain.WorkflowRelease, error) {
	if err := domain.ValidateWorkflowRelease(release); err != nil {
		return nil, err
	}
	nodeIDs := make(map[string]bool, len(release.Definition.Nodes))
	for _, node := range release.Definition.Nodes {
		nodeIDs[node.ID] = true
	}
	for bindingKey, promptID := range release.PromptBundle {
		nodeID, slot, valid := parseWorkflowPromptBindingKey(bindingKey)
		if !valid || strings.TrimSpace(promptID) == "" {
			return nil, errors.New("prompt bundle contains an empty node or prompt id")
		}
		if !nodeIDs[nodeID] {
			return nil, fmt.Errorf("prompt bundle references unknown workflow node %s", nodeID)
		}
		if slot != "" && !validWorkflowPromptSlot(slot) {
			return nil, fmt.Errorf("prompt bundle key %s has an invalid slot", bindingKey)
		}
		if _, err := s.repo.GetPromptVersion(ctx, promptID); err != nil {
			return nil, fmt.Errorf("prompt bundle node %s references unavailable prompt %s: %w", nodeID, promptID, err)
		}
	}
	providedChecksum := release.Checksum
	calculated, err := checksumValue(struct {
		Key          string                    `json:"key"`
		Version      int                       `json:"version"`
		Manifest     map[string]any            `json:"manifest,omitempty"`
		Definition   domain.WorkflowDefinition `json:"definition"`
		PromptBundle map[string]string         `json:"promptBundle,omitempty"`
		Policies     domain.WorkflowPolicies   `json:"policies,omitempty"`
	}{release.Key, release.Version, release.Manifest, release.Definition, release.PromptBundle, release.Policies})
	if err != nil {
		return nil, err
	}
	if providedChecksum != "" && providedChecksum != calculated {
		return nil, errors.New("workflow release checksum mismatch")
	}
	release.Checksum = calculated
	now := time.Now().UTC()
	if release.PublishedAt.IsZero() {
		release.PublishedAt = now
	}
	if release.CreatedAt.IsZero() {
		release.CreatedAt = now
	}
	if err := s.repo.SaveWorkflowRelease(ctx, release); err != nil {
		return nil, err
	}
	stored, err := s.repo.GetWorkflowRelease(ctx, release.ID)
	if err != nil {
		return nil, err
	}
	s.cache(ctx, "workflow:release:"+stored.ID, stored)
	return stored, nil
}

func parseWorkflowPromptBindingKey(key string) (string, string, bool) {
	key = strings.TrimSpace(key)
	if key == "" || strings.Count(key, ":") > 1 {
		return "", "", false
	}
	nodeID, slot, hasSlot := strings.Cut(key, ":")
	if strings.TrimSpace(nodeID) == "" || (hasSlot && strings.TrimSpace(slot) == "") {
		return "", "", false
	}
	return strings.TrimSpace(nodeID), strings.TrimSpace(slot), true
}

func validWorkflowPromptSlot(slot string) bool {
	for _, char := range slot {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func (s *WorkflowRegistryService) GetRelease(ctx context.Context, id string) (*domain.WorkflowRelease, error) {
	var cached domain.WorkflowRelease
	if s.readCache(ctx, "workflow:release:"+strings.TrimSpace(id), &cached) {
		return &cached, nil
	}
	release, err := s.repo.GetWorkflowRelease(ctx, id)
	if err == nil {
		s.cache(ctx, "workflow:release:"+release.ID, release)
	}
	return release, err
}

func (s *WorkflowRegistryService) SaveBinding(ctx context.Context, binding *domain.WorkflowBinding) (*domain.WorkflowBinding, error) {
	if binding == nil || strings.TrimSpace(binding.Surface) == "" || strings.TrimSpace(binding.Action) == "" || strings.TrimSpace(binding.ReleaseID) == "" {
		return nil, errors.New("workflow binding surface, action and release id are required")
	}
	release, err := s.GetRelease(ctx, binding.ReleaseID)
	if err != nil {
		return nil, fmt.Errorf("resolve workflow release: %w", err)
	}
	if binding.WorkflowKey == "" {
		binding.WorkflowKey = release.Key
	}
	if binding.WorkflowKey != release.Key {
		return nil, errors.New("workflow binding key does not match release")
	}
	if err := ValidateWorkflowBindingConditions(binding.Conditions); err != nil {
		return nil, fmt.Errorf("invalid workflow binding conditions: %w", err)
	}
	if binding.ID == "" {
		binding.ID = "wfb_" + uuid.NewString()
	}
	now := time.Now().UTC()
	if binding.CreatedAt.IsZero() {
		binding.CreatedAt = now
	}
	binding.UpdatedAt = now
	if err := s.repo.SaveWorkflowBinding(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *WorkflowRegistryService) PauseReleaseBindings(ctx context.Context, releaseID string) (int64, error) {
	releaseID = strings.TrimSpace(releaseID)
	if releaseID == "" {
		return 0, errors.New("workflow release id is required")
	}
	if _, err := s.GetRelease(ctx, releaseID); err != nil {
		return 0, fmt.Errorf("resolve workflow release: %w", err)
	}
	return s.repo.DisableWorkflowBindingsByRelease(ctx, releaseID)
}

func (s *WorkflowRegistryService) RebindWorkflowBindings(ctx context.Context, surface, action, workflowKey, releaseID string) (int64, error) {
	surface, action, workflowKey, releaseID = strings.TrimSpace(surface), strings.TrimSpace(action), strings.TrimSpace(workflowKey), strings.TrimSpace(releaseID)
	if surface == "" || action == "" || workflowKey == "" || releaseID == "" {
		return 0, errors.New("surface, action, workflow key and release id are required")
	}
	release, err := s.GetRelease(ctx, releaseID)
	if err != nil {
		return 0, fmt.Errorf("resolve workflow release: %w", err)
	}
	if release.Key != workflowKey {
		return 0, errors.New("workflow binding key does not match release")
	}
	return s.repo.RebindWorkflowBindings(ctx, surface, action, workflowKey, releaseID)
}

func (s *WorkflowRegistryService) Catalog(ctx context.Context, surface, action, tenantID string) ([]*domain.WorkflowCatalogEntry, error) {
	return s.repo.ListWorkflowCatalog(ctx, surface, action, tenantID)
}

func (s *WorkflowRegistryService) ReleaseStats(ctx context.Context, days int) ([]domain.WorkflowReleaseStats, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		return nil, errors.New("workflow stats range cannot exceed 365 days")
	}
	return s.repo.ListWorkflowReleaseStats(ctx, time.Now().UTC().Add(-time.Duration(days)*24*time.Hour))
}

func (s *WorkflowRegistryService) Resolve(ctx context.Context, surface, action, tenantID string) (*domain.WorkflowCatalogEntry, error) {
	resolution, err := s.ResolveForInput(ctx, surface, action, tenantID, nil)
	if err != nil {
		return nil, err
	}
	return &resolution.Entry, nil
}

// ResolveForInput performs deterministic, explainable content-aware routing.
// Catalog ordering remains the priority order; conditional bindings are tried
// first and an unconditional binding is treated as the safe fallback.
func (s *WorkflowRegistryService) ResolveForInput(ctx context.Context, surface, action, tenantID string, input map[string]any) (*domain.WorkflowResolution, error) {
	entries, err := s.Catalog(ctx, surface, action, tenantID)
	if err != nil {
		return nil, err
	}
	profile := BuildWorkflowContentProfile(surface, action, input)
	return ResolveWorkflowEntries(entries, profile)
}

// ResolvePinnedPromptSnapshots verifies that a client-discovered release is
// still the active product binding, then resolves all immutable prompt content
// server-side. Callers must never accept prompt bodies supplied by a client.
func (s *WorkflowRegistryService) ResolvePinnedPromptSnapshots(ctx context.Context, surface, action, tenantID, releaseID string) (*domain.WorkflowCatalogEntry, map[string]domain.PromptTemplateVersion, error) {
	return s.ResolvePinnedPromptSnapshotsForInput(ctx, surface, action, tenantID, releaseID, nil)
}

func (s *WorkflowRegistryService) ResolvePinnedPromptSnapshotsForInput(ctx context.Context, surface, action, tenantID, releaseID string, input map[string]any) (*domain.WorkflowCatalogEntry, map[string]domain.PromptTemplateVersion, error) {
	resolution, err := s.ResolveForInput(ctx, surface, action, tenantID, input)
	if err != nil {
		return nil, nil, err
	}
	entry := &resolution.Entry
	if entry.Release.ID != strings.TrimSpace(releaseID) {
		return nil, nil, errors.New("workflow release changed; refresh and retry")
	}
	snapshots := make(map[string]domain.PromptTemplateVersion, len(entry.Release.PromptBundle))
	for nodeID, promptID := range entry.Release.PromptBundle {
		prompt, err := s.GetPromptVersion(ctx, promptID)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve prompt for node %s: %w", nodeID, err)
		}
		if prompt == nil || prompt.ID != promptID || strings.TrimSpace(prompt.Checksum) == "" {
			return nil, nil, fmt.Errorf("resolve prompt for node %s: invalid immutable prompt version", nodeID)
		}
		snapshots[nodeID] = *prompt
	}
	return entry, snapshots, nil
}

func checksumValue(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal checksum payload: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (s *WorkflowRegistryService) cache(ctx context.Context, key string, value any) {
	if s.redis == nil {
		return
	}
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	if err := s.redis.Set(ctx, key, b, workflowArtifactCacheTTL).Err(); err != nil && s.logger != nil {
		s.logger.Warn("failed to cache workflow artifact", zap.String("key", key), zap.Error(err))
	}
}

func (s *WorkflowRegistryService) readCache(ctx context.Context, key string, target any) bool {
	if s.redis == nil {
		return false
	}
	b, err := s.redis.Get(ctx, key).Bytes()
	if err != nil || json.Unmarshal(b, target) != nil {
		return false
	}
	return true
}
