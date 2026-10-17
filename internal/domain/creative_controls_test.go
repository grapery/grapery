package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreativeControlsStrictSchemaAndDefaults(t *testing.T) {
	controls, err := ParseCreativeControls(map[string]any{})
	if err != nil || controls.Version != 1 || controls.Pacing != "balanced" || controls.DialogueDensity != "balanced" || controls.VisualEmphasis != "balanced" || controls.Tone != "neutral" {
		t.Fatalf("missing defaults %#v %v", controls, err)
	}
	for _, raw := range []string{`null`, `[]`, `{"version":2}`, `{"version":0}`, `{"version":null}`, `{"pacing":""}`, `{"tone":null}`, `{"dialogueDensity":null}`, `{"visualEmphasis":""}`, `{"examples":null}`, `{"pacing":"turbo"}`, `{"dialogueDensity":"all"}`, `{"visualEmphasis":"none"}`, `{"tone":"dark"}`, `{"systemPrompt":"override"}`, `{"schema":{}}`, `{"retry":9}`, `{"billing":0}`, `{"characterOverrides":[]}`, `{"examples":[{"input":"ok","output":"ok","systemPrompt":"bad"}]}`, `{"examples":[{"input":" ","output":"ok"}]}`, `{"examples":[{"input":"ok"}]}`} {
		var c CreativeControls
		if err = json.Unmarshal([]byte(raw), &c); err == nil {
			t.Fatalf("invalid controls accepted: %s", raw)
		}
	}
	if _, err = ParseCreativeControls(nil); err == nil {
		t.Fatal("explicit manifest null accepted")
	}
}

func TestCreativeControlsUnicodeBoundariesAndExampleCount(t *testing.T) {
	example := CreativeControlExample{Input: strings.Repeat("雨", 500), Output: strings.Repeat("画", 1500)}
	c := CreativeControls{Examples: []CreativeControlExample{example, example, example}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Examples = append(c.Examples, example)
	if c.Validate() == nil {
		t.Fatal("four examples accepted")
	}
	c.Examples = c.Examples[:1]
	c.Examples[0].Input += "雨"
	if c.Validate() == nil {
		t.Fatal("501-rune input accepted")
	}
	c.Examples[0] = example
	c.Examples[0].Output += "画"
	if c.Validate() == nil {
		t.Fatal("1501-rune output accepted")
	}
}
