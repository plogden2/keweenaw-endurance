package bridgeapp

import "strings"

type procSnapshot struct {
	PID       int
	ParentPID int
	Name      string
}

func isProxmarkProcess(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "proxmark3.exe" || n == "proxmark3"
}

// orphanProxmarkPIDs returns proxmark PIDs whose parent is not keepParentPID.
// If keepParentPID is 0, every proxmark process is treated as an orphan.
func orphanProxmarkPIDs(procs []procSnapshot, keepParentPID int) []int {
	var out []int
	for _, p := range procs {
		if !isProxmarkProcess(p.Name) {
			continue
		}
		if keepParentPID != 0 && p.ParentPID == keepParentPID {
			continue
		}
		out = append(out, p.PID)
	}
	return out
}
