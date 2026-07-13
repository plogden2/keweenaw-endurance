package rfid

import (
	"errors"
	"os"
	"sync"
)

var ErrHardwareUnavailable = errors.New("rfid hardware unavailable")

// Reader abstracts Proxmark3 (or compatible) RFID hardware.
type Reader interface {
	WriteTag(tagUID, participantID string) error
	Poll() (tagUID string, err error)
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

func (p *Proxmark3) Poll() (string, error) {
	if p == nil || p.reader == nil || !p.reader.IsAvailable() {
		return "", ErrHardwareUnavailable
	}
	return p.reader.Poll()
}

func (p *Proxmark3) IsAvailable() bool {
	return p != nil && p.reader != nil && p.reader.IsAvailable()
}

// MockReader records writes and serves scripted UIDs for unit/e2e tests.
type MockReader struct {
	Available bool
	WriteErr  error
	LastUID   string
	LastData  string

	mu   sync.Mutex
	uids []string
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

// PushUID enqueues a tag UID for a subsequent Poll.
func (m *MockReader) PushUID(uid string) {
	m.Enqueue(uid)
}

// Enqueue is an alias for PushUID.
func (m *MockReader) Enqueue(uid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uids = append(m.uids, uid)
}

func (m *MockReader) Poll() (string, error) {
	if !m.Available {
		return "", ErrHardwareUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.uids) == 0 {
		return "", nil
	}
	uid := m.uids[0]
	m.uids = m.uids[1:]
	return uid, nil
}

func (m *MockReader) IsAvailable() bool {
	return m.Available
}

// NoOpReader is used when no hardware is connected.
type NoOpReader struct{}

func (n *NoOpReader) WriteTag(string, string) error {
	return ErrHardwareUnavailable
}

func (n *NoOpReader) Poll() (string, error) {
	return "", ErrHardwareUnavailable
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
