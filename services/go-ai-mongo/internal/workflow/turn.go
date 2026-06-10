package workflow

type ExtractedState struct {
	ChiefComplaint string   `json:"chief_complaint"`
	Symptoms       []string `json:"symptoms"`
	Onset          string   `json:"onset"`
	Severity       string   `json:"severity"`
	Location       string   `json:"location"`
	Modifiers      []string `json:"modifiers"`
}

func MergeState(state *State, ext ExtractedState) {
	if ext.ChiefComplaint != "" {
		state.ChiefComplaint = ext.ChiefComplaint
	}
	if ext.Onset != "" {
		state.Onset = ext.Onset
	}
	if ext.Severity != "" {
		state.Severity = ext.Severity
	}
	if ext.Location != "" {
		state.Location = ext.Location
	}
	state.Symptoms = unionSorted(state.Symptoms, ext.Symptoms)
	state.Modifiers = unionSorted(state.Modifiers, ext.Modifiers)
}

func AppendUserMessage(state *State, content string) {
	state.Messages = append(state.Messages, Message{Role: "user", Content: content})
	TrimMessages(state)
}
