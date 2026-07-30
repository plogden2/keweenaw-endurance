-- Continuous finish-line arm for Keweenaw Endurance.
-- Stays in one Proxmark client process: wait for card, one Type-2 READ of
-- pages 4–7 (16 bytes), signal the bridge, loop. Avoids COM reconnect latency.
--
-- THRESH is replaced by the Go bridge before launch (hw sethfthresh).

local thresh = tonumber("{{THRESH}}") or 7
core.console(string.format("hw sethfthresh -t %d", thresh))
print("KEWEENAW_ARM_READY")

while true do
  -- -k keeps the field up so the following raw READ does not re-select.
  core.console("hf 14a reader -w --skip -k")
  -- ISO14443 Type-2 READ page 4 returns pages 4–7 (16 bytes) + CRC.
  core.console("hf 14a raw -c -t 500 3004")
  print("KEWEENAW_TAP_END")
end
