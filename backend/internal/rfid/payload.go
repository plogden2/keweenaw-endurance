package rfid

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func EncodeLogicalUUID(s string) ([]byte, error) {
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("logical tag id must be a UUID: %w", err)
	}
	b := id[:]
	out := make([]byte, 16)
	copy(out, b[:])
	return out, nil
}

func DecodeLogicalUUID(b []byte) (string, error) {
	if len(b) < 16 {
		return "", fmt.Errorf("payload too short")
	}
	id, err := uuid.FromBytes(b[:16])
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func EncodeLogicalUUIDHex(s string) (string, error) {
	raw, err := EncodeLogicalUUID(s)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", raw), nil
}
