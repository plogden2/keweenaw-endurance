//go:build windows

package rfid

import "syscall"

var (
	modUser32      = syscall.NewLazyDLL("user32.dll")
	procMessageBeep = modUser32.NewProc("MessageBeep")
)

type windowsBeeper struct{}

func (windowsBeeper) Beep() {
	// MB_OK (0) — standard system beep / default sound.
	_, _, _ = procMessageBeep.Call(0)
}

func defaultBeeper() Beeper {
	return windowsBeeper{}
}
