// Package triage ports the immutable notebook dialogue loop (Cell 14) to Go:
// M0 attachments -> M1 scope -> M2 extract/merge -> UMLS -> red-flag fast path ->
// M3 sufficiency -> M4 clarify -> BERT -> M5 triage -> escalate-only reconcile ->
// M6 patient response -> FHIR. Persona/profile extraction is intentionally absent
// from clinical state; approved profile context is passed as a bounded prompt.
package triage

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"orsa.ai/go-ai-mongo/internal/llm"
	"orsa.ai/go-ai-mongo/internal/umls"
)

const (
	loopCap     = 5
	maxMessages = 20
)

// Message is one chat turn stored on the thread.
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func newMessage(role, content string) Message {
	return Message{Role: role, Content: content, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
}

// State is the notebook dialogue state. JSON tags match the notebook keys so the
// stored thread JSON is identical to the reference implementation.
type State struct {
	ChiefComplaint      string           `json:"chief_complaint"`
	Symptoms            []string         `json:"symptoms"`
	Onset               string           `json:"onset"`
	Severity            string           `json:"severity"`
	Location            string           `json:"location"`
	Modifiers           []string         `json:"modifiers"`
	Demographics        Demographics     `json:"demographics"`
	Vitals              Vitals           `json:"vitals"`
	RiskFactors         []string         `json:"risk_factors"`
	RedFlags            []string         `json:"red_flags"`
	AttachmentsSummary  []map[string]any `json:"attachments_summary"`
	ReportReviewActive  bool             `json:"report_review_active,omitempty"`
	GeneralHealthActive bool             `json:"general_health_active,omitempty"`
	TurnCount           int              `json:"turn_count"`
	Messages            []Message        `json:"messages"`
}

type Demographics struct {
	Age *int    `json:"age"`
	Sex *string `json:"sex"`
}

type Vitals struct {
	HR   *float64 `json:"hr"`
	RR   *float64 `json:"rr"`
	SpO2 *float64 `json:"spo2"`
	Temp *float64 `json:"temp"`
	BP   *string  `json:"bp"`
}

// ProfileContext is user-approved personalization context supplied per thread.
// It must never become clinical evidence or reduce triage urgency.
type ProfileContext struct {
	ConsentStatus    string `json:"consentStatus,omitempty"`
	PersonaSummary   string `json:"personaSummary,omitempty"`
	WorkflowBoundary string `json:"workflowBoundary,omitempty"`
	BoundaryPrompt   string `json:"boundaryPrompt,omitempty"`
}

func NewState() State {
	return State{
		Symptoms:           []string{},
		Modifiers:          []string{},
		RiskFactors:        []string{},
		RedFlags:           []string{},
		AttachmentsSummary: []map[string]any{},
		Messages:           []Message{},
	}
}

// BertPredictor is the specialist ESI model signal. Implementations live in the
// bert package; a nil predictor yields the safe escalate-only default (ESI-5).
type BertPredictor interface {
	Predict(text string) (level int, confidence float64)
}

// ReconcileResult is the escalate-only reconciliation outcome.
type ReconcileResult struct {
	FinalESI            int  `json:"final_esi"`
	BertESI             int  `json:"bert_esi"`
	GptESI              int  `json:"gpt_esi"`
	RedFlagFloor        *int `json:"red_flag_floor"`
	Disagreement        bool `json:"disagreement"`
	GptWantedDeescalate bool `json:"gpt_wanted_deescalate"`
}

// Result is the outcome of one pipeline turn.
type Result struct {
	Type         string           `json:"type"` // refusal | clarify | triage | report_review
	Text         string           `json:"text"`
	Question     string           `json:"question,omitempty"`
	Reconcile    *ReconcileResult `json:"reconcile,omitempty"`
	Differential []any            `json:"differential,omitempty"`
	FHIRBundle   string           `json:"fhir_bundle_json,omitempty"`
	Warnings     []string         `json:"warnings,omitempty"`
}

// Engine runs the dialogue loop using the GPT-OSS client and an optional BERT model.
type Engine struct {
	llm  *llm.Client
	bert BertPredictor
	umls *umls.Client
}

func NewEngine(client *llm.Client, bert BertPredictor, umlsClient *umls.Client) *Engine {
	return &Engine{llm: client, bert: bert, umls: umlsClient}
}

// RunTurn executes one full pipeline turn, mutating state in place.
// attachments is the optional list of attachment refs from the chat request;
// each entry may carry a "summary" field already produced by M0 vision analysis.
func (e *Engine) RunTurn(ctx context.Context, state *State, userInput string, attachments []map[string]any, profile ProfileContext) Result {
	// M0 - merge any pre-analysed attachment summaries into state.
	rememberAttachmentRefs(state, attachments)

	previousAssistant := lastAssistantMessage(state)
	state.Messages = append(state.Messages, newMessage("user", userInput))
	var warnings []string

	// M1 - scope/safety gate.
	// Only applied on the very first turn (TurnCount == 0). Follow-up messages
	// within an established dialogue are always treated as in-scope: short
	// contextual replies such as "70", "yes", or "since this morning" carry no
	// medical framing on their own and would be incorrectly refused if re-gated.
	if state.TurnCount == 0 {
		scope := e.m1Scope(ctx, userInput)
		inScope, _ := scope["in_scope"].(bool)
		intent, _ := scope["intent"].(string)

		if inScope {
			// Route to the conversational general-health assistant when the model
			// labels the intent as general_health, OR when the opening message reads
			// like a general health / nutrition / body-science question rather than a
			// symptom complaint. The second clause keeps general Q&A working even when
			// the LLM is unavailable (M1 falls back to intent="triage") or mislabels a
			// borderline educational question.
			if intent == "general_health" || hasGeneralHealthIntent(userInput) {
				state.GeneralHealthActive = true
			}
		} else {
			text, _ := scope["refusal_reason"].(string)
			if strings.TrimSpace(text) == "" {
				text = "I can only help with health/triage concerns."
			}
			state.Messages = append(state.Messages, newMessage("assistant", text))
			trim(state)
			return Result{Type: "refusal", Text: text}
		}
	}

	if shouldRunReportReview(userInput, attachments, state) {
		state.ReportReviewActive = true
		reply := e.reportReview(ctx, userInput, reportAttachmentPayload(attachments, state), recentMessages(state), profile)
		state.Messages = append(state.Messages, newMessage("assistant", reply))
		state.TurnCount++
		trim(state)
		if !e.llm.Available() {
			warnings = append(warnings, "GPT-OSS token not configured; using safe fallback responses.")
		}
		return Result{Type: "report_review", Text: reply, Warnings: warnings}
	}

	if shouldRunGeneralHealth(userInput, attachments, state) {
		state.GeneralHealthActive = true
		reply := e.generalHealth(ctx, userInput, recentMessages(state), profile)
		state.Messages = append(state.Messages, newMessage("assistant", reply))
		state.TurnCount++
		trim(state)
		if !e.llm.Available() {
			warnings = append(warnings, "GPT-OSS token not configured; using safe fallback responses.")
		}
		return Result{Type: "general_health", Text: reply, Warnings: warnings}
	}

	// M2 - extract + merge state.
	ext := e.m2Extract(ctx, userInput, clinicalState(state))
	mergeState(state, ext)
	applyDeterministicClinicalExtraction(state, userInput, previousAssistant)

	// Red-flag fast path.
	flags := detectRedFlags(userInput, state)
	state.RedFlags = sortedUnion(state.RedFlags, flags)

	// M3/M4 - sufficiency gate, skipped when a red flag is present.
	if len(flags) == 0 {
		suff := e.m3Sufficiency(ctx, clinicalState(state))
		sufficient, _ := suff["sufficient"].(bool)
		missing := filterAnsweredMissing(suff["missing"], state)
		if len(missing) == 0 && hasMinimumTriageContext(state) {
			sufficient = true
		}
		if state.TurnCount >= 1 && hasCoreTriageContext(state) {
			sufficient = true
		}
		if !sufficient && state.TurnCount < loopCap {
			state.TurnCount++
			question := e.m4Clarify(ctx, missing, userInput, profile)
			state.Messages = append(state.Messages, newMessage("assistant", question))
			trim(state)
			return Result{Type: "clarify", Text: question, Question: question}
		}
	}

	// BERT + UMLS concept lookup + M5 + escalate-only reconcile.
	triageText := firstNonEmpty(state.ChiefComplaint, userInput, "unspecified")
	bertLevel, bertConf := e.predictBert(triageText)

	// UMLS: encode extracted symptoms to SNOMED CT CUIs for M5 enrichment.
	umlsConcepts := e.lookupUMLS(ctx, state.Symptoms)
	m5 := e.m5Triage(ctx, clinicalState(state), bertLevel, bertConf, recentMessages(state), umlsConcepts, profile)

	gptLevel := asInt(m5["esi_level"], bertLevel)
	gptLevel = clampESI(gptLevel)

	var rfFloor *int
	if len(flags) > 0 {
		two := 2
		rfFloor = &two
	} else {
		rfFloor = dangerZoneVitals(state.Vitals, state.Demographics.Age)
	}
	rec := reconcile(bertLevel, gptLevel, rfFloor)

	dangerousMimic, _ := m5["dangerous_mimic"].(bool)
	confidence := asString(m5["confidence"], "medium")
	likely := m5["likely_condition"]
	differential := asSlice(m5["differential"])
	specialty := asString(m5["recommended_specialty"], "")
	if strings.TrimSpace(specialty) == "" {
		specialty = defaultSpecialty(rec.FinalESI)
	}

	actions := []string{"Monitor symptoms"}
	if rec.FinalESI <= 2 {
		actions = append(actions, "Seek emergency care now")
	}
	allowCondition := likely != nil && !dangerousMimic

	reply := e.m6Respond(ctx, m6Input{
		state:            clinicalState(state),
		finalESI:         rec.FinalESI,
		confidence:       confidence,
		modelDisagree:    rec.Disagreement,
		dangerousMimic:   dangerousMimic,
		allowCondition:   allowCondition,
		likelyCondition:  likely,
		differential:     differential,
		recommendedSpec:  specialty,
		recommendActions: actions,
		messages:         recentMessages(state),
		profile:          profile,
	})
	state.Messages = append(state.Messages, newMessage("assistant", reply))

	fhir := buildFHIRBundle(state, rec.FinalESI, asString(m5["rationale"], ""), umlsConcepts, likely, specialty)
	state.TurnCount++
	trim(state)

	if !e.llm.Available() {
		warnings = append(warnings, "GPT-OSS token not configured; using safe fallback responses.")
	}

	return Result{
		Type:         "triage",
		Text:         reply,
		Reconcile:    &rec,
		Differential: differential,
		FHIRBundle:   fhir,
		Warnings:     warnings,
	}
}

func (e *Engine) predictBert(text string) (int, float64) {
	if e.bert == nil {
		return 5, 0.0 // escalate-only safe default
	}
	level, conf := e.bert.Predict(text)
	return clampESI(level), conf
}

// ---- M-step wrappers ----

func (e *Engine) m1Scope(ctx context.Context, text string) map[string]any {
	return e.llm.JSON(ctx, m1System, "User message:\n"+text, 0.1, 1500,
		map[string]any{"in_scope": true, "intent": "triage", "refusal_reason": nil})
}

func (e *Engine) m2Extract(ctx context.Context, text string, prior map[string]any) map[string]any {
	user := "Existing structured state:\n" + mustJSON(prior) + "\n\nNew patient message:\n" + text
	return e.llm.JSON(ctx, m2System, user, 0.1, 1500, map[string]any{
		"symptoms": []any{text}, "onset": "", "severity": "", "location": "",
		"modifiers": []any{}, "chief_complaint": text,
		"demographics": map[string]any{"age": nil, "sex": nil},
		"vitals":       map[string]any{"hr": nil, "rr": nil, "spo2": nil, "temp": nil, "bp": ""},
		"risk_factors": []any{},
	})
}

func (e *Engine) m3Sufficiency(ctx context.Context, state map[string]any) map[string]any {
	return e.llm.JSON(ctx, m3System, "Structured patient state:\n"+mustJSON(state), 0.1, 1500,
		map[string]any{"sufficient": true, "missing": []any{}})
}

func (e *Engine) m4Clarify(ctx context.Context, missing any, patientText string, profile ProfileContext) string {
	user := "Missing information:\n" + mustJSON(missing) + "\n\nPatient's latest message:\n" + patientText
	out := e.llm.JSON(ctx, systemWithProfileBoundary(m4System, profile), user, 0.1, 1500,
		map[string]any{"question": "Can you tell me more about your main symptom and when it started?"})
	if q, ok := out["question"].(string); ok && strings.TrimSpace(q) != "" {
		return q
	}
	return "Can you tell me more about your symptoms?"
}

func (e *Engine) reportReview(ctx context.Context, patientText string, attachments []map[string]any, messages []Message, profile ProfileContext) string {
	user := mustJSON(map[string]any{
		"patient_message":      patientText,
		"attachments":          attachments,
		"conversation_history": messages,
		"profile_context":      profile.llmPayload(),
	})
	reply := e.llm.Text(ctx, systemWithProfileBoundary(reportReviewSystem, profile), user, 0.3, 2500)
	if strings.TrimSpace(reply) == "" || strings.HasPrefix(reply, "[GPT-OSS") {
		return fallbackReportReviewReply(patientText, attachments)
	}
	return reply
}

func (e *Engine) generalHealth(ctx context.Context, patientText string, messages []Message, profile ProfileContext) string {
	user := mustJSON(map[string]any{
		"patient_message":      patientText,
		"conversation_history": messages,
		"profile_context":      profile.llmPayload(),
		"response_language":    preferredResponseLanguage(patientText, messages),
	})
	reply := e.llm.Text(ctx, systemWithProfileBoundary(generalHealthSystem, profile), user, 0.4, 3200)
	if strings.TrimSpace(reply) == "" || strings.HasPrefix(reply, "[GPT-OSS") {
		return fallbackGeneralHealthReply(patientText, messages)
	}
	return reply
}

func (e *Engine) lookupUMLS(ctx context.Context, symptoms []string) []umls.Concept {
	if !e.umls.Available() || len(symptoms) == 0 {
		return nil
	}
	// Limit to the first 5 symptoms to stay within rate limits.
	terms := symptoms
	if len(terms) > 5 {
		terms = terms[:5]
	}
	return e.umls.SearchMany(ctx, terms)
}

func (e *Engine) m5Triage(ctx context.Context, state map[string]any, bertLevel int, bertConf float64, messages []Message, concepts []umls.Concept, profile ProfileContext) map[string]any {
	// Serialize UMLS concepts into a compact form for the LLM.
	umlsPayload := make([]map[string]any, 0, len(concepts))
	for _, c := range concepts {
		umlsPayload = append(umlsPayload, map[string]any{
			"cui": c.CUI, "name": c.Name, "snomed_code": c.SnomedCode,
		})
	}
	user := mustJSON(map[string]any{
		"state":         state,
		"umls_concepts": umlsPayload,
		"specialist_model_prediction": map[string]any{
			"esi_level": bertLevel, "confidence": round3(bertConf),
		},
		"recent_conversation": messages,
		"profile_context":     profile.llmPayload(),
	})
	return e.llm.JSON(ctx, systemWithProfileBoundary(m5System, profile), user, 0.1, 2500, map[string]any{
		"decision_A": "", "decision_B": "", "decision_C": "", "decision_D": "",
		"esi_level": bertLevel, "confidence": "low", "uncertain": true,
		"dangerous_mimic": false, "likely_condition": nil,
		"recommended_specialty": defaultSpecialty(bertLevel),
		"rationale":             "fallback to specialist", "differential": []any{},
	})
}

type m6Input struct {
	state            map[string]any
	finalESI         int
	confidence       string
	modelDisagree    bool
	dangerousMimic   bool
	allowCondition   bool
	likelyCondition  any
	differential     []any
	recommendedSpec  string
	recommendActions []string
	messages         []Message
	profile          ProfileContext
}

func (e *Engine) m6Respond(ctx context.Context, in m6Input) string {
	user := mustJSON(map[string]any{
		"triage_result": map[string]any{
			"esi_level": in.finalESI, "confidence": in.confidence,
			"model_disagreement": in.modelDisagree, "dangerous_mimic": in.dangerousMimic,
			"allow_condition": in.allowCondition, "likely_condition": in.likelyCondition,
			"recommended_specialty": in.recommendedSpec,
			"differential":          in.differential, "recommended_actions": in.recommendActions,
		},
		"state":                in.state,
		"conversation_history": in.messages,
		"profile_context":      in.profile.llmPayload(),
		"response_language":    preferredResponseLanguage("", in.messages),
	})
	reply := e.llm.Text(ctx, systemWithProfileBoundary(m6System, in.profile), user, 0.4, 2500)
	if strings.TrimSpace(reply) == "" || strings.HasPrefix(reply, "[GPT-OSS") {
		// Safe, contentful fallback when the model is unavailable or returns nothing.
		return fallbackReply(in)
	}
	return reply
}

// ---- ported helpers (Cell 14 + Cell 12) ----

var redFlagTerms = []string{
	"chest pain", "crushing", "short of breath", "shortness of breath", "difficulty breathing",
	"unresponsive", "unconscious", "not breathing", "cardiac arrest", "stroke", "facial droop",
	"slurred speech", "severe bleeding", "uncontrolled bleeding", "anaphylaxis", "seizure",
	"blue lips", "suicidal", "overdose", "stiff neck and fever",
}

// negationCues precede a symptom to mark it as explicitly denied ("no chest
// pain", "denies shortness of breath"). Matching the whole serialized state used
// to flag denied symptoms — e.g. the modifier "denies chest pain" contains the
// substring "chest pain" — producing false ESI-2 floors.
var negationCues = []string{
	"no ", "not ", "without ", "denies ", "deny ", "denied ", "negative for ",
	"never ", "free of ", "no history of ", "n't ", " no ongoing ",
}

// detectRedFlags scans only affirmed clinical sources (the patient message,
// chief complaint, and symptom list — not denial modifiers) and skips
// occurrences that are explicitly negated. The escalate-when-in-doubt bias is
// preserved: a flag is only suppressed when clearly negated.
func detectRedFlags(text string, state *State) []string {
	sources := make([]string, 0, len(state.Symptoms)+2)
	sources = append(sources, text, state.ChiefComplaint)
	sources = append(sources, state.Symptoms...)

	seen := map[string]struct{}{}
	var found []string
	for _, src := range sources {
		for _, f := range redFlagTerms {
			if _, ok := seen[f]; ok {
				continue
			}
			if containsAffirmed(src, f) {
				found = append(found, f)
				seen[f] = struct{}{}
			}
		}
	}
	return found
}

// containsAffirmed reports whether phrase appears in src without an immediately
// preceding negation cue.
func containsAffirmed(src, phrase string) bool {
	s := strings.ToLower(src)
	p := strings.ToLower(phrase)
	for from := 0; from < len(s); {
		i := strings.Index(s[from:], p)
		if i < 0 {
			return false
		}
		pos := from + i
		start := pos - 24
		if start < 0 {
			start = 0
		}
		if !hasNegationCue(s[start:pos]) {
			return true
		}
		from = pos + len(p)
	}
	return false
}

func hasNegationCue(prefix string) bool {
	for _, cue := range negationCues {
		if strings.Contains(prefix, cue) {
			return true
		}
	}
	return false
}

func dangerZoneVitals(v Vitals, age *int) *int {
	two := 2
	if v.SpO2 != nil && *v.SpO2 < 92 {
		return &two
	}
	if v.RR != nil && (*v.RR > 28 || *v.RR < 8) {
		return &two
	}
	if v.HR != nil && (*v.HR > 130 || *v.HR < 40) {
		return &two
	}
	return nil
}

func reconcile(bertLevel, gptLevel int, redFlagFloor *int) ReconcileResult {
	final := bertLevel
	if gptLevel < final {
		final = gptLevel
	}
	if redFlagFloor != nil && *redFlagFloor < final {
		final = *redFlagFloor
	}
	disagreement := abs(bertLevel-gptLevel) >= 2
	return ReconcileResult{
		FinalESI:            final,
		BertESI:             bertLevel,
		GptESI:              gptLevel,
		RedFlagFloor:        redFlagFloor,
		Disagreement:        disagreement,
		GptWantedDeescalate: gptLevel > bertLevel,
	}
}

func mergeState(state *State, ext map[string]any) {
	if v := asString(ext["chief_complaint"], ""); v != "" {
		state.ChiefComplaint = v
	}
	if v := asString(ext["onset"], ""); v != "" {
		state.Onset = v
	}
	if v := asString(ext["severity"], ""); v != "" {
		state.Severity = v
	}
	if v := asString(ext["location"], ""); v != "" {
		state.Location = v
	}
	state.Symptoms = sortedUnion(state.Symptoms, sanitizeClinicalTerms(asStringSlice(ext["symptoms"])))
	state.Modifiers = sortedUnion(state.Modifiers, sanitizeClinicalTerms(asStringSlice(ext["modifiers"])))
	state.RiskFactors = sortedUnion(state.RiskFactors, sanitizeClinicalTerms(asStringSlice(ext["risk_factors"])))
	mergeDemographics(state, ext["demographics"])
	mergeVitals(state, ext["vitals"])
}

func sanitizeClinicalTerms(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		term := strings.TrimSpace(item)
		normalized := strings.ToLower(term)
		if term == "" || len(term) > 80 || strings.Contains(term, "?") || strings.Contains(normalized, "what should i do") {
			continue
		}
		out = append(out, term)
	}
	return out
}

func mergeDemographics(state *State, value any) {
	data, ok := value.(map[string]any)
	if !ok {
		return
	}
	if age, ok := asFloat(data["age"]); ok {
		ageInt := int(age)
		if ageInt > 0 {
			state.Demographics.Age = &ageInt
		}
	}
	if sex := strings.TrimSpace(asString(data["sex"], "")); sex != "" {
		state.Demographics.Sex = &sex
	}
}

func mergeVitals(state *State, value any) {
	data, ok := value.(map[string]any)
	if !ok {
		return
	}
	if hr, ok := asFloat(data["hr"]); ok {
		state.Vitals.HR = &hr
	}
	if rr, ok := asFloat(data["rr"]); ok {
		state.Vitals.RR = &rr
	}
	if spo2, ok := asFloat(data["spo2"]); ok {
		state.Vitals.SpO2 = &spo2
	}
	if temp, ok := asFloat(data["temp"]); ok {
		state.Vitals.Temp = &temp
	}
	if bp := strings.TrimSpace(asString(data["bp"], "")); bp != "" {
		state.Vitals.BP = &bp
	}
}

var (
	agePattern       = regexp.MustCompile(`(?i)\b(?:i am|i'm|im|age|aged)\s*(\d{1,3})\b`)
	bpPattern        = regexp.MustCompile(`\b(\d{2,3})\s*/\s*(\d{2,3})\b`)
	hrPattern        = regexp.MustCompile(`(?i)\b(?:heart rate|hr|pulse)\D{0,16}(\d{2,3})\b`)
	rrPattern        = regexp.MustCompile(`(?i)\b(?:respiratory rate|breathing rate|rr)\D{0,16}(\d{1,2})\b`)
	spo2Pattern      = regexp.MustCompile(`(?i)\b(?:spo2|sp02|oxygen|o2|saturation)\D{0,16}(\d{2,3})\s*%?`)
	tempPattern      = regexp.MustCompile(`(?i)\b(?:temperature|temp)\D{0,16}(\d{2,3}(?:\.\d+)?)\b`)
	severityPattern  = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(?:/|out of)\s*10\b`)
	onsetPattern     = regexp.MustCompile(`(?i)\b(?:started|began|onset)\s+([^.;,]+?\s+ago)\b`)
	numberOnlyPatten = regexp.MustCompile(`^\s*\d{1,3}(?:\.\d+)?\s*%?\s*$`)
)

func applyDeterministicClinicalExtraction(state *State, text, previousAssistant string) {
	if match := agePattern.FindStringSubmatch(text); len(match) == 2 {
		if age, ok := parsePositiveInt(match[1]); ok {
			state.Demographics.Age = &age
		}
	}
	if strings.Contains(strings.ToLower(previousAssistant), "how old") {
		if age, ok := parsePositiveInt(text); ok {
			state.Demographics.Age = &age
		}
	}

	if match := bpPattern.FindStringSubmatch(text); len(match) == 3 {
		bp := match[1] + "/" + match[2]
		state.Vitals.BP = &bp
	}
	applyNumberFromPattern(&state.Vitals.HR, hrPattern, text)
	applyNumberFromPattern(&state.Vitals.RR, rrPattern, text)
	applyNumberFromPattern(&state.Vitals.SpO2, spo2Pattern, text)
	applyNumberFromPattern(&state.Vitals.Temp, tempPattern, text)
	applyContextualVitalAnswer(state, text, previousAssistant)

	if match := severityPattern.FindStringSubmatch(text); len(match) == 2 {
		state.Severity = match[1] + "/10"
	}
	if match := onsetPattern.FindStringSubmatch(text); len(match) == 2 && strings.TrimSpace(state.Onset) == "" {
		state.Onset = strings.TrimSpace(match[1])
	}

	normalized := strings.ToLower(text)
	if strings.Contains(normalized, "chest tightness") {
		state.Symptoms = sortedUnion(state.Symptoms, []string{"chest tightness"})
		if strings.TrimSpace(state.ChiefComplaint) == "" || len(state.ChiefComplaint) > 80 {
			state.ChiefComplaint = "chest tightness"
		}
	}
	if strings.Contains(normalized, "no trouble breathing") ||
		strings.Contains(normalized, "not having trouble breathing") ||
		strings.Contains(normalized, "no shortness of breath") ||
		strings.Contains(normalized, "breathing normally") {
		addModifier(state, "denies trouble breathing")
	}
	if strings.Contains(strings.ToLower(previousAssistant), "trouble breathing") {
		if isNegativeAnswer(text) {
			addModifier(state, "denies trouble breathing")
		} else if isPositiveAnswer(text) {
			state.Symptoms = sortedUnion(state.Symptoms, []string{"shortness of breath"})
			state.RedFlags = sortedUnion(state.RedFlags, []string{"shortness of breath"})
		}
	}
	for phrase, modifier := range map[string]string{
		"no fainting":       "denies fainting",
		"no fever":          "denies fever",
		"no severe pain":    "denies severe pain",
		"no chest pain":     "denies chest pain",
		"no chest pressure": "denies chest pressure",
	} {
		if strings.Contains(normalized, phrase) {
			addModifier(state, modifier)
		}
	}
}

func applyNumberFromPattern(target **float64, pattern *regexp.Regexp, text string) {
	match := pattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return
	}
	if value, err := strconv.ParseFloat(match[1], 64); err == nil {
		*target = &value
	}
}

func applyContextualVitalAnswer(state *State, text, previousAssistant string) {
	if !numberOnlyPatten.MatchString(text) {
		return
	}
	value, err := strconv.ParseFloat(strings.Trim(strings.TrimSpace(text), "%"), 64)
	if err != nil {
		return
	}
	question := strings.ToLower(previousAssistant)
	switch {
	case strings.Contains(question, "heart rate") || strings.Contains(question, "pulse"):
		state.Vitals.HR = &value
	case strings.Contains(question, "respiratory rate") || strings.Contains(question, "breathing rate"):
		state.Vitals.RR = &value
	case strings.Contains(question, "oxygen") || strings.Contains(question, "spo2") || strings.Contains(question, "sp02"):
		state.Vitals.SpO2 = &value
	case strings.Contains(question, "temperature"):
		state.Vitals.Temp = &value
	}
}

func parsePositiveInt(text string) (int, bool) {
	cleaned := strings.Trim(strings.TrimSpace(text), " .,%")
	value, err := strconv.Atoi(cleaned)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func addModifier(state *State, modifier string) {
	state.Modifiers = sortedUnion(state.Modifiers, []string{modifier})
}

func clinicalState(state *State) map[string]any {
	var m map[string]any
	raw, _ := json.Marshal(state)
	_ = json.Unmarshal(raw, &m)
	delete(m, "messages")
	return m
}

func recentMessages(state *State) []Message {
	if len(state.Messages) <= maxMessages {
		return state.Messages
	}
	return state.Messages[len(state.Messages)-maxMessages:]
}

func lastAssistantMessage(state *State) string {
	for i := len(state.Messages) - 1; i >= 0; i-- {
		if state.Messages[i].Role == "assistant" {
			return state.Messages[i].Content
		}
	}
	return ""
}

func filterAnsweredMissing(missing any, state *State) []any {
	items := asSlice(missing)
	if len(items) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		label := strings.ToLower(strings.TrimSpace(asString(item, "")))
		if label == "" || isMissingItemAnswered(label, state) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func isMissingItemAnswered(label string, state *State) bool {
	switch {
	case strings.Contains(label, "age") || strings.Contains(label, "demographic"):
		return state.Demographics.Age != nil
	case strings.Contains(label, "breath") || strings.Contains(label, "shortness"):
		return hasBreathingStatus(state)
	case strings.Contains(label, "severity") || strings.Contains(label, "pain score"):
		return strings.TrimSpace(state.Severity) != ""
	case strings.Contains(label, "duration") || strings.Contains(label, "onset") || strings.Contains(label, "started"):
		return strings.TrimSpace(state.Onset) != ""
	case strings.Contains(label, "blood pressure"):
		return state.Vitals.BP != nil
	case strings.Contains(label, "heart rate") || strings.Contains(label, "pulse"):
		return state.Vitals.HR != nil
	case strings.Contains(label, "oxygen") || strings.Contains(label, "spo2") || strings.Contains(label, "sp02"):
		return state.Vitals.SpO2 != nil
	case strings.Contains(label, "respiratory rate"):
		return state.Vitals.RR != nil
	case strings.Contains(label, "vital"):
		return hasAnyVital(state)
	case strings.Contains(label, "main symptom") || strings.Contains(label, "chief") || strings.Contains(label, "complaint"):
		return strings.TrimSpace(state.ChiefComplaint) != "" || len(state.Symptoms) > 0
	default:
		return false
	}
}

func hasMinimumTriageContext(state *State) bool {
	hasConcern := strings.TrimSpace(state.ChiefComplaint) != "" || len(state.Symptoms) > 0
	hasTimingOrSeverity := strings.TrimSpace(state.Onset) != "" || strings.TrimSpace(state.Severity) != ""
	return hasConcern && hasTimingOrSeverity
}

func hasCoreTriageContext(state *State) bool {
	if !hasMinimumTriageContext(state) {
		return false
	}
	if requiresBreathingStatus(state) && !hasBreathingStatus(state) {
		return false
	}
	return state.Demographics.Age != nil || hasAnyVital(state)
}

func requiresBreathingStatus(state *State) bool {
	blob := strings.ToLower(state.ChiefComplaint + " " + strings.Join(state.Symptoms, " ") + " " + state.Location)
	return strings.Contains(blob, "chest") ||
		strings.Contains(blob, "breath") ||
		strings.Contains(blob, "tightness") ||
		strings.Contains(blob, "pressure")
}

func hasBreathingStatus(state *State) bool {
	for _, symptom := range state.Symptoms {
		if strings.Contains(strings.ToLower(symptom), "shortness of breath") || strings.Contains(strings.ToLower(symptom), "difficulty breathing") {
			return true
		}
	}
	for _, modifier := range state.Modifiers {
		if strings.Contains(strings.ToLower(modifier), "breath") {
			return true
		}
	}
	return false
}

func hasAnyVital(state *State) bool {
	return state.Vitals.BP != nil || state.Vitals.HR != nil || state.Vitals.RR != nil || state.Vitals.SpO2 != nil || state.Vitals.Temp != nil
}

func isNegativeAnswer(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.Trim(normalized, ".!؟? ")
	return normalized == "no" || normalized == "nope" || normalized == "not now" || normalized == "لا" || normalized == "لا شيء"
}

func isPositiveAnswer(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.Trim(normalized, ".!؟? ")
	return normalized == "yes" || normalized == "yeah" || normalized == "yep" || normalized == "نعم" || normalized == "ايوه"
}

func trim(state *State) {
	if len(state.Messages) > maxMessages {
		state.Messages = state.Messages[len(state.Messages)-maxMessages:]
	}
}

func defaultSpecialty(finalESI int) string {
	if finalESI <= 2 {
		return "emergency medicine"
	}
	return "general practice / primary care"
}

func fallbackReply(in m6Input) string {
	if shouldAnswerArabic("", in.messages) {
		return fallbackReplyArabic(in)
	}
	clinician := clinicianLabel(in.recommendedSpec, in.finalESI)
	urgency := fallbackUrgency(in.finalESI, clinician)
	known := fallbackKnownFacts(in.state)
	guidance := fallbackGeneralGuidance(in.state, in.finalESI)
	escalation := fallbackEscalationAdvice(in.state, in.finalESI)

	parts := []string{"## What to do now\n\n" + urgency}
	if known != "" {
		parts = append(parts, "## What I am basing this on\n\n"+known)
	}
	parts = append(parts, guidance, escalation)
	parts = append(parts, "## Important note\n\nThis assessment is intended for triage and guidance only. It is not a medical diagnosis or a substitute for professional medical care. Only a qualified clinician can diagnose medical conditions.")
	return strings.Join(parts, "\n\n")
}

func fallbackReplyArabic(in m6Input) string {
	urgency := fallbackUrgencyArabic(in.finalESI)
	known := fallbackKnownFactsArabic(in.state)
	guidance := fallbackGeneralGuidanceArabic(in.finalESI)
	escalation := fallbackEscalationAdviceArabic(in.finalESI)

	parts := []string{"## ما عليك فعله الآن\n\n" + urgency}
	if known != "" {
		parts = append(parts, "## ما أعتمد عليه\n\n"+known)
	}
	parts = append(parts, guidance, escalation)
	parts = append(parts, "## ملاحظة مهمة\n\nهذا التقييم مخصص للفرز والإرشاد فقط. ليس تشخيصا طبيا ولا بديلا عن الرعاية الطبية المتخصصة. يمكن للطبيب أو المختص المؤهل فقط تشخيص الحالات الطبية.")
	return strings.Join(parts, "\n\n")
}

func fallbackUrgency(finalESI int, clinician string) string {
	switch {
	case finalESI <= 2:
		return "Based on what you've described, this could be serious and you should seek emergency care now, ideally through the nearest emergency department or local emergency services."
	case finalESI == 3:
		return "Based on what you've described, it would be safest to be assessed today or within 24 hours by " + clinician + ", especially if symptoms are not clearly improving."
	case finalESI == 4:
		return "Based on what you've described, this does not sound like an immediate emergency right now, but you should arrange follow-up within the next day or two with " + clinician + " if symptoms persist, recur, or are worrying."
	default:
		return "Based on what you've described, this sounds appropriate for routine guidance and monitoring, with follow-up from " + clinician + " if it does not improve or you remain concerned."
	}
}

func fallbackUrgencyArabic(finalESI int) string {
	switch {
	case finalESI <= 2:
		return "بناء على ما ذكرته، قد يكون الأمر خطيرا ويجب طلب رعاية طارئة الآن من قسم الطوارئ أو رقم الطوارئ المحلي."
	case finalESI == 3:
		return "بناء على ما ذكرته، الأفضل أن يتم تقييمك اليوم أو خلال 24 ساعة، خصوصا إذا لم تتحسن الأعراض بوضوح."
	case finalESI == 4:
		return "بناء على ما ذكرته، لا يبدو كحالة طارئة فورية الآن، لكن رتّب متابعة خلال يوم أو يومين إذا استمرت الأعراض أو تكررت أو كانت مقلقة."
	default:
		return "بناء على ما ذكرته، يبدو مناسبا للمراقبة والإرشاد العام، مع طلب متابعة طبية إذا لم يتحسن الأمر أو بقيت قلقا."
	}
}

func fallbackKnownFacts(state map[string]any) string {
	var facts []string
	if demographics, ok := state["demographics"].(map[string]any); ok {
		if age, ok := asFloat(demographics["age"]); ok && age > 0 {
			facts = append(facts, "age: "+formatNumber(age))
		}
		if sex := strings.TrimSpace(asString(demographics["sex"], "")); sex != "" {
			facts = append(facts, "sex: "+sex)
		}
	}
	if chief := strings.TrimSpace(asString(state["chief_complaint"], "")); chief != "" {
		facts = append(facts, "main concern: "+chief)
	}
	if symptoms := asStringSlice(state["symptoms"]); len(symptoms) > 0 {
		facts = append(facts, "symptoms: "+strings.Join(symptoms, ", "))
	}
	if onset := strings.TrimSpace(asString(state["onset"], "")); onset != "" {
		facts = append(facts, "onset: "+onset)
	}
	if severity := strings.TrimSpace(asString(state["severity"], "")); severity != "" {
		facts = append(facts, "severity: "+severity)
	}
	if location := strings.TrimSpace(asString(state["location"], "")); location != "" {
		facts = append(facts, "location: "+location)
	}
	if vitals := fallbackVitalsSummary(state); vitals != "" {
		facts = append(facts, "vitals: "+vitals)
	}
	if len(facts) == 0 {
		return ""
	}
	return "I am basing this on the details you gave: " + strings.Join(facts, "; ") + "."
}

func fallbackKnownFactsArabic(state map[string]any) string {
	var facts []string
	if demographics, ok := state["demographics"].(map[string]any); ok {
		if age, ok := asFloat(demographics["age"]); ok && age > 0 {
			facts = append(facts, "العمر: "+formatNumber(age))
		}
		if sex := strings.TrimSpace(asString(demographics["sex"], "")); sex != "" {
			facts = append(facts, "الجنس: "+sex)
		}
	}
	if chief := strings.TrimSpace(asString(state["chief_complaint"], "")); chief != "" {
		facts = append(facts, "المشكلة الأساسية: "+chief)
	}
	if symptoms := asStringSlice(state["symptoms"]); len(symptoms) > 0 {
		facts = append(facts, "الأعراض: "+strings.Join(symptoms, ", "))
	}
	if onset := strings.TrimSpace(asString(state["onset"], "")); onset != "" {
		facts = append(facts, "البداية: "+onset)
	}
	if severity := strings.TrimSpace(asString(state["severity"], "")); severity != "" {
		facts = append(facts, "الشدة: "+severity)
	}
	if location := strings.TrimSpace(asString(state["location"], "")); location != "" {
		facts = append(facts, "المكان: "+location)
	}
	if vitals := fallbackVitalsSummary(state); vitals != "" {
		facts = append(facts, "العلامات الحيوية: "+vitals)
	}
	if len(facts) == 0 {
		return ""
	}
	return "أعتمد على التفاصيل التي ذكرتها: " + strings.Join(facts, "؛ ") + "."
}

func fallbackVitalsSummary(state map[string]any) string {
	data, ok := state["vitals"].(map[string]any)
	if !ok {
		return ""
	}
	var vitals []string
	if bp := strings.TrimSpace(asString(data["bp"], "")); bp != "" {
		vitals = append(vitals, "blood pressure "+bp)
	}
	if hr, ok := asFloat(data["hr"]); ok {
		vitals = append(vitals, "heart rate "+formatNumber(hr))
	}
	if rr, ok := asFloat(data["rr"]); ok {
		vitals = append(vitals, "respiratory rate "+formatNumber(rr))
	}
	if spo2, ok := asFloat(data["spo2"]); ok {
		vitals = append(vitals, "oxygen saturation "+formatNumber(spo2)+"%")
	}
	if temp, ok := asFloat(data["temp"]); ok {
		vitals = append(vitals, "temperature "+formatNumber(temp))
	}
	return strings.Join(vitals, ", ")
}

func fallbackGeneralGuidance(state map[string]any, finalESI int) string {
	if finalESI <= 2 {
		return "## General care while you monitor\n\n- While getting help, avoid exertion, sit or lie in the position that makes breathing easiest, and keep your phone nearby.\n- Have your medication list, allergies, medical conditions, and symptom timeline ready for the emergency team.\n- Do not drive yourself if you feel faint, short of breath, confused, weak, or unsafe."
	}

	guidance := []string{
		"Rest and avoid strenuous activity until you know symptoms are stable.",
		"Drink fluids if you can keep them down.",
		"Track what changes the symptoms, their severity, and any temperature, pulse, breathing changes, or blood pressure readings you have.",
		"Continue prescribed medicines as directed.",
	}
	if hasAnyStateSymptom(state, "sore throat", "cough", "runny nose", "congestion") {
		guidance = append(guidance, "Use usual cold or throat comfort measures that are normally safe for you.")
	}
	if hasAnyStateSymptom(state, "pain", "headache", "fever") {
		guidance = append(guidance, "Use over-the-counter symptom relief only if it is normally safe for you and according to the label; ask a pharmacist or clinician if you have medical conditions, pregnancy, allergies, or take other medicines.")
	}
	return "## General care while you monitor\n\n- " + strings.Join(guidance, "\n- ")
}

func fallbackGeneralGuidanceArabic(finalESI int) string {
	if finalESI <= 2 {
		return "## ما يمكنك فعله الآن\n\n- أثناء انتظار المساعدة، تجنب المجهود واجلس أو استلق بالوضع الأكثر راحة للتنفس.\n- جهز قائمة الأدوية والحساسيات والأمراض المزمنة وتوقيت بداية الأعراض.\n- لا تقد السيارة بنفسك إذا شعرت بإغماء أو ضيق نفس أو ارتباك أو ضعف أو عدم أمان."
	}
	return "## ما يمكنك فعله الآن\n\n- استرح وتجنب المجهود الشديد حتى تتأكد أن الأعراض مستقرة.\n- اشرب سوائل إذا كنت تستطيع الاحتفاظ بها.\n- راقب تغير الأعراض وشدتها وأي قراءات حرارة أو نبض أو تنفس أو ضغط متاحة.\n- استمر على أدويتك الموصوفة كما وُصفت لك."
}

func fallbackEscalationAdvice(state map[string]any, finalESI int) string {
	signs := []string{
		"trouble breathing",
		"chest pain or pressure",
		"fainting",
		"new confusion",
		"blue lips",
		"sudden weakness or trouble speaking",
		"severe or rapidly worsening pain",
		"uncontrolled bleeding",
		"you cannot keep fluids down",
	}
	if hasAnyStateSymptom(state, "fever") {
		signs = append(signs, "stiff neck, rash, severe drowsiness, or fever that is very high or not improving")
	}
	if hasAnyStateSymptom(state, "abdominal pain") {
		signs = append(signs, "severe abdominal pain, a rigid abdomen, vomiting blood, or black stools")
	}
	bullets := "- " + strings.Join(signs, "\n- ")
	if finalESI <= 2 {
		return "## Escalate to urgent or emergency care\n\nYou should seek emergency care now if any of these are present or develop:\n\n" + bullets
	}
	return "## Escalate to urgent or emergency care\n\nGet urgent or emergency care if you develop any of these, or if the overall situation is getting worse quickly:\n\n" + bullets
}

func fallbackEscalationAdviceArabic(finalESI int) string {
	signs := []string{
		"صعوبة في التنفس",
		"ألم أو ضغط في الصدر",
		"إغماء",
		"ارتباك جديد",
		"ازرقاق الشفاه",
		"ضعف مفاجئ أو صعوبة في الكلام",
		"ألم شديد أو يزداد بسرعة",
		"نزيف لا يتوقف",
		"عدم القدرة على الاحتفاظ بالسوائل",
	}
	bullets := "- " + strings.Join(signs, "\n- ")
	if finalESI <= 2 {
		return "## اطلب مساعدة عاجلة إذا\n\nاطلب رعاية طارئة الآن إذا كان أي من هذه موجودا أو ظهر:\n\n" + bullets
	}
	return "## اطلب مساعدة عاجلة إذا\n\nاطلب رعاية عاجلة أو طارئة إذا ظهر أي من هذه العلامات، أو إذا كان الوضع يزداد سوءا بسرعة:\n\n" + bullets
}

func hasAnyStateSymptom(state map[string]any, terms ...string) bool {
	blob := strings.ToLower(strings.Join(asStringSlice(state["symptoms"]), " ") + " " + asString(state["chief_complaint"], "") + " " + asString(state["location"], ""))
	for _, term := range terms {
		if strings.Contains(blob, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func clinicianLabel(spec string, finalESI int) string {
	normalized := strings.ToLower(strings.TrimSpace(spec))
	if finalESI <= 2 || normalized == "emergency medicine" {
		return "an emergency clinician"
	}
	switch normalized {
	case "", "general practice / primary care", "primary care", "general practice":
		return "your primary care clinician"
	case "cardiology":
		return "a cardiology clinician"
	case "neurology":
		return "a neurology clinician"
	case "orthopedics":
		return "an orthopedic clinician"
	case "dermatology":
		return "a dermatology clinician"
	case "gastroenterology":
		return "a gastroenterology clinician"
	case "obstetrics and gynecology":
		return "an obstetrics and gynecology clinician"
	case "psychiatry":
		return "a mental health clinician"
	case "ophthalmology":
		return "an eye care clinician"
	case "urology":
		return "a urology clinician"
	case "pulmonology":
		return "a pulmonology clinician"
	default:
		return "a " + normalized + " clinician"
	}
}

func shouldRunGeneralHealth(text string, attachments []map[string]any, state *State) bool {
	if hasAnyAttachmentRef(attachments, state) || hasLikelySymptomComplaint(text) {
		return false
	}
	return state.GeneralHealthActive
}

func hasGeneralHealthIntent(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" || hasLikelySymptomComplaint(text) {
		return false
	}
	if !containsAnyTerm(normalized, generalHealthTerms) {
		return false
	}
	return strings.ContainsAny(normalized, "?؟") || containsAnyTerm(normalized, generalHealthQuestionCues)
}

var generalHealthTerms = []string{
	"health", "healthy", "medical", "medicine", "medication", "drug", "doctor", "clinician",
	"nutrition", "diet", "meal", "meals", "food", "eat", "hydration", "water", "exercise",
	"workout", "sleep", "weight", "calorie", "protein", "carb", "salt", "sugar", "vitamin",
	"supplement", "blood pressure", "cholesterol", "diabetes", "vaccine", "screening",
	"prevention", "wellness", "pregnancy", "smoking", "alcohol", "allergy",
	// Science of the human body / physiology education.
	"body", "anatomy", "physiology", "organ", "organs", "muscle", "muscles", "bone", "bones",
	"heart", "brain", "lung", "lungs", "liver", "kidney", "kidneys", "stomach", "intestine",
	"hormone", "hormones", "metabolism", "immune", "immunity", "nervous system", "blood",
	"cell", "cells", "dna", "gene", "genes", "digestion", "nutrient", "nutrients",
	"جسم", "تشريح", "وظائف الأعضاء", "عضو", "أعضاء", "عضلة", "عضلات", "عظم", "عظام",
	"قلب", "دماغ", "رئة", "رئتين", "كبد", "كلية", "كلى", "معدة", "أمعاء",
	"هرمون", "هرمونات", "أيض", "مناعة", "جهاز عصبي", "دم", "خلية", "خلايا", "هضم",
	"صحة", "صحي", "طبي", "دواء", "ادوية", "أدوية", "علاج", "طبيب", "تغذية", "غذاء",
	"نظام غذائي", "حمية", "وجبة", "وجبات", "اكل", "أكل", "طعام", "ماء", "سوائل",
	"رياضة", "تمارين", "نوم", "وزن", "سعرات", "بروتين", "سكر", "ملح", "فيتامين",
	"مكمل", "ضغط", "سكري", "كوليسترول", "لقاح", "تطعيم", "فحص", "وقاية", "حمل",
	"تدخين", "حساسية",
}

var generalHealthQuestionCues = []string{
	"what", "how", "should", "can i", "could i", "recommend", "advice", "advise", "tips",
	"guide", "help", "explain", "normal", "best", "safe",
	"هل", "ما", "ماذا", "كيف", "كم", "تنصح", "انصح", "أنصح", "نصيحة", "نصائح",
	"اشرح", "افضل", "أفضل", "ينفع", "مسموح", "طبيعي",
}

func containsAnyTerm(text string, terms []string) bool {
	normalizedText := strings.ToLower(text)
	for _, term := range terms {
		normalizedTerm := strings.ToLower(term)
		if isASCIITerm(normalizedTerm) {
			if containsBoundedASCIITerm(normalizedText, normalizedTerm) {
				return true
			}
			continue
		}
		if strings.Contains(normalizedText, normalizedTerm) {
			return true
		}
	}
	return false
}

func isASCIITerm(term string) bool {
	for i := 0; i < len(term); i++ {
		if term[i] > 127 {
			return false
		}
	}
	return true
}

func containsBoundedASCIITerm(text, term string) bool {
	for start := 0; start < len(text); {
		index := strings.Index(text[start:], term)
		if index < 0 {
			return false
		}
		pos := start + index
		beforeOK := pos == 0 || !isASCIIAlphaNum(text[pos-1])
		after := pos + len(term)
		afterOK := after == len(text) || !isASCIIAlphaNum(text[after])
		if beforeOK && afterOK {
			return true
		}
		start = pos + len(term)
	}
	return false
}

func isASCIIAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func fallbackGeneralHealthReply(patientText string, messages []Message) string {
	if shouldAnswerArabic(patientText, messages) {
		if containsAnyTerm(strings.ToLower(patientText), []string{"وجبة", "وجبات", "تغذية", "غذاء", "اكل", "أكل", "طعام", "نظام غذائي"}) {
			return "## إجابة عامة\n\nبالنسبة لمعظم البالغين الأصحاء، لا يوجد رقم واحد مثالي للجميع. غالبا تكون وجبتان إلى ثلاث وجبات رئيسية يوميا مناسبة، حسب نشاطك، شهيتك، مواعيدك، وأهدافك الصحية.\n\n## إرشادات عملية\n\n- اجعل كل وجبة متوازنة: بروتين، خضار أو فاكهة، وكربوهيدرات كاملة أو مصدر طاقة مناسب.\n- إذا كنت تجوع بين الوجبات، يمكن إضافة وجبة خفيفة صحية بدلا من إجبار نفسك على وجبات كبيرة.\n- ركز على الانتظام وجودة الطعام أكثر من عدد الوجبات وحده.\n- إذا لديك سكري، حمل، مرض كلى، اضطراب أكل، أو هدف علاجي محدد، الأفضل سؤال طبيب أو أخصائي تغذية لخطة مناسبة لك.\n\nهذه معلومات صحية عامة وليست تشخيصا أو بديلا عن رعاية طبية متخصصة."
		}
		return "## إجابة عامة\n\nأستطيع مساعدتك في الأسئلة الطبية والصحية العامة. أعطني تفاصيل أكثر عن سؤالك الصحي، وسأقدم إرشادا عاما وآمنا بدون تشخيص أو خطة علاج شخصية.\n\nهذه معلومات صحية عامة وليست تشخيصا أو بديلا عن رعاية طبية متخصصة."
	}

	normalized := strings.ToLower(patientText)
	if containsAnyTerm(normalized, []string{"meal", "meals", "nutrition", "diet", "food", "eat"}) {
		return "## General answer\n\nFor most healthy adults, there is no single perfect number. Two to three main meals per day is usually reasonable, depending on your schedule, appetite, activity level, and health goals.\n\n## Practical guidance\n\n- Build meals around protein, vegetables or fruit, and a whole-grain or other suitable energy source.\n- If you get hungry between meals, a healthy snack can be better than forcing very large meals.\n- Focus more on consistency and food quality than on meal count alone.\n- If you have diabetes, pregnancy, kidney disease, an eating disorder, or a specific treatment goal, ask a clinician or dietitian for personalized guidance.\n\nThis is general health information, not a diagnosis or a substitute for professional medical care."
	}
	return "## General answer\n\nI can help with general medical and health questions. Share a little more detail about the health topic, and I can give safe general guidance without making a diagnosis or a personal treatment plan.\n\nThis is general health information, not a diagnosis or a substitute for professional medical care."
}

func hasReportReviewIntent(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	if hasLikelySymptomComplaint(text) {
		return false
	}
	multilingualTerms := []string{
		"dolor", "fiebre", "tos", "nausea", "nauseas", "vomito", "vomitos", "mareo", "sangrado",
		"debilidad", "cansancio", "erupcion", "hinchazon", "pecho", "abdomen", "respirar",
		"douleur", "fievre", "fièvre", "toux", "nausée", "nausee", "vertige", "saignement",
		"faiblesse", "eruption", "éruption", "gonflement", "poitrine", "ventre", "respir",
		"schmerz", "fieber", "husten", "ubelkeit", "übelkeit", "erbrechen", "schwindel",
		"blutung", "schwache", "schwäche", "mudigkeit", "müdigkeit", "ausschlag", "schwellung", "brust", "bauch", "atem",
		"दर्द", "बुखार", "खांसी", "उल्टी", "मतली", "चक्कर", "सांस", "साँस", "खून", "कमजोरी", "थकान", "सूजन", "छाती", "पेट",
		"痛", "疼", "发烧", "發燒", "咳嗽", "恶心", "噁心", "呕吐", "嘔吐", "头晕", "頭暈", "出血", "无力", "無力", "乏力", "皮疹", "肿", "腫", "胸", "腹", "呼吸",
	}
	for _, term := range multilingualTerms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	terms := []string{
		"analysis", "analyses", "lab", "labs", "laboratory", "blood test", "test result", "results",
		"report", "scan", "xray", "x-ray", "mri", "ct", "ultrasound", "based on this", "based on these",
		"تحليل", "تحاليل", "نتيجة", "نتائج", "فحص", "فحوصات", "مختبر", "تقرير", "اشعة", "أشعة",
		"بناء على", "بناءً على",
	}
	for _, term := range terms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}

func shouldRunReportReview(text string, attachments []map[string]any, state *State) bool {
	if state.ReportReviewActive {
		return !hasLikelySymptomComplaint(text)
	}
	if !hasAnyAttachmentRef(attachments, state) {
		return false
	}
	if hasReportReviewIntent(text) {
		return true
	}
	if hasLikelySymptomComplaint(text) {
		return false
	}
	return hasDocumentLikeAttachment(attachments, state) || hasReadableAttachmentSummary(reportAttachmentPayload(attachments, state))
}

func hasLikelySymptomComplaint(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	multilingualTerms := []string{
		"dolor", "fiebre", "tos", "nausea", "nauseas", "vomito", "vomitos", "mareo", "sangrado",
		"debilidad", "cansancio", "erupcion", "hinchazon", "pecho", "abdomen", "respirar",
		"douleur", "fievre", "toux", "nausee", "vertige", "saignement",
		"faiblesse", "eruption", "gonflement", "poitrine", "ventre", "respir",
		"schmerz", "fieber", "husten", "ubelkeit", "erbrechen", "schwindel",
		"blutung", "schwache", "mudigkeit", "ausschlag", "schwellung", "brust", "bauch", "atem",
		"\u0623\u0644\u0645", "\u0648\u062c\u0639", "\u062d\u0631\u0627\u0631\u0629", "\u062d\u0645\u0649", "\u063a\u062b\u064a\u0627\u0646", "\u0642\u064a\u0621", "\u0627\u0633\u062a\u0641\u0631\u0627\u063a", "\u0643\u062d\u0629", "\u0633\u0639\u0627\u0644", "\u062a\u0646\u0641\u0633",
		"\u0646\u0641\u0633", "\u062f\u0648\u062e\u0629", "\u0635\u062f\u0627\u0639", "\u0646\u0632\u064a\u0641", "\u0636\u0639\u0641", "\u062a\u0639\u0628", "\u0625\u0631\u0647\u0627\u0642", "\u0637\u0641\u062d", "\u062a\u0648\u0631\u0645", "\u0635\u062f\u0631", "\u0628\u0637\u0646",
		"\u0926\u0930\u094d\u0926", "\u092c\u0941\u0916\u093e\u0930", "\u0916\u093e\u0902\u0938\u0940", "\u0909\u0932\u094d\u091f\u0940", "\u092e\u0924\u0932\u0940", "\u091a\u0915\u094d\u0915\u0930", "\u0938\u093e\u0902\u0938", "\u0938\u093e\u0901\u0938", "\u0916\u0942\u0928", "\u0915\u092e\u091c\u094b\u0930\u0940", "\u0925\u0915\u093e\u0928", "\u0938\u0942\u091c\u0928", "\u091b\u093e\u0924\u0940", "\u092a\u0947\u091f",
		"\u75db", "\u75bc", "\u53d1\u70e7", "\u767c\u71d2", "\u54b3\u55fd", "\u6076\u5fc3", "\u5641\u5fc3", "\u5455\u5410", "\u5614\u5410", "\u5934\u6655", "\u982d\u6688", "\u51fa\u8840", "\u65e0\u529b", "\u7121\u529b", "\u4e4f\u529b", "\u76ae\u75b9", "\u80bf", "\u816b", "\u80f8", "\u8179", "\u547c\u5438",
	}
	for _, term := range multilingualTerms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	terms := []string{
		"pain", "fever", "nausea", "vomit", "cough", "breath", "dizzy", "headache", "bleeding",
		"weakness", "fatigue", "rash", "swelling", "injury", "chest", "abdominal",
		"ألم", "وجع", "حرارة", "حمى", "غثيان", "قيء", "استفراغ", "كحة", "سعال", "تنفس",
		"نفس", "دوخة", "صداع", "نزيف", "ضعف", "تعب", "إرهاق", "طفح", "تورم", "صدر", "بطن",
	}
	for _, term := range terms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}

func hasAnyAttachmentRef(attachments []map[string]any, state *State) bool {
	return len(attachments) > 0 || len(state.AttachmentsSummary) > 0
}

func hasDocumentLikeAttachment(attachments []map[string]any, state *State) bool {
	for _, att := range reportAttachmentPayload(attachments, state) {
		contentType := strings.ToLower(strings.TrimSpace(asString(att["contentType"], "")))
		fileName := strings.ToLower(strings.TrimSpace(asString(att["fileName"], "")))
		if contentType == "application/pdf" || strings.HasSuffix(fileName, ".pdf") {
			return true
		}
		terms := []string{
			"lab", "labs", "blood", "test", "result", "report", "scan", "xray", "x-ray", "mri", "ct", "ultrasound",
			"تحليل", "تحاليل", "نتيجة", "نتائج", "تقرير", "فحص", "مختبر", "اشعة", "أشعة",
			"analisis", "análisis", "laboratorio", "resultado", "informe",
			"analyse", "laboratoire", "résultat", "resultat", "rapport",
			"blut", "labor", "befund", "bericht", "ergebnis",
		}
		for _, term := range terms {
			if strings.Contains(fileName, term) {
				return true
			}
		}
	}
	return false
}

func rememberAttachmentRefs(state *State, attachments []map[string]any) {
	for _, att := range attachments {
		meta := normalizeAttachmentForPrompt(att)
		if attachmentAlreadyStored(state.AttachmentsSummary, meta) {
			continue
		}
		state.AttachmentsSummary = append(state.AttachmentsSummary, meta)
	}
}

func attachmentAlreadyStored(existing []map[string]any, next map[string]any) bool {
	nextID := strings.TrimSpace(asString(next["id"], ""))
	nextFileName := strings.TrimSpace(asString(next["fileName"], ""))
	for _, item := range existing {
		if nextID != "" && nextID == strings.TrimSpace(asString(item["id"], "")) {
			return true
		}
		if nextFileName != "" && nextFileName == strings.TrimSpace(asString(item["fileName"], "")) {
			return true
		}
	}
	return false
}

func reportAttachmentPayload(attachments []map[string]any, state *State) []map[string]any {
	out := make([]map[string]any, 0, len(state.AttachmentsSummary)+len(attachments))
	seen := map[string]struct{}{}
	for _, att := range state.AttachmentsSummary {
		out = appendAttachmentPayload(out, seen, att)
	}
	for _, att := range attachments {
		out = appendAttachmentPayload(out, seen, att)
	}
	return out
}

func appendAttachmentPayload(out []map[string]any, seen map[string]struct{}, att map[string]any) []map[string]any {
	meta := normalizeAttachmentForPrompt(att)
	key := attachmentIdentity(meta)
	if key != "" {
		if _, ok := seen[key]; ok {
			return out
		}
		seen[key] = struct{}{}
	}
	return append(out, meta)
}

func normalizeAttachmentForPrompt(att map[string]any) map[string]any {
	summary := strings.TrimSpace(asString(att["summary"], ""))
	status := strings.TrimSpace(asString(att["analysisStatus"], ""))
	readable := isReadableAttachment(status, summary)
	return map[string]any{
		"id":             att["id"],
		"fileName":       att["fileName"],
		"contentType":    att["contentType"],
		"analysisStatus": status,
		"summary":        summary,
		"readable":       readable,
	}
}

func isReadableAttachment(status, summary string) bool {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	if normalizedStatus == "readable" {
		return strings.TrimSpace(summary) != ""
	}
	if normalizedStatus != "" && normalizedStatus != "complete" {
		return false
	}
	normalizedSummary := strings.ToLower(strings.TrimSpace(summary))
	if normalizedSummary == "" {
		return false
	}
	unreadableTerms := []string{
		"unavailable",
		"failed",
		"too large",
		"not supported",
		"unsupported",
		"unreadable",
		"blurry",
		"cropped",
		"no extractable",
		"no readable",
		"empty summary",
	}
	for _, term := range unreadableTerms {
		if strings.Contains(normalizedSummary, term) {
			return false
		}
	}
	return true
}

func attachmentIdentity(att map[string]any) string {
	if id := strings.TrimSpace(asString(att["id"], "")); id != "" {
		return "id:" + id
	}
	if name := strings.TrimSpace(asString(att["fileName"], "")); name != "" {
		return "file:" + name
	}
	return ""
}

func fallbackReportReviewReply(patientText string, attachments []map[string]any) string {
	if !hasReadableAttachmentSummary(attachments) {
		if containsArabic(patientText) {
			return "أرى أنك أرفقت ملفا وتريد مراجعته، لكن لا توجد لدي قيم أو نتائج مقروءة من المرفق الآن. الصق أسماء الفحوصات والقيم والوحدات والمدى المرجعي وتاريخ التحليل، أو ارفع صورة واضحة للتقرير، وسأشرح لك معناها بشكل آمن دون تشخيص نهائي. إذا لديك ألم صدر، ضيق نفس شديد، إغماء، ضعف مفاجئ، نزيف شديد، أو تدهور سريع فاذهب للطوارئ الآن. هذا تقييم للفرز والإرشاد فقط وليس تشخيصا طبيا أو بديلا عن رعاية الطبيب."
		}
		return "I can see that you attached a file for review, but I do not have readable test values or findings from it yet. Please paste the test names, values, units, reference ranges, and report date, or upload a clear image of the report, and I can explain it safely without making a diagnosis. If you have chest pain, severe shortness of breath, fainting, sudden weakness, severe bleeding, or rapid worsening, seek emergency care now. This is triage guidance only, not a medical diagnosis or a substitute for professional care."
	}
	if containsArabic(patientText) {
		return "راجعت ملخص المرفق المتاح، لكن خدمة GPT-OSS غير متاحة الآن لإنتاج شرح كامل. أستطيع مساعدتك إذا أرسلت القيم المهمة كنص، وسأوضح الطبيعي وغير الطبيعي وما يحتاج متابعة طبية دون تقديم تشخيص نهائي. هذا تقييم للفرز والإرشاد فقط وليس تشخيصا طبيا أو بديلا عن رعاية الطبيب."
	}
	return "I reviewed the available attachment summary, but GPT-OSS is not available right now to produce a full explanation. Paste the key values as text and I can help distinguish normal from abnormal items and what needs clinician follow-up without making a diagnosis. This is triage guidance only, not a medical diagnosis or a substitute for professional care."
}

func hasReadableAttachmentSummary(attachments []map[string]any) bool {
	for _, att := range attachments {
		if readable, ok := att["readable"].(bool); ok && readable {
			return true
		}
	}
	return false
}

func containsArabic(text string) bool {
	for _, r := range text {
		if r >= '\u0600' && r <= '\u06FF' {
			return true
		}
	}
	return false
}

func preferredResponseLanguage(patientText string, messages []Message) string {
	if shouldAnswerArabic(patientText, messages) {
		return "Arabic"
	}
	return "same as the latest substantive patient message"
}

func shouldAnswerArabic(patientText string, messages []Message) bool {
	latest := latestPatientText(patientText, messages)
	if containsArabic(latest) {
		return true
	}
	if !isLanguageNeutralMessage(latest) {
		return false
	}
	skippedLatest := false
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		content := strings.TrimSpace(messages[i].Content)
		if !skippedLatest && content == latest {
			skippedLatest = true
			continue
		}
		if containsArabic(content) {
			return true
		}
		if !isLanguageNeutralMessage(content) {
			return false
		}
	}
	return false
}

func latestPatientText(patientText string, messages []Message) string {
	if text := strings.TrimSpace(patientText); text != "" {
		return text
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

var neutralVitalsRE = regexp.MustCompile(`(?i)^(?:bp|b/p|hr|rr|spo2|o2|temp|pulse|oxygen|blood pressure)?\s*[:=]?\s*\d+(?:[./]\d+)?\s*(?:%|c|f|bpm|mmhg)?$`)

func isLanguageNeutralMessage(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.Trim(normalized, ".!,;:؟? ")
	if normalized == "" {
		return false
	}
	switch normalized {
	case "yes", "no", "y", "n", "ok", "okay", "sure":
		return true
	}
	return neutralVitalsRE.MatchString(normalized)
}

func (p ProfileContext) consentEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(p.ConsentStatus), "enabled")
}

func (p ProfileContext) promptForLLM() string {
	if !p.consentEnabled() {
		return "Personalization consent is disabled. Do not use stored persona summary or workflow boundary in this thread."
	}
	if prompt := strings.TrimSpace(p.BoundaryPrompt); prompt != "" {
		return prompt
	}

	var parts []string
	parts = append(parts,
		"User-approved profile context is available for GPT-OSS in this thread.",
		"Use it only to respect communication preferences and workflow boundaries.")
	if summary := strings.TrimSpace(p.PersonaSummary); summary != "" {
		parts = append(parts, "Persona summary: "+summary)
	}
	if boundary := strings.TrimSpace(p.WorkflowBoundary); boundary != "" {
		parts = append(parts, "Workflow boundary: "+boundary)
	}
	parts = append(parts, "This context is not clinical evidence. Do not infer symptoms, history, risk, diagnoses, or severity from it. Never let it reduce urgency, override safety rules, or bypass escalation.")
	return strings.Join(parts, " ")
}

func (p ProfileContext) llmPayload() map[string]any {
	prompt := p.promptForLLM()
	if !p.consentEnabled() {
		return map[string]any{
			"consent_status":  "disabled",
			"boundary_prompt": prompt,
		}
	}
	return map[string]any{
		"consent_status":    "enabled",
		"persona_summary":   strings.TrimSpace(p.PersonaSummary),
		"workflow_boundary": strings.TrimSpace(p.WorkflowBoundary),
		"boundary_prompt":   prompt,
	}
}

func systemWithProfileBoundary(system string, profile ProfileContext) string {
	prompt := profile.promptForLLM()
	if strings.TrimSpace(prompt) == "" {
		return system
	}
	return system + "\n\nUSER-CONTROLLED PROFILE BOUNDARY\n" + prompt
}

// ---- small generic helpers ----

func sortedUnion(a, b []string) []string {
	set := map[string]struct{}{}
	for _, x := range a {
		if strings.TrimSpace(x) != "" {
			set[x] = struct{}{}
		}
	}
	for _, x := range b {
		if strings.TrimSpace(x) != "" {
			set[x] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func clampESI(v int) int {
	if v < 1 {
		return 1
	}
	if v > 5 {
		return 5
	}
	return v
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func asInt(v any, fallback int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	}
	return fallback
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		if f, err := n.Float64(); err == nil {
			return f, true
		}
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func formatNumber(n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}

func asString(v any, fallback string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fallback
}

func asSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return []any{}
}

func asStringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}

func mustJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
