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
