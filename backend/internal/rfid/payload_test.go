package rfid

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPayloadRoundTrip(t *testing.T) {
	id := uuid.MustParse("1441674d-a011-471a-a601-722b88b117f5")
	raw, err := EncodeLogicalUUID(id.String())
	require.NoError(t, err)
	require.Len(t, raw, 16)
	got, err := DecodeLogicalUUID(raw)
	require.NoError(t, err)
	require.Equal(t, id.String(), got)
}

func TestEncodeRejectsNonUUID(t *testing.T) {
	_, err := EncodeLogicalUUID("DEMO-TAG-0001")
	require.Error(t, err)
}

func TestEncodeLogicalUUIDHex(t *testing.T) {
	id := uuid.MustParse("1441674d-a011-471a-a601-722b88b117f5")
	got, err := EncodeLogicalUUIDHex(id.String())
	require.NoError(t, err)
	require.Equal(t, "1441674da011471aa601722b88b117f5", got)
}

func TestDecodeLogicalUUIDShortPayload(t *testing.T) {
	_, err := DecodeLogicalUUID([]byte{1, 2, 3})
	require.Error(t, err)
	require.Contains(t, err.Error(), "payload too short")
}
