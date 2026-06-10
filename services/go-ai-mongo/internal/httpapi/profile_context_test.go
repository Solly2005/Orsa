package httpapi

import (
	"strings"
	"testing"

	"orsa.ai/go-ai-mongo/internal/triage"
)

func TestNormalizeProfileContextDisabledClearsStoredContext(t *testing.T) {
	context := normalizeProfileContext(triage.ProfileContext{
		ConsentStatus:    "disabled",
		PersonaSummary:   "Prefers concise answers",
		WorkflowBoundary: "Never use profile context",
		BoundaryPrompt:   "Persona summary: Prefers concise answers",
	})

	if context.PersonaSummary != "" {
		t.Fatalf("expected disabled consent to clear persona summary, got %q", context.PersonaSummary)
	}
	if context.WorkflowBoundary != "" {
		t.Fatalf("expected disabled consent to clear workflow boundary, got %q", context.WorkflowBoundary)
	}
	if strings.Contains(context.BoundaryPrompt, "Prefers concise answers") {
		t.Fatalf("expected disabled consent prompt to omit stored details, got %q", context.BoundaryPrompt)
	}
}

func TestNormalizeProfileContextEnabledBuildsBoundaryPrompt(t *testing.T) {
	context := normalizeProfileContext(triage.ProfileContext{
		ConsentStatus:    "enabled",
		PersonaSummary:   "Prefers concise answers",
		WorkflowBoundary: "Use profile context for tone only",
	})

	if !strings.Contains(context.BoundaryPrompt, "GPT-OSS") {
		t.Fatalf("expected GPT-OSS boundary prompt, got %q", context.BoundaryPrompt)
	}
	if !strings.Contains(context.BoundaryPrompt, "not clinical evidence") {
		t.Fatalf("expected safety boundary in prompt, got %q", context.BoundaryPrompt)
	}
}

func TestAttachmentAnalysisStatusClassifiesVisionExtraction(t *testing.T) {
	if got := attachmentAnalysisStatus("Hemoglobin 12.9 g/dL within reference range."); got != "readable" {
		t.Fatalf("expected readable status, got %q", got)
	}
	if got := attachmentAnalysisStatus("The upload is blurry and no readable values are visible."); got != "unreadable" {
		t.Fatalf("expected unreadable status, got %q", got)
	}
	if got := attachmentAnalysisStatus(""); got != "unavailable" {
		t.Fatalf("expected unavailable status, got %q", got)
	}
}
