package uuidutil

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuffix(t *testing.T) {
	id := uuid.MustParse("11111111-1111-4111-8111-111111111101")
	assert.Equal(t, "111101", Suffix(id))
}

func TestPublicUUIDMarshalJSON(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	payload, err := json.Marshal(PublicUUID(id))
	require.NoError(t, err)
	assert.Equal(t, `"440000"`, string(payload))
}

func TestPublicUUIDUnmarshalJSONFullUUID(t *testing.T) {
	var parsed PublicUUID
	require.NoError(t, json.Unmarshal([]byte(`"550e8400-e29b-41d4-a716-446655440000"`), &parsed))
	assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), parsed.UUID())
}
