//go:build !windows

package bridgeapp

// ListSerialPorts is a no-op off Windows.
func ListSerialPorts() ([]string, error) {
	return nil, nil
}
