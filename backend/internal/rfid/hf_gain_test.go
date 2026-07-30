package rfid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHFThreshFromGain(t *testing.T) {
	// Raw mapping would be 1, but thresh < 3 breaks anticollision on this PM3.
	assert.Equal(t, HFThreshMinSafe, HFThreshFromGain(63))
	assert.Equal(t, HFThreshMinSafe, HFThreshFromGain(62)) // raw 2 → floor
	assert.Equal(t, 63, HFThreshFromGain(1))
	assert.Equal(t, 7, HFThreshFromGain(57)) // 64-57
}

func TestClampHFGain(t *testing.T) {
	assert.Equal(t, 63, ClampHFGain(0))
	assert.Equal(t, 63, ClampHFGain(99))
	assert.Equal(t, 1, ClampHFGain(1))
	assert.Equal(t, HFGainDefault, ClampHFGain(-1))
}
