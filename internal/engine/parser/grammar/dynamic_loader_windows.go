//go:build windows

package grammar

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// LoadDynamic returns an error on Windows as dynamic grammar loading is not yet supported.
func LoadDynamic(path, langName string) (*sitter.Language, error) {
	return nil, fmt.Errorf("dynamic grammar loading is currently not supported on Windows")
}
