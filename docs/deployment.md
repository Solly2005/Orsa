# ORSA Deployment Guide

This guide deploys ORSA as:

- Vercel: Angular frontend.
- Hugging Face Docker Space: combined backend container.

The backend container keeps the microservice architecture internally:

- C# Supabase engine runs as its own process.
- Go AI gateway runs as its own process.
- Go calls C# over local gRPC at `127.0.0.1:50053`.
- Nginx exposes the single public Hugging Face Space port `7860`.

## Public Route Map

Vercel keeps the browser API same-origin:

| Browser path | Vercel destination | Backend service |
| --- | --- | --- |
| `/api/auth/*` | `${ORSA_BACKEND_URL}/auth/*` | C# REST auth |
| `/api/*` | `${ORSA_BACKEND_URL}/*` | Go REST gateway |
| `/*` | `/index.html` | Angular SPA |

Hugging Face Space routes:

| Space path | Internal destination |
| --- | --- |
| `/auth/*` | C# REST auth on `127.0.0.1:8085` |
| `/csharp/healthz` | C# health on `127.0.0.1:8085/healthz` |
| `/*` | Go REST gateway on `127.0.0.1:3000` |

## 1. Prepare the Backend Space Directory

From the repository root:

```powershell
.\scripts\prepare-hf-backend-space.ps1
```

This creates:

```text
dist\huggingface-backend-space
```

That directory is the Hugging Face Space repo content. It includes:

- `Dockerfile` at the Space root.
- Space `README.md` with `sdk: docker` and `app_port: 7860`.
- C# service source.
- Go service source.
- Shared proto contract.
- BERT ONNX runtime artifacts from `models/bert_esi_onnx`.
- Nginx and process startup files.

## 2. Create the Hugging Face Backend Space

Create a new Hugging Face Space:

- Owner: your Hugging Face account or organization.
- Name: recommended `orsa-backend`.
- SDK: Docker.
- Visibility: private for production unless you intentionally want a public API.

The final Space URL will look like:

```text
https://<owner>-orsa-backend.hf.space
```

Use this URL as `ORSA_BACKEND_URL` in Vercel with no trailing slash.

## 3. Push the Backend Space

Clone the empty Space repo:

```powershell
git lfs install
git clone https://huggingface.co/spaces/<owner>/orsa-backend
```

Copy the generated Space files into the clone:

```powershell
Copy-Item -Recurse -Force .\dist\huggingface-backend-space\* .\orsa-backend\
Copy-Item -Force .\dist\huggingface-backend-space\.gitattributes .\orsa-backend\
Copy-Item -Force .\dist\huggingface-backend-space\.dockerignore .\orsa-backend\
```

Commit and push:

```powershell
cd .\orsa-backend
git add .
git commit -m "Deploy ORSA backend"
git push
```

Large `model.onnx` files should be tracked by Git LFS because the generated Space includes `.gitattributes`.

## 4. Configure Hugging Face Space Secrets

Set these as Space secrets:

| Secret | Required | Purpose |
| --- | --- | --- |
| `ORSA_SESSION_SECRET` | Yes | Shared JWT secret used by C# to mint and Go to verify sessions. |
| `SUPABASE_DB_CONNECTION_STRING` | Yes | Supabase/Postgres connection string for users/settings/legal data. |
| `MONGODB_ATLAS_URI` | Yes | MongoDB Atlas URI for chat thread persistence. |
| `GPT_OSS_AUTH_TOKEN` | Recommended | Enables GPT-OSS triage through Hugging Face Router. |
| `GITHUB_TOKEN` | Recommended | Enables GitHub Models triage (GPT-4.1, Llama-4-Maverick) and vision analysis in the round-robin pool. |
| `RESEND_API_KEY` | For email verification | Enables sending verification emails via Resend. If unset, email/password signups stay unverified and cannot use chat/upload. |
| `GEMINI_API_KEY` | Recommended | Adds gemini-2.5-flash (text + vision) to the round-robin model pool. |
| `UMLS_API_KEY` | Recommended | Enables UMLS/SNOMED concept enrichment. |
| `GOOGLE_CLIENT_SECRET` | If Google login is enabled | Google OAuth client secret. |

Use a strong production `ORSA_SESSION_SECRET`; do not use the local dev fallback.

## 5. Configure Hugging Face Space Variables

Set these as Space variables:

| Variable | Value |
| --- | --- |
| `CORS_ALLOWED_ORIGINS` | `https://<your-vercel-domain>` |
| `MONGODB_DATABASE` | `orsa` |
| `GPT_OSS_ENDPOINT` | `https://router.huggingface.co/v1` |
| `GPT_OSS_MODEL_ID` | `openai/gpt-oss-120b` |
| `GITHUB_MODELS_BASE` | `https://models.inference.ai.azure.com` |
| `VISION_MODEL_ID` | `meta/Llama-3.2-90B-Vision-Instruct` |
| `GITHUB_GPT_MODEL_ID` | `openai/gpt-4.1` (round-robin text+vision via GitHub Models) |
| `GITHUB_LLAMA_MODEL_ID` | `meta/Llama-4-Maverick-17B-128E-Instruct-FP8` |
| `GEMINI_BASE` | `https://generativelanguage.googleapis.com/v1beta/openai` |
| `GEMINI_MODEL_ID` | `gemini-2.5-flash` |
| `RESEND_FROM` | `ORSA <noreply@your-verified-domain>` (must be a Resend-verified sender) |
| `APP_BASE_URL` | `https://<your-vercel-domain>` (used to build the email verification link) |
| `GOOGLE_CLIENT_ID` | Your Google OAuth client id, if enabled. |
| `GOOGLE_REDIRECT_URI` | `https://<your-vercel-domain>/auth/google/callback`, if enabled. |

The Docker image sets these internal defaults:

| Variable | Default |
| --- | --- |
| `HTTP_PORT` | `3000` |
| `GRPC_PORT` | `50053` |
| `HEALTH_PORT` | `8085` |
| `CSHARP_SUPABASE_GRPC_URL` | `127.0.0.1:50053` |
| `BERT_ESI_ONNX_DIR` | `/home/user/app/models/bert_esi_onnx` |

Do not override those defaults unless you also update Nginx and startup wiring.

## 6. Verify the Backend Space

After the Space finishes building, check:

```powershell
curl https://<owner>-orsa-backend.hf.space/healthz
curl https://<owner>-orsa-backend.hf.space/csharp/healthz
```

Expected:

- `/healthz` returns the Go gateway health response.
- `/csharp/healthz` returns the C# service health response.

If `/healthz` works but `/csharp/healthz` fails, inspect the Space logs for C# startup/database errors.

If `/csharp/healthz` works but `/healthz` fails, inspect the Space logs for Go startup, Mongo, ONNX, or config errors.

## 7. Configure Vercel

The repository root has `vercel.json`.

Set this Vercel environment variable for production and preview:

| Variable | Value |
| --- | --- |
| `ORSA_BACKEND_URL` | `https://<owner>-orsa-backend.hf.space` |

No trailing slash.

Vercel build settings are controlled by `vercel.json`:

- Build command: `cd frontend && npm ci && npm run build`
- Output directory: `frontend/dist/frontend/browser`
- SPA fallback: `/index.html`
- API proxy: `${ORSA_BACKEND_URL}`

Deploy:

```powershell
vercel --prod
```

Or connect the repository in the Vercel dashboard and let Vercel deploy from git.

## 8. Verify Vercel

After Vercel deploys:

```powershell
curl https://<your-vercel-domain>/api/healthz
```

Then open:

```text
https://<your-vercel-domain>
```

Check:

- Email registration.
- Email login.
- Chat page loads.
- A basic chat turn reaches Go.
- Settings/profile pages reach C# through Go gRPC.
- Attachment upload quota is enforced.

## 9. Google OAuth Checklist

If using Google login:

1. In Google Cloud Console, add this authorized redirect URI:

```text
https://<your-vercel-domain>/auth/google/callback
```

2. Set the same value in the Hugging Face Space variable:

```text
GOOGLE_REDIRECT_URI=https://<your-vercel-domain>/auth/google/callback
```

3. Set `GOOGLE_CLIENT_ID` as a Space variable.
4. Set `GOOGLE_CLIENT_SECRET` as a Space secret.
5. Verify `/api/auth/google` redirects to Google.

## 10. Local Backend Container Test

From the repository root:

```powershell
docker build -f deploy/huggingface/backend/Dockerfile -t orsa-hf-backend .
docker run --rm -p 7860:7860 --env-file .env orsa-hf-backend
```

Then test:

```powershell
curl http://127.0.0.1:7860/healthz
curl http://127.0.0.1:7860/csharp/healthz
```

## Notes

- Hugging Face Spaces restart containers; persistent data should live in Supabase/Postgres and MongoDB Atlas, not on local disk.
- The backend Space can run without GPT-OSS, vision, UMLS, MongoDB, or BERT, but it will use degraded safe fallbacks.
- Production should not rely on in-memory database or chat persistence fallbacks.
- Keep the backend Space private unless the API is intentionally public.
