# Build reader-gui.exe (Fyne + CGO). Requires 64-bit MinGW.
$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$Backend = Join-Path $RepoRoot "backend"
$MingwCandidates = @(
  "C:\Users\gener\sdk\ProxSpace\msys2\mingw64\bin",
  "C:\msys64\mingw64\bin",
  "$env:USERPROFILE\sdk\ProxSpace\msys2\mingw64\bin"
)
$mingw = $MingwCandidates | Where-Object { Test-Path (Join-Path $_ "gcc.exe") } | Select-Object -First 1
if (-not $mingw) {
  Write-Error "64-bit mingw64 gcc not found. Install MSYS2 mingw64 or ProxSpace."
}
$env:Path = "$mingw;" + $env:Path
$env:CGO_ENABLED = "1"
$env:CC = Join-Path $mingw "gcc.exe"
Write-Host "Using CC=$env:CC"
Set-Location $Backend
go build -o reader-gui.exe ./cmd/reader-gui
Write-Host "Built $(Join-Path $Backend 'reader-gui.exe')"
