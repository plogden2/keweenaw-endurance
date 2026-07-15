package rfid

import (
	"errors"
	"os"
	"strings"
	"sync"
)

var ErrHardwareUnavailable = errors.New("rfid hardware unavailable")

// Reader abstracts Proxmark3 (or compatible) RFID hardware.
// Identity is always a logical UUID stored in user memory — never the silicon UID.
type Reader interface {
	// WriteLogicalUUID programs the chip currently on the antenna with the racer's logical UUID.
	WriteLogicalUUID(logicalUUID string) error
	// Poll reads user memory and returns the logical UUID, or "" if no tag / empty.
	Poll() (logicalUUID string, err error)
	IsAvailable() bool
}

// Proxmark3 wraps a Reader for field deployment.
type Proxmark3 struct {
	reader Reader
}

func NewProxmark3(reader Reader) *Proxmark3 {
	return &Proxmark3{reader: reader}
}

func (p *Proxmark3) WriteLogicalUUID(logicalUUID string) error {
	if p == nil || p.reader == nil || !p.reader.IsAvailable() {
		return ErrHardwareUnavailable
	}
	return p.reader.WriteLogicalUUID(logicalUUID)
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

// MockReader simulates a single chip's user memory for CI.
type MockReader struct {
	Available bool
	WriteErr  error
	mu        sync.Mutex
	memory    string   // last programmed logical UUID
	queue     []string // inject/scripted polls (optional override)
}

func NewMockReader() *MockReader {
	return &MockReader{Available: true}
}

func (m *MockReader) WriteLogicalUUID(logicalUUID string) error {
	if !m.Available {
		return ErrHardwareUnavailable
	}
	if m.WriteErr != nil {
		return m.WriteErr
	}
	if _, err := EncodeLogicalUUID(logicalUUID); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory = strings.ToLower(logicalUUID)
	return nil
}

func (m *MockReader) Poll() (string, error) {
	if !m.Available {
		return "", ErrHardwareUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.queue) > 0 {
		uid := m.queue[0]
		m.queue = m.queue[1:]
		return uid, nil
	}
	return m.memory, nil
}

// PushUID enqueues a logical UUID for a subsequent Poll.
func (m *MockReader) PushUID(uid string) {
	m.Enqueue(uid)
}

// Enqueue injects a scripted poll result ahead of chip memory.
func (m *MockReader) Enqueue(logicalUUID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queue = append(m.queue, strings.ToLower(logicalUUID))
}

func (m *MockReader) IsAvailable() bool {
	return m.Available
}

// NoOpReader is used when no hardware is connected.
type NoOpReader struct{}

func (n *NoOpReader) WriteLogicalUUID(string) error {
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
