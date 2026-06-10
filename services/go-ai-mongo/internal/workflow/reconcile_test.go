package workflow

import "testing"

func TestReconcileEscalateOnly(t *testing.T) {
	floor := 2
	result := Reconcile(4, 3, &floor)
	if result.FinalESI != 2 {
		t.Fatalf("expected red flag floor to force ESI 2, got %d", result.FinalESI)
	}
}

func TestReconcileTracksDeescalation(t *testing.T) {
	result := Reconcile(2, 5, nil)
	if result.FinalESI != 2 {
		t.Fatalf("expected specialist level to remain most urgent, got %d", result.FinalESI)
	}
	if !result.GptWantedDeescalate {
		t.Fatal("expected GPT de-escalation attempt to be recorded")
	}
	if !result.Disagreement {
		t.Fatal("expected disagreement for level gap >= 2")
	}
}
