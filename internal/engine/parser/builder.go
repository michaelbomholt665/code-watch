package parser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Builder struct {
	WorkDir string
}

func NewBuilder() (*Builder, error) {
	wd, err := os.MkdirTemp("", "circular-build")
	if err != nil {
		return nil, err
	}
	return &Builder{WorkDir: wd}, nil
}

func (b *Builder) Cleanup() {
	os.RemoveAll(b.WorkDir)
}

func (b *Builder) Build(name, repoURL string) (string, string, error) {
	// 1. Git Clone
	repoDir := filepath.Join(b.WorkDir, name)
	if err := runCmd(b.WorkDir, "git", "clone", "--depth", "1", repoURL, repoDir); err != nil {
		return "", "", fmt.Errorf("git clone: %w", err)
	}

	// 2. Tree-sitter generate
	srcDir := filepath.Join(repoDir, "src")
	if _, err := os.Stat(filepath.Join(srcDir, "parser.c")); os.IsNotExist(err) {
		if err := runCmd(repoDir, "tree-sitter", "generate"); err != nil {
			return "", "", fmt.Errorf("tree-sitter generate: %w", err)
		}
	}

	// 3. Compile
	soName := name + ".so"
	if runtime.GOOS == "darwin" {
		soName = name + ".dylib"
	} else if runtime.GOOS == "windows" {
		soName = name + ".dll"
	}
	soPath := filepath.Join(b.WorkDir, soName)

	args := []string{"-o", soPath, "-I", srcDir, "-shared"}
	if runtime.GOOS != "windows" {
		args = append(args, "-fPIC")
	}
	// macOS needs -dynamiclib sometimes but -shared usually works with clang
	// but let's stick to standard flags
	if runtime.GOOS == "darwin" {
		// args = append(args, "-dynamiclib") 
		// clang -shared works on macos too usually.
	}

	args = append(args, filepath.Join(srcDir, "parser.c"))

	scannerC := filepath.Join(srcDir, "scanner.c")
	scannerCC := filepath.Join(srcDir, "scanner.cc")
	
	hasCpp := false
	if _, err := os.Stat(scannerC); err == nil {
		args = append(args, scannerC)
	} else if _, err := os.Stat(scannerCC); err == nil {
		args = append(args, scannerCC)
		hasCpp = true
	}

	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	
	if hasCpp && runtime.GOOS != "windows" {
		// Link stdc++ if C++ scanner
		// But usually we need to use c++ compiler if mixed?
		// cc usually invokes clang/gcc which handles extension.
		// Just linking stdc++ might be enough.
		// Or assume user has c++ installed.
	}

	if err := runCmd(repoDir, cc, args...); err != nil {
		return "", "", fmt.Errorf("compile: %w", err)
	}

	// 4. Node Types
	nodeTypesPath := filepath.Join(srcDir, "node-types.json")
	if _, err := os.Stat(nodeTypesPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("node-types.json not found")
	}

	return soPath, nodeTypesPath, nil
}

func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
