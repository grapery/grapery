package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

type workflowRegistryMemoryRepo struct {
	prompts  map[string]*domain.PromptTemplateVersion
	releases map[string]*domain.WorkflowRelease
	bindings map[string]*domain.WorkflowBinding
}

func newWorkflowRegistryMemoryRepo() *workflowRegistryMemoryRepo {
	return &workflowRegistryMemoryRepo{
		prompts: map[string]*domain.PromptTemplateVersion{}, releases: map[string]*domain.WorkflowRelease{}, bindings: map[string]*domain.WorkflowBinding{},
	}
}

func (r *workflowRegistryMemoryRepo) SavePromptVersion(_ context.Context, prompt *domain.PromptTemplateVersion) error {
	if existing := r.prompts[prompt.ID]; existing != nil && existing.Checksum != prompt.Checksum {
		return domain.ErrPromptVersionImmutable
	}
	copy := *prompt
	r.prompts[prompt.ID] = &copy
	return nil
}

func (r *workflowRegistryMemoryRepo) GetPromptVersion(_ context.Context, id string) (*domain.PromptTemplateVersion, error) {
	prompt := r.prompts[id]
	if prompt == nil {
		return nil, errors.New("not found")
	}
	copy := *prompt
	return &copy, nil
}

func (r *workflowRegistryMemoryRepo) SaveWorkflowRelease(_ context.Context, release *domain.WorkflowRelease) error {
	if existing := r.releases[release.ID]; existing != nil && existing.Checksum != release.Checksum {
		return domain.ErrWorkflowReleaseImmutable
	}
	copy := *release
	r.releases[release.ID] = &copy
	return nil
}

func (r *workflowRegistryMemoryRepo) GetWorkflowRelease(_ context.Context, id string) (*domain.WorkflowRelease, error) {
	release := r.releases[id]
	if release == nil {
		return nil, errors.New("not found")
	}
	copy := *release
	return &copy, nil
}

func (r *workflowRegistryMemoryRepo) SaveWorkflowBinding(_ context.Context, binding *domain.WorkflowBinding) error {
	copy := *binding
	r.bindings[binding.ID] = &copy
	return nil
}

func (r *workflowRegistryMemoryRepo) ListWorkflowCatalog(_ context.Context, surface, action, tenantID string) ([]*domain.WorkflowCatalogEntry, error) {
	var out []*domain.WorkflowCatalogEntry
	for _, binding := range r.bindings {
		if binding.Enabled && binding.Surface == surface && binding.Action == action && (binding.TenantID == "" || binding.TenantID == tenantID) {
			out = append(out, &domain.WorkflowCatalogEntry{Binding: *binding, Release: *r.releases[binding.ReleaseID]})
		}
	}
	return out, nil
}

func (r *workflowRegistryMemoryRepo) ListWorkflowReleaseStats(_ context.Context, _ time.Time) ([]domain.WorkflowReleaseStats, error) {
	return nil, nil
}

func TestWorkflowRegistryPublishResolveAndPinVersions(t *testing.T) {
	ctx := context.Background()
	repo := newWorkflowRegistryMemoryRepo()
	svc := NewWorkflowRegistryService(repo, nil, nil)
	prompt, err := svc.PublishPromptVersion(ctx, &domain.PromptTemplateVersion{
		ID: "prompt_storyboard_bible_1", Key: "storyboard.bible", Version: 1, Type: "chat", SystemTemplate: "strict json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if prompt.Checksum == "" {
		t.Fatal("expected prompt checksum")
	}

	release, err := svc.PublishRelease(ctx, &domain.WorkflowRelease{
		ID: "wfr_storyboard_1", Key: "storyboard", Version: 1, Name: "Storyboard", Status: "released",
		Definition:   domain.WorkflowDefinition{Nodes: []domain.WorkflowNode{{ID: "generate", Type: "activity", Activity: "legacy.storyboard.generate"}}},
		PromptBundle: map[string]string{"generate": prompt.ID, "generate:scene_plan": prompt.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if release.Checksum == "" {
		t.Fatal("expected workflow checksum")
	}
	if _, err := svc.PublishRelease(ctx, &domain.WorkflowRelease{
		ID: "wfr_storyboard_bad", Key: "storyboard", Version: 2, Name: "Bad", Status: "released",
		Definition:   domain.WorkflowDefinition{Nodes: []domain.WorkflowNode{{ID: "generate", Type: "activity", Activity: "legacy.storyboard.generate"}}},
		PromptBundle: map[string]string{"missing": prompt.ID},
	}); err == nil {
		t.Fatal("expected prompt bundle referencing an unknown node to fail")
	}
	release.CreatedAt = release.CreatedAt.Add(time.Hour)
	release.PublishedAt = release.PublishedAt.Add(time.Hour)
	if _, err := svc.PublishRelease(ctx, release); err != nil {
		t.Fatalf("metadata-only retry must remain idempotent: %v", err)
	}

	_, err = svc.SaveBinding(ctx, &domain.WorkflowBinding{
		ID: "binding_storyboard", Surface: "voyager.create", Action: "storyboard", ReleaseID: release.ID, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := svc.Resolve(ctx, "voyager.create", "storyboard", "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Release.ID != release.ID || resolved.Release.PromptBundle["generate"] != prompt.ID {
		t.Fatalf("unexpected resolved release: %+v", resolved.Release)
	}
	pinned, snapshots, err := svc.ResolvePinnedPromptSnapshots(ctx, "voyager.create", "storyboard", "", release.ID)
	if err != nil || pinned.Release.ID != release.ID || snapshots["generate"].ID != prompt.ID || snapshots["generate:scene_plan"].ID != prompt.ID {
		t.Fatalf("failed to resolve pinned prompt snapshots: entry=%+v snapshots=%+v err=%v", pinned, snapshots, err)
	}
	if _, _, err := svc.ResolvePinnedPromptSnapshots(ctx, "voyager.create", "storyboard", "", "wfr_stale"); err == nil {
		t.Fatal("expected stale client release to be rejected")
	}
}

func TestWorkflowRegistryRejectsInvalidPromptTemplateSyntax(t *testing.T) {
	svc := NewWorkflowRegistryService(newWorkflowRegistryMemoryRepo(), nil, nil)
	if _, err := svc.PublishPromptVersion(context.Background(), &domain.PromptTemplateVersion{
		ID: "prompt_invalid", Key: "storyboard.invalid", Version: 1, Type: "chat", SystemTemplate: "{{.broken",
	}); err == nil {
		t.Fatal("expected invalid template syntax to be rejected")
	}
}
