package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// StoryboardRevisionBaseline is stored in the generation run before any model
// call. It is the recoverable accepted result, not a model-written summary.
type StoryboardRevisionBaseline struct {
	Revision       string            `json:"revision"`
	Title          string            `json:"title"`
	Content        string            `json:"content"`
	RawInput       string            `json:"rawInput"`
	Scenes         []StoryboardScene `json:"scenes"`
	TargetSceneIDs []string          `json:"targetSceneIds,omitempty"`
}

// StoryboardContentRevision deliberately excludes engagement counters and wall
// clock timestamps. A view or a summary refresh cannot invalidate an edit.
func StoryboardContentRevision(board *Storyboard, scenes []StoryboardScene) string {
	if board == nil {
		return ""
	}
	ordered := append([]StoryboardScene(nil), scenes...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Sequence == ordered[j].Sequence {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Sequence < ordered[j].Sequence
	})
	// Linked story locations are mutable enrichment; only their stable IDs belong
	// in a draft's revision. Generated media is included to reject stale rewrites.
	for i := range ordered {
		ordered[i].StoryScene = nil
		ordered[i].UpdatedAt = 0
		ordered[i].CreatedAt = 0
	}
	payload, _ := json.Marshal(struct {
		ID, Title, Content, RawInput string
		Scenes                       []StoryboardScene
	}{board.ID, board.Title, board.Content, board.RawInput, ordered})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// StoryboardRevisionHistory exposes recoverable accepted data only; system
// prompts, alignment preferences and private model traces are never returned.
type StoryboardRevisionHistory struct {
	GenerationRunID string                     `json:"generationRunId"`
	Status          string                     `json:"status"`
	CreatedAt       int64                      `json:"createdAt"`
	Baseline        StoryboardRevisionBaseline `json:"acceptedBaseline"`
}
