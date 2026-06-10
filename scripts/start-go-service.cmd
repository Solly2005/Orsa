@echo off
rem Start the Go AI+Mongo service from cmd.exe with the correct 64-bit GCC for CGo.
rem The 32-bit MinGW at C:\MinGW\bin cannot compile amd64; we pin CC to the 64-bit
rem gcc and put its bin dir FIRST in PATH so all subtools (ld/as/collect2) resolve 64-bit.
setlocal

set "MINGW64=C:\ProgramData\chocolatey\lib\mingw-w64\tools\install\mingw64\bin"
if not exist "%MINGW64%\gcc.exe" (
  echo ERROR: 64-bit MinGW gcc not found at "%MINGW64%\gcc.exe".
  echo Install mingw-w64 ^(choco install mingw^) or edit MINGW64 in this script.
  exit /b 1
)

set "CC=%MINGW64%\gcc.exe"
set "CGO_ENABLED=1"
set "PATH=%MINGW64%;%PATH%"

cd /d "%~dp0..\services\go-ai-mongo"
echo Starting go-ai-mongo (CC=%CC%)...
go run ./cmd/orsa-ai-mongo
