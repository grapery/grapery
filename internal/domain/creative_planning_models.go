package domain

// CreativeEditPlan turns a conversational request into an executable, reviewable plan.
// TargetIndexes are 1-based so they map directly to the labels shown in Voyager.
type CreativeEditPlan struct {
	Operation                  string   `json:"operation"`
	TargetIndexes              []int    `json:"targetIndexes,omitempty"`
	RequestedChanges           []string `json:"requestedChanges,omitempty"`
	Preserve                   []string `json:"preserve,omitempty"`
	NeedsClarification         bool     `json:"needsClarification"`
	ClarificationQuestion      string   `json:"clarificationQuestion,omitempty"`
	EstimatedRegenerationCount int      `json:"estimatedRegenerationCount,omitempty"`
}
