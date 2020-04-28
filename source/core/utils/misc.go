package utils

import (
	"fmt"

	"github.com/borchero/switchboard/backends"
)

// FullHost returns the full hostname for the given subdomain and the domain defined by the given
// zone.
func FullHost(host string, zone backends.DNSZone) string {
	domain := zone.Domain()
	if host == "@" {
		return domain
	}
	return fmt.Sprintf("%s.%s", host, domain)
}

// Any returns true if any of the given booleans are true.
func Any(items ...bool) bool {
	for _, item := range items {
		if item {
			return true
		}
	}
	return false
}

// Equal returns whether the two slices contain the same values (in the same order).
func Equal(lhs, rhs []string) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for i := range lhs {
		if lhs[i] != rhs[i] {
			return false
		}
	}
	return true
}
