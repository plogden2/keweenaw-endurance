package uuidutil

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type resolveTestRow struct {
	ID PublicUUID `gorm:"type:uuid;primary_key"`
}

func (resolveTestRow) TableName() string {
	return "resolve_test_rows"
}

func TestResolveShortIDSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&resolveTestRow{}))

	id := uuid.New()
	require.NoError(t, db.Create(&resolveTestRow{ID: PublicUUID(id)}).Error)

	resolved, err := Resolve(db, &resolveTestRow{}, Suffix(id))
	require.NoError(t, err)
	assert.Equal(t, id, resolved)
}
