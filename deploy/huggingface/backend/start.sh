#!/usr/bin/env bash
set -euo pipefail

export HTTP_PORT="${HTTP_PORT:-3000}"
export GRPC_PORT="${GRPC_PORT:-50053}"
export HEALTH_PORT="${HEALTH_PORT:-8085}"
export CSHARP_SUPABASE_GRPC_URL="${CSHARP_SUPABASE_GRPC_URL:-127.0.0.1:50053}"
mkdir -p /tmp/orsa-nginx-client-body /tmp/orsa-nginx-proxy /tmp/orsa-nginx-fastcgi /tmp/orsa-nginx-uwsgi /tmp/orsa-nginx-scgi

cleanup() {
  local code=$?
  for pid in "${nginx_pid:-}" "${go_pid:-}" "${csharp_pid:-}"; do
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      kill -TERM "$pid" 2>/dev/null || true
    fi
  done
  wait 2>/dev/null || true
  exit "$code"
}
trap cleanup INT TERM EXIT

dotnet "$APP_HOME/csharp/Orsa.SupabaseEngine.dll" &
csharp_pid=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${HEALTH_PORT}/healthz" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$csharp_pid" 2>/dev/null; then
    echo "C# service exited before becoming healthy" >&2
    exit 1
  fi
  sleep 1
done

orsa-ai-mongo &
go_pid=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${HTTP_PORT}/healthz" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$go_pid" 2>/dev/null; then
    echo "Go service exited before becoming healthy" >&2
    exit 1
  fi
  sleep 1
done

nginx -c "$APP_HOME/deploy/huggingface/backend/nginx.conf" -g "daemon off;" &
nginx_pid=$!

wait -n "$csharp_pid" "$go_pid" "$nginx_pid"
