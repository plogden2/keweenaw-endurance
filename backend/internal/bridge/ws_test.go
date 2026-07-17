package bridge

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWSReadSender_MessageIncludesSourceLapID(t *testing.T) {
	lap := PendingLap{
		ID:          uuid.New().String(),
		LogicalUUID: "9fe78eeb-a21c-594a-acc2-7e1efe378201",
		TS:          time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
		DeviceID:    "laptop-finish-1",
	}

	msg := services.BridgeMessage{
		Type:        "read",
		LogicalUUID: lap.LogicalUUID,
		SourceLapID: lap.ID,
		TS:          lap.TS.UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"source_lap_id":"`+lap.ID+`"`)
}
