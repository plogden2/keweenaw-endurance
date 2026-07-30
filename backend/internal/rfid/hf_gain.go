package rfid

const (
	HFGainMin     = 1
	HFGainMax     = 63
	HFGainDefault = 63
	// HFThreshMinSafe is the lowest Proxmark HF reader threshold that still
	// completes ISO14443-A anticollision on the finish-line RRG client.
	// Gain 63 maps to raw thresh 1, which returns BCC0 errors and aborts reads.
	HFThreshMinSafe = 3
)

func ClampHFGain(g int) int {
	if g < HFGainMin || g > HFGainMax {
		return HFGainDefault
	}
	return g
}

func HFThreshFromGain(g int) int {
	g = ClampHFGain(g)
	t := 64 - g
	if t < HFThreshMinSafe {
		return HFThreshMinSafe
	}
	return t
}
