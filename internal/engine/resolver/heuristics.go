// # internal/resolver/heuristics.go
package resolver

import "strings"

func IsKnownNonModule(name string, excluded []string) bool {
	parts := strings.Split(name, ".")
	prefix := parts[0]

	for _, sym := range excluded {
		if sym == prefix || sym == prefix+"." {
			return true
		}
	}

	return false
}
