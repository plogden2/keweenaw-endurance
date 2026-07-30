# Live monitor for wave-through RFID tap testing.
# Usage (tag already programmed, bridge running):
#   powershell -File scripts\watch-tap-test.ps1
#
# Keep the chip OFF the antenna between taps. Wave on briefly, lift off.
# Each accepted read prints a line and the bridge should beep.

$ErrorActionPreference = 'Stop'
$url = 'http://127.0.0.1:8091/status'
Write-Host @"
=== Instant / wave-through tap test ===
Bridge: $url
1. Leave the chip OFF the HF antenna.
2. Touch it briefly to the HF coil, then lift off.
3. Watch for TAP lines + listen for beeps.
4. Ctrl+C to stop.

"@

$lastAt = ''
$n = 0
while ($true) {
  try {
    $st = Invoke-RestMethod -Uri $url -TimeoutSec 2
    $at = [string]$st.last_read_at
    $uid = [string]$st.last_read
    if ($at -and $at -ne $lastAt) {
      $n++
      $ago = ''
      try {
        $ts = [datetime]::Parse($at).ToUniversalTime()
        $ago = ('{0:n0}ms ago' -f (([datetime]::UtcNow) - $ts).TotalMilliseconds)
      } catch {}
      $short = if ($uid.Length -gt 13) { $uid.Substring(0, 13) + '…' } else { $uid }
      Write-Host ("[{0:HH:mm:ss.fff}] TAP #{1}  {2}  {3}  mode={4}" -f (Get-Date), $n, $short, $ago, $st.mode) -ForegroundColor Green
      $lastAt = $at
    } else {
      Write-Host ("`r[{0:HH:mm:ss}] waiting for tap…  last={1}   " -f (Get-Date), $(if ($lastAt) { $lastAt } else { 'none' })) -NoNewline
    }
  } catch {
    Write-Host ("`r[{0:HH:mm:ss}] bridge unreachable: {1}   " -f (Get-Date), $_.Exception.Message) -NoNewline
  }
  Start-Sleep -Milliseconds 100
}
