//go:build windows

package rfid

import (
	"encoding/binary"
	"math"
	"sync"
	"syscall"
	"unsafe"
)

const (
	sndAsync     = 0x0001
	sndNoDefault = 0x0002
	sndMemory    = 0x0004
)

var (
	modWinmm       = syscall.NewLazyDLL("winmm.dll")
	procPlaySoundA = modWinmm.NewProc("PlaySoundA") // SND_MEMORY expects byte* via PlaySoundA

	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	procBeep    = modKernel32.NewProc("Beep")

	beepWAVOnce sync.Once
	beepWAV     []byte
	beepSilent  []byte
)

type windowsBeeper struct{}

func (windowsBeeper) Beep() {
	PlayTapBeep()
}

func defaultBeeper() Beeper {
	return windowsBeeper{}
}

// PlayTapBeep plays the finish-line tap tone via the sound card (WAV).
// kernel32.Beep / MessageBeep are often silent on modern Windows sound schemes.
//
// SND_ASYNC|SND_MEMORY is unreliable on Windows (often plays synchronously and
// can delay the caller). Always synthesize once, then play from a goroutine so
// the RFID path never waits on the audio stack.
func PlayTapBeep() {
	PrewarmTapBeep()
	if len(beepWAV) == 0 {
		go func() { _, _, _ = procBeep.Call(1000, 120) }()
		return
	}
	go playTapBeepWAV()
}

// PrewarmTapBeep synthesizes the WAV so the first tap does not pay synth cost
// on the critical path. A short silent PlaySound primes winmm once at startup.
func PrewarmTapBeep() {
	beepWAVOnce.Do(func() {
		beepWAV = synthBeepWAV(1200, 120, 22050)
		beepSilent = synthBeepWAV(1200, 30, 22050)
		for i := 44; i < len(beepSilent); i++ {
			beepSilent[i] = 0
		}
		if len(beepSilent) > 0 {
			// Synchronous prime (no SND_ASYNC): buffer stays valid; ~30ms once.
			_, _, _ = procPlaySoundA.Call(
				uintptr(unsafe.Pointer(&beepSilent[0])),
				0,
				sndMemory|sndNoDefault,
			)
		}
	})
}

func playTapBeepWAV() {
	// Buffer must stay alive for SND_MEMORY — package-level beepWAV.
	r1, _, _ := procPlaySoundA.Call(
		uintptr(unsafe.Pointer(&beepWAV[0])),
		0,
		sndAsync|sndMemory|sndNoDefault,
	)
	if r1 == 0 {
		_, _, _ = procBeep.Call(1200, 120)
	}
}

func synthBeepWAV(freqHz, durationMs, sampleRate int) []byte {
	n := sampleRate * durationMs / 1000
	if n <= 0 {
		return nil
	}
	dataBytes := n * 2
	buf := make([]byte, 44+dataBytes)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataBytes))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16) // PCM chunk size
	binary.LittleEndian.PutUint16(buf[20:22], 1)  // PCM
	binary.LittleEndian.PutUint16(buf[22:24], 1)  // mono
	binary.LittleEndian.PutUint32(buf[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(buf[32:34], 2)  // block align
	binary.LittleEndian.PutUint16(buf[34:36], 16) // bits
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataBytes))

	amp := 0.55 * 32767
	for i := 0; i < n; i++ {
		// Short attack/release so the tone is obvious on laptop speakers.
		env := 1.0
		if i < sampleRate/100 {
			env = float64(i) / float64(sampleRate/100)
		} else if rem := n - i; rem < sampleRate/50 {
			env = float64(rem) / float64(sampleRate/50)
		}
		sample := int16(amp * env * math.Sin(2*math.Pi*float64(freqHz)*float64(i)/float64(sampleRate)))
		binary.LittleEndian.PutUint16(buf[44+i*2:], uint16(sample))
	}
	return buf
}
