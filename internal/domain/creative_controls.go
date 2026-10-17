package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// CreativeControls v1 contains editorial preferences only. It cannot alter
// model routing, schemas, retries, billing, identity or confirmed story facts.
type CreativeControls struct {
	Version         int                      `json:"version,omitempty"`
	Pacing          string                   `json:"pacing,omitempty"`
	DialogueDensity string                   `json:"dialogueDensity,omitempty"`
	VisualEmphasis  string                   `json:"visualEmphasis,omitempty"`
	Tone            string                   `json:"tone,omitempty"`
	Examples        []CreativeControlExample `json:"examples,omitempty"`
}

type CreativeControlExample struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

func (c *CreativeControls) UnmarshalJSON(raw []byte) error {
	if value := bytes.TrimSpace(raw); len(value) == 0 || value[0] != '{' {
		return fmt.Errorf("creativeControls must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for key, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("creativeControls.%s cannot be null", key)
		}
		if key == "version" {
			var version int
			if err := json.Unmarshal(value, &version); err != nil || version != 1 {
				return fmt.Errorf("creativeControls.version must be 1")
			}
		}
		if key == "pacing" || key == "dialogueDensity" || key == "visualEmphasis" || key == "tone" {
			var text string
			if err := json.Unmarshal(value, &text); err != nil || text == "" {
				return fmt.Errorf("creativeControls.%s must be a nonempty enum", key)
			}
		}
	}
	type plain CreativeControls
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("creativeControls: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("creativeControls must contain one object")
	}
	*c = CreativeControls(value)
	return c.Validate()
}

func (c *CreativeControls) Validate() error {
	if c == nil {
		return nil
	}
	if c.Version != 0 && c.Version != 1 {
		return fmt.Errorf("creativeControls.version must be 1")
	}
	checks := []struct {
		name, value string
		allowed     []string
	}{
		{"pacing", c.Pacing, []string{"balanced", "slow", "fast"}},
		{"dialogueDensity", c.DialogueDensity, []string{"balanced", "sparse", "rich"}},
		{"visualEmphasis", c.VisualEmphasis, []string{"balanced", "atmosphere", "action", "character"}},
		{"tone", c.Tone, []string{"neutral", "warm", "tense", "playful"}},
	}
	for _, check := range checks {
		if check.value == "" {
			continue
		}
		valid := false
		for _, value := range check.allowed {
			if check.value == value {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("creativeControls.%s has unsupported value %q", check.name, check.value)
		}
	}
	if len(c.Examples) > 3 {
		return fmt.Errorf("creativeControls.examples allows at most 3 examples")
	}
	for i, example := range c.Examples {
		if strings.TrimSpace(example.Input) == "" || strings.TrimSpace(example.Output) == "" {
			return fmt.Errorf("creativeControls.examples[%d] requires nonempty input and output", i)
		}
		if !utf8.ValidString(example.Input) || !utf8.ValidString(example.Output) || utf8.RuneCountInString(example.Input) > 500 || utf8.RuneCountInString(example.Output) > 1500 {
			return fmt.Errorf("creativeControls.examples[%d] exceeds input/output limits", i)
		}
	}
	return nil
}

func (c *CreativeControls) Normalized() CreativeControls {
	value := CreativeControls{Version: 1, Pacing: "balanced", DialogueDensity: "balanced", VisualEmphasis: "balanced", Tone: "neutral"}
	if c == nil {
		return value
	}
	if c.Pacing != "" {
		value.Pacing = c.Pacing
	}
	if c.DialogueDensity != "" {
		value.DialogueDensity = c.DialogueDensity
	}
	if c.VisualEmphasis != "" {
		value.VisualEmphasis = c.VisualEmphasis
	}
	if c.Tone != "" {
		value.Tone = c.Tone
	}
	value.Examples = append([]CreativeControlExample(nil), c.Examples...)
	return value
}

// ParseCreativeControls handles explicitly present registry values. Missing keys
// are handled by the caller; an explicitly present null is rejected.
func ParseCreativeControls(value any) (*CreativeControls, error) {
	if value == nil {
		return nil, fmt.Errorf("creativeControls must be an object, not null")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("creativeControls: %w", err)
	}
	var controls CreativeControls
	if err = json.Unmarshal(raw, &controls); err != nil {
		return nil, err
	}
	normalized := controls.Normalized()
	return &normalized, nil
}

// ValidateCreativeControlsEnvelope distinguishes an omitted optional object from
// explicit null, which ordinary *CreativeControls JSON fields otherwise erase.
func ValidateCreativeControlsEnvelope(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	value, present := fields["creativeControls"]
	if !present {
		return nil
	}
	var controls CreativeControls
	return json.Unmarshal(value, &controls)
}
