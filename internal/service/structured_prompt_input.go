package service

import (
	"encoding/json"
	"strings"
)

type PromptDSLSection struct {
	Title   string `json:"title"`
	Kind    string `json:"kind"` // text | json
	Body    string `json:"body,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

type PromptDSL struct {
	Role           string             `json:"role"`
	Task           string             `json:"task"`
	Inputs         any                `json:"inputs,omitempty"`
	GlobalConfig   any                `json:"globalConfig,omitempty"`
	OutputContract string             `json:"outputContract,omitempty"`
	Sections       []PromptDSLSection `json:"sections,omitempty"`
}

func promptJSONBlock(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

func promptSection(builder *strings.Builder, title, body string) {
	if builder == nil {
		return
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	builder.WriteString("\n## ")
	builder.WriteString(title)
	builder.WriteString("\n")
	if strings.TrimSpace(body) != "" {
		builder.WriteString(strings.TrimSpace(body))
		builder.WriteString("\n")
	}
}

func promptJSONSection(builder *strings.Builder, title string, payload any) {
	promptSection(builder, title, promptJSONBlock(payload))
}

func renderPromptDSL(dsl PromptDSL) string {
	var b strings.Builder
	b.WriteString("# Role\n")
	b.WriteString(strings.TrimSpace(dsl.Role))
	b.WriteString("\n")
	promptSection(&b, "Task", dsl.Task)
	if dsl.Inputs != nil {
		promptJSONSection(&b, "Inputs", dsl.Inputs)
	}
	if dsl.GlobalConfig != nil {
		switch v := dsl.GlobalConfig.(type) {
		case string:
			promptSection(&b, "Global Visual Config", v)
		default:
			promptJSONSection(&b, "Global Visual Config", v)
		}
	}
	for _, sec := range dsl.Sections {
		title := strings.TrimSpace(sec.Title)
		if title == "" {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(sec.Kind))
		if kind == "json" {
			promptJSONSection(&b, title, sec.Payload)
			continue
		}
		body := strings.TrimSpace(sec.Body)
		if body == "" && sec.Payload != nil {
			body = promptJSONBlock(sec.Payload)
		}
		promptSection(&b, title, body)
	}
	if strings.TrimSpace(dsl.OutputContract) != "" {
		promptSection(&b, "Output Contract", dsl.OutputContract)
	}
	return strings.TrimSpace(b.String())
}
