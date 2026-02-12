package common

import (
	"encoding/json"
	"testing"
)

// TestBaseModelFields verifies that BaseModel has all expected fields with correct tags
func TestBaseModelFields(t *testing.T) {
	model := BaseModel{
		ID:        "test-123",
		CreatedAt: 1234567890,
		UpdatedAt: 1234567899,
	}

	// Test JSON serialization
	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Failed to marshal BaseModel: %v", err)
	}

	expected := `{"id":"test-123","createdAt":1234567890,"updatedAt":1234567899}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}

	// Test JSON deserialization
	var decoded BaseModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal BaseModel: %v", err)
	}

	if decoded.ID != model.ID {
		t.Errorf("Expected ID %s, got %s", model.ID, decoded.ID)
	}
	if decoded.CreatedAt != model.CreatedAt {
		t.Errorf("Expected CreatedAt %d, got %d", model.CreatedAt, decoded.CreatedAt)
	}
	if decoded.UpdatedAt != model.UpdatedAt {
		t.Errorf("Expected UpdatedAt %d, got %d", model.UpdatedAt, decoded.UpdatedAt)
	}
}

// TestTimestampsFields verifies that Timestamps has all expected fields with correct tags
func TestTimestampsFields(t *testing.T) {
	timestamps := Timestamps{
		CreatedAt: 1234567890,
		UpdatedAt: 1234567899,
	}

	// Test JSON serialization
	data, err := json.Marshal(timestamps)
	if err != nil {
		t.Fatalf("Failed to marshal Timestamps: %v", err)
	}

	expected := `{"createdAt":1234567890,"updatedAt":1234567899}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// TestEngagementStatsFields verifies that EngagementStats has all expected fields
func TestEngagementStatsFields(t *testing.T) {
	stats := EngagementStats{
		Likes:    100,
		Comments: 50,
		Shares:   25,
		Views:    1000,
	}

	// Test JSON serialization
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal EngagementStats: %v", err)
	}

	expected := `{"likes":100,"comments":50,"shares":25,"views":1000}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}

	// Test JSON deserialization
	var decoded EngagementStats
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal EngagementStats: %v", err)
	}

	if decoded.Likes != stats.Likes {
		t.Errorf("Expected Likes %d, got %d", stats.Likes, decoded.Likes)
	}
	if decoded.Comments != stats.Comments {
		t.Errorf("Expected Comments %d, got %d", stats.Comments, decoded.Comments)
	}
	if decoded.Shares != stats.Shares {
		t.Errorf("Expected Shares %d, got %d", stats.Shares, decoded.Shares)
	}
	if decoded.Views != stats.Views {
		t.Errorf("Expected Views %d, got %d", stats.Views, decoded.Views)
	}
}

// TestSocialStatsFields verifies that SocialStats has all expected fields
func TestSocialStatsFields(t *testing.T) {
	stats := SocialStats{
		Followers: 500,
		Following: 200,
	}

	// Test JSON serialization
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal SocialStats: %v", err)
	}

	expected := `{"followers":500,"following":200}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}

	// Test JSON deserialization
	var decoded SocialStats
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal SocialStats: %v", err)
	}

	if decoded.Followers != stats.Followers {
		t.Errorf("Expected Followers %d, got %d", stats.Followers, decoded.Followers)
	}
	if decoded.Following != stats.Following {
		t.Errorf("Expected Following %d, got %d", stats.Following, decoded.Following)
	}
}

// TestEntityRefFields verifies that EntityRef has all expected fields
func TestEntityRefFields(t *testing.T) {
	ref := EntityRef{
		ID:     "entity-123",
		Name:   "Test Entity",
		Avatar: "https://example.com/avatar.png",
	}

	// Test JSON serialization
	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Failed to marshal EntityRef: %v", err)
	}

	expected := `{"id":"entity-123","name":"Test Entity","avatar":"https://example.com/avatar.png"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}

	// Test JSON deserialization
	var decoded EntityRef
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal EntityRef: %v", err)
	}

	if decoded.ID != ref.ID {
		t.Errorf("Expected ID %s, got %s", ref.ID, decoded.ID)
	}
	if decoded.Name != ref.Name {
		t.Errorf("Expected Name %s, got %s", ref.Name, decoded.Name)
	}
	if decoded.Avatar != ref.Avatar {
		t.Errorf("Expected Avatar %s, got %s", ref.Avatar, decoded.Avatar)
	}
}

// TestEntityRefOmitEmpty verifies that omitempty tags work correctly
func TestEntityRefOmitEmpty(t *testing.T) {
	ref := EntityRef{
		ID: "entity-123",
		// Name and Avatar are empty
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Failed to marshal EntityRef: %v", err)
	}

	expected := `{"id":"entity-123"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// TestUserRefFields verifies that UserRef has all expected fields
func TestUserRefFields(t *testing.T) {
	ref := UserRef{
		ID:       "user-123",
		Username: "testuser",
		Name:     "Test User",
		Avatar:   "https://example.com/avatar.png",
	}

	// Test JSON serialization
	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Failed to marshal UserRef: %v", err)
	}

	expected := `{"id":"user-123","username":"testuser","name":"Test User","avatar":"https://example.com/avatar.png"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}

	// Test JSON deserialization
	var decoded UserRef
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal UserRef: %v", err)
	}

	if decoded.ID != ref.ID {
		t.Errorf("Expected ID %s, got %s", ref.ID, decoded.ID)
	}
	if decoded.Username != ref.Username {
		t.Errorf("Expected Username %s, got %s", ref.Username, decoded.Username)
	}
	if decoded.Name != ref.Name {
		t.Errorf("Expected Name %s, got %s", ref.Name, decoded.Name)
	}
	if decoded.Avatar != ref.Avatar {
		t.Errorf("Expected Avatar %s, got %s", ref.Avatar, decoded.Avatar)
	}
}

// TestUserRefOmitEmpty verifies that omitempty tags work correctly for UserRef
func TestUserRefOmitEmpty(t *testing.T) {
	ref := UserRef{
		ID:       "user-123",
		Username: "testuser",
		Name:     "Test User",
		// Avatar is empty
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Failed to marshal UserRef: %v", err)
	}

	expected := `{"id":"user-123","username":"testuser","name":"Test User"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}
