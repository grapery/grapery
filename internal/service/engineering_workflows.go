package service

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"strings"
)

//go:embed engineering_workflows.json
var engineeringWorkflowJSON []byte

// Decode on each lookup so callers cannot mutate a process-global release.
func engineeringReleases() ([]domain.WorkflowRelease, error) {
	var releases []domain.WorkflowRelease
	if err := json.Unmarshal(engineeringWorkflowJSON, &releases); err != nil {
		return nil, fmt.Errorf("decode engineering workflows: %w", err)
	}
	for i := range releases {
		release := &releases[i]
		if err := domain.ValidateWorkflowRelease(release); err != nil {
			return nil, err
		}
		checksum, err := checksumValue(struct {
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
		release.Checksum = checksum
	}
	return releases, nil
}

func engineeringRelease(id string) (*domain.WorkflowRelease, error) {
	releases, err := engineeringReleases()
	if err != nil {
		return nil, err
	}
	for _, release := range releases {
		if release.ID == id {
			return &release, nil
		}
	}
	return nil, domain.ErrNotFound
}
func engineeringCatalog(surface, action, tenantID string) ([]*domain.WorkflowCatalogEntry, error) {
	releases, err := engineeringReleases()
	if err != nil {
		return nil, err
	}
	out := []*domain.WorkflowCatalogEntry{}
	for _, release := range releases {
		if release.Manifest["surface"] != strings.TrimSpace(surface) || release.Manifest["action"] != strings.TrimSpace(action) {
			continue
		}
		out = append(out, &domain.WorkflowCatalogEntry{Release: release, Binding: domain.WorkflowBinding{
			ID: "engineering.default." + release.Key, Surface: surface, Action: action, TenantID: tenantID, WorkflowKey: release.Key,
			ReleaseID: release.ID, Priority: -10000, Enabled: true, CreatedBy: "engineering",
		}})
	}
	return out, nil
}
func validateWorkflowCreativeControls(manifest map[string]any) error {
	if raw, exists := manifest["creativeControls"]; exists {
		_, err := domain.ParseCreativeControls(raw)
		return err
	}
	return nil
}
func normalizeWorkflowCreativeControls(manifest map[string]any) error {
	raw, exists := manifest["creativeControls"]
	if !exists {
		return nil
	}
	controls, err := domain.ParseCreativeControls(raw)
	if err != nil {
		return err
	}
	data, err := json.Marshal(controls)
	if err != nil {
		return err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	manifest["creativeControls"] = object
	return nil
}

func engineeringManagedCreationSurface(surface string) bool {
	switch strings.TrimSpace(surface) {
	case "voyager.fragment", "voyager.storyboard", "voyager.character":
		return true
	default:
		return false
	}
}
