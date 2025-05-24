package mcps

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// handleCreateStory creates a new story
func (s *McpService) handleCreateStory(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	title, ok := req["title"].(string)
	if !ok {
		return nil, fmt.Errorf("missing title")
	}

	description, _ := req["description"].(string)
	content, _ := req["content"].(string)

	story := &Story{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Content:     content,
		Characters:  make([]string, 0),
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	s.mu.Lock()
	s.stories[story.ID] = story
	s.mu.Unlock()

	return json.Marshal(map[string]interface{}{
		"status": "success",
		"story":  story,
	})
}

// handleGetStory retrieves a story by ID
func (s *McpService) handleGetStory(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	id, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story id")
	}

	s.mu.RLock()
	story, exists := s.stories[id]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("story not found")
	}

	return json.Marshal(map[string]interface{}{
		"status": "success",
		"story":  story,
	})
}

// handleCreateCharacter creates a new character
func (s *McpService) handleCreateCharacter(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	name, ok := req["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing character name")
	}

	storyID, ok := req["story_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story id")
	}

	description, _ := req["description"].(string)
	personality, _ := req["personality"].(string)

	character := &Character{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Personality: personality,
		StoryID:     storyID,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	s.mu.Lock()
	s.characters[character.ID] = character
	if story, exists := s.stories[storyID]; exists {
		story.Characters = append(story.Characters, character.ID)
		story.UpdatedAt = time.Now().Unix()
	}
	s.mu.Unlock()

	return json.Marshal(map[string]interface{}{
		"status":    "success",
		"character": character,
	})
}

// handleGetCharacter retrieves a character by ID
func (s *McpService) handleGetCharacter(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	id, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing character id")
	}

	s.mu.RLock()
	character, exists := s.characters[id]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("character not found")
	}

	return json.Marshal(map[string]interface{}{
		"status":    "success",
		"character": character,
	})
}

// handleGenerateImage generates an image for a story or character
func (s *McpService) handleGenerateImage(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	imageType, ok := req["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing image type")
	}

	var storyID, characterID string
	if imageType == "story" {
		storyID, ok = req["story_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing story id")
		}
	} else if imageType == "character" {
		characterID, ok = req["character_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing character id")
		}
	} else {
		return nil, fmt.Errorf("invalid image type")
	}

	// TODO: Implement actual image generation logic here
	// This is a placeholder that would be replaced with actual image generation
	image := &StoryImage{
		ID:          uuid.New().String(),
		StoryID:     storyID,
		CharacterID: characterID,
		URL:         fmt.Sprintf("https://example.com/images/%s.jpg", uuid.New().String()),
		Type:        imageType,
		CreatedAt:   time.Now().Unix(),
	}

	s.mu.Lock()
	s.images[image.ID] = image
	s.mu.Unlock()

	return json.Marshal(map[string]interface{}{
		"status": "success",
		"image":  image,
	})
}

// handleCharacterChat handles chat interactions with a character
func (s *McpService) handleCharacterChat(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	characterID, ok := req["character_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing character id")
	}

	message, ok := req["message"].(string)
	if !ok {
		return nil, fmt.Errorf("missing message")
	}

	s.mu.RLock()
	character, exists := s.characters[characterID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("character not found")
	}

	// TODO: Implement actual character chat logic here
	// This is a placeholder that would be replaced with actual AI chat implementation
	response := fmt.Sprintf("As %s, I would say: %s", character.Name, message)

	return json.Marshal(map[string]interface{}{
		"status":   "success",
		"response": response,
	})
}

// handleFollowCharacter handles following a character
func (s *McpService) handleFollowCharacter(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	characterID, ok := req["character_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing character id")
	}

	userID, ok := req["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	character, exists := s.characters[characterID]
	if !exists {
		return nil, fmt.Errorf("character not found")
	}

	// Check if already following
	for _, follower := range character.Followers {
		if follower == userID {
			return json.Marshal(map[string]interface{}{
				"status":  "success",
				"message": "already following",
			})
		}
	}

	character.Followers = append(character.Followers, userID)
	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"message": "followed character successfully",
	})
}

// handleUnfollowCharacter handles unfollowing a character
func (s *McpService) handleUnfollowCharacter(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	characterID, ok := req["character_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing character id")
	}

	userID, ok := req["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	character, exists := s.characters[characterID]
	if !exists {
		return nil, fmt.Errorf("character not found")
	}

	// Remove follower
	newFollowers := make([]string, 0)
	for _, follower := range character.Followers {
		if follower != userID {
			newFollowers = append(newFollowers, follower)
		}
	}
	character.Followers = newFollowers

	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"message": "unfollowed character successfully",
	})
}

// handleLikeStory handles liking a story
func (s *McpService) handleLikeStory(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	storyID, ok := req["story_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	story, exists := s.stories[storyID]
	if !exists {
		return nil, fmt.Errorf("story not found")
	}

	story.Likes++
	return json.Marshal(map[string]interface{}{
		"status": "success",
		"likes":  story.Likes,
	})
}

// handleUnlikeStory handles unliking a story
func (s *McpService) handleUnlikeStory(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	storyID, ok := req["story_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	story, exists := s.stories[storyID]
	if !exists {
		return nil, fmt.Errorf("story not found")
	}

	if story.Likes > 0 {
		story.Likes--
	}

	return json.Marshal(map[string]interface{}{
		"status": "success",
		"likes":  story.Likes,
	})
}

// handleRegenerateCharacter handles regenerating a character's description
func (s *McpService) handleRegenerateCharacter(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	characterID, ok := req["character_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing character id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	character, exists := s.characters[characterID]
	if !exists {
		return nil, fmt.Errorf("character not found")
	}

	// TODO: Implement AI-based description generation
	// This is a placeholder that would be replaced with actual AI implementation
	character.Description = fmt.Sprintf("Regenerated description for %s", character.Name)
	character.UpdatedAt = time.Now().Unix()

	return json.Marshal(map[string]interface{}{
		"status":    "success",
		"character": character,
	})
}

// handleFollowStory handles following a story
func (s *McpService) handleFollowStory(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	storyID, ok := req["story_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story id")
	}

	userID, ok := req["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	story, exists := s.stories[storyID]
	if !exists {
		return nil, fmt.Errorf("story not found")
	}

	// Check if already following
	for _, follower := range story.Followers {
		if follower == userID {
			return json.Marshal(map[string]interface{}{
				"status":  "success",
				"message": "already following",
			})
		}
	}

	story.Followers = append(story.Followers, userID)
	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"message": "followed story successfully",
	})
}

// handleUnfollowStory handles unfollowing a story
func (s *McpService) handleUnfollowStory(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	storyID, ok := req["story_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story id")
	}

	userID, ok := req["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	story, exists := s.stories[storyID]
	if !exists {
		return nil, fmt.Errorf("story not found")
	}

	// Remove follower
	newFollowers := make([]string, 0)
	for _, follower := range story.Followers {
		if follower != userID {
			newFollowers = append(newFollowers, follower)
		}
	}
	story.Followers = newFollowers

	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"message": "unfollowed story successfully",
	})
}

// handleCreateStoryVersion handles creating a new story version
func (s *McpService) handleCreateStoryVersion(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	storyID, ok := req["story_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story id")
	}

	content, ok := req["content"].(string)
	if !ok {
		return nil, fmt.Errorf("missing content")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if story exists
	if _, exists := s.stories[storyID]; !exists {
		return nil, fmt.Errorf("story not found")
	}

	version := &StoryVersion{
		ID:        uuid.New().String(),
		StoryID:   storyID,
		Content:   content,
		CreatedAt: time.Now().Unix(),
		Likes:     0,
		Followers: make([]string, 0),
	}

	s.storyVersions[version.ID] = version
	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"version": version,
	})
}

// handleContinueStoryVersion handles continuing a story version
func (s *McpService) handleContinueStoryVersion(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	versionID, ok := req["version_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing version id")
	}

	additionalContent, ok := req["content"].(string)
	if !ok {
		return nil, fmt.Errorf("missing content")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	version, exists := s.storyVersions[versionID]
	if !exists {
		return nil, fmt.Errorf("story version not found")
	}

	version.Content += "\n" + additionalContent
	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"version": version,
	})
}

// handleFollowUser handles following a user
func (s *McpService) handleFollowUser(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	followerID, ok := req["follower_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing follower id")
	}

	followingID, ok := req["following_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing following id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	follower, exists := s.users[followerID]
	if !exists {
		return nil, fmt.Errorf("follower not found")
	}

	following, exists := s.users[followingID]
	if !exists {
		return nil, fmt.Errorf("user to follow not found")
	}

	// Add to following list
	follower.Following = append(follower.Following, followingID)
	// Add to followers list
	following.Followers = append(following.Followers, followerID)

	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"message": "followed user successfully",
	})
}

// handleUnfollowUser handles unfollowing a user
func (s *McpService) handleUnfollowUser(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	followerID, ok := req["follower_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing follower id")
	}

	followingID, ok := req["following_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing following id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	follower, exists := s.users[followerID]
	if !exists {
		return nil, fmt.Errorf("follower not found")
	}

	following, exists := s.users[followingID]
	if !exists {
		return nil, fmt.Errorf("user to unfollow not found")
	}

	// Remove from following list
	newFollowing := make([]string, 0)
	for _, id := range follower.Following {
		if id != followingID {
			newFollowing = append(newFollowing, id)
		}
	}
	follower.Following = newFollowing

	// Remove from followers list
	newFollowers := make([]string, 0)
	for _, id := range following.Followers {
		if id != followerID {
			newFollowers = append(newFollowers, id)
		}
	}
	following.Followers = newFollowers

	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"message": "unfollowed user successfully",
	})
}

// handleAnalyzeCharacterStoryline handles analyzing a character's storyline
func (s *McpService) handleAnalyzeCharacterStoryline(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	characterID, ok := req["character_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing character id")
	}

	s.mu.RLock()
	character, exists := s.characters[characterID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("character not found")
	}

	// TODO: Implement AI-based storyline analysis
	// This is a placeholder that would be replaced with actual AI implementation
	analysis := fmt.Sprintf("Analysis of %s's storyline: This character has shown significant development throughout the story.", character.Name)

	return json.Marshal(map[string]interface{}{
		"status":   "success",
		"analysis": analysis,
	})
}

// handleSubscribeVIP handles subscribing to VIP membership
func (s *McpService) handleSubscribeVIP(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	userID, ok := req["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user id")
	}

	duration, ok := req["duration"].(int64)
	if !ok {
		return nil, fmt.Errorf("missing duration")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	user.IsVIP = true
	user.VIPExpireTime = time.Now().Add(time.Duration(duration) * time.Hour).Unix()

	return json.Marshal(map[string]interface{}{
		"status":      "success",
		"message":     "subscribed to VIP successfully",
		"expire_time": user.VIPExpireTime,
	})
}

// handleUnsubscribeVIP handles unsubscribing from VIP membership
func (s *McpService) handleUnsubscribeVIP(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	userID, ok := req["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	user.IsVIP = false
	user.VIPExpireTime = 0

	return json.Marshal(map[string]interface{}{
		"status":  "success",
		"message": "unsubscribed from VIP successfully",
	})
}

// handleCreateStoryPoint handles creating a new story point
func (s *McpService) handleCreateStoryPoint(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	storyID, ok := req["story_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story id")
	}

	description, ok := req["description"].(string)
	if !ok {
		return nil, fmt.Errorf("missing description")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	story, exists := s.stories[storyID]
	if !exists {
		return nil, fmt.Errorf("story not found")
	}

	storyPoint := &StoryPoint{
		ID:          uuid.New().String(),
		StoryID:     storyID,
		Description: description,
		Reward:      0,
		Status:      "open",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	s.storyPoints[storyPoint.ID] = storyPoint
	story.StoryPoints = append(story.StoryPoints, storyPoint.ID)

	return json.Marshal(map[string]interface{}{
		"status":      "success",
		"story_point": storyPoint,
	})
}

// handleSetStoryPointReward handles setting a reward for a story point
func (s *McpService) handleSetStoryPointReward(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	storyPointID, ok := req["story_point_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing story point id")
	}

	reward, ok := req["reward"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing reward amount")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	storyPoint, exists := s.storyPoints[storyPointID]
	if !exists {
		return nil, fmt.Errorf("story point not found")
	}

	storyPoint.Reward = reward
	storyPoint.UpdatedAt = time.Now().Unix()

	return json.Marshal(map[string]interface{}{
		"status":      "success",
		"story_point": storyPoint,
	})
}

// handleLikeStoryVersion handles liking a story version
func (s *McpService) handleLikeStoryVersion(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	versionID, ok := req["version_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing version id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	version, exists := s.storyVersions[versionID]
	if !exists {
		return nil, fmt.Errorf("story version not found")
	}

	version.Likes++
	return json.Marshal(map[string]interface{}{
		"status": "success",
		"likes":  version.Likes,
	})
}

// handleUnlikeStoryVersion handles unliking a story version
func (s *McpService) handleUnlikeStoryVersion(ctx context.Context, req map[string]interface{}) ([]byte, error) {
	versionID, ok := req["version_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing version id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	version, exists := s.storyVersions[versionID]
	if !exists {
		return nil, fmt.Errorf("story version not found")
	}

	if version.Likes > 0 {
		version.Likes--
	}

	return json.Marshal(map[string]interface{}{
		"status": "success",
		"likes":  version.Likes,
	})
}
