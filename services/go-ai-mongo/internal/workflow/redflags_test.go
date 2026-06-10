package workflow

import "testing"

func TestDangerZoneVitals(t *testing.T) {
	spo2 := 91.0
	result := DangerZoneVitals(Vitals{SpO2: &spo2}, nil)
	if result == nil || *result != 2 {
		t.Fatal("expected low SpO2 to create ESI 2 floor")
	}
}

func TestDetectRedFlags(t *testing.T) {
	flags := DetectRedFlags("I have chest pain and shortness of breath", NewState())
	if len(flags) < 2 {
		t.Fatalf("expected red flags, got %#v", flags)
	}
}
