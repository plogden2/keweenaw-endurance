package rfid

import (
	"errors"
	"os"
)

var ErrHardwareUnavailable = errors.New("rfid hardware unavailable")

// Reader abstracts Proxmark3 (or compatible) RFID hardware.
type Reader interface {
	WriteTag(tagUID, participantID string) error
	IsAvailable() bool
}

// Proxmark3 wraps a Reader for field deployment.
type Proxmark3 struct {
	reader Reader
}

func NewProxmark3(reader Reader) *Proxmark3 {
	return &Proxmark3{reader: reader}
}

func (p *Proxmark3) WriteTag(tagUID, participantID string) error {
	if p == nil || p.reader == nil || !p.reader.IsAvailable() {
		return ErrHardwareUnavailable
	}
	return p.reader.WriteTag(tagUID, participantID)
}

func (p *Proxmark3) IsAvailable() bool {
	return p != nil && p.reader != nil && p.reader.IsAvailable()
}

// MockReader records writes for unit tests.
type MockReader struct {
	Available bool
	WriteErr  error
	LastUID   string
	LastData  string
}

func NewMockReader() *MockReader {
	return &MockReader{Available: true}
}

func (m *MockReader) WriteTag(tagUID, participantID string) error {
	if !m.Available {
		return ErrHardwareUnavailable
	}
	if m.WriteErr != nil {
		return m.WriteErr
	}
	m.LastUID = tagUID
	m.LastData = participantID
	return nil
}

func (m *MockReader) IsAvailable() bool {
	return m.Available
}

// NoOpReader is used when no hardware is connected.
type NoOpReader struct{}

func (n *NoOpReader) WriteTag(string, string) error {
	return ErrHardwareUnavailable
}

func (n *NoOpReader) IsAvailable() bool {
	return false
}

// DefaultReader returns a mock reader in test environments or when
// PROXMARK3_ENABLED=true; otherwise a no-op reader.
func DefaultReader() Reader {
	if os.Getenv("GO_ENV") == "test" || os.Getenv("PROXMARK3_ENABLED") == "true" {
		return NewMockReader()
	}
	return &NoOpReader{}
}
