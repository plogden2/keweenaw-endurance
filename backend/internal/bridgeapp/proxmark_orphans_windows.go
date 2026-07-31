//go:build windows

package bridgeapp

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// KillProxmarkOrphans terminates proxmark3.exe processes that are not children
// of keepParentPID (pass os.Getpid() from reader-gui to preserve the active arm).
// Returns how many processes were killed.
func KillProxmarkOrphans(keepParentPID int) (int, error) {
	procs, err := listProcessSnapshots()
	if err != nil {
		return 0, err
	}
	pids := orphanProxmarkPIDs(procs, keepParentPID)
	killed := 0
	var errs []string
	for _, pid := range pids {
		if err := terminatePID(pid); err != nil {
			errs = append(errs, fmt.Sprintf("pid %d: %v", pid, err))
			continue
		}
		killed++
	}
	if len(errs) > 0 {
		return killed, fmt.Errorf("kill orphans: %s", strings.Join(errs, "; "))
	}
	return killed, nil
}

func listProcessSnapshots() ([]procSnapshot, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("process snapshot: %w", err)
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snap, &entry); err != nil {
		return nil, fmt.Errorf("process32 first: %w", err)
	}

	var out []procSnapshot
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		out = append(out, procSnapshot{
			PID:       int(entry.ProcessID),
			ParentPID: int(entry.ParentProcessID),
			Name:      name,
		})
		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	return out, nil
}

func terminatePID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid")
	}
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.TerminateProcess(h, 1)
}
