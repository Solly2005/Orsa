package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"orsa.ai/go-ai-mongo/internal/auth"
)

const testSessionSecret = "test-session-secret"

func testAuthToken(t *testing.T) string {
	t.Helper()
	token, err := auth.Sign(testSessionSecret, "00000000-0000-4000-8000-000000000001", "test@example.com", true, time.Hour)
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return token
}

type fakeAttachmentAnalyzer struct {
	seen []string
}

func (f *fakeAttachmentAnalyzer) Available() bool { return true }

func (f *fakeAttachmentAnalyzer) Analyze(_ context.Context, _ []byte, mimeType, fileName string) (string, error) {
	f.seen = append(f.seen, fileName+"|"+mimeType)
	return "Readable values extracted from " + fileName, nil
}

func TestAttachmentsAcceptMultipleFilesAndDetectMIME(t *testing.T) {
	analyzer := &fakeAttachmentAnalyzer{}
	server := NewServer(nil, nil, nil, analyzer, testSessionSecret, nil)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	addUploadPart(t, writer, "labs.pdf", []byte("%PDF-1.7\nfake pdf bytes"))
	addUploadPart(t, writer, "rash.png", tinyPNG(t))
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/attachments", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testAuthToken(t))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Uploaded    int `json:"uploaded"`
		Attachments []struct {
			FileName       string `json:"fileName"`
			ContentType    string `json:"contentType"`
			AnalysisStatus string `json:"analysisStatus"`
			Summary        string `json:"summary"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Uploaded != 2 || len(payload.Attachments) != 2 {
		t.Fatalf("expected two uploaded attachments, got uploaded=%d len=%d", payload.Uploaded, len(payload.Attachments))
	}
	if payload.Attachments[0].ContentType != "application/pdf" {
		t.Fatalf("expected PDF MIME detection, got %q", payload.Attachments[0].ContentType)
	}
	if payload.Attachments[1].ContentType != "image/png" {
		t.Fatalf("expected PNG MIME detection, got %q", payload.Attachments[1].ContentType)
	}
	for _, attachment := range payload.Attachments {
		if attachment.AnalysisStatus != "readable" {
			t.Fatalf("expected readable attachment %#v", attachment)
		}
		if attachment.Summary == "" {
			t.Fatalf("expected extracted summary for %#v", attachment)
		}
	}
	if len(analyzer.seen) != 2 {
		t.Fatalf("expected analyzer to read both files, got %#v", analyzer.seen)
	}
}

func TestDetectAttachmentContentTypeUsesBytesAndExtensionFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		header   string
		data     []byte
		want     string
	}{
		{name: "pdf bytes", fileName: "unknown.bin", header: "application/octet-stream", data: []byte("%PDF-1.7"), want: "application/pdf"},
		{name: "image bytes", fileName: "unknown.bin", header: "", data: tinyPNG(t), want: "image/png"},
		{name: "extension", fileName: "scan.jpg", header: "application/octet-stream", data: []byte("not enough"), want: "image/jpeg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectAttachmentContentType(tt.fileName, tt.header, tt.data); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png fixture: %v", err)
	}
	return data
}

func addUploadPart(t *testing.T, writer *multipart.Writer, fileName string, data []byte) {
	t.Helper()
	part, err := writer.CreateFormFile("files", fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write form file: %v", err)
	}
}
