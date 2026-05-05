package service

import (
	"encoding/json"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestFragmentTraceCharacterDisplayNames_PrefersBibleThenEvidence(t *testing.T) {
	raw := `{
		"visualBible": {
			"characters": [{"key": "char_0", "name": "", "immutableTraits": ["young man"]}],
			"sourceEvidence": [{
				"entities": [{"key": "char_0", "kind": "character", "name": "林间少年"}]
			}]
		}
	}`
	var trace domain.FragmentGenerationTrace
	if err := json.Unmarshal([]byte(raw), &trace); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	m := fragmentTraceCharacterDisplayNames(&trace)
	if got := m["char_0"]; got != "林间少年" {
		t.Fatalf("expected evidence name for char_0, got %q", got)
	}
}

func TestDisplayNameFromStableCharacterKey_AsTitleCaseSlug(t *testing.T) {
	if got := displayNameFromStableCharacterKey("char-main"); got != "Char Main" {
		t.Fatalf("expected Char Main, got %q", got)
	}
}
