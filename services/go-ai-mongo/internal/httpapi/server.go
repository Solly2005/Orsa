// Package httpapi is the external REST/JSON API consumed by the browser (via the
// Angular dev proxy). It is the gateway that replaced the Node orchestrator:
// REST for the browser, gRPC for service-to-service calls to the C# engine.
package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"orsa.ai/go-ai-mongo/internal/auth"
	"orsa.ai/go-ai-mongo/internal/store"
	"orsa.ai/go-ai-mongo/internal/triage"
	"orsa.ai/go-ai-mongo/internal/userclient"
)

// ctxKey is an unexported context key type to avoid collisions.
type ctxKey string

const userIDCtxKey ctxKey = "orsa-user-id"
const emailVerifiedCtxKey ctxKey = "orsa-email-verified"

const (
	defaultPersonaSummary   = "Persona extraction is stored separately from triage and only runs with explicit consent."
	defaultWorkflowBoundary = "Stored profile context can personalize response style only when consent is enabled. It must not change clinical urgency, diagnosis, or safety escalation."

	maxAttachmentFiles        = 5
	dailyAttachmentLimit      = 5
	maxAttachmentReadBytes    = 12 << 20
	maxAttachmentRequestBytes = 64 << 20
)

// UserService is the subset of the C# client the API needs (nil-safe wrapper).
type UserService interface {
	GetSettings(ctx context.Context, userID string) (userclient.Settings, error)
	UpdateSettings(ctx context.Context, userID string, memory, reminders *bool) (userclient.Settings, error)
	GetProfile(ctx context.Context, userID string) (userclient.Profile, error)
	UpdateProfile(ctx context.Context, userID string, memory *bool, summary, boundary *string) (userclient.Profile, error)
	RecordLegalAcceptance(ctx context.Context, userID, terms, privacy, consent, acceptedAtISO string) error
	GetAttachmentUsage(ctx context.Context, userID string) (userclient.AttachmentUsage, error)
	ConsumeAttachment(ctx context.Context, userID string, count, limit int) (userclient.AttachmentUsage, error)
	DeleteUser(ctx context.Context, userID string) error
}

// Server holds the API dependencies.
type Server struct {
	engine        *triage.Engine
	store         store.Store
	users         UserService
	vision        AttachmentAnalyzer
	sessionSecret string
	corsOrigins   []string
	quota         *dailyQuota
	chatLimiter   *rateLimiter
}

type AttachmentAnalyzer interface {
	Available() bool
	Analyze(ctx context.Context, data []byte, mimeType, fileName string) (string, error)
}

func NewServer(engine *triage.Engine, st store.Store, users UserService, vis AttachmentAnalyzer, sessionSecret string, corsOrigins []string) *Server {
	return &Server{
		engine:        engine,
		store:         st,
		users:         users,
		vision:        vis,
		sessionSecret: sessionSecret,
		corsOrigins:   corsOrigins,
		quota:         newDailyQuota(),
		chatLimiter:   newRateLimiter(30, time.Minute),
	}
}

// Handler builds the routed, CORS-wrapped HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /conversations", s.listConversations)
	mux.HandleFunc("GET /conversations/{threadId}/messages", s.getMessages)
	mux.HandleFunc("DELETE /conversations/{threadId}", s.deleteConversation)
	mux.HandleFunc("POST /conversations/{threadId}/restore", s.restoreConversation)
	mux.HandleFunc("POST /chat", s.chat)
	mux.HandleFunc("POST /attachments", s.attachments)
	mux.HandleFunc("GET /settings", s.getSettings)
	mux.HandleFunc("PATCH /settings", s.patchSettings)
	mux.HandleFunc("GET /profile", s.getProfile)
	mux.HandleFunc("PATCH /profile", s.patchProfile)
	mux.HandleFunc("DELETE /account", s.deleteAccount)
	// Auth (register / login / Google OAuth) is handled by the C# Supabase engine
	// on port 8085.  The Angular proxy routes /api/auth/* there directly.
	mux.HandleFunc("GET /notifications", s.notifications)
	// withCORS runs first so even 401 responses carry CORS headers; withAuth then
	// enforces a valid session token on every route except /healthz.
	return s.withCORS(s.withAuth(mux))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "go-ai-mongo"})
}

func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	convs, err := s.store.ListConversations(r.Context(), userID)
	if err != nil || convs == nil {
		convs = []store.ConversationSummary{}
	}
	writeJSON(w, http.StatusOK, convs)
}

func (s *Server) getMessages(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	threadID := r.PathValue("threadId")
	thread, err := s.store.GetThread(r.Context(), userID, threadID)
	if err != nil || thread == nil || len(thread.State.Messages) == 0 {
		writeJSON(w, http.StatusOK, []triage.Message{{
			Role:      "assistant",
			Content:   "Tell me what is going on and I will help route the next step safely.",
			CreatedAt: nowISO(),
		}})
		return
	}
	writeJSON(w, http.StatusOK, thread.State.Messages)
}

func (s *Server) deleteConversation(w http.ResponseWriter, r *http.Request) {
	_ = s.store.SetDeleted(r.Context(), getUserID(r), r.PathValue("threadId"), true)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) restoreConversation(w http.ResponseWriter, r *http.Request) {
	_ = s.store.SetDeleted(r.Context(), getUserID(r), r.PathValue("threadId"), false)
	writeJSON(w, http.StatusOK, map[string]any{"restored": true})
}

type chatRequest struct {
	ThreadID       string                `json:"threadId"`
	Content        string                `json:"content"`
	Attachments    []map[string]any      `json:"attachments"`
	ProfileContext triage.ProfileContext `json:"profileContext"`
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	if !requireVerified(w, r) {
		return
	}
	userID := getUserID(r)
	if !s.chatLimiter.allow(userID) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "too many messages; please slow down and try again shortly"})
		return
	}
	var body chatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "content is required"})
		return
	}

	threadID := body.ThreadID
	if strings.TrimSpace(threadID) == "" {
		threadID = newID()
	}

	thread, err := s.store.GetThread(r.Context(), userID, threadID)
	if err != nil || thread == nil {
		thread = &store.Thread{
			UserID:    userID,
			ThreadID:  threadID,
			State:     triage.NewState(),
			CreatedAt: time.Now().UTC(),
		}
	}

	profileContext := normalizeProfileContext(body.ProfileContext)
	if !hasProfileContext(body.ProfileContext) {
		profileContext = s.profileContextForUser(r.Context(), userID)
	}

	result := s.engine.RunTurn(r.Context(), &thread.State, body.Content, body.Attachments, profileContext)

	if thread.Title == "" {
		thread.Title = makeTitle(body.Content)
	}
	thread.UpdatedAt = time.Now().UTC()
	if err := s.store.SaveThread(r.Context(), thread); err != nil {
		// Persistence failure must not drop the reply the patient already received.
		result.Warnings = append(result.Warnings, "chat history could not be saved this turn")
	}

	resp := map[string]any{
		"role":          "assistant",
		"content":       result.Text,
		"threadId":      threadID,
		"messageId":     newID(),
		"createdAt":     nowISO(),
		"type":          result.Type,
		"serviceStatus": "go_ai_connected",
	}
	if result.Reconcile != nil {
		resp["triage"] = map[string]any{
			"type":           result.Type,
			"reconcile":      result.Reconcile,
			"warnings":       result.Warnings,
			"fhirBundleJson": result.FHIRBundle,
			"differential":   result.Differential,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) attachments(w http.ResponseWriter, r *http.Request) {
	if !requireVerified(w, r) {
		return
	}
	userID := getUserID(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentRequestBytes)
	if err := r.ParseMultipartForm(maxAttachmentRequestBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid multipart form"})
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) > maxAttachmentFiles {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "too many attachments",
			"quota": map[string]any{"allowed": false, "used": s.quotaUsed(r.Context(), userID), "limit": dailyAttachmentLimit, "resetAt": tomorrowISO()},
		})
		return
	}
	// Enforce the per-user daily limit. The count is authoritative in Postgres
	// (via the C# engine) and survives restarts; the in-memory counter is a
	// fallback used only when the user service is unreachable.
	ok, used := s.quotaConsume(r.Context(), userID, len(files), dailyAttachmentLimit)
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error": "daily attachment limit reached",
			"quota": map[string]any{"allowed": false, "used": used, "limit": dailyAttachmentLimit, "resetAt": tomorrowISO()},
		})
		return
	}
	refs := make([]map[string]any, 0, len(files))
	for _, fh := range files {
		id := newID()
		var data []byte
		var readErr error
		if f, err := fh.Open(); err == nil {
			data, readErr = io.ReadAll(io.LimitReader(f, maxAttachmentReadBytes+1))
			f.Close()
		} else {
			readErr = err
		}
		ct := detectAttachmentContentType(fh.Filename, fh.Header.Get("Content-Type"), data)
		ref := map[string]any{
			"id":             id,
			"fileName":       fh.Filename,
			"contentType":    ct,
			"storageUri":     "memory://" + userID + "/" + id + "/" + fh.Filename,
			"caption":        "",
			"summary":        "",
			"analysisStatus": "unavailable",
		}
		// M0: run Llama vision extraction before the chat turn consumes the upload.
		switch {
		case readErr != nil:
			ref["summary"] = "Attachment read failed: " + readErr.Error()
			ref["analysisStatus"] = "failed"
		case len(data) > maxAttachmentReadBytes:
			ref["summary"] = "Attached file is too large for extraction in this request."
			ref["analysisStatus"] = "unreadable"
		case s.vision != nil:
			if summary, err := s.vision.Analyze(r.Context(), data, ct, fh.Filename); err == nil {
				ref["summary"] = summary
				ref["analysisStatus"] = attachmentAnalysisStatus(summary)
			} else {
				ref["summary"] = "Vision extraction failed: " + err.Error()
				ref["analysisStatus"] = "failed"
			}
		}
		refs = append(refs, ref)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"uploaded":    len(files),
		"attachments": refs,
		"quota":       map[string]any{"allowed": true, "used": used, "limit": dailyAttachmentLimit, "resetAt": tomorrowISO()},
	})
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	usedToday := s.quotaUsed(r.Context(), userID)
	if s.users != nil {
		if settings, err := s.users.GetSettings(r.Context(), userID); err == nil {
			writeJSON(w, http.StatusOK, settingsView(userID, settings, usedToday))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"userId": userID, "memoryExtractionEnabled": false, "remindersEnabled": true,
		"attachmentCountToday": usedToday, "attachmentLimit": dailyAttachmentLimit,
	})
}

type patchSettingsBody struct {
	MemoryExtractionEnabled *bool `json:"memoryExtractionEnabled"`
	RemindersEnabled        *bool `json:"remindersEnabled"`
}

func (s *Server) patchSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	usedToday := s.quotaUsed(r.Context(), userID)
	var body patchSettingsBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	if s.users != nil {
		if settings, err := s.users.UpdateSettings(r.Context(), userID, body.MemoryExtractionEnabled, body.RemindersEnabled); err == nil {
			writeJSON(w, http.StatusOK, settingsView(userID, settings, usedToday))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"userId":                  userID,
		"memoryExtractionEnabled": valueOr(body.MemoryExtractionEnabled, false),
		"remindersEnabled":        valueOr(body.RemindersEnabled, true),
		"attachmentCountToday":    usedToday, "attachmentLimit": dailyAttachmentLimit,
	})
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if s.users != nil {
		if profile, err := s.users.GetProfile(r.Context(), userID); err == nil {
			writeJSON(w, http.StatusOK, profileView(userID, profile))
			return
		}
	}
	writeJSON(w, http.StatusOK, fallbackProfileView(userID, nil, nil, nil))
}

type patchProfileBody struct {
	MemoryExtractionEnabled *bool   `json:"memoryExtractionEnabled"`
	ConsentStatus           *string `json:"consentStatus"`
	PersonaSummary          *string `json:"summary"`
	WorkflowBoundary        *string `json:"workflowBoundary"`
}

func (s *Server) patchProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var body patchProfileBody
	_ = json.NewDecoder(r.Body).Decode(&body)

	memory := body.MemoryExtractionEnabled
	if body.ConsentStatus != nil {
		enabled := strings.EqualFold(strings.TrimSpace(*body.ConsentStatus), "enabled")
		memory = &enabled
	}

	if s.users != nil {
		if profile, err := s.users.UpdateProfile(r.Context(), userID, memory, body.PersonaSummary, body.WorkflowBoundary); err == nil {
			writeJSON(w, http.StatusOK, profileView(userID, profile))
			return
		}
	}

	writeJSON(w, http.StatusOK, fallbackProfileView(userID, memory, body.PersonaSummary, body.WorkflowBoundary))
}

// deleteAccount permanently removes the authenticated user's chat history (Mongo)
// and their account/profile data (Postgres, via the C# engine). It does not
// require a verified email — a user must always be able to delete their account.
func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
		return
	}

	// Best-effort chat history wipe; a failure here should not block account removal.
	if err := s.store.DeleteUserThreads(r.Context(), userID); err != nil {
		// Continue: the account row removal below is the authoritative deletion.
		_ = err
	}

	if s.users != nil {
		if err := s.users.DeleteUser(r.Context(), userID); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "could not delete account; please try again"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) notifications(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]any{
		{"id": "n-1", "label": "Follow-up queued", "status": "queued", "createdAt": nowISO()},
		{"id": "n-2", "label": "Upload reviewed", "status": "read", "createdAt": nowISO()},
	})
}

// ---- helpers ----

func profileView(userID string, profile userclient.Profile) map[string]any {
	context := profileContextFromProfile(profile)
	display := profile.DisplayName
	if strings.TrimSpace(display) == "" {
		display = "ORSA User"
	}

	return map[string]any{
		"userId":           userID,
		"displayName":      display,
		"location":         joinNonEmpty(", ", profile.City, profile.Region, profile.Country),
		"summary":          context.PersonaSummary,
		"workflowBoundary": context.WorkflowBoundary,
		"consentStatus":    context.ConsentStatus,
		"boundaryPrompt":   context.BoundaryPrompt,
		"lastPersonaRunAt": emptyToNil(profile.PersonaUpdatedAt),
	}
}

func fallbackProfileView(userID string, memory *bool, summary, boundary *string) map[string]any {
	consentEnabled := valueOr(memory, false)
	personaSummary := defaultPersonaSummary
	if summary != nil {
		personaSummary = strings.TrimSpace(*summary)
	}
	workflowBoundary := defaultWorkflowBoundary
	if boundary != nil {
		workflowBoundary = strings.TrimSpace(*boundary)
	}
	return map[string]any{
		"userId":           userID,
		"displayName":      "ORSA User",
		"location":         "",
		"summary":          personaSummary,
		"workflowBoundary": workflowBoundary,
		"consentStatus":    enabledStatus(consentEnabled),
		"boundaryPrompt":   buildBoundaryPrompt(consentEnabled, personaSummary, workflowBoundary),
	}
}

func (s *Server) profileContextForUser(ctx context.Context, userID string) triage.ProfileContext {
	if s.users == nil {
		return normalizeProfileContext(triage.ProfileContext{ConsentStatus: "disabled"})
	}
	profile, err := s.users.GetProfile(ctx, userID)
	if err != nil {
		return normalizeProfileContext(triage.ProfileContext{ConsentStatus: "disabled"})
	}
	return profileContextFromProfile(profile)
}

func profileContextFromProfile(profile userclient.Profile) triage.ProfileContext {
	data := map[string]any{}
	if strings.TrimSpace(profile.PersonaJSON) != "" {
		_ = json.Unmarshal([]byte(profile.PersonaJSON), &data)
	}

	summary := firstNonEmpty(profile.PersonaSummary, stringFromMap(data, "personaSummary"), defaultPersonaSummary)
	boundary := firstNonEmpty(profile.WorkflowBoundary, stringFromMap(data, "workflowBoundary"), defaultWorkflowBoundary)
	consent := profile.ConsentStatus
	if consent != "enabled" && consent != "disabled" {
		consent = enabledStatus(boolFromMap(data, "memoryExtractionEnabled", false))
	}
	prompt := firstNonEmpty(profile.BoundaryPrompt, stringFromMap(data, "boundaryPrompt"), buildBoundaryPrompt(consent == "enabled", summary, boundary))

	return normalizeProfileContext(triage.ProfileContext{
		ConsentStatus:    consent,
		PersonaSummary:   summary,
		WorkflowBoundary: boundary,
		BoundaryPrompt:   prompt,
	})
}

func hasProfileContext(context triage.ProfileContext) bool {
	return strings.TrimSpace(context.ConsentStatus) != "" ||
		strings.TrimSpace(context.PersonaSummary) != "" ||
		strings.TrimSpace(context.WorkflowBoundary) != "" ||
		strings.TrimSpace(context.BoundaryPrompt) != ""
}

func normalizeProfileContext(context triage.ProfileContext) triage.ProfileContext {
	consent := strings.ToLower(strings.TrimSpace(context.ConsentStatus))
	if consent != "enabled" {
		consent = "disabled"
	}
	summary := strings.TrimSpace(context.PersonaSummary)
	boundary := strings.TrimSpace(context.WorkflowBoundary)
	prompt := strings.TrimSpace(context.BoundaryPrompt)
	if prompt == "" {
		prompt = buildBoundaryPrompt(consent == "enabled", summary, boundary)
	}
	if consent == "disabled" {
		summary = ""
		boundary = ""
		prompt = buildBoundaryPrompt(false, "", "")
	}
	return triage.ProfileContext{
		ConsentStatus:    consent,
		PersonaSummary:   summary,
		WorkflowBoundary: boundary,
		BoundaryPrompt:   prompt,
	}
}

func settingsView(userID string, st userclient.Settings, usedToday int) map[string]any {
	return map[string]any{
		"userId":                  userID,
		"memoryExtractionEnabled": st.MemoryExtractionEnabled,
		"remindersEnabled":        st.RemindersEnabled,
		"attachmentCountToday":    usedToday,
		"attachmentLimit":         dailyAttachmentLimit,
	}
}

// getUserID returns the authenticated user id placed in the request context by
// withAuth. It is never derived from a client-supplied header.
func getUserID(r *http.Request) string {
	if v, ok := r.Context().Value(userIDCtxKey).(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// isEmailVerified reports the verification state carried by the session token.
func isEmailVerified(r *http.Request) bool {
	v, _ := r.Context().Value(emailVerifiedCtxKey).(bool)
	return v
}

// requireVerified writes a 403 and returns false when the caller's email is not
// yet verified. Google sign-ins are always verified, so this only blocks
// unconfirmed email/password accounts.
func requireVerified(w http.ResponseWriter, r *http.Request) bool {
	if isEmailVerified(r) {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]any{
		"error": "please verify your email to use this feature",
		"code":  "email_unverified",
	})
	return false
}

// withAuth requires a valid `Authorization: Bearer <token>` session token on
// every route except the public health check, and stashes the verified user id
// (the token subject / user UUID) in the request context.
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		token := bearerToken(r)
		claims, err := auth.Verify(s.sessionSecret, token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
			return
		}
		ctx := context.WithValue(r.Context(), userIDCtxKey, claims.Subject)
		ctx = context.WithValue(ctx, emailVerifiedCtxKey, claims.EmailVerified)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return ""
	}
	if parts := strings.SplitN(header, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// withCORS reflects only allow-listed origins (no wildcard). Identity travels in
// the Authorization header, so an open `*` policy previously let any website
// read a user's data; the allowlist closes that.
func (s *Server) withCORS(next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(s.corsOrigins))
	for _, o := range s.corsOrigins {
		allowed[strings.ToLower(strings.TrimRight(o, "/"))] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if _, ok := allowed[strings.ToLower(strings.TrimRight(origin, "/"))]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func nowISO() string      { return time.Now().UTC().Format(time.RFC3339) }
func tomorrowISO() string { return time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339) }

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func makeTitle(content string) string {
	title := strings.TrimSpace(content)
	if len(title) > 40 {
		title = title[:40] + "…"
	}
	if title == "" {
		title = "New conversation"
	}
	return title
}

func valueOr(p *bool, fallback bool) bool {
	if p != nil {
		return *p
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func attachmentAnalysisStatus(summary string) string {
	normalized := strings.ToLower(strings.TrimSpace(summary))
	if normalized == "" {
		return "unavailable"
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
		if strings.Contains(normalized, term) {
			return "unreadable"
		}
	}
	return "readable"
}

func detectAttachmentContentType(fileName, headerContentType string, data []byte) string {
	normalized := normalizeMIME(headerContentType)
	if normalized != "" && normalized != "application/octet-stream" {
		return normalized
	}
	if len(data) >= 4 && string(data[:4]) == "%PDF" {
		return "application/pdf"
	}
	if len(data) >= 8 && bytesHasPrefix(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png"
	}
	if len(data) >= 3 && bytesHasPrefix(data, []byte{0xff, 0xd8, 0xff}) {
		return "image/jpeg"
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	if len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a") {
		return "image/gif"
	}
	if ext := strings.ToLower(filepath.Ext(fileName)); ext != "" {
		if ext == ".pdf" {
			return "application/pdf"
		}
		if byExt := normalizeMIME(mime.TypeByExtension(ext)); byExt != "" {
			return byExt
		}
	}
	if sniffed := normalizeMIME(http.DetectContentType(data)); sniffed != "" && sniffed != "application/octet-stream" {
		return sniffed
	}
	return "application/octet-stream"
}

func bytesHasPrefix(data, prefix []byte) bool {
	if len(data) < len(prefix) {
		return false
	}
	for i := range prefix {
		if data[i] != prefix[i] {
			return false
		}
	}
	return true
}

func normalizeMIME(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	if semi := strings.Index(value, ";"); semi >= 0 {
		value = strings.TrimSpace(value[:semi])
	}
	return value
}

func joinNonEmpty(sep string, parts ...string) string {
	var kept []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

func stringFromMap(data map[string]any, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func boolFromMap(data map[string]any, key string, fallback bool) bool {
	if value, ok := data[key].(bool); ok {
		return value
	}
	return fallback
}

func enabledStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func buildBoundaryPrompt(consentEnabled bool, personaSummary, workflowBoundary string) string {
	if !consentEnabled {
		return "Personalization consent is disabled. Do not use stored persona summary or workflow boundary in this thread."
	}
	parts := []string{
		"User-approved profile context is available for GPT-OSS in this thread.",
		"Use it only to respect communication preferences and workflow boundaries.",
	}
	if summary := strings.TrimSpace(personaSummary); summary != "" {
		parts = append(parts, "Persona summary: "+summary)
	}
	if boundary := strings.TrimSpace(workflowBoundary); boundary != "" {
		parts = append(parts, "Workflow boundary: "+boundary)
	}
	parts = append(parts, "This context is not clinical evidence. Do not infer symptoms, history, risk, diagnoses, or severity from it. Never let it reduce urgency, override safety rules, or bypass escalation.")
	return strings.Join(parts, " ")
}

func emptyToNil(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
