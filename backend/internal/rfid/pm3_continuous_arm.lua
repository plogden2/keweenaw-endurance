-- Continuous finish-line arm for Keweenaw Endurance.
-- Stays in one Proxmark client process: wait for card, classify by SAK from
-- the select transcript (printed to stdout), read logical UUID, signal the
-- bridge, loop. Avoids COM reconnect latency.
--
-- Ultralight/NTAG: Type-2 READ of pages 4–7 (16 bytes)
-- Classic 1K:      hf mf rdbl block 1 with key FFFFFFFFFFFF
--
-- THRESH is replaced by the Go bridge before launch (hw sethfthresh).

local thresh = tonumber("{{THRESH}}") or 7
core.console(string.format("hw sethfthresh -t %d", thresh))
print("KEWEENAW_ARM_READY")

-- Best-effort SAK parse from the last console line buffer is unavailable in
-- stock pm3 Lua; instead run family reads that leave parseable dumps for Go.
-- Ultralight taps produce the Type-2 raw line; Classic taps produce mf rdbl.
-- Unsupported chips may yield empty dumps — Go soft-skips those taps.
while true do
  -- -k keeps the field up so the following reads do not re-select.
  core.console("hf 14a reader -w --skip -k")
  -- ISO14443 Type-2 READ page 4 returns pages 4–7 (16 bytes) + CRC (NTAG/UL).
  core.console("hf 14a raw -c -t 500 3004")
  -- MIFARE Classic 1K logical UUID in block 1 (default transport key).
  core.console("hf mf rdbl --blk 1 -k FFFFFFFFFFFF")
  print("KEWEENAW_TAP_END")
end
