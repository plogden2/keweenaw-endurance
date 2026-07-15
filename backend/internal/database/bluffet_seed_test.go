package database

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Stable IDs matching frontend/e2e/fixtures/rfid.ts BLUFFET + generate_bluffet_seed.py
const (
	bluffetEventID   = "1441674d-a011-471a-a601-722b88b117f5"
	bluffetRace12H   = "17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e"
	bluffetRace6H    = "209769a1-f723-4f70-ae90-466a46338684"
	bluffetRaceKids  = "0e45ee85-800c-4e1f-a95b-4b92462e790a"
	bluffetEventName = "All You Can East Bluffet"
)

func bluffetSeedSQLCandidates(t *testing.T) []string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(thisFile)
	backendRoot := filepath.Clean(filepath.Join(dir, "..", ".."))
	repoRoot := filepath.Clean(filepath.Join(backendRoot, ".."))
	return []string{
		filepath.Join(repoRoot, "database", "seed", "03-bluffet-2026.sql"),
		filepath.Join(backendRoot, "..", "database", "seed", "03-bluffet-2026.sql"),
		"/database/seed/03-bluffet-2026.sql", // docker-compose.test.yml mount
	}
}

func readBluffetSeedSQL(t *testing.T) string {
	t.Helper()
	var lastErr error
	for _, p := range bluffetSeedSQLCandidates(t) {
		raw, err := os.ReadFile(p)
		if err == nil {
			return string(raw)
		}
		lastErr = err
	}
	require.NoError(t, lastErr, "bluffet seed SQL not found in any candidate path")
	return ""
}

// T064 — assert shipped Bluffet seed SQL invariants (3 races, categories, 100 racers).
func TestBluffetSeedSQL_Invariants(t *testing.T) {
	sql := readBluffetSeedSQL(t)

	assert.Contains(t, sql, bluffetEventName)
	assert.Contains(t, sql, bluffetEventID)
	assert.Contains(t, sql, bluffetRace12H)
	assert.Contains(t, sql, bluffetRace6H)
	assert.Contains(t, sql, bluffetRaceKids)
	assert.Contains(t, sql, "2026-08-01")
	assert.Contains(t, sql, "2026-08-01 08:00:00-04")
	assert.Contains(t, sql, "2026-08-01 15:00:00-04")

	assert.Equal(t, 1, strings.Count(sql, "'12 Hour'"))
	assert.Equal(t, 1, strings.Count(sql, "'6 Hour'"))
	assert.Equal(t, 1, strings.Count(sql, "'90-Minute Kids'"))

	// 4 adult cats × 2 races + 2 kids = 10 category name mentions (adults appear twice)
	assert.Equal(t, 2, strings.Count(sql, "'Intermediate Men'"))
	assert.Equal(t, 2, strings.Count(sql, "'Intermediate Women'"))
	assert.Equal(t, 2, strings.Count(sql, "'Advanced Men'"))
	assert.Equal(t, 2, strings.Count(sql, "'Advanced Women'"))
	assert.Contains(t, sql, "'Men'")
	assert.Contains(t, sql, "'Women'")

	assert.NotContains(t, sql, "DEMO-TAG-")
	// Per-race first tags (match frontend/e2e/fixtures/rfid.ts DEMO_TAG_*)
	assert.Contains(t, sql, "23657b2d-aa08-5fe8-8553-e9e3affb4678") // tag:12-hour:1
	assert.Contains(t, sql, "2fe0e039-60a4-50a8-90af-e14ff61371fc") // tag:6-hour:1
	assert.Contains(t, sql, "7dca226d-4eb6-500d-916e-c1044c107ffd") // tag:90-minute-kids:1
	// Consecutive 12-hour tags for multi-station/offline (BLUFFET.demoTags)
	assert.Contains(t, sql, "bdfd9257-7f51-5012-a9b1-a36617846ce5") // tag:12-hour:2
	assert.Contains(t, sql, "cb60c4cd-8c3e-5bbb-be05-e3f6f34c6313") // tag:12-hour:3

	// Each participant INSERT value block includes 'registered'
	assert.Equal(t, 100, strings.Count(sql, "'registered'"))
}
