package rfid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyISO14443A_UltralightSAK00(t *testing.T) {
	stdout := "[+] UID: 04 12 34 56 78 9A 80\n[+] ATQA: 00 44\n[+] SAK: 00\n"
	fam, err := ClassifyISO14443A(stdout)
	require.NoError(t, err)
	assert.Equal(t, TagFamilyUltralight, fam)
}

func TestClassifyISO14443A_Classic1K_SAK08(t *testing.T) {
	stdout := "[+] UID: 11 22 33 44\n[+] ATQA: 00 04\n[+] SAK: 08\n"
	fam, err := ClassifyISO14443A(stdout)
	require.NoError(t, err)
	assert.Equal(t, TagFamilyClassic1K, fam)
}

func TestClassifyISO14443A_UnsupportedSAK20(t *testing.T) {
	stdout := "[+] UID: AA BB CC DD\n[+] ATQA: 03 44\n[+] SAK: 20\n"
	fam, err := ClassifyISO14443A(stdout)
	require.NoError(t, err)
	assert.Equal(t, TagFamilyUnsupported, fam)
}

func TestClassifyISO14443A_NoneWhenEmpty(t *testing.T) {
	fam, err := ClassifyISO14443A("")
	require.NoError(t, err)
	assert.Equal(t, TagFamilyNone, fam)
}

func TestClassifyISO14443A_NoneWhenNoSAK(t *testing.T) {
	stdout := "[+] Using UART port COM3\n[+] Communicating with PM3 over USB-CDC\n"
	fam, err := ClassifyISO14443A(stdout)
	require.NoError(t, err)
	assert.Equal(t, TagFamilyNone, fam)
}

func TestClassifyISO14443A_SAKHexVariants(t *testing.T) {
	for _, stdout := range []string{
		"[+] SAK: 0x08\n",
		"[+] SAK : 08 [ ]\n",
		"SAK: 08\n",
	} {
		fam, err := ClassifyISO14443A(stdout)
		require.NoError(t, err, stdout)
		assert.Equal(t, TagFamilyClassic1K, fam, stdout)
	}
}

func TestUnsupportedTagTypeMessage(t *testing.T) {
	msg := unsupportedTagTypeMessage(0x20)
	assert.Contains(t, msg, "unsupported tag type")
	assert.Contains(t, msg, "20")
	assert.Contains(t, msg, "NTAG/Ultralight or Classic 1K")
}
