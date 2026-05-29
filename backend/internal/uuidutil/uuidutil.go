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
	idSuffixSQLExpr = "RIGHT(id::text, 6)"
)

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

// Parse accepts a full UUID or a six-character suffix (suffix cannot be resolved without a database).
func Parse(value string) (uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil, ErrInvalidID
	}
	if id, err := uuid.Parse(value); err == nil {
		return id, nil
	}
	if len(value) == SuffixLength && suffixHex.MatchString(strings.ToLower(value)) {
		return uuid.Nil, fmt.Errorf("%w: short id requires database lookup", ErrInvalidID)
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
	if err := db.Model(model).
		Where(fmt.Sprintf("%s = ?", idSuffixSQLExpr), value).
		Pluck("id", &ids).Error; err != nil {
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
