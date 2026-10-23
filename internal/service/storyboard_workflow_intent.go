package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// This versioned object records the original request choices separately from
// the effective settings/baseline recorded by the generation run. In particular,
// omitted baseRevision must stay omitted across a retry after content changed.
type storyboardWorkflowIntent struct {
	Version             int                         `json:"version"`
	RawInput            string                      `json:"rawInput"`
	StoryID             string                      `json:"storyId"`
	ParentID            string                      `json:"parentId"`
	IsStandalone        bool                        `json:"isStandalone"`
	ComicPage           bool                        `json:"useComicPagePipeline"`
	RegenerateStructure bool                        `json:"regenerateStructure"`
	UserDirective       string                      `json:"userDirective"`
	SceneCount          int                         `json:"sceneCount"`
	ComicStyle          string                      `json:"comicStyle"`
	Language            string                      `json:"language"`
	TargetSceneIDs      []string                    `json:"targetSceneIds"`
	BaseRevision        string                      `json:"baseRevision"`
	CreativeControls    domain.CreativeControls     `json:"creativeControls"`
	CharacterRefs       []storyboardIntentCharacter `json:"characterRefs"`
	SceneRefs           []storyboardIntentLocation  `json:"sceneRefs"`
}

type storyboardIntentCharacter struct {
	ID    string `json:"id"`
	Role  string `json:"role"`
	Order int    `json:"order"`
	Notes string `json:"notes"`
}
type storyboardIntentLocation struct {
	ID       string `json:"id"`
	Sequence int    `json:"sequence"`
	Primary  bool   `json:"primary"`
}

type storyboardWorkflowRequestFinder interface {
	FindStoryboardGenerationRunsByClientRequest(context.Context, string, string, string) ([]*domain.StoryboardGenerationRun, error)
}

func normalizedStoryboardTargets(ids []string) []string {
	set := map[string]bool{}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			set[id] = true
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
func newStoryboardWorkflowIntent(board *domain.Storyboard, opts StoryboardWorkflowStageOptions, controls *domain.CreativeControls) storyboardWorkflowIntent {
	language := strings.TrimSpace(opts.Language)
	if language == "" {
		language = "zh-Hans"
	}
	count := opts.SceneCount
	if count == 0 {
		count = board.SceneCount
	}
	if count < MinStoryboardSceneCount || count > MaxStoryboardSceneCount {
		count = DefaultStoryboardSceneCount
	}
	style := strings.TrimSpace(opts.ComicStyle)
	if style == "" {
		style = strings.TrimSpace(board.ContinuationComicStyle)
	}
	characters := make([]storyboardIntentCharacter, 0, len(board.CharacterRefs))
	for _, ref := range board.CharacterRefs {
		characters = append(characters, storyboardIntentCharacter{ID: ref.CharacterID, Role: ref.Role, Order: ref.Order, Notes: ref.Notes})
	}
	sort.Slice(characters, func(i, j int) bool {
		a, _ := json.Marshal(characters[i])
		b, _ := json.Marshal(characters[j])
		return string(a) < string(b)
	})
	locations := make([]storyboardIntentLocation, 0, len(board.SceneRefs))
	for _, ref := range board.SceneRefs {
		locations = append(locations, storyboardIntentLocation{ID: ref.StorySceneID, Sequence: ref.Sequence, Primary: ref.IsPrimaryScene})
	}
	sort.Slice(locations, func(i, j int) bool {
		a, _ := json.Marshal(locations[i])
		b, _ := json.Marshal(locations[j])
		return string(a) < string(b)
	})
	return storyboardWorkflowIntent{Version: 1, RawInput: strings.TrimSpace(board.RawInput), StoryID: board.StoryID, ParentID: board.ParentID, IsStandalone: board.IsStandalone, ComicPage: board.UseComicPagePipeline, RegenerateStructure: opts.RegenerateStructure, UserDirective: strings.TrimSpace(opts.UserDirective), SceneCount: count, ComicStyle: style, Language: language, TargetSceneIDs: normalizedStoryboardTargets(opts.TargetSceneIDs), BaseRevision: strings.TrimSpace(opts.BaseRevision), CreativeControls: controls.Normalized(), CharacterRefs: characters, SceneRefs: locations}

}

// ErrStoryboardRequestIntentConflict distinguishes an idempotency conflict from
// ordinary duplicate-entry business responses at the HTTP boundary.
var ErrStoryboardRequestIntentConflict = errors.New("storyboard_request_intent_conflict")

func storyboardIntentConflict(detail string) error {
	return fmt.Errorf("%w: %w (%s); reuse the original request or start a new clientRequestId", domain.ErrAlreadyExists, ErrStoryboardRequestIntentConflict, detail)
}

// A generationRunId-only continuation uses the frozen run, not live board
// settings. If the caller supplies creation/revision choices they must agree.
func validateStoryboardWorkflowRunIntent(run *domain.StoryboardGenerationRun, board *domain.Storyboard, opts StoryboardWorkflowStageOptions, full bool) error {
	var request storyboardWorkflowRunRequest
	if run == nil || json.Unmarshal([]byte(run.RequestJSON), &request) != nil {
		return storyboardIntentConflict("invalid persisted request")
	}
	if id := strings.TrimSpace(opts.ClientRequestID); (id != "" || opts.ProvidedFields["clientRequestId"]) && id != strings.TrimSpace(request.ClientRequestID) {
		return storyboardIntentConflict("clientRequestId")
	}
	if opts.CreativeControls != nil && !sameCreativeControls(opts.CreativeControls, request.CreativeControls) {
		return storyboardIntentConflict("creativeControls")
	}
	if request.RequestIntent == nil {
		return validateLegacyStoryboardWorkflowIntent(run, board, opts, full)
	}
	frozen := *request.RequestIntent
	if frozen.Version != 1 {
		return storyboardIntentConflict("unsupported request intent version")
	}
	if !full {
		if (opts.RegenerateStructure || opts.ProvidedFields["regenerateStructure"]) && opts.RegenerateStructure != frozen.RegenerateStructure {
			return storyboardIntentConflict("regenerateStructure")
		}
		if (opts.UserDirective != "" || opts.ProvidedFields["userDirective"]) && strings.TrimSpace(opts.UserDirective) != frozen.UserDirective {
			return storyboardIntentConflict("userDirective")
		}
		if (opts.SceneCount != 0 || opts.ProvidedFields["sceneCount"]) && opts.SceneCount != frozen.SceneCount {
			return storyboardIntentConflict("sceneCount")
		}
		if (opts.ComicStyle != "" || opts.ProvidedFields["comicStyle"]) && strings.TrimSpace(opts.ComicStyle) != frozen.ComicStyle {
			return storyboardIntentConflict("comicStyle")
		}
		if (opts.Language != "" || opts.ProvidedFields["language"]) && strings.TrimSpace(opts.Language) != frozen.Language {
			return storyboardIntentConflict("language")
		}
		if (opts.TargetSceneIDs != nil || opts.ProvidedFields["targetSceneIds"]) && !reflect.DeepEqual(normalizedStoryboardTargets(opts.TargetSceneIDs), frozen.TargetSceneIDs) {
			return storyboardIntentConflict("targetSceneIds")
		}
		if (opts.BaseRevision != "" || opts.ProvidedFields["baseRevision"]) && strings.TrimSpace(opts.BaseRevision) != frozen.BaseRevision {
			return storyboardIntentConflict("baseRevision")
		}
		return nil
	}
	// Missing controls on a legitimate retry mean use the pinned run's controls.
	controls := request.CreativeControls
	if opts.CreativeControls != nil {
		controls = opts.CreativeControls
	}
	// Resolve omitted count/style against this run's frozen choices, not a
	// newer edit of the mutable draft. An explicit different value conflicts.
	if opts.SceneCount == 0 {
		opts.SceneCount = frozen.SceneCount
	}
	if strings.TrimSpace(opts.ComicStyle) == "" {
		opts.ComicStyle = frozen.ComicStyle
	}
	incoming := newStoryboardWorkflowIntent(board, opts, controls)
	if !reflect.DeepEqual(frozen, incoming) {
		return storyboardIntentConflict("request fields differ")
	}
	return nil
}

func validateLegacyStoryboardWorkflowIntent(run *domain.StoryboardGenerationRun, board *domain.Storyboard, opts StoryboardWorkflowStageOptions, full bool) error {
	// Old records never preserved the original optional choices. A bare pinned
	// stage continuation is safe; a keyed replay cannot prove omission/defaults.
	if full {
		return storyboardIntentConflict("legacy record has no complete request fingerprint")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(run.RequestJSON), &fields); err != nil {
		return storyboardIntentConflict("invalid legacy request")
	}
	request := storyboardWorkflowRunRequestFromRun(run)
	checks := map[string]any{}
	if opts.UserDirective != "" || opts.ProvidedFields["userDirective"] {
		checks["userDirective"] = strings.TrimSpace(opts.UserDirective)
	}
	if opts.SceneCount != 0 || opts.ProvidedFields["sceneCount"] {
		checks["sceneCount"] = opts.SceneCount
	}
	if opts.ComicStyle != "" || opts.ProvidedFields["comicStyle"] {
		checks["comicStyle"] = strings.TrimSpace(opts.ComicStyle)
	}
	if opts.Language != "" || opts.ProvidedFields["language"] {
		checks["language"] = strings.TrimSpace(opts.Language)
	}
	if opts.TargetSceneIDs != nil || opts.ProvidedFields["targetSceneIds"] {
		checks["targetSceneIds"] = normalizedStoryboardTargets(opts.TargetSceneIDs)
	}
	if opts.BaseRevision != "" || opts.ProvidedFields["baseRevision"] {
		checks["baseRevision"] = strings.TrimSpace(opts.BaseRevision)
	}
	if (opts.RegenerateStructure || opts.ProvidedFields["regenerateStructure"]) && opts.RegenerateStructure != request.RegenerateStructure {
		return storyboardIntentConflict("regenerateStructure")
	}
	for key, expected := range checks {
		raw, exists := fields[key]
		if !exists {
			return storyboardIntentConflict("legacy record lacks " + key)
		}
		if key == "targetSceneIds" {
			var ids []string
			if json.Unmarshal(raw, &ids) != nil || !reflect.DeepEqual(normalizedStoryboardTargets(ids), expected) {
				return storyboardIntentConflict(key)
			}
			continue
		}
		value, _ := json.Marshal(expected)
		if string(value) != string(raw) {
			return storyboardIntentConflict(key)
		}
	}
	return nil
}
