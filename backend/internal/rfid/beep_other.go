//go:build !windows

package rfid

import (
	"fmt"
	"os"
)

type asciiBeeper struct{}

func (asciiBeeper) Beep() {
	fmt.Fprint(os.Stderr, "\a")
}

func defaultBeeper() Beeper {
	return asciiBeeper{}
}

// PlayTapBeep is the non-Windows tap feedback (terminal bell).
func PlayTapBeep() {
	fmt.Fprint(os.Stderr, "\a")
}

// PrewarmTapBeep is a no-op outside Windows.
func PrewarmTapBeep() {}
