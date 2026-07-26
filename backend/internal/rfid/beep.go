package rfid

// Beeper plays immediate local feedback when a tag UUID is successfully read.
type Beeper interface {
	Beep()
}

type nopBeeper struct{}

func (nopBeeper) Beep() {}

// recordingBeeper is used by unit tests.
type recordingBeeper struct {
	calls int
}

func (b *recordingBeeper) Beep() {
	if b == nil {
		return
	}
	b.calls++
}
