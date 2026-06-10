# Start the Go AI+Mongo service with the correct 64-bit GCC for CGo (onnxruntime).
# Run from any directory; the script resolves the repo root automatically.

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path $PSScriptRoot -Parent
$svcDir = Join-Path $repoRoot "services\go-ai-mongo"

# The 64-bit MinGW from Chocolatey is required for CGo (onnxruntime_go).
# The 32-bit MinGW at C:\MinGW\bin is first in PATH by default and CANNOT compile
# amd64 ("sorry, unimplemented: 64-bit mode not compiled in"). Pinning CC alone is
# not enough: gcc resolves its own subtools (ld, as, collect2) via PATH, so a 32-bit
# bin that comes first will break the link with a bare "exit status 1". We therefore
# both pin CC and force the 64-bit bin to the front of PATH (and drop the 32-bit ones).
$gcc = "C:\ProgramData\chocolatey\lib\mingw-w64\tools\install\mingw64\bin\gcc.exe"
if (!(Test-Path -LiteralPath $gcc)) {
  throw "64-bit MinGW GCC was not found at $gcc. Install mingw-w64 or set CC to a 64-bit gcc.exe."
}
$mingwBin = Split-Path $gcc -Parent

$env:CC = $gcc
$env:CGO_ENABLED = "1"

# 64-bit bin first; strip the 32-bit C:\MinGW\bin and any mingw32\bin for this session.
$clean = ($env:Path -split ';' | Where-Object { $_ -and $_ -notmatch '\\MinGW\\bin' -and $_ -notmatch 'mingw32\\bin' })
$env:Path = $mingwBin + ';' + ($clean -join ';')

if (!$env:ONNX_RUNTIME_LIB -and $env:ONNXRUNTIME_SHARED_LIBRARY_PATH) {
  $env:ONNX_RUNTIME_LIB = $env:ONNXRUNTIME_SHARED_LIBRARY_PATH
}

Set-Location $svcDir
Write-Host "Starting go-ai-mongo (CC=$($env:CC))..."
go run ./cmd/orsa-ai-mongo
