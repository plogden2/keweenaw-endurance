# Copy a slim Proxmark3 Windows runtime (exe + walked MinGW/Qt DLLs + platforms plugin).
param(
  [Parameter(Mandatory = $true)][string]$DestDir,
  [string]$ProxSpaceRoot = "C:\Users\gener\sdk\ProxSpace",
  [string]$ProxmarkExe = ""
)

$ErrorActionPreference = "Stop"

if (-not $ProxmarkExe) {
  $ProxmarkExe = Join-Path $ProxSpaceRoot "pm3\proxmark3\client\proxmark3.exe"
}
if (-not (Test-Path $ProxmarkExe)) {
  throw "proxmark3.exe not found at $ProxmarkExe"
}

$mingwBin = Join-Path $ProxSpaceRoot "msys2\mingw64\bin"
$objdump = Join-Path $mingwBin "objdump.exe"
if (-not (Test-Path $objdump)) {
  throw "objdump.exe not found at $objdump (need ProxSpace mingw64)"
}

$systemDlls = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
@(
  'ADVAPI32.dll','KERNEL32.dll','WS2_32.dll','msvcrt.dll','USER32.dll','GDI32.dll','SHELL32.dll',
  'ole32.dll','OLEAUT32.dll','COMDLG32.dll','COMCTL32.dll','WINMM.dll','IMM32.dll','VERSION.dll',
  'SETUPAPI.dll','CRYPT32.dll','bcrypt.dll','ntdll.dll','RPCRT4.dll','SHLWAPI.dll','UxTheme.dll',
  'dwmapi.dll','WINSPOOL.DRV','WLDAP32.dll','IPHLPAPI.DLL','DNSAPI.dll','Secur32.dll','NETAPI32.dll',
  'USERENV.dll','PROPSYS.dll','CLBCatQ.dll','MSIMG32.dll','OpenGL32.dll','DWrite.dll','d3d11.dll',
  'dxgi.dll','d3d9.dll','HID.DLL','CFGMGR32.dll','powrprof.dll','WTSAPI32.dll','CRYPTBASE.dll',
  'SSPICLI.DLL','sechost.dll','MPR.dll','NSI.dll','PSAPI.DLL','dbghelp.dll','WINTRUST.dll'
) | ForEach-Object { [void]$systemDlls.Add($_) }

$searchDirs = @(
  (Split-Path -Parent $ProxmarkExe),
  $mingwBin,
  (Join-Path $ProxSpaceRoot "msys2\mingw64\lib")
)

function Get-DllDeps([string]$path) {
  & $objdump -p $path 2>$null |
    Select-String "DLL Name:\s+(\S+)" |
    ForEach-Object { $_.Matches.Groups[1].Value }
}

function Find-Dll([string]$name) {
  foreach ($d in $searchDirs) {
    $p = Join-Path $d $name
    if (Test-Path $p) { return (Resolve-Path $p).Path }
  }
  $null
}

New-Item -ItemType Directory -Force -Path $DestDir | Out-Null
Copy-Item -Force $ProxmarkExe (Join-Path $DestDir "proxmark3.exe")

$queue = [System.Collections.Generic.Queue[string]]::new()
$visited = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
$queue.Enqueue((Resolve-Path (Join-Path $DestDir "proxmark3.exe")).Path)
[void]$visited.Add((Resolve-Path (Join-Path $DestDir "proxmark3.exe")).Path)

# Also walk original exe (same binary) deps from mingw search path
$queue.Enqueue((Resolve-Path $ProxmarkExe).Path)
[void]$visited.Add((Resolve-Path $ProxmarkExe).Path)

while ($queue.Count -gt 0) {
  $cur = $queue.Dequeue()
  foreach ($dll in (Get-DllDeps $cur)) {
    if ($systemDlls.Contains($dll)) { continue }
    if ($dll -match '^api-ms-win-') { continue }
    $found = Find-Dll $dll
    if (-not $found) { continue }
    $target = Join-Path $DestDir (Split-Path -Leaf $found)
    if (-not (Test-Path $target)) {
      Copy-Item -Force $found $target
    }
    if ($visited.Add($found)) {
      $queue.Enqueue($found)
    }
  }
}

$qwindows = Join-Path $ProxSpaceRoot "msys2\mingw64\share\qt5\plugins\platforms\qwindows.dll"
if (-not (Test-Path $qwindows)) {
  throw "qwindows.dll not found at $qwindows"
}
$platforms = Join-Path $DestDir "platforms"
New-Item -ItemType Directory -Force -Path $platforms | Out-Null
Copy-Item -Force $qwindows (Join-Path $platforms "qwindows.dll")

# Walk qwindows deps too
$queue.Enqueue((Resolve-Path (Join-Path $platforms "qwindows.dll")).Path)
while ($queue.Count -gt 0) {
  $cur = $queue.Dequeue()
  foreach ($dll in (Get-DllDeps $cur)) {
    if ($systemDlls.Contains($dll)) { continue }
    if ($dll -match '^api-ms-win-') { continue }
    $found = Find-Dll $dll
    if (-not $found) { continue }
    $target = Join-Path $DestDir (Split-Path -Leaf $found)
    if (-not (Test-Path $target)) {
      Copy-Item -Force $found $target
    }
    if ($visited.Add($found)) {
      $queue.Enqueue($found)
    }
  }
}

$count = (Get-ChildItem $DestDir -Recurse -File).Count
$sizeMB = [math]::Round(((Get-ChildItem $DestDir -Recurse -File | Measure-Object Length -Sum).Sum / 1MB), 1)
Write-Host "Proxmark runtime: $count files, ${sizeMB} MB -> $DestDir"
