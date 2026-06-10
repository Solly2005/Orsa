# ORSA Architecture Analysis

## Source Inputs
- Active PRD/prompt: pasted ORSA Architecture Integration & Implementation Prompt.
- Frontend source of truth: `C:\Users\Basil\Downloads\Orsa.zip`.
- Immutable chatbot workflow source: `C:\Users\Basil\Downloads\AI_Doctor_Triage_fixed (4).ipynb`.

## Current Supplied State
- Repository was initially empty except for `README.md`.
- The ZIP contains static HTML/CSS/JS, screenshots, and one uploaded image. It is not an Angular project.
- The notebook defines the clinical workflow, prompts M0-M6, BERT-ESI integration, UMLS coding, attachment ingestion, OCR/vision handling, red-flag detection, escalate-only reconciliation, FHIR R5 output, and the dialogue loop.
- The supplied Postgres schema contains `users`, `user_settings`, and `user_files`. `user_settings.health_profile_data` is the preserved JSONB location for profile/persona data.

## Resolved Conflict
- Requirement says the supplied Angular frontend is source of truth.
- Supplied artifact is static frontend.
- Resolution from user: preserve the ZIP UI as the source of truth and rebuild it in Angular.

## Architecture Implemented
- `frontend`: Angular app preserving ORSA visual direction and UX surfaces.
- `services/node-orchestrator`: REST gateway, cache/quota utilities, WebSocket entrypoint, gRPC proto loading, and persona extraction boundary.
- `services/go-ai-mongo`: gRPC runtime and notebook workflow invariants/parity tests.
- `services/csharp-supabase`: gRPC service for settings, consent, profile/persona storage, and audit writes.
- `proto`: versioned proto3 contracts for AI, user, chat, and notifications.

## Hard Boundary
Persona/profile extraction is out-of-band. It reads changed thread JSON only for consenting users and never feeds persona/profile data into M1-M6, BERT, reconciliation, or FHIR.

## Postgres Schema Alignment
- `users`: preserved as the user identity table with `id`, `email`, `username`, `created_at`, and `last_login`.
- `user_settings`: preserved as the settings/profile table with `user_id`, `theme_preference`, `health_profile_data`, and `updated_at`.
- `user_files`: preserved as the uploaded-file metadata table with `id`, `user_id`, `conversation_id`, `file_name`, `file_url`, `file_type`, and `uploaded_at`.
- Additional legal and persona audit tables are additive requirements for consent versioning and auditability.

## Legal Documents
- Terms & Conditions: `docs/legal/terms-2026-06-09.md`
- Privacy Policy: `docs/legal/privacy-2026-06-09.md`
- Health Data & AI Processing Consent: `docs/legal/health-data-ai-consent-2026-06-09.md`
