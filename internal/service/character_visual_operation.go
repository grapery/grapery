package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func characterViewSheet(ch *domain.Character) string {
	if ch.Views != nil {
		return ch.Views.Sheet
	}
	return ""
}

// A client action freezes its provider input in the image ledger. Replaying a
// successful action must not adopt the very portrait/reference it just wrote.
func (s *Service) prepareCharacterVisualOperation(ctx context.Context, ch *domain.Character, kind, key string, choices any, req *GenerateImageRequest) error {
	if req.OperationID != "" {
		return nil
	} // Durable character task already owns this operation.
	key = strings.TrimSpace(key)
	if len(key) > 256 {
		return fmt.Errorf("clientRequestId exceeds 256 bytes")
	}
	raw, err := json.Marshal(struct {
		Choices any
		Inputs  string
	}{choices, domain.CharacterVisualInputSignature(ch)})
	if err != nil {
		return err
	}
	intent := fmt.Sprintf("%x", sha256.Sum256(raw))
	if key == "" {
		// Simultaneous legacy calls with the same frozen input still share an image
		// claim. A later intentional regenerate gets a new source-image identity.
		key = intent + ":" + fmt.Sprint(req.ReferenceImages)
	}
	req.OperationID = "character:" + ch.ID + ":" + kind + ":" + fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	recordID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(req.UserID+":image:"+req.OperationID)).String()
	previous, err := s.repo.GetAIGenerationRecord(ctx, recordID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	req.Metadata = cloneImageGenerationMetadata(req.Metadata)
	req.Metadata["characterVisualIntent"] = intent
	req.Metadata["characterVisualPortrait"] = ch.Portrait
	req.Metadata["characterVisualViews"] = characterViewSheet(ch)
	if previous == nil {
		return nil
	}
	if previous.UserID != req.UserID || previous.RelatedEntityID != ch.ID || previous.RelatedEntityType != "character" || previous.Metadata["characterVisualIntent"] != intent {
		return fmt.Errorf("%w: visual request key was already used for another intent", domain.ErrCharacterTaskRequestConflict)
	}
	var output struct {
		ImageURLs []string `json:"imageURLs"`
	}
	_ = json.Unmarshal([]byte(previous.OutputResult), &output)
	outputURL := ""
	if len(output.ImageURLs) > 0 {
		outputURL = output.ImageURLs[0]
	}
	oldPortrait, _ := previous.Metadata["characterVisualPortrait"].(string)
	oldViews, _ := previous.Metadata["characterVisualViews"].(string)
	portraitMatches := ch.Portrait == oldPortrait || (kind == "portrait" && outputURL != "" && ch.Portrait == outputURL)
	viewsMatches := characterViewSheet(ch) == oldViews || (kind == "threeViews" && outputURL != "" && characterViewSheet(ch) == outputURL)
	if !portraitMatches || !viewsMatches {
		return domain.ErrCharacterEditConflict
	}
	frozen := *req
	if err := json.Unmarshal([]byte(previous.InputParams), &frozen); err != nil {
		return fmt.Errorf("invalid frozen character visual request: %w", err)
	}
	frozen.Metadata = cloneImageGenerationMetadata(previous.Metadata)
	frozen.Metadata["characterVisualReplay"] = true
	*req = frozen
	return nil
}
