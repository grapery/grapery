package mcps

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
)

func getStringField(m map[string]interface{}, key string) (string, error) {
	if value, ok := getStringFromMap(m, key); ok {
		if value == "" {
			return "", fmt.Errorf("%s cannot be empty", key)
		}
		return value, nil
	}
	return "", fmt.Errorf("missing %s", key)
}

func getStringFieldWithFallback(m map[string]interface{}, keys ...string) (string, error) {
	if value, ok := getStringOrDefaultRaw(m, keys, ""); ok && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("missing %s", strings.Join(keys, "/"))
}

func getStringOrDefault(m map[string]interface{}, keys []string, def string) string {
	if value, ok := getStringOrDefaultRaw(m, keys, def); ok {
		return value
	}
	return def
}

func getStringOrDefaultRaw(m map[string]interface{}, keys []string, def string) (string, bool) {
	for _, key := range keys {
		if value, ok := getStringFromMap(m, key); ok {
			return value, true
		}
	}
	return def, false
}

func getStringFromMap(m map[string]interface{}, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	if value, ok := m[key]; ok {
		return toString(value)
	}
	return "", false
}

func getInt64Field(m map[string]interface{}, key string) (int64, error) {
	if value, ok := getOptionalInt64(m, key); ok {
		return value, nil
	}
	return 0, fmt.Errorf("missing %s", key)
}

func getInt64FieldWithFallback(m map[string]interface{}, keys ...string) (int64, error) {
	for _, key := range keys {
		if value, ok := getOptionalInt64(m, key); ok {
			return value, nil
		}
	}
	return 0, fmt.Errorf("missing %s", strings.Join(keys, "/"))
}

func getOptionalInt64(m map[string]interface{}, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	raw, ok := m[key]
	if !ok {
		return 0, false
	}
	value, err := toInt64(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func getOptionalBool(m map[string]interface{}, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	raw, ok := m[key]
	if !ok {
		return false, false
	}
	switch v := raw.(type) {
	case bool:
		return v, true
	case string:
		if v == "" {
			return false, false
		}
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return false, false
		}
		return parsed, true
	case float64:
		return v != 0, true
	default:
		return false, false
	}
}

func getOptionalMap(m map[string]interface{}, key string) (map[string]interface{}, bool) {
	if m == nil {
		return nil, false
	}
	raw, ok := m[key]
	if !ok {
		return nil, false
	}
	switch v := raw.(type) {
	case map[string]interface{}:
		return v, true
	case nil:
		return nil, false
	default:
		return nil, false
	}
}

func toInt64Slice(raw interface{}) ([]int64, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case []interface{}:
		result := make([]int64, 0, len(v))
		for _, item := range v {
			parsed, err := toInt64(item)
			if err != nil {
				return nil, err
			}
			result = append(result, parsed)
		}
		return result, nil
	case []int64:
		return v, nil
	case []float64:
		result := make([]int64, 0, len(v))
		for _, item := range v {
			result = append(result, int64(item))
		}
		return result, nil
	case []string:
		result := make([]int64, 0, len(v))
		for _, item := range v {
			parsed, err := strconv.ParseInt(strings.TrimSpace(item), 10, 64)
			if err != nil {
				return nil, err
			}
			result = append(result, parsed)
		}
		return result, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		parts := strings.Split(v, ",")
		return toInt64Slice(parts)
	default:
		parsed, err := toInt64(v)
		if err != nil {
			return nil, err
		}
		return []int64{parsed}, nil
	}
}

func toStringSlice(raw interface{}) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case []string:
		return v, nil
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			str, ok := toString(item)
			if !ok {
				return nil, fmt.Errorf("cannot convert value to string: %v", item)
			}
			if str != "" {
				result = append(result, str)
			}
		}
		return result, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts, nil
	default:
		str, ok := toString(v)
		if !ok {
			return nil, fmt.Errorf("cannot convert value to string: %v", raw)
		}
		if str == "" {
			return nil, nil
		}
		return []string{str}, nil
	}
}

func toMapSlice(raw interface{}) ([]map[string]interface{}, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case []map[string]interface{}:
		return v, nil
	case []interface{}:
		result := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			switch itemVal := item.(type) {
			case map[string]interface{}:
				result = append(result, itemVal)
			case string:
				parsed := make(map[string]interface{})
				if err := json.Unmarshal([]byte(itemVal), &parsed); err != nil {
					return nil, err
				}
				result = append(result, parsed)
			default:
				return nil, fmt.Errorf("invalid role definition: %v", itemVal)
			}
		}
		return result, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		parsed := make([]map[string]interface{}, 0)
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("unsupported type for map slice: %T", raw)
	}
}

func toInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("invalid number")
		}
		return int64(v), nil
	case float32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, fmt.Errorf("empty string")
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", value)
	}
}

func toString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), true
	case fmt.Stringer:
		return strings.TrimSpace(v.String()), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case int, int32, int64:
		return fmt.Sprintf("%v", v), true
	case bool:
		return strconv.FormatBool(v), true
	default:
		return "", false
	}
}

func isNil(v interface{}) bool {
	switch v := v.(type) {
	case nil:
		return true
	default:
		return v == nil
	}
}

func convertAPIStoryToMCP(story *api.Story) *Story {
	if story == nil {
		return nil
	}
	summary := &Story{
		ID:          fmt.Sprintf("%d", story.GetId()),
		Title:       story.GetTitle(),
		Description: story.GetDesc(),
		Characters:  make([]string, 0),
		StoryPoints: make([]string, 0),
		Followers:   make([]string, 0),
		Likes:       int(story.GetLikeCount()),
		AuthorID:    fmt.Sprintf("%d", story.GetCreatorId()),
		CreatedAt:   story.GetCtime(),
		UpdatedAt:   story.GetMtime(),
	}
	if story.GetParams() != nil {
		desc := story.GetParams().GetStoryDescription()
		if desc != "" {
			summary.Content = desc
		}
		background := story.GetParams().GetBackground()
		if background != "" {
			summary.StoryPoints = append(summary.StoryPoints, background)
		}
	}
	if summary.Content == "" {
		summary.Content = summary.Description
	}
	return summary
}

func modelRoleToCharacter(role *models.StoryRole) *Character {
	if role == nil {
		return nil
	}
	character := &Character{
		ID:          fmt.Sprintf("%d", role.ID),
		Name:        role.CharacterName,
		Description: role.CharacterDescription,
		StoryID:     fmt.Sprintf("%d", role.StoryID),
		CreatedAt:   role.CreateAt.Unix(),
		UpdatedAt:   role.UpdateAt.Unix(),
		Followers:   make([]string, 0),
	}

	var detail api.CharacterDetail
	if err := json.Unmarshal([]byte(role.CharacterDetail), &detail); err == nil {
		character.Personality = detail.GetPersonality()
		if detail.GetBackground() != "" {
			character.Storyline = detail.GetBackground()
		}
	}

	return character
}

func mapToCharacterDetail(detail map[string]interface{}) *api.CharacterDetail {
	if len(detail) == 0 {
		return nil
	}
	cd := &api.CharacterDetail{}
	if value, ok := getStringFromMap(detail, "description"); ok {
		cd.Description = value
	}
	if value, ok := getStringFromMap(detail, "short_term_goal"); ok {
		cd.ShortTermGoal = value
	}
	if value, ok := getStringFromMap(detail, "long_term_goal"); ok {
		cd.LongTermGoal = value
	}
	if value, ok := getStringFromMap(detail, "personality"); ok {
		cd.Personality = value
	}
	if value, ok := getStringFromMap(detail, "background"); ok {
		cd.Background = value
	}
	if value, ok := getStringFromMap(detail, "handling_style"); ok {
		cd.HandlingStyle = value
	}
	if value, ok := getStringFromMap(detail, "cognition_range"); ok {
		cd.CognitionRange = value
	}
	if value, ok := getStringFromMap(detail, "ability_features"); ok {
		cd.AbilityFeatures = value
	}
	if value, ok := getStringFromMap(detail, "appearance"); ok {
		cd.Appearance = value
	}
	if value, ok := getStringFromMap(detail, "dress_preference"); ok {
		cd.DressPreference = value
	}

	if cd.Description == "" && cd.ShortTermGoal == "" && cd.LongTermGoal == "" && cd.Personality == "" && cd.Background == "" && cd.HandlingStyle == "" && cd.CognitionRange == "" && cd.AbilityFeatures == "" && cd.Appearance == "" && cd.DressPreference == "" {
		return nil
	}
	return cd
}

func mapToStoryBoardParams(m map[string]interface{}) *api.StoryBoardParams {
	if len(m) == 0 {
		return nil
	}
	params := &api.StoryBoardParams{}
	if value, ok := getStringFromMap(m, "style"); ok {
		params.Style = value
	}
	if value, ok := getStringFromMap(m, "negative_prompt"); ok {
		params.NegativePrompt = value
	}
	if value, ok := getStringFromMap(m, "ref_image"); ok {
		params.RefImage = value
	}
	if value, ok := getStringFromMap(m, "layout_style"); ok {
		params.LayoutStyle = value
	}
	if value, ok := getStringFromMap(m, "story_description"); ok {
		params.StoryDescription = value
	}
	if value, ok := getOptionalInt64(m, "scene_count"); ok {
		params.SceneCount = int32(value)
	}
	return params
}

func (s *McpService) resolveStoryboardRoles(ctx context.Context, storyID, creatorID int64, req map[string]interface{}) ([]*api.StoryRole, error) {
	roles := make([]*api.StoryRole, 0)

	if rawIDs, ok := req["role_ids"]; ok {
		roleIDs, err := toInt64Slice(rawIDs)
		if err != nil {
			return nil, fmt.Errorf("invalid role_ids: %w", err)
		}
		for _, roleID := range roleIDs {
			detailResp, err := s.storyEngine.GetStoryRoleDetail(ctx, &api.GetStoryRoleDetailRequest{RoleId: roleID})
			if err != nil {
				return nil, fmt.Errorf("failed to fetch role %d: %w", roleID, err)
			}
			if detailResp == nil || detailResp.GetInfo() == nil {
				return nil, fmt.Errorf("role %d not found", roleID)
			}
			roleInfo := detailResp.GetInfo()
			if err := ensureStoryboardRoleValid(roleInfo, storyID, creatorID); err != nil {
				return nil, err
			}
			roles = append(roles, roleInfo)
		}
	}

	if rawRoles, ok := req["roles"]; ok {
		roleMaps, err := toMapSlice(rawRoles)
		if err != nil {
			return nil, fmt.Errorf("invalid roles payload: %w", err)
		}
		for _, roleMap := range roleMaps {
			name, err := getStringFieldWithFallback(roleMap, "character_name", "name")
			if err != nil {
				return nil, err
			}
			desc, err := getStringFieldWithFallback(roleMap, "character_description", "description")
			if err != nil {
				return nil, err
			}
			avatar := getStringOrDefault(roleMap, []string{"character_avatar", "avatar"}, "")
			if avatar == "" {
				return nil, fmt.Errorf("role %s missing avatar", name)
			}

			role := &api.StoryRole{
				StoryId:              storyID,
				CreatorId:            creatorID,
				CharacterName:        name,
				CharacterDescription: desc,
				CharacterAvatar:      avatar,
				CharacterPrompt:      getStringOrDefault(roleMap, []string{"character_prompt", "prompt"}, ""),
				CharacterType:        getStringOrDefault(roleMap, []string{"character_type"}, ""),
			}

			if refImages, err := toStringSlice(roleMap["ref_images"]); err == nil && len(refImages) > 0 {
				role.CharacterRefImages = refImages
			}
			if id, ok := getOptionalInt64(roleMap, "role_id"); ok {
				role.RoleId = id
			} else if characterID, ok := getStringFromMap(roleMap, "role_id"); ok {
				if parsed, err := strconv.ParseInt(characterID, 10, 64); err == nil {
					role.RoleId = parsed
				}
			}
			if detailMap, ok := getOptionalMap(roleMap, "detail"); ok {
				if detail := mapToCharacterDetail(detailMap); detail != nil {
					role.CharacterDetail = detail
				}
			}
			if err := ensureStoryboardRoleValid(role, storyID, creatorID); err != nil {
				return nil, err
			}
			roles = append(roles, role)
		}
	}

	return roles, nil
}

func ensureStoryboardRoleValid(role *api.StoryRole, storyID, creatorID int64) error {
	if role == nil {
		return fmt.Errorf("role cannot be nil")
	}
	if role.GetCharacterAvatar() == "" {
		return fmt.Errorf("role %s missing avatar", role.GetCharacterName())
	}
	if role.GetCharacterDescription() == "" {
		return fmt.Errorf("role %s missing description", role.GetCharacterName())
	}
	if role.GetStoryId() == 0 {
		role.StoryId = storyID
	}
	if role.GetCreatorId() == 0 {
		role.CreatorId = creatorID
	}
	return nil
}
