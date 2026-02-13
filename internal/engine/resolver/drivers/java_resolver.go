package drivers

import "strings"

type JavaResolver struct{}

func NewJavaResolver() *JavaResolver {
	return &JavaResolver{}
}

func (r *JavaResolver) ResolveModuleName(modulePath string) string {
	modulePath = strings.TrimSpace(modulePath)
	modulePath = strings.TrimSuffix(modulePath, ".*")
	modulePath = strings.TrimPrefix(modulePath, "static ")
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return ""
	}

	parts := strings.Split(modulePath, ".")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part != "" {
			return part
		}
	}
	return modulePath
}
