package common_test

import (
	"encoding/json"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
)

// TestCommonPackageImport verifies that the common package can be imported and used correctly
func TestCommonPackageImport(t *testing.T) {
	// Test BaseModel
	model := common.BaseModel{
		ID:        "test-id",
		CreatedAt: 1234567890,
		UpdatedAt: 1234567899,
	}
	if model.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", model.ID)
	}

	// Test EngagementStats
	stats := common.EngagementStats{
		Likes:    100,
		Comments: 50,
	}
	if stats.Likes != 100 {
		t.Errorf("Expected Likes 100, got %d", stats.Likes)
	}

	// Test BaseStatus
	status := common.StatusActive
	if !status.IsValid() {
		t.Error("Expected StatusActive to be valid")
	}
	if !status.IsActive() {
		t.Error("Expected StatusActive to be active")
	}

	// Test TaskStatus
	taskStatus := common.TaskStatusPending
	if !taskStatus.IsValid() {
		t.Error("Expected TaskStatusPending to be valid")
	}

	// Test status transition
	if !taskStatus.CanTransitionTo(common.TaskStatusProcessing) {
		t.Error("Expected transition from pending to processing to be valid")
	}

	// Test ContentStatus
	contentStatus := common.ContentStatusPublished
	if !contentStatus.IsPublic() {
		t.Error("Expected ContentStatusPublished to be public")
	}

	// Test UserRef
	userRef := common.UserRef{
		ID:       "user-123",
		Username: "testuser",
		Name:     "Test User",
	}

	data, err := json.Marshal(userRef)
	if err != nil {
		t.Fatalf("Failed to marshal UserRef: %v", err)
	}

	var decoded common.UserRef
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal UserRef: %v", err)
	}

	if decoded.ID != userRef.ID {
		t.Errorf("Expected ID %s, got %s", userRef.ID, decoded.ID)
	}

	// Test all status types are properly exported and usable
	statuses := []interface {
		IsValid() bool
		String() string
	}{
		common.StatusDraft,
		common.TaskStatusCompleted,
		common.ContentStatusArchived,
		common.MembershipStatusActive,
		common.OrderStatusPaid,
		common.DeletionStatusPending,
		common.PublicationStatusPublished,
	}

	for _, s := range statuses {
		if !s.IsValid() {
			t.Errorf("Expected status %s to be valid", s.String())
		}
	}
}

// TestCommonPackageIntegration tests integration between different types
func TestCommonPackageIntegration(t *testing.T) {
	// Create a struct that embeds BaseModel
	type Post struct {
		common.BaseModel
		Title   string
		Content string
		Status  common.ContentStatus
		Stats   common.EngagementStats
		Author  common.UserRef
	}

	post := Post{
		BaseModel: common.BaseModel{
			ID:        "post-123",
			CreatedAt: 1234567890,
			UpdatedAt: 1234567899,
		},
		Title:   "Test Post",
		Content: "This is a test post",
		Status:  common.ContentStatusPublished,
		Stats: common.EngagementStats{
			Likes:    100,
			Comments: 50,
			Shares:   25,
			Views:    1000,
		},
		Author: common.UserRef{
			ID:       "user-123",
			Username: "testuser",
			Name:     "Test User",
		},
	}

	// Verify all fields
	if post.ID != "post-123" {
		t.Errorf("Expected ID 'post-123', got '%s'", post.ID)
	}

	if !post.Status.IsPublic() {
		t.Error("Expected post status to be public")
	}

	if post.Stats.Likes != 100 {
		t.Errorf("Expected 100 likes, got %d", post.Stats.Likes)
	}

	if post.Author.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", post.Author.Username)
	}

	// Test JSON serialization
	data, err := json.Marshal(post)
	if err != nil {
		t.Fatalf("Failed to marshal Post: %v", err)
	}

	var decoded Post
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Post: %v", err)
	}

	if decoded.ID != post.ID {
		t.Errorf("Expected ID %s, got %s", post.ID, decoded.ID)
	}

	if decoded.Title != post.Title {
		t.Errorf("Expected Title %s, got %s", post.Title, decoded.Title)
	}
}
