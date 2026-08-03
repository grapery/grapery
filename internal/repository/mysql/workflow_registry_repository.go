package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

type WorkflowRegistryRepository struct{ db *gorm.DB }

func NewWorkflowRegistryRepository(db *gorm.DB) *WorkflowRegistryRepository {
	return &WorkflowRegistryRepository{db: db}
}

func (r *WorkflowRegistryRepository) SavePromptVersion(ctx context.Context, prompt *domain.PromptTemplateVersion) error {
	row, err := promptVersionToDB(prompt)
	if err != nil {
		return err
	}
	var existing PromptTemplateVersionDB
	err = r.db.WithContext(ctx).Where("id = ?", row.ID).Take(&existing).Error
	if err == nil {
		if existing.Checksum == row.Checksum {
			return nil
		}
		return domain.ErrPromptVersionImmutable
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *WorkflowRegistryRepository) GetPromptVersion(ctx context.Context, id string) (*domain.PromptTemplateVersion, error) {
	var row PromptTemplateVersionDB
	if err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).Take(&row).Error; err != nil {
		return nil, err
	}
	return promptVersionFromDB(&row)
}

func (r *WorkflowRegistryRepository) SaveWorkflowRelease(ctx context.Context, release *domain.WorkflowRelease) error {
	row, err := workflowReleaseToDB(release)
	if err != nil {
		return err
	}
	var existing WorkflowReleaseDB
	err = r.db.WithContext(ctx).Where("id = ?", row.ID).Take(&existing).Error
	if err == nil {
		if existing.Checksum == row.Checksum {
			return nil
		}
		return domain.ErrWorkflowReleaseImmutable
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *WorkflowRegistryRepository) GetWorkflowRelease(ctx context.Context, id string) (*domain.WorkflowRelease, error) {
	var row WorkflowReleaseDB
	if err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).Take(&row).Error; err != nil {
		return nil, err
	}
	return workflowReleaseFromDB(&row)
}

func (r *WorkflowRegistryRepository) SaveWorkflowBinding(ctx context.Context, binding *domain.WorkflowBinding) error {
	conditions, err := json.Marshal(binding.Conditions)
	if err != nil {
		return fmt.Errorf("marshal workflow binding conditions: %w", err)
	}
	row := WorkflowBindingDB{
		ID: binding.ID, Surface: binding.Surface, Action: binding.Action, TenantID: binding.TenantID,
		WorkflowKey: binding.WorkflowKey, ReleaseID: binding.ReleaseID, Priority: binding.Priority,
		Enabled: binding.Enabled, ConditionsJSON: string(conditions), CreatedBy: binding.CreatedBy,
		CreatedAt: binding.CreatedAt, UpdatedAt: binding.UpdatedAt,
	}
	return r.db.WithContext(ctx).Save(&row).Error
}

func (r *WorkflowRegistryRepository) ListWorkflowCatalog(ctx context.Context, surface, action, tenantID string) ([]*domain.WorkflowCatalogEntry, error) {
	q := r.db.WithContext(ctx).Where("enabled = ?", true)
	if surface = strings.TrimSpace(surface); surface != "" {
		q = q.Where("surface = ?", surface)
	}
	if action = strings.TrimSpace(action); action != "" {
		q = q.Where("action = ?", action)
	}
	if tenantID = strings.TrimSpace(tenantID); tenantID != "" {
		q = q.Where("tenant_id IN ?", []string{"", tenantID})
	} else {
		q = q.Where("tenant_id = ''")
	}
	var bindings []WorkflowBindingDB
	if err := q.Order("priority DESC, updated_at DESC").Find(&bindings).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.WorkflowCatalogEntry, 0, len(bindings))
	for i := range bindings {
		release, err := r.GetWorkflowRelease(ctx, bindings[i].ReleaseID)
		if err != nil {
			return nil, err
		}
		binding, err := workflowBindingFromDB(&bindings[i])
		if err != nil {
			return nil, err
		}
		out = append(out, &domain.WorkflowCatalogEntry{Binding: *binding, Release: *release})
	}
	return out, nil
}

func workflowReleaseToDB(release *domain.WorkflowRelease) (*WorkflowReleaseDB, error) {
	manifest, err := json.Marshal(release.Manifest)
	if err != nil {
		return nil, err
	}
	definition, err := json.Marshal(release.Definition)
	if err != nil {
		return nil, err
	}
	bundle, err := json.Marshal(release.PromptBundle)
	if err != nil {
		return nil, err
	}
	policies, err := json.Marshal(release.Policies)
	if err != nil {
		return nil, err
	}
	approved, err := json.Marshal(release.ApprovedBy)
	if err != nil {
		return nil, err
	}
	return &WorkflowReleaseDB{
		ID: release.ID, WorkflowKey: release.Key, Version: release.Version, Name: release.Name,
		Description: release.Description, Status: release.Status, ManifestJSON: string(manifest),
		DefinitionJSON: string(definition), PromptBundleJSON: string(bundle), PoliciesJSON: string(policies),
		Checksum: release.Checksum, CreatedBy: release.CreatedBy, ApprovedByJSON: string(approved),
		PublishedAt: release.PublishedAt, CreatedAt: release.CreatedAt,
	}, nil
}

func workflowReleaseFromDB(row *WorkflowReleaseDB) (*domain.WorkflowRelease, error) {
	out := &domain.WorkflowRelease{
		ID: row.ID, Key: row.WorkflowKey, Version: row.Version, Name: row.Name, Description: row.Description,
		Status: row.Status, Checksum: row.Checksum, CreatedBy: row.CreatedBy, PublishedAt: row.PublishedAt, CreatedAt: row.CreatedAt,
	}
	if err := json.Unmarshal([]byte(defaultJSON(row.ManifestJSON, "{}")), &out.Manifest); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(row.DefinitionJSON), &out.Definition); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(defaultJSON(row.PromptBundleJSON, "{}")), &out.PromptBundle); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(defaultJSON(row.PoliciesJSON, "{}")), &out.Policies); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(defaultJSON(row.ApprovedByJSON, "[]")), &out.ApprovedBy); err != nil {
		return nil, err
	}
	return out, nil
}

func promptVersionToDB(prompt *domain.PromptTemplateVersion) (*PromptTemplateVersionDB, error) {
	variables, err := json.Marshal(prompt.VariablesSchema)
	if err != nil {
		return nil, err
	}
	output, err := json.Marshal(prompt.OutputSchema)
	if err != nil {
		return nil, err
	}
	config, err := json.Marshal(prompt.ModelConfig)
	if err != nil {
		return nil, err
	}
	return &PromptTemplateVersionDB{
		ID: prompt.ID, PromptKey: prompt.Key, Version: prompt.Version, Type: prompt.Type,
		SystemTemplate: prompt.SystemTemplate, UserTemplate: prompt.UserTemplate,
		VariablesSchemaJSON: string(variables), OutputSchemaJSON: string(output), ModelConfigJSON: string(config),
		Checksum: prompt.Checksum, CreatedBy: prompt.CreatedBy, CreatedAt: prompt.CreatedAt,
	}, nil
}

func promptVersionFromDB(row *PromptTemplateVersionDB) (*domain.PromptTemplateVersion, error) {
	out := &domain.PromptTemplateVersion{
		ID: row.ID, Key: row.PromptKey, Version: row.Version, Type: row.Type,
		SystemTemplate: row.SystemTemplate, UserTemplate: row.UserTemplate,
		Checksum: row.Checksum, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt,
	}
	if err := json.Unmarshal([]byte(defaultJSON(row.VariablesSchemaJSON, "{}")), &out.VariablesSchema); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(defaultJSON(row.OutputSchemaJSON, "{}")), &out.OutputSchema); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(defaultJSON(row.ModelConfigJSON, "{}")), &out.ModelConfig); err != nil {
		return nil, err
	}
	return out, nil
}

func workflowBindingFromDB(row *WorkflowBindingDB) (*domain.WorkflowBinding, error) {
	out := &domain.WorkflowBinding{
		ID: row.ID, Surface: row.Surface, Action: row.Action, TenantID: row.TenantID,
		WorkflowKey: row.WorkflowKey, ReleaseID: row.ReleaseID, Priority: row.Priority,
		Enabled: row.Enabled, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if err := json.Unmarshal([]byte(defaultJSON(row.ConditionsJSON, "{}")), &out.Conditions); err != nil {
		return nil, err
	}
	return out, nil
}

func defaultJSON(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
