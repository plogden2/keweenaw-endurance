# Build KeweenawReader-Setup.exe (reader-gui + slim Proxmark) into dist/reader-setup for USB copy.
param(
  [string]$ProxSpaceRoot = "C:\Users\gener\sdk\ProxSpace",
  [switch]$SkipGuiBuild
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$Backend = Join-Path $RepoRoot "backend"
$SetupCmd = Join-Path $Backend "cmd\reader-setup"
$Stage = Join-Path $RepoRoot "dist\reader-setup\stage"
$OutDir = Join-Path $RepoRoot "dist\reader-setup"
$PayloadZip = Join-Path $SetupCmd "payload.zip"
$SetupTxtSrc = Join-Path $RepoRoot "deploy\reader-setup\SETUP.txt"

Write-Host "== Pack reader setup =="

if (Test-Path $Stage) { Remove-Item -Recurse -Force $Stage }
New-Item -ItemType Directory -Force -Path $Stage | Out-Null
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

# 1) reader-gui.exe
$GuiExe = Join-Path $Backend "reader-gui.exe"
if (-not $SkipGuiBuild -or -not (Test-Path $GuiExe)) {
  Write-Host "Building reader-gui.exe..."
  & powershell -File (Join-Path $PSScriptRoot "build-reader-gui.ps1")
}
if (-not (Test-Path $GuiExe)) {
  throw "reader-gui.exe missing at $GuiExe"
}
Copy-Item -Force $GuiExe (Join-Path $Stage "reader-gui.exe")

# 2) slim Proxmark
Write-Host "Copying slim Proxmark runtime..."
& powershell -File (Join-Path $PSScriptRoot "Copy-ProxmarkRuntime.ps1") `
  -DestDir (Join-Path $Stage "proxmark") `
  -ProxSpaceRoot $ProxSpaceRoot

# 3) SETUP.txt + Bluffet paste sheet
Copy-Item -Force $SetupTxtSrc (Join-Path $Stage "SETUP.txt")
$IdsSrc = Join-Path $RepoRoot "deploy\reader-setup\BLUFFET-IDS.txt"
if (Test-Path $IdsSrc) {
  Copy-Item -Force $IdsSrc (Join-Path $Stage "BLUFFET-IDS.txt")
}

# 4) Zip payload for go:embed (overwrite tiny placeholder; restore after build)
$PlaceholderDir = Join-Path $env:TEMP "keweenaw-reader-setup-placeholder"
if (Test-Path $PlaceholderDir) { Remove-Item -Recurse -Force $PlaceholderDir }
New-Item -ItemType Directory -Force -Path $PlaceholderDir | Out-Null
Set-Content -Path (Join-Path $PlaceholderDir "SETUP.txt") -Value 'placeholder - run scripts/pack-reader-setup.ps1' -Encoding UTF8

if (Test-Path $PayloadZip) { Remove-Item -Force $PayloadZip }
Write-Host "Creating payload.zip..."
Compress-Archive -Path (Join-Path $Stage "*") -DestinationPath $PayloadZip -Force

# 5) Build setup.exe (no CGO)
Write-Host "Building KeweenawReader-Setup.exe..."
$env:CGO_ENABLED = "0"
Push-Location $Backend
try {
  go build -o (Join-Path $OutDir "KeweenawReader-Setup.exe") ./cmd/reader-setup
} finally {
  Pop-Location
  # Restore tiny placeholder so the large Proxmark zip is not left for accidental commit
  if (Test-Path $PayloadZip) { Remove-Item -Force $PayloadZip }
  Compress-Archive -Path (Join-Path $PlaceholderDir "*") -DestinationPath $PayloadZip -Force
}

# 6) USB folder extras
$UsbReadme = @"
Keweenaw Reader — USB pack
==========================

1. Copy this whole folder to a USB stick (or just KeweenawReader-Setup.exe).
2. On the race laptop, double-click KeweenawReader-Setup.exe.
3. Follow SETUP.txt after install (also installed next to reader-gui.exe).

Default install: %LOCALAPPDATA%\KeweenawReader\
No admin required. Proxmark must already be flashed; use Zadig once if no COM port appears.

Rebuild this pack from the repo:
  powershell -File scripts\pack-reader-setup.ps1
"@
Set-Content -Path (Join-Path $OutDir "README-USB.txt") -Value $UsbReadme -Encoding UTF8
Copy-Item -Force $SetupTxtSrc (Join-Path $OutDir "SETUP.txt")
if (Test-Path $IdsSrc) {
  Copy-Item -Force $IdsSrc (Join-Path $OutDir "BLUFFET-IDS.txt")
}

$setupSize = [math]::Round((Get-Item (Join-Path $OutDir "KeweenawReader-Setup.exe")).Length / 1MB, 1)
Write-Host ""
Write-Host "Done. USB folder:"
Write-Host "  $OutDir"
Write-Host "  KeweenawReader-Setup.exe ($setupSize MB)"
Write-Host "  README-USB.txt"
Write-Host "  SETUP.txt"
