// # internal/resolver/go_resolver.go
package drivers

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type GoResolver struct {
	goModPath  string
	moduleName string
	moduleRoot string
}

func NewGoResolver() *GoResolver {
	return &GoResolver{}
}

func (r *GoResolver) FindGoMod(startPath string) error {
	current := filepath.Dir(startPath)
	for {
		modPath := filepath.Join(current, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			r.goModPath = modPath
			r.moduleRoot = current
			return r.parseGoMod()
		}

		parent := filepath.Dir(current)
		if parent == current {
			return errors.New("no go.mod found")
		}
		current = parent
	}
}

func (r *GoResolver) parseGoMod() error {
	data, err := os.ReadFile(r.goModPath)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`module\s+(\S+)`)
	matches := re.FindSubmatch(data)
	if len(matches) > 1 {
		r.moduleName = string(matches[1])
	}
	return nil
}

func (r *GoResolver) GetModuleRoot() string {
	return r.moduleRoot
}

func (r *GoResolver) ModulePath() string {
	return r.moduleName
}

func (r *GoResolver) GetModuleName(filePath string) string {
	rel, err := filepath.Rel(r.moduleRoot, filePath)
	if err != nil {
		return ""
	}

	dir := filepath.Dir(rel)
	if dir == "." {
		return r.moduleName
	}

	return r.moduleName + "/" + dir
}

func (r *GoResolver) ModuleBaseName(modulePath string) string {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return ""
	}
	parts := strings.Split(modulePath, "/")
	return parts[len(parts)-1]
}
