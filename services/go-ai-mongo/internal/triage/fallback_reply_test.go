package triage

import (
	"context"
	"strings"
	"testing"

	"orsa.ai/go-ai-mongo/internal/llm"
	"orsa.ai/go-ai-mongo/internal/modelpool"
)

func TestFallbackReplyIncludesGeneralGuidanceAndEscalation(t *testing.T) {
	reply := fallbackReply(m6Input{
		state: map[string]any{
			"chief_complaint": "chest tightness",
			"symptoms":        []any{"chest tightness"},
			"onset":           "30 minutes ago",
		},
		finalESI:        4,
		recommendedSpec: "general practice / primary care",
	})

	required := []string{
		"General care while you monitor",
		"Escalate to urgent or emergency care",
		"chest tightness",
		"primary care clinician",
	}
	for _, part := range required {
		if !strings.Contains(reply, part) {
			t.Fatalf("expected fallback reply to contain %q, got %q", part, reply)
		}
	}
	if strings.Contains(reply, "please share more detail") {
		t.Fatalf("fallback should not ask for details already gathered, got %q", reply)
	}
}

func TestFallbackReplyEmergencyAdviceIsActionable(t *testing.T) {
	reply := fallbackReply(m6Input{
		state: map[string]any{
			"chief_complaint": "shortness of breath",
			"symptoms":        []any{"shortness of breath"},
		},
		finalESI:        2,
		recommendedSpec: "emergency medicine",
	})

	required := []string{
		"seek emergency care now",
		"Do not drive yourself",
		"trouble breathing",
	}
	for _, part := range required {
		if !strings.Contains(reply, part) {
			t.Fatalf("expected emergency fallback reply to contain %q, got %q", part, reply)
		}
	}
}

func TestFallbackReplyKeepsArabicForNeutralVitalFollowup(t *testing.T) {
	reply := fallbackReply(m6Input{
		state: map[string]any{
			"chief_complaint": "leg pain",
			"symptoms":        []any{"leg pain"},
			"vitals":          map[string]any{"bp": "130/85"},
		},
		finalESI:        4,
		recommendedSpec: "general practice / primary care",
		messages: []Message{
			{Role: "user", Content: "ركضت نصف ساعة متواصلة وأشعر بألم في السيقان"},
			{Role: "assistant", Content: "هل يمكنك إخباري بضغط دمك؟"},
			{Role: "user", Content: "130/85"},
		},
	})

	if !strings.Contains(reply, "ما عليك فعله الآن") {
		t.Fatalf("expected Arabic fallback headings after neutral vital follow-up, got %q", reply)
	}
	if strings.Contains(reply, "What to do now") {
		t.Fatalf("expected fallback not to switch to English, got %q", reply)
	}
}

func TestGeneralHealthIntentDetectsArabicMealQuestion(t *testing.T) {
	text := "تنصحني بكم وجبة اساسية في اليوم"
	if !hasGeneralHealthIntent(text) {
		t.Fatal("expected Arabic meal advice to be treated as a general health request")
	}
	if hasGeneralHealthIntent("how do I create an app") {
		t.Fatal("expected non-health programming question not to match the eat term inside create")
	}
	if shouldRunGeneralHealth("I have chest pain", nil, &State{}) {
		t.Fatal("expected symptom complaints to stay in triage")
	}
}

func TestRunTurnAnswersArabicGeneralHealthWhenLLMUnavailable(t *testing.T) {
	// Empty pool → LLM unavailable, exercising the deterministic fallback path.
	engine := NewEngine(llm.New(modelpool.NewPool(nil)), nil, nil)
	state := NewState()

	result := engine.RunTurn(context.Background(), &state, "تنصحني بكم وجبة اساسية في اليوم", nil, ProfileContext{})

	if result.Type != "general_health" {
		t.Fatalf("expected general_health result, got %q with text %q", result.Type, result.Text)
	}
	if !strings.Contains(result.Text, "وجبتان إلى ثلاث وجبات") {
		t.Fatalf("expected Arabic meal guidance, got %q", result.Text)
	}
	if strings.Contains(result.Text, "طلب غير طبي") {
		t.Fatalf("expected health question not to be refused, got %q", result.Text)
	}
}

func TestMergeStateStoresDemographicsVitalsAndRiskFactors(t *testing.T) {
	state := NewState()
	mergeState(&state, map[string]any{
		"demographics": map[string]any{"age": float64(20), "sex": "female"},
		"vitals": map[string]any{
			"bp":   "120/80",
			"hr":   float64(72),
			"rr":   float64(14),
			"spo2": float64(98),
			"temp": float64(37),
		},
		"risk_factors": []any{"asthma"},
	})

	if state.Demographics.Age == nil || *state.Demographics.Age != 20 {
		t.Fatalf("expected age 20, got %#v", state.Demographics.Age)
	}
	if state.Demographics.Sex == nil || *state.Demographics.Sex != "female" {
		t.Fatalf("expected sex female, got %#v", state.Demographics.Sex)
	}
	if state.Vitals.BP == nil || *state.Vitals.BP != "120/80" {
		t.Fatalf("expected BP 120/80, got %#v", state.Vitals.BP)
	}
	if state.Vitals.HR == nil || *state.Vitals.HR != 72 {
		t.Fatalf("expected HR 72, got %#v", state.Vitals.HR)
	}
	if state.Vitals.RR == nil || *state.Vitals.RR != 14 {
		t.Fatalf("expected RR 14, got %#v", state.Vitals.RR)
	}
	if state.Vitals.SpO2 == nil || *state.Vitals.SpO2 != 98 {
		t.Fatalf("expected SpO2 98, got %#v", state.Vitals.SpO2)
	}
	if len(state.RiskFactors) != 1 || state.RiskFactors[0] != "asthma" {
		t.Fatalf("expected asthma risk factor, got %#v", state.RiskFactors)
	}
}

func TestMergeStateSanitizesVerboseSymptomText(t *testing.T) {
	state := NewState()
	mergeState(&state, map[string]any{
		"symptoms": []any{
			"I am 20. I have mild chest tightness that started 2 hours ago. What should I do?",
			"chest tightness",
		},
	})

	if len(state.Symptoms) != 1 || state.Symptoms[0] != "chest tightness" {
		t.Fatalf("expected concise symptom only, got %#v", state.Symptoms)
	}
}

func TestDeterministicExtractionCapturesVitalsAndBreathingDenial(t *testing.T) {
	state := NewState()
	applyDeterministicClinicalExtraction(&state,
		"I am 20. Chest tightness started 2 hours ago. It is 2 out of 10. I am not having trouble breathing.",
		"",
	)
	applyDeterministicClinicalExtraction(&state,
		"Blood pressure 120/80, heart rate 72, oxygen 98%, respiratory rate 14.",
		"",
	)

	if state.Demographics.Age == nil || *state.Demographics.Age != 20 {
		t.Fatalf("expected age 20, got %#v", state.Demographics.Age)
	}
	if state.Vitals.BP == nil || *state.Vitals.BP != "120/80" {
		t.Fatalf("expected BP 120/80, got %#v", state.Vitals.BP)
	}
	if state.Vitals.HR == nil || *state.Vitals.HR != 72 {
		t.Fatalf("expected HR 72, got %#v", state.Vitals.HR)
	}
	if state.Vitals.RR == nil || *state.Vitals.RR != 14 {
		t.Fatalf("expected RR 14, got %#v", state.Vitals.RR)
	}
	if state.Vitals.SpO2 == nil || *state.Vitals.SpO2 != 98 {
		t.Fatalf("expected SpO2 98, got %#v", state.Vitals.SpO2)
	}
	if !hasBreathingStatus(&state) {
		t.Fatalf("expected breathing denial to count as breathing status, modifiers=%#v", state.Modifiers)
	}
}

func TestFilterAnsweredMissingDropsAnsweredFields(t *testing.T) {
	state := NewState()
	state.ChiefComplaint = "chest tightness"
	state.Onset = "2 hours ago"
	state.Severity = "2/10"
	addModifier(&state, "denies trouble breathing")
	bp := "120/80"
	hr := 72.0
	spo2 := 98.0
	rr := 14.0
	state.Vitals.BP = &bp
	state.Vitals.HR = &hr
	state.Vitals.SpO2 = &spo2
	state.Vitals.RR = &rr

	missing := filterAnsweredMissing([]any{
		"main symptom",
		"symptom duration",
		"symptom severity",
		"breathing status",
		"blood pressure",
		"heart rate",
		"oxygen saturation",
		"respiratory rate",
	}, &state)

	if len(missing) != 0 {
		t.Fatalf("expected all missing items to be filtered, got %#v", missing)
	}
}

func TestCoreTriageContextAllowsGuidanceAfterClarification(t *testing.T) {
	state := NewState()
	state.ChiefComplaint = "chest tightness"
	state.Symptoms = []string{"chest tightness"}
	state.Onset = "2 hours ago"
	state.Severity = "2/10"
	addModifier(&state, "denies trouble breathing")
	age := 20
	state.Demographics.Age = &age
	spo2 := 98.0
	state.Vitals.SpO2 = &spo2

	if !hasCoreTriageContext(&state) {
		t.Fatal("expected core triage context to be complete")
	}
}
