//go:build !windows

package bridgeapp

// KillProxmarkOrphans is a no-op off Windows.
func KillProxmarkOrphans(keepParentPID int) (int, error) {
	_ = keepParentPID
	return 0, nil
}
