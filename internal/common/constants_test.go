package common

import (
	"testing"
)

// TestBaseStatusConst verifies all BaseStatus constants are defined correctly
func TestBaseStatusConst(t *testing.T) {
	tests := []struct {
		status BaseStatus
		value  string
	}{
		{StatusActive, "active"},
		{StatusDraft, "draft"},
		{StatusPending, "pending"},
		{StatusDeleted, "deleted"},
		{StatusFailed, "failed"},
		{StatusCancelled, "cancelled"},
		{StatusSuspended, "suspended"},
		{StatusExpired, "expired"},
		{StatusPendingDeletion, "pending_deletion"},
		{StatusSystem, "system"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s, got %s", tt.value, tt.status)
		}
	}
}

// TestBaseStatusIsValid verifies the IsValid method for BaseStatus
func TestBaseStatusIsValid(t *testing.T) {
	validStatuses := []BaseStatus{
		StatusActive, StatusDraft, StatusPending, StatusDeleted,
		StatusFailed, StatusCancelled, StatusSuspended, StatusExpired,
		StatusPendingDeletion, StatusSystem,
	}

	for _, status := range validStatuses {
		if !status.IsValid() {
			t.Errorf("Expected %s to be valid", status)
		}
	}

	invalidStatus := BaseStatus("invalid")
	if invalidStatus.IsValid() {
		t.Error("Expected invalid status to be invalid")
	}
}

// TestBaseStatusString verifies the String method for BaseStatus
func TestBaseStatusString(t *testing.T) {
	status := StatusActive
	if status.String() != "active" {
		t.Errorf("Expected 'active', got '%s'", status.String())
	}
}

// TestBaseStatusIsTerminal verifies the IsTerminal method for BaseStatus
func TestBaseStatusIsTerminal(t *testing.T) {
	terminalStatuses := []BaseStatus{
		StatusDeleted, StatusFailed, StatusCancelled, StatusExpired,
	}

	for _, status := range terminalStatuses {
		if !status.IsTerminal() {
			t.Errorf("Expected %s to be terminal", status)
		}
	}

	nonTerminalStatuses := []BaseStatus{
		StatusActive, StatusDraft, StatusPending, StatusSuspended,
		StatusPendingDeletion, StatusSystem,
	}

	for _, status := range nonTerminalStatuses {
		if status.IsTerminal() {
			t.Errorf("Expected %s to not be terminal", status)
		}
	}
}

// TestBaseStatusIsActive verifies the IsActive method for BaseStatus
func TestBaseStatusIsActive(t *testing.T) {
	if !StatusActive.IsActive() {
		t.Error("Expected StatusActive to be active")
	}

	otherStatuses := []BaseStatus{
		StatusDraft, StatusPending, StatusDeleted, StatusFailed,
		StatusCancelled, StatusSuspended, StatusExpired,
		StatusPendingDeletion, StatusSystem,
	}

	for _, status := range otherStatuses {
		if status.IsActive() {
			t.Errorf("Expected %s to not be active", status)
		}
	}
}

// TestTaskStatusConst verifies all TaskStatus constants are defined correctly
func TestTaskStatusConst(t *testing.T) {
	tests := []struct {
		status TaskStatus
		value  string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusProcessing, "processing"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
		{TaskStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s, got %s", tt.value, tt.status)
		}
	}
}

// TestTaskStatusIsValid verifies the IsValid method for TaskStatus
func TestTaskStatusIsValid(t *testing.T) {
	validStatuses := []TaskStatus{
		TaskStatusPending, TaskStatusProcessing, TaskStatusCompleted,
		TaskStatusFailed, TaskStatusCancelled,
	}

	for _, status := range validStatuses {
		if !status.IsValid() {
			t.Errorf("Expected %s to be valid", status)
		}
	}

	invalidStatus := TaskStatus("invalid")
	if invalidStatus.IsValid() {
		t.Error("Expected invalid status to be invalid")
	}
}

// TestTaskStatusString verifies the String method for TaskStatus
func TestTaskStatusString(t *testing.T) {
	status := TaskStatusProcessing
	if status.String() != "processing" {
		t.Errorf("Expected 'processing', got '%s'", status.String())
	}
}

// TestTaskStatusIsTerminal verifies the IsTerminal method for TaskStatus
func TestTaskStatusIsTerminal(t *testing.T) {
	terminalStatuses := []TaskStatus{
		TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled,
	}

	for _, status := range terminalStatuses {
		if !status.IsTerminal() {
			t.Errorf("Expected %s to be terminal", status)
		}
	}

	nonTerminalStatuses := []TaskStatus{
		TaskStatusPending, TaskStatusProcessing,
	}

	for _, status := range nonTerminalStatuses {
		if status.IsTerminal() {
			t.Errorf("Expected %s to not be terminal", status)
		}
	}
}

// TestTaskStatusIsInProgress verifies the IsInProgress method for TaskStatus
func TestTaskStatusIsInProgress(t *testing.T) {
	if !TaskStatusProcessing.IsInProgress() {
		t.Error("Expected TaskStatusProcessing to be in progress")
	}

	otherStatuses := []TaskStatus{
		TaskStatusPending, TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled,
	}

	for _, status := range otherStatuses {
		if status.IsInProgress() {
			t.Errorf("Expected %s to not be in progress", status)
		}
	}
}

// TestTaskStatusCanTransitionTo verifies the CanTransitionTo method for TaskStatus
func TestTaskStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		current       TaskStatus
		new           TaskStatus
		canTransition bool
	}{
		{TaskStatusPending, TaskStatusProcessing, true},
		{TaskStatusPending, TaskStatusCancelled, true},
		{TaskStatusPending, TaskStatusCompleted, false},
		{TaskStatusProcessing, TaskStatusCompleted, true},
		{TaskStatusProcessing, TaskStatusFailed, true},
		{TaskStatusProcessing, TaskStatusCancelled, true},
		{TaskStatusProcessing, TaskStatusPending, false},
		{TaskStatusCompleted, TaskStatusFailed, false},
		{TaskStatusFailed, TaskStatusProcessing, false},
	}

	for _, tt := range tests {
		result := tt.current.CanTransitionTo(tt.new)
		if result != tt.canTransition {
			t.Errorf("Expected transition from %s to %s to be %v, got %v",
				tt.current, tt.new, tt.canTransition, result)
		}
	}
}

// TestContentStatusConst verifies all ContentStatus constants are defined correctly
func TestContentStatusConst(t *testing.T) {
	tests := []struct {
		status ContentStatus
		value  string
	}{
		{ContentStatusDraft, "draft"},
		{ContentStatusPublished, "published"},
		{ContentStatusArchived, "archived"},
		{ContentStatusRendering, "rendering"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s, got %s", tt.value, tt.status)
		}
	}
}

// TestContentStatusIsValid verifies the IsValid method for ContentStatus
func TestContentStatusIsValid(t *testing.T) {
	validStatuses := []ContentStatus{
		ContentStatusDraft, ContentStatusPublished,
		ContentStatusArchived, ContentStatusRendering,
	}

	for _, status := range validStatuses {
		if !status.IsValid() {
			t.Errorf("Expected %s to be valid", status)
		}
	}

	invalidStatus := ContentStatus("invalid")
	if invalidStatus.IsValid() {
		t.Error("Expected invalid status to be invalid")
	}
}

// TestContentStatusString verifies the String method for ContentStatus
func TestContentStatusString(t *testing.T) {
	status := ContentStatusPublished
	if status.String() != "published" {
		t.Errorf("Expected 'published', got '%s'", status.String())
	}
}

// TestContentStatusIsPublic verifies the IsPublic method for ContentStatus
func TestContentStatusIsPublic(t *testing.T) {
	if !ContentStatusPublished.IsPublic() {
		t.Error("Expected ContentStatusPublished to be public")
	}

	otherStatuses := []ContentStatus{
		ContentStatusDraft, ContentStatusArchived, ContentStatusRendering,
	}

	for _, status := range otherStatuses {
		if status.IsPublic() {
			t.Errorf("Expected %s to not be public", status)
		}
	}
}

// TestContentStatusIsEditable verifies the IsEditable method for ContentStatus
func TestContentStatusIsEditable(t *testing.T) {
	editableStatuses := []ContentStatus{
		ContentStatusDraft, ContentStatusArchived,
	}

	for _, status := range editableStatuses {
		if !status.IsEditable() {
			t.Errorf("Expected %s to be editable", status)
		}
	}

	nonEditableStatuses := []ContentStatus{
		ContentStatusPublished, ContentStatusRendering,
	}

	for _, status := range nonEditableStatuses {
		if status.IsEditable() {
			t.Errorf("Expected %s to not be editable", status)
		}
	}
}

// TestMembershipStatusConst verifies all MembershipStatus constants are defined correctly
func TestMembershipStatusConst(t *testing.T) {
	tests := []struct {
		status MembershipStatus
		value  string
	}{
		{MembershipStatusActive, "active"},
		{MembershipStatusExpired, "expired"},
		{MembershipStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s, got %s", tt.value, tt.status)
		}
	}
}

// TestMembershipStatusIsValid verifies the IsValid method for MembershipStatus
func TestMembershipStatusIsValid(t *testing.T) {
	validStatuses := []MembershipStatus{
		MembershipStatusActive, MembershipStatusExpired, MembershipStatusCancelled,
	}

	for _, status := range validStatuses {
		if !status.IsValid() {
			t.Errorf("Expected %s to be valid", status)
		}
	}

	invalidStatus := MembershipStatus("invalid")
	if invalidStatus.IsValid() {
		t.Error("Expected invalid status to be invalid")
	}
}

// TestMembershipStatusString verifies the String method for MembershipStatus
func TestMembershipStatusString(t *testing.T) {
	status := MembershipStatusActive
	if status.String() != "active" {
		t.Errorf("Expected 'active', got '%s'", status.String())
	}
}

// TestMembershipStatusIsActive verifies the IsActive method for MembershipStatus
func TestMembershipStatusIsActive(t *testing.T) {
	if !MembershipStatusActive.IsActive() {
		t.Error("Expected MembershipStatusActive to be active")
	}

	otherStatuses := []MembershipStatus{
		MembershipStatusExpired, MembershipStatusCancelled,
	}

	for _, status := range otherStatuses {
		if status.IsActive() {
			t.Errorf("Expected %s to not be active", status)
		}
	}
}

// TestOrderStatusConst verifies all OrderStatus constants are defined correctly
func TestOrderStatusConst(t *testing.T) {
	tests := []struct {
		status OrderStatus
		value  string
	}{
		{OrderStatusPending, "pending"},
		{OrderStatusPaid, "paid"},
		{OrderStatusFailed, "failed"},
		{OrderStatusRefunded, "refunded"},
		{OrderStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s, got %s", tt.value, tt.status)
		}
	}
}

// TestOrderStatusIsValid verifies the IsValid method for OrderStatus
func TestOrderStatusIsValid(t *testing.T) {
	validStatuses := []OrderStatus{
		OrderStatusPending, OrderStatusPaid, OrderStatusFailed,
		OrderStatusRefunded, OrderStatusCancelled,
	}

	for _, status := range validStatuses {
		if !status.IsValid() {
			t.Errorf("Expected %s to be valid", status)
		}
	}

	invalidStatus := OrderStatus("invalid")
	if invalidStatus.IsValid() {
		t.Error("Expected invalid status to be invalid")
	}
}

// TestOrderStatusString verifies the String method for OrderStatus
func TestOrderStatusString(t *testing.T) {
	status := OrderStatusPaid
	if status.String() != "paid" {
		t.Errorf("Expected 'paid', got '%s'", status.String())
	}
}

// TestOrderStatusIsSuccessful verifies the IsSuccessful method for OrderStatus
func TestOrderStatusIsSuccessful(t *testing.T) {
	if !OrderStatusPaid.IsSuccessful() {
		t.Error("Expected OrderStatusPaid to be successful")
	}

	otherStatuses := []OrderStatus{
		OrderStatusPending, OrderStatusFailed, OrderStatusRefunded, OrderStatusCancelled,
	}

	for _, status := range otherStatuses {
		if status.IsSuccessful() {
			t.Errorf("Expected %s to not be successful", status)
		}
	}
}

// TestOrderStatusCanBeRefunded verifies the CanBeRefunded method for OrderStatus
func TestOrderStatusCanBeRefunded(t *testing.T) {
	if !OrderStatusPaid.CanBeRefunded() {
		t.Error("Expected OrderStatusPaid to be refundable")
	}

	otherStatuses := []OrderStatus{
		OrderStatusPending, OrderStatusFailed, OrderStatusRefunded, OrderStatusCancelled,
	}

	for _, status := range otherStatuses {
		if status.CanBeRefunded() {
			t.Errorf("Expected %s to not be refundable", status)
		}
	}
}

// TestDeletionStatusConst verifies all DeletionStatus constants are defined correctly
func TestDeletionStatusConst(t *testing.T) {
	tests := []struct {
		status DeletionStatus
		value  string
	}{
		{DeletionStatusPending, "pending"},
		{DeletionStatusProcessing, "processing"},
		{DeletionStatusCompleted, "completed"},
		{DeletionStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s, got %s", tt.value, tt.status)
		}
	}
}

// TestDeletionStatusIsValid verifies the IsValid method for DeletionStatus
func TestDeletionStatusIsValid(t *testing.T) {
	validStatuses := []DeletionStatus{
		DeletionStatusPending, DeletionStatusProcessing,
		DeletionStatusCompleted, DeletionStatusCancelled,
	}

	for _, status := range validStatuses {
		if !status.IsValid() {
			t.Errorf("Expected %s to be valid", status)
		}
	}

	invalidStatus := DeletionStatus("invalid")
	if invalidStatus.IsValid() {
		t.Error("Expected invalid status to be invalid")
	}
}

// TestDeletionStatusString verifies the String method for DeletionStatus
func TestDeletionStatusString(t *testing.T) {
	status := DeletionStatusProcessing
	if status.String() != "processing" {
		t.Errorf("Expected 'processing', got '%s'", status.String())
	}
}

// TestDeletionStatusIsTerminal verifies the IsTerminal method for DeletionStatus
func TestDeletionStatusIsTerminal(t *testing.T) {
	terminalStatuses := []DeletionStatus{
		DeletionStatusCompleted, DeletionStatusCancelled,
	}

	for _, status := range terminalStatuses {
		if !status.IsTerminal() {
			t.Errorf("Expected %s to be terminal", status)
		}
	}

	nonTerminalStatuses := []DeletionStatus{
		DeletionStatusPending, DeletionStatusProcessing,
	}

	for _, status := range nonTerminalStatuses {
		if status.IsTerminal() {
			t.Errorf("Expected %s to not be terminal", status)
		}
	}
}

// TestDeletionStatusCanBeCancelled verifies the CanBeCancelled method for DeletionStatus
func TestDeletionStatusCanBeCancelled(t *testing.T) {
	cancelableStatuses := []DeletionStatus{
		DeletionStatusPending, DeletionStatusProcessing,
	}

	for _, status := range cancelableStatuses {
		if !status.CanBeCancelled() {
			t.Errorf("Expected %s to be cancelable", status)
		}
	}

	nonCancelableStatuses := []DeletionStatus{
		DeletionStatusCompleted, DeletionStatusCancelled,
	}

	for _, status := range nonCancelableStatuses {
		if status.CanBeCancelled() {
			t.Errorf("Expected %s to not be cancelable", status)
		}
	}
}

// TestPublicationStatusConst verifies all PublicationStatus constants are defined correctly
func TestPublicationStatusConst(t *testing.T) {
	tests := []struct {
		status PublicationStatus
		value  string
	}{
		{PublicationStatusPublished, "published"},
		{PublicationStatusUnpublished, "unpublished"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.value {
			t.Errorf("Expected %s, got %s", tt.value, tt.status)
		}
	}
}

// TestPublicationStatusIsValid verifies the IsValid method for PublicationStatus
func TestPublicationStatusIsValid(t *testing.T) {
	validStatuses := []PublicationStatus{
		PublicationStatusPublished, PublicationStatusUnpublished,
	}

	for _, status := range validStatuses {
		if !status.IsValid() {
			t.Errorf("Expected %s to be valid", status)
		}
	}

	invalidStatus := PublicationStatus("invalid")
	if invalidStatus.IsValid() {
		t.Error("Expected invalid status to be invalid")
	}
}

// TestPublicationStatusString verifies the String method for PublicationStatus
func TestPublicationStatusString(t *testing.T) {
	status := PublicationStatusPublished
	if status.String() != "published" {
		t.Errorf("Expected 'published', got '%s'", status.String())
	}
}

// TestPublicationStatusIsPublished verifies the IsPublished method for PublicationStatus
func TestPublicationStatusIsPublished(t *testing.T) {
	if !PublicationStatusPublished.IsPublished() {
		t.Error("Expected PublicationStatusPublished to be published")
	}

	if PublicationStatusUnpublished.IsPublished() {
		t.Error("Expected PublicationStatusUnpublished to not be published")
	}
}
