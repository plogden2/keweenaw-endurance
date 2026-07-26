//go:build windows

package bridgeapp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ListSerialPorts returns COM port names available on this Windows machine.
func ListSerialPorts() ([]string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DEVICEMAP\SERIALCOMM`, registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("open SERIALCOMM: %w", err)
	}
	defer k.Close()

	names, err := k.ReadValueNames(0)
	if err != nil {
		return nil, err
	}
	ports := make([]string, 0, len(names))
	for _, name := range names {
		val, _, err := k.GetStringValue(name)
		if err != nil {
			continue
		}
		val = strings.TrimSpace(val)
		if val != "" {
			ports = append(ports, val)
		}
	}
	sort.Slice(ports, func(i, j int) bool {
		return comSortKey(ports[i]) < comSortKey(ports[j])
	})
	return ports, nil
}

func comSortKey(port string) int {
	p := strings.ToUpper(strings.TrimSpace(port))
	if strings.HasPrefix(p, "COM") {
		n, err := strconv.Atoi(strings.TrimPrefix(p, "COM"))
		if err == nil {
			return n
		}
	}
	return 9999
}
