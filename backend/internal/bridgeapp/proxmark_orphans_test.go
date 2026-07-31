package bridgeapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrphanProxmarkPIDs_KeepsChildrenOfKeepParent(t *testing.T) {
	procs := []procSnapshot{
		{PID: 100, ParentPID: 1, Name: "reader-gui.exe"},
		{PID: 200, ParentPID: 100, Name: "proxmark3.exe"}, // keep
		{PID: 300, ParentPID: 999, Name: "proxmark3.exe"}, // orphan
		{PID: 301, ParentPID: 1, Name: "PROXMARK3.EXE"},   // orphan, case
		{PID: 400, ParentPID: 100, Name: "notepad.exe"},
	}
	got := orphanProxmarkPIDs(procs, 100)
	assert.Equal(t, []int{300, 301}, got)
}

func TestOrphanProxmarkPIDs_AllWhenKeepZero(t *testing.T) {
	procs := []procSnapshot{
		{PID: 200, ParentPID: 100, Name: "proxmark3.exe"},
		{PID: 300, ParentPID: 999, Name: "proxmark3.exe"},
	}
	got := orphanProxmarkPIDs(procs, 0)
	assert.Equal(t, []int{200, 300}, got)
}

func TestOrphanProxmarkPIDs_Empty(t *testing.T) {
	assert.Empty(t, orphanProxmarkPIDs(nil, 100))
}
