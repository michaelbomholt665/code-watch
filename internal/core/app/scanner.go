package app

import (
	"circular/internal/core/app/helpers"
	"circular/internal/core/ports"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"circular/internal/shared/observability"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobwas/glob"
)

func (a *App) InitialScan(ctx context.Context) error {
	ctx, span := observability.Tracer.Start(ctx, "App.InitialScan")
	defer span.End()

	finalPaths := helpers.UniqueScanRoots(a.Config.WatchPaths)
	expandedPaths := append([]string(nil), finalPaths...)
	for _, p := range finalPaths {
		r := resolver.NewGoResolver()
		if err := r.FindGoMod(p); err == nil {
			if absRoot, err := filepath.Abs(r.GetModuleRoot()); err == nil {
				expandedPaths = append(expandedPaths, filepath.Clean(absRoot))
			}
		}
	}
	finalPaths = helpers.UniqueScanRoots(expandedPaths)

	files, err := a.ScanDirectories(finalPaths, a.Config.Exclude.Dirs, a.Config.Exclude.Files)
	if err != nil {
		return err
	}

	var batch batchUpserter
	if a.symbolStore != nil && !a.Config.WriteQueue.QueueEnabled() {
		b, err := a.symbolStore.BeginBatch()
		if err != nil {
			slog.Warn("failed to begin symbol store batch", "error", err)
		} else {
			batch = b
		}
	}

	for _, filePath := range files {
		if err := a.processFileWithUpserter(filePath, batch); err != nil {
			slog.Warn("failed to process file", "path", filePath, "error", err)
		}
	}
	if batch != nil {
		if err := batch.Commit(); err != nil {
			_ = batch.Rollback()
			return err
		}
	}
	if err := a.enqueueSymbolWrite(ports.WriteRequest{
		Operation: ports.WriteOperationPruneToPaths,
		Paths:     a.currentGraphPaths(),
	}); err != nil {
		slog.Warn("failed to prune persisted symbol rows after initial scan", "error", err)
	}
	return nil
}

func (a *App) ScanDirectories(paths []string, excludeDirs, excludeFiles []string) ([]string, error) {
	var files []string

	dirGlobs := make([]glob.Glob, 0, len(excludeDirs))
	for _, p := range excludeDirs {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude dir pattern %q: %w", p, err)
		}
		dirGlobs = append(dirGlobs, g)
	}

	fileGlobs := make([]glob.Glob, 0, len(excludeFiles))
	for _, p := range excludeFiles {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude file pattern %q: %w", p, err)
		}
		fileGlobs = append(fileGlobs, g)
	}

	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			base := filepath.Base(path)
			if d.IsDir() {
				for _, g := range dirGlobs {
					if g.Match(base) {
						return filepath.SkipDir
					}
				}
				return nil
			}

			if !a.codeParser.IsSupportedPath(path) {
				return nil
			}

			// Exclude test files if requested
			if !a.IncludeTests && a.codeParser.IsTestFile(path) {
				return nil
			}

			for _, g := range fileGlobs {
				if g.Match(base) {
					return nil
				}
			}

			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (a *App) ProcessFile(path string) error {
	return a.processFileWithUpserter(path, nil)
}

type fileUpserter interface {
	UpsertFile(file *parser.File) error
}

type batchUpserter interface {
	fileUpserter
	Commit() error
	Rollback() error
}

func (a *App) processFileWithUpserter(path string, upserter fileUpserter) error {
	start := time.Now()
	lang := a.codeParser.GetLanguage(path)
	defer func() {
		observability.ParsingDuration.WithLabelValues(lang).Observe(time.Since(start).Seconds())
	}()

	previousContent := a.contentForPath(path)
	previousFile, _ := a.Graph.GetFile(path)
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Skip generated files: check after reading so we have the real content.
	if parser.IsGeneratedFile(content) {
		slog.Debug("skipping generated file", "path", path)
		return nil
	}

	file, err := a.codeParser.ParseFile(path, content)
	if err != nil {
		return err
	}

	switch file.Language {
	case "python":
		if len(a.Config.WatchPaths) == 0 {
			return fmt.Errorf("python resolver requires at least one watch path")
		}
		matchingPath, err := helpers.FindContainingWatchPath(path, a.Config.WatchPaths)
		if err != nil {
			return err
		}
		r := resolver.NewPythonResolver(matchingPath)
		file.Module = r.GetModuleName(path)
	case "go":
		moduleName, ok, err := a.resolveGoModule(path)
		if err != nil {
			return err
		}
		if ok {
			file.Module = moduleName
		}
	}

	// Update FullName for all definitions now that we have the module name
	if file.Module != "" {
		for i := range file.Definitions {
			if !strings.HasPrefix(file.Definitions[i].FullName, file.Module+".") {
				file.Definitions[i].FullName = file.Module + "." + file.Definitions[i].FullName
			}
		}
	}

	if a.secretScanner != nil && !a.shouldSkipSecretScan(path) {
		previousSecrets := []parser.Secret(nil)
		if previousFile != nil {
			previousSecrets = append(previousSecrets, previousFile.Secrets...)
		}
		file.Secrets = helpers.DetectSecrets(a.secretScanner, path, previousContent, content, previousSecrets)
	}
	a.Graph.AddFile(file)
	a.cacheContent(path, content)
	if upserter != nil {
		if err := upserter.UpsertFile(file); err != nil {
			slog.Warn("failed to upsert persisted symbol rows", "path", path, "error", err)
		}
	} else if err := a.enqueueSymbolWrite(ports.WriteRequest{
		Operation: ports.WriteOperationUpsertFile,
		File:      file,
		FilePath:  file.Path,
	}); err != nil {
		slog.Warn("failed to upsert persisted symbol rows", "path", path, "error", err)
	}
	return nil
}

func (a *App) shouldSkipSecretScan(path string) bool {
	base := filepath.Base(path)
	for _, g := range a.secretExcludeFiles {
		if g.Match(base) {
			return true
		}
	}

	dir := filepath.Dir(path)
	for {
		name := filepath.Base(dir)
		for _, g := range a.secretExcludeDirs {
			if g.Match(name) {
				return true
			}
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return false
}
