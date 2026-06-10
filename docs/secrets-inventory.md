# ORSA Secrets Inventory

No secret values are committed. Values must be supplied through environment variables or the deployment secret store.

## Node Orchestrator
- `JWT_SECRET`
- `SESSION_SECRET`
- `GOOGLE_OAUTH_CLIENT_ID`
- `GOOGLE_OAUTH_CLIENT_SECRET`
- `UPSTASH_REDIS_REST_URL`
- `UPSTASH_REDIS_REST_TOKEN`
- `GPT_OSS_ENDPOINT`
- `GPT_OSS_MODEL_ID`
- `GPT_OSS_AUTH_TOKEN`
- `HF_BASE`
- `PERSONA_EXTRACTION_PROMPT_PATH`
- `GO_AI_GRPC_URL`
- `CSHARP_SUPABASE_GRPC_URL`

## Go AI + Mongo
- `MONGODB_ATLAS_URI`
- `MONGODB_DATABASE`
- `UMLS_API_KEY`
- `BERT_BASE_MODEL_ID`
- `BERT_ESI_MODEL_DIR`
- `BERT_ESI_ONNX_DIR`
- `GITHUB_MODELS_BASE`
- `VISION_MODEL_ID`
- `HUGGING_FACE_API_KEY`
- `HF_TOKEN`
- `GITHUB_TOKEN`

## C# Supabase
- `SUPABASE_PROJECT_REF`
- `SUPABASE_URL`
- `SUPABASE_ANON_KEY`
- `SUPABASE_SERVICE_ROLE_KEY`
- `SUPABASE_DB_CONNECTION_STRING`

## Legal Versions
- `TERMS_VERSION`
- `PRIVACY_VERSION`
- `CONSENT_VERSION`

## Required User-Supplied Prompt
- The persona/profile extraction prompt must be provided as a file path through `PERSONA_EXTRACTION_PROMPT_PATH`.
- The file content is passed unchanged to the GPT-OSS invocation.
- The service fails closed if the path is missing or empty.
