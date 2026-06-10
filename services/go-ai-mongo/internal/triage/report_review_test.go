package triage

import "testing"

func TestReportReviewIntentDetectsArabicLabRequest(t *testing.T) {
	text := "\u0623\u062e\u0628\u0631\u0646\u064a \u0639\u0646 \u0635\u062d\u062a\u064a \u0628\u0646\u0627\u0621 \u0639\u0644\u0649 \u0647\u0630\u0647 \u0627\u0644\u062a\u062d\u0627\u0644\u064a\u0644"
	if !hasReportReviewIntent(text) {
		t.Fatal("expected Arabic lab-review request to be detected")
	}
}

func TestReportReviewStaysActiveForNonSymptomFollowup(t *testing.T) {
	state := NewState()
	state.ReportReviewActive = true

	if !shouldRunReportReview("\u0644\u0627 \u0634\u064a\u0621", nil, &state) {
		t.Fatal("expected report-review mode to handle non-symptom follow-up")
	}
}

func TestPDFUploadRoutesToReportReviewWithoutLanguageKeywords(t *testing.T) {
	state := NewState()
	attachments := []map[string]any{{
		"id":             "a1",
		"fileName":       "documento.pdf",
		"contentType":    "application/pdf",
		"analysisStatus": "readable",
		"summary":        "CBC report values extracted.",
	}}

	if !shouldRunReportReview("Puedes revisarlo?", attachments, &state) {
		t.Fatal("expected PDF-backed non-symptom request to route to report review")
	}
}

func TestReadableMediaUploadRoutesToReportReviewWithoutLanguageKeywords(t *testing.T) {
	state := NewState()
	attachments := []map[string]any{{
		"id":             "a1",
		"fileName":       "image.png",
		"contentType":    "image/png",
		"analysisStatus": "readable",
		"summary":        "Visible lab report values extracted from the image.",
	}}

	if !shouldRunReportReview("\u8bf7\u770b\u4e00\u4e0b\u8fd9\u4e2a", attachments, &state) {
		t.Fatal("expected readable media-backed non-symptom request to route to report review")
	}
}

func TestAttachmentDoesNotOverrideNonEnglishSymptomComplaint(t *testing.T) {
	state := NewState()
	attachments := []map[string]any{{
		"id":          "a1",
		"fileName":    "documento.pdf",
		"contentType": "application/pdf",
	}}

	if shouldRunReportReview("Tengo dolor en el pecho", attachments, &state) {
		t.Fatal("expected Spanish chest-pain complaint to stay in symptom triage")
	}
}

func TestDocumentAttachmentStateRecoversNumericFollowupToReportReview(t *testing.T) {
	state := NewState()
	state.AttachmentsSummary = []map[string]any{{
		"id":             "a1",
		"fileName":       "blood-test.pdf",
		"contentType":    "application/pdf",
		"analysisStatus": "readable",
		"summary":        "CBC report values extracted.",
		"readable":       true,
	}}

	if !shouldRunReportReview("120/80", nil, &state) {
		t.Fatal("expected numeric follow-up with document attachment state to route to report review")
	}
}

func TestReportReviewYieldsToSymptomComplaint(t *testing.T) {
	state := NewState()
	state.ReportReviewActive = true

	if shouldRunReportReview("I have chest pain", nil, &state) {
		t.Fatal("expected symptom complaint to leave report-review mode")
	}
}

func TestRememberAttachmentRefsStoresUnreadableUpload(t *testing.T) {
	state := NewState()
	attachments := []map[string]any{{
		"id":          "a1",
		"fileName":    "labs.pdf",
		"contentType": "application/pdf",
	}}

	rememberAttachmentRefs(&state, attachments)

	if len(state.AttachmentsSummary) != 1 {
		t.Fatalf("expected one stored attachment ref, got %d", len(state.AttachmentsSummary))
	}
	if readable, _ := state.AttachmentsSummary[0]["readable"].(bool); readable {
		t.Fatal("expected upload without extracted summary to be marked unreadable")
	}
}

func TestAttachmentAnalysisStatusControlsReadability(t *testing.T) {
	readable := normalizeAttachmentForPrompt(map[string]any{
		"id":             "a1",
		"fileName":       "labs.pdf",
		"contentType":    "application/pdf",
		"analysisStatus": "readable",
		"summary":        "Hemoglobin 12.9 g/dL within reference range.",
	})
	if ok, _ := readable["readable"].(bool); !ok {
		t.Fatal("expected readable status and non-empty summary to be readable")
	}

	unreadable := normalizeAttachmentForPrompt(map[string]any{
		"id":             "a2",
		"fileName":       "labs.pdf",
		"contentType":    "application/pdf",
		"analysisStatus": "unreadable",
		"summary":        "The PDF is blurry and no readable values are visible.",
	})
	if ok, _ := unreadable["readable"].(bool); ok {
		t.Fatal("expected unreadable status to override summary text")
	}
}
