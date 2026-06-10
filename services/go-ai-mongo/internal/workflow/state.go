package workflow

const (
	LoopCap     = 5
	MaxMessages = 20
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AttachmentSummary struct {
	DocumentType     string   `json:"document_type"`
	KeyFindings      []string `json:"key_findings"`
	Values           []string `json:"values"`
	Dates            []string `json:"dates"`
	RelevantToTriage string   `json:"relevant_to_triage"`
	Limitations      string   `json:"limitations"`
	SourceFile       string   `json:"_source_file"`
	RawDescription   string   `json:"_raw_description"`
}

type Demographics struct {
	Age *float64 `json:"age"`
	Sex *string  `json:"sex"`
}

type Vitals struct {
	HR   *float64 `json:"hr"`
	RR   *float64 `json:"rr"`
	SpO2 *float64 `json:"spo2"`
	Temp *float64 `json:"temp"`
	BP   *string  `json:"bp"`
}

type State struct {
	ChiefComplaint string              `json:"chief_complaint"`
	Symptoms       []string            `json:"symptoms"`
	Onset          string              `json:"onset"`
	Severity       string              `json:"severity"`
	Location       string              `json:"location"`
	Modifiers      []string            `json:"modifiers"`
	Demographics   Demographics        `json:"demographics"`
	Vitals         Vitals              `json:"vitals"`
	RiskFactors    []string            `json:"risk_factors"`
	RedFlags       []string            `json:"red_flags"`
	Attachments    []AttachmentSummary `json:"attachments_summary"`
	TurnCount      int                 `json:"turn_count"`
	Messages       []Message           `json:"messages"`
}

func NewState() State {
	return State{
		Symptoms:    []string{},
		Modifiers:   []string{},
		RiskFactors: []string{},
		RedFlags:    []string{},
		Attachments: []AttachmentSummary{},
		Messages:    []Message{},
	}
}

func TrimMessages(state *State) {
	if len(state.Messages) > MaxMessages {
		state.Messages = state.Messages[len(state.Messages)-MaxMessages:]
	}
}
