---
title: ORSA Backend
sdk: docker
app_port: 7860
---

# ORSA Backend

This Space runs the ORSA backend as separate internal services in one Docker container:

- C# Supabase engine: REST auth on `8085`, gRPC UserService on `50053`
- Go AI gateway: REST API on `3000`, calling C# over local gRPC
- Nginx: public Space entrypoint on `7860`

Public routes:

- `/auth/*` proxies to C# auth REST.
- `/csharp/healthz` proxies to C# health.
- All other paths proxy to the Go AI gateway.

Required runtime secrets:

- `ORSA_SESSION_SECRET`
- `SUPABASE_DB_CONNECTION_STRING`
- `MONGODB_ATLAS_URI`

Recommended runtime variables:

- `CORS_ALLOWED_ORIGINS`
- `GOOGLE_CLIENT_ID`
- `GOOGLE_CLIENT_SECRET`
- `GOOGLE_REDIRECT_URI`
- `GPT_OSS_AUTH_TOKEN`
- `GITHUB_TOKEN`
- `UMLS_API_KEY`
