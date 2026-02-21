package cli

import (
	"circular/internal/core/config"
	"circular/internal/engine/parser"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
)

func runGrammarsCommand(cfg *config.Config, args []string) int {
	if len(args) == 0 {
		printGrammarHelp()
		return 1
	}

	cmd := args[0]
	switch cmd {
	case "add":
		if len(args) < 3 {
			fmt.Println("Usage: circular grammars add <name> <repo_url>")
			return 1
		}
		return runGrammarsAdd(cfg, args[1], args[2])
	case "list":
		return runGrammarsList(cfg)
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: circular grammars remove <name>")
			return 1
		}
		return runGrammarsRemove(cfg, args[1])
	default:
		fmt.Printf("Unknown grammar command: %s\n", cmd)
		printGrammarHelp()
		return 1
	}
}

func printGrammarHelp() {
	fmt.Println("Usage: circular grammars <command> [args]")
	fmt.Println("\nCommands:")
	fmt.Println("  add <name> <repo_url>   Download and build a grammar")
	fmt.Println("  list                    List installed grammars")
	fmt.Println("  remove <name>           Remove a grammar")
}

func runGrammarsAdd(cfg *config.Config, name, url string) int {
	fmt.Printf("Building grammar %s from %s...\n", name, url)

	builder, err := parser.NewBuilder()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing builder: %v\n", err)
		return 1
	}
	defer builder.Cleanup()

	soPath, nodeTypesPath, err := builder.Build(name, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		return 1
	}

	// Move artifacts
	destDir := filepath.Join(cfg.GrammarsPath, name)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create destination dir: %v\n", err)
		return 1
	}

	destSO := filepath.Join(destDir, filepath.Base(soPath))
	if err := copyFile(soPath, destSO); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to copy shared object: %v\n", err)
		return 1
	}
	destNodeTypes := filepath.Join(destDir, "node-types.json")
	if err := copyFile(nodeTypesPath, destNodeTypes); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to copy node-types.json: %v\n", err)
		return 1
	}

	// Update Manifest
	manifestPath := filepath.Join(cfg.GrammarsPath, "manifest.toml")
	m, err := parser.LoadManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			m = &parser.Manifest{Version: 1}
		} else {
			fmt.Fprintf(os.Stderr, "Failed to load manifest: %v\n", err)
			return 1
		}
	}

	soHash, _ := parser.CalculateSHA256(destSO)
	ntHash, _ := parser.CalculateSHA256(destNodeTypes)

	m.AddArtifact(parser.Artifact{
		Language:        name,
		AIBVersion:      15, // Defaulting to 15
		SOPath:          filepath.Join(name, filepath.Base(soPath)),
		SOSHA256:        soHash,
		NodeTypesPath:   filepath.Join(name, "node-types.json"),
		NodeTypesSHA256: ntHash,
		Source:          url,
	})

	if err := m.Save(manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save manifest: %v\n", err)
		return 1
	}

	fmt.Printf("Grammar %s installed successfully.\n", name)
	return 0
}

func runGrammarsList(cfg *config.Config) int {
	manifestPath := filepath.Join(cfg.GrammarsPath, "manifest.toml")
	m, err := parser.LoadManifest(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load manifest: %v\n", err)
		return 1
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "LANGUAGE\tVERSION\tSOURCE\tAPPROVED")
	for _, art := range m.Artifacts {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", art.Language, art.AIBVersion, art.Source, art.ApprovedDate)
	}
	w.Flush()
	return 0
}

func runGrammarsRemove(cfg *config.Config, name string) int {
	manifestPath := filepath.Join(cfg.GrammarsPath, "manifest.toml")
	m, err := parser.LoadManifest(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load manifest: %v\n", err)
		return 1
	}

	m.RemoveArtifact(name)
	if err := m.Save(manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save manifest: %v\n", err)
		return 1
	}

	// Remove dir
	destDir := filepath.Join(cfg.GrammarsPath, name)
	if err := os.RemoveAll(destDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to remove directory: %v\n", err)
		return 1
	}

	fmt.Printf("Grammar %s removed.\n", name)
	return 0
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
