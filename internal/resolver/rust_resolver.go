package resolver

import "strings"

type RustResolver struct{}

func NewRustResolver() *RustResolver {
	return &RustResolver{}
}

func (r *RustResolver) ResolveModuleName(modulePath string) string {
	modulePath = strings.TrimSpace(modulePath)
	modulePath = strings.TrimPrefix(modulePath, "crate::")
	modulePath = strings.TrimPrefix(modulePath, "self::")
	modulePath = strings.TrimPrefix(modulePath, "super::")
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return ""
	}

	parts := strings.Split(modulePath, "::")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part != "" {
			return part
		}
	}
	return modulePath
}
