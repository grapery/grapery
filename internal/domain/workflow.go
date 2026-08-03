package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrWorkflowReleaseImmutable = errors.New("workflow release is immutable")
	ErrPromptVersionImmutable   = errors.New("prompt version is immutable")
)

// WorkflowNode is a declarative, provider-neutral node. Activity contains a
// code-registered capability; operators can configure it but cannot inject code.
type WorkflowNode struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Activity  string         `json:"activity,omitempty"`
	DependsOn []string       `json:"dependsOn,omitempty"`
	Config    map[string]any `json:"config,omitempty"`
}

type WorkflowDefinition struct {
	InputSchema  map[string]any `json:"inputSchema,omitempty"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
	Nodes        []WorkflowNode `json:"nodes"`
}

type WorkflowPolicies struct {
	MaxDurationSeconds int `json:"maxDurationSeconds,omitempty"`
	MaxParallelism     int `json:"maxParallelism,omitempty"`
	MaxAttempts        int `json:"maxAttempts,omitempty"`
}

// WorkflowRelease is an immutable executable artifact published by Forge.
type WorkflowRelease struct {
	ID           string             `json:"id"`
	Key          string             `json:"key"`
	Version      int                `json:"version"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Status       string             `json:"status"`
	Manifest     map[string]any     `json:"manifest,omitempty"`
	Definition   WorkflowDefinition `json:"definition"`
	PromptBundle map[string]string  `json:"promptBundle,omitempty"`
	Policies     WorkflowPolicies   `json:"policies,omitempty"`
	Checksum     string             `json:"checksum"`
	CreatedBy    string             `json:"createdBy,omitempty"`
	ApprovedBy   []string           `json:"approvedBy,omitempty"`
	PublishedAt  time.Time          `json:"publishedAt"`
	CreatedAt    time.Time          `json:"createdAt"`
}

// PromptTemplateVersion is immutable and can be referenced by release bundles.
type PromptTemplateVersion struct {
	ID              string         `json:"id"`
	Key             string         `json:"key"`
	Version         int            `json:"version"`
	Type            string         `json:"type"`
	SystemTemplate  string         `json:"systemTemplate,omitempty"`
	UserTemplate    string         `json:"userTemplate,omitempty"`
	VariablesSchema map[string]any `json:"variablesSchema,omitempty"`
	OutputSchema    map[string]any `json:"outputSchema,omitempty"`
	ModelConfig     map[string]any `json:"modelConfig,omitempty"`
	Checksum        string         `json:"checksum"`
	CreatedBy       string         `json:"createdBy,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
}

// WorkflowBinding activates a release for a product surface/action pair.
type WorkflowBinding struct {
	ID          string         `json:"id"`
	Surface     string         `json:"surface"`
	Action      string         `json:"action"`
	TenantID    string         `json:"tenantId,omitempty"`
	WorkflowKey string         `json:"workflowKey"`
	ReleaseID   string         `json:"releaseId"`
	Priority    int            `json:"priority,omitempty"`
	Enabled     bool           `json:"enabled"`
	Conditions  map[string]any `json:"conditions,omitempty"`
	CreatedBy   string         `json:"createdBy,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type WorkflowCatalogEntry struct {
	Binding WorkflowBinding `json:"binding"`
	Release WorkflowRelease `json:"release"`
}

type WorkflowRegistryRepository interface {
	SavePromptVersion(ctx context.Context, prompt *PromptTemplateVersion) error
	GetPromptVersion(ctx context.Context, id string) (*PromptTemplateVersion, error)
	SaveWorkflowRelease(ctx context.Context, release *WorkflowRelease) error
	GetWorkflowRelease(ctx context.Context, id string) (*WorkflowRelease, error)
	SaveWorkflowBinding(ctx context.Context, binding *WorkflowBinding) error
	ListWorkflowCatalog(ctx context.Context, surface, action, tenantID string) ([]*WorkflowCatalogEntry, error)
}

func ValidateWorkflowRelease(release *WorkflowRelease) error {
	if release == nil {
		return errors.New("workflow release is required")
	}
	if strings.TrimSpace(release.ID) == "" || strings.TrimSpace(release.Key) == "" || release.Version <= 0 {
		return errors.New("workflow release id, key and positive version are required")
	}
	if release.Status != "released" && release.Status != "active" {
		return errors.New("workflow release status must be released or active")
	}
	if len(release.Definition.Nodes) == 0 || len(release.Definition.Nodes) > 200 {
		return errors.New("workflow must contain between 1 and 200 nodes")
	}
	if release.Policies.MaxDurationSeconds < 0 || release.Policies.MaxDurationSeconds > 12*60*60 {
		return errors.New("workflow max duration must be between 0 and 43200 seconds")
	}
	if release.Policies.MaxParallelism < 0 || release.Policies.MaxParallelism > 32 {
		return errors.New("workflow max parallelism must be between 0 and 32")
	}
	if release.Policies.MaxAttempts < 0 || release.Policies.MaxAttempts > 10 {
		return errors.New("workflow max attempts must be between 0 and 10")
	}
	return validateWorkflowGraph(release.Definition.Nodes)
}

func validateWorkflowGraph(nodes []WorkflowNode) error {
	allowedTypes := map[string]bool{
		"activity": true, "condition": true, "parallel": true, "foreach": true,
		"wait": true, "human_input": true, "sub_workflow": true, "persist": true,
	}
	byID := make(map[string]WorkflowNode, len(nodes))
	allowedActivities := map[string]bool{
		"legacy.storyboard.generate":     true,
		"storyboard.ensure_draft":        true,
		"storyboard.generate_bible_plan": true,
		"storyboard.generate_scene_plan": true,
		"storyboard.persist_content":     true,
		"storyboard.await_content":       true,
		"storyboard.ensure_images":       true,
	}
	for _, node := range nodes {
		node.ID = strings.TrimSpace(node.ID)
		if node.ID == "" || !allowedTypes[node.Type] {
			return fmt.Errorf("invalid workflow node id or type: %q/%q", node.ID, node.Type)
		}
		if _, exists := byID[node.ID]; exists {
			return fmt.Errorf("duplicate workflow node id: %s", node.ID)
		}
		if (node.Type == "activity" || node.Type == "persist") && strings.TrimSpace(node.Activity) == "" {
			return fmt.Errorf("workflow node %s requires an activity", node.ID)
		}
		if (node.Type == "activity" || node.Type == "persist") && !allowedActivities[strings.TrimSpace(node.Activity)] {
			return fmt.Errorf("workflow node %s uses an unregistered activity %s", node.ID, node.Activity)
		}
		byID[node.ID] = node
	}
	indegree := make(map[string]int, len(nodes))
	children := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		seen := map[string]bool{}
		for _, dependency := range node.DependsOn {
			dependency = strings.TrimSpace(dependency)
			if dependency == node.ID {
				return fmt.Errorf("workflow node %s cannot depend on itself", node.ID)
			}
			if _, ok := byID[dependency]; !ok {
				return fmt.Errorf("workflow node %s depends on unknown node %s", node.ID, dependency)
			}
			if seen[dependency] {
				continue
			}
			seen[dependency] = true
			indegree[node.ID]++
			children[dependency] = append(children[dependency], node.ID)
		}
	}
	queue := make([]string, 0, len(nodes))
	for id := range byID {
		if indegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, child := range children[id] {
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}
	if visited != len(nodes) {
		return errors.New("workflow graph contains a cycle")
	}
	return nil
}
