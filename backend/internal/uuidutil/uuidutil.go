package uuidutil

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const SuffixLength = 6

var (
	ErrInvalidID    = errors.New("invalid id")
	ErrAmbiguousID  = errors.New("ambiguous id")
	suffixHex       = regexp.MustCompile(`^[0-9a-f]{6}$`)
)

func idSuffixWhere(db *gorm.DB, value string) *gorm.DB {
	switch db.Dialector.Name() {
	case "postgres":
		return db.Where("RIGHT(id::text, 6) = ?", value)
	default:
		return db.Where("SUBSTR(CAST(id AS TEXT), -6) = ?", value)
	}
}

// Suffix returns the last six characters of the canonical UUID string (lowercase).
func Suffix(id uuid.UUID) string {
	s := strings.ToLower(id.String())
	if len(s) < SuffixLength {
		return s
	}
	return s[len(s)-SuffixLength:]
}

// PublicUUID is stored as a full UUID in the database but exposed in JSON as a six-character suffix.
type PublicUUID uuid.UUID

func (p PublicUUID) UUID() uuid.UUID {
	return uuid.UUID(p)
}

func (p PublicUUID) IsZero() bool {
	return uuid.UUID(p) == uuid.Nil
}

func (p PublicUUID) String() string {
	return uuid.UUID(p).String()
}

func (p PublicUUID) Short() string {
	return Suffix(p.UUID())
}

func (p PublicUUID) MarshalJSON() ([]byte, error) {
	if p.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(p.Short())
}

func (p *PublicUUID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*p = PublicUUID(uuid.Nil)
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	id, err := Parse(value)
	if err != nil {
		return err
	}
	*p = PublicUUID(id)
	return nil
}

func (p PublicUUID) Value() (driver.Value, error) {
	if p.IsZero() {
		return nil, nil
	}
	return p.UUID().String(), nil
}

func (p *PublicUUID) Scan(value interface{}) error {
	if value == nil {
		*p = PublicUUID(uuid.Nil)
		return nil
	}
	switch v := value.(type) {
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			return err
		}
		*p = PublicUUID(id)
		return nil
	case []byte:
		id, err := uuid.ParseBytes(v)
		if err != nil {
			return err
		}
		*p = PublicUUID(id)
		return nil
	default:
		return fmt.Errorf("unsupported UUID scan type %T", value)
	}
}

// UUIDFromSuffix builds a placeholder UUID whose canonical string ends with the given suffix.
// Used only for JSON unmarshaling of API responses; database lookups must use Resolve.
func UUIDFromSuffix(suffix string) (uuid.UUID, error) {
	suffix = strings.ToLower(strings.TrimSpace(suffix))
	if len(suffix) != SuffixLength || !suffixHex.MatchString(suffix) {
		return uuid.Nil, ErrInvalidID
	}
	return uuid.Parse("00000000-0000-0000-0000-" + strings.Repeat("0", 6) + suffix)
}

// Parse accepts a full UUID or a six-character suffix placeholder for in-memory use.
func Parse(value string) (uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil, ErrInvalidID
	}
	if id, err := uuid.Parse(value); err == nil {
		return id, nil
	}
	if len(value) == SuffixLength && suffixHex.MatchString(strings.ToLower(value)) {
		return UUIDFromSuffix(value)
	}
	return uuid.Nil, ErrInvalidID
}

// Resolve accepts a full UUID or six-character suffix and looks up the matching row id.
func Resolve(db *gorm.DB, model interface{}, value string) (uuid.UUID, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return uuid.Nil, ErrInvalidID
	}

	if id, err := uuid.Parse(value); err == nil {
		return id, nil
	}

	if len(value) != SuffixLength || !suffixHex.MatchString(value) {
		return uuid.Nil, ErrInvalidID
	}

	var ids []uuid.UUID
	if err := idSuffixWhere(db.Model(model), value).Pluck("id", &ids).Error; err != nil {
		return uuid.Nil, err
	}

	switch len(ids) {
	case 0:
		return uuid.Nil, gorm.ErrRecordNotFound
	case 1:
		return ids[0], nil
	default:
		return uuid.Nil, ErrAmbiguousID
	}
}

// NewPublicUUID creates a PublicUUID from a standard UUID.
func NewPublicUUID(id uuid.UUID) PublicUUID {
	return PublicUUID(id)
}
