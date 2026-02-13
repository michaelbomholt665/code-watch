package drivers

import (
	"path/filepath"
	"strings"
)

type JavaScriptResolver struct{}

func NewJavaScriptResolver() *JavaScriptResolver {
	return &JavaScriptResolver{}
}

func (r *JavaScriptResolver) ResolveModuleName(modulePath string) string {
	modulePath = strings.TrimSpace(modulePath)
	modulePath = strings.Trim(modulePath, "\"'`")
	modulePath = strings.TrimPrefix(modulePath, "node:")

	for strings.HasPrefix(modulePath, "./") {
		modulePath = strings.TrimPrefix(modulePath, "./")
	}
	for strings.HasPrefix(modulePath, "../") {
		modulePath = strings.TrimPrefix(modulePath, "../")
	}

	if modulePath == "" {
		return ""
	}

	parts := strings.Split(modulePath, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" || part == "." || part == ".." {
			continue
		}
		ext := filepath.Ext(part)
		if ext != "" {
			part = strings.TrimSuffix(part, ext)
		}
		if part != "" {
			return part
		}
	}

	return modulePath
}
