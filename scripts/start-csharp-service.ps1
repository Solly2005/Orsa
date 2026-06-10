# Start the C# Supabase engine (gRPC + REST auth).
# Listens on port 50053 (gRPC / HTTP2) and port 8085 (REST auth / HTTP1).
# Run from any directory; the script resolves the repo root automatically.

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path $PSScriptRoot -Parent
$svcDir = Join-Path $repoRoot "services\csharp-supabase"

Set-Location $svcDir
Write-Host "Starting csharp-supabase (gRPC :50053  REST :8085)..."
dotnet run
