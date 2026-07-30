package rfid

const (
	HFGainMin     = 1
	HFGainMax     = 63
	HFGainDefault = 63
)

func ClampHFGain(g int) int {
	if g < HFGainMin || g > HFGainMax {
		return HFGainDefault
	}
	return g
}

func HFThreshFromGain(g int) int {
	g = ClampHFGain(g)
	return 64 - g
}
