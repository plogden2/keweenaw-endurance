// Package eventpolicy holds small, dependency-free event identity helpers
// shared across bridgeapp, services, and handlers. It must NOT import
// services or bridgeapp to avoid import cycles.
package eventpolicy

import "strings"

// Canonical All You Can East Bluffet 2026 event IDs (seed + production short form).
const (
	BluffetEventIDFull  = "1441674d-a011-471a-a601-722b88b117f5"
	BluffetEventIDShort = "b117f5"
)

// IsBluffetEventID reports whether id refers to the All You Can East Bluffet
// 2026 event. Matches the full UUID, the known short public id, and any
// suffix-matched short id (case-insensitive), mirroring bridgeapp.CanonicalEventID.
func IsBluffetEventID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	lower := strings.ToLower(id)
	fullLower := strings.ToLower(BluffetEventIDFull)
	if lower == fullLower || lower == BluffetEventIDShort {
		return true
	}
	if len(lower) == len(BluffetEventIDShort) && strings.HasSuffix(fullLower, lower) {
		return true
	}
	return false
}
