package triage

import "testing"

func TestDetectRedFlagsAffirmed(t *testing.T) {
	state := NewState()
	flags := detectRedFlags("I have chest pain and shortness of breath", &state)
	if len(flags) < 2 {
		t.Fatalf("expected chest pain and shortness of breath flagged, got %#v", flags)
	}
}

func TestDetectRedFlagsSuppressesNegated(t *testing.T) {
	state := NewState()
	// A patient explicitly denying symptoms must not trigger the ESI-2 red-flag floor.
	if flags := detectRedFlags("no chest pain and denies shortness of breath", &state); len(flags) != 0 {
		t.Fatalf("expected negated mentions to be suppressed, got %#v", flags)
	}
}

func TestDetectRedFlagsIgnoresDenialModifiers(t *testing.T) {
	state := NewState()
	// The "denies chest pain" modifier previously matched via whole-state
	// serialization, producing a false red flag. Only affirmed sources count now.
	state.Modifiers = []string{"denies chest pain"}
	if flags := detectRedFlags("my ankle hurts", &state); len(flags) != 0 {
		t.Fatalf("expected denial modifier to be ignored, got %#v", flags)
	}
}

func TestDetectRedFlagsMixedAffirmedAndNegated(t *testing.T) {
	state := NewState()
	flags := detectRedFlags("chest pain but no shortness of breath", &state)
	if len(flags) != 1 || flags[0] != "chest pain" {
		t.Fatalf("expected only chest pain flagged, got %#v", flags)
	}
}
