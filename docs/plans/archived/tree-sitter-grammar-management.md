# Implementation Plan: Tree-sitter Grammar Management CLI

**ID:** PLAN-008  
**Status:** Draft  
**Target Package:** `internal/ui/cli`, `internal/engine/parser`  
**User Context:** Local first, single user (Extending language support)

## Overview

Provide a seamless way for users to add new Tree-sitter grammars to Circular using a new CLI command. The system will handle downloading, compiling, and registering the grammar while keeping only the essential binary artifacts (`.so` and `node-types.json`) to keep the installation lean.

## Current State

Grammars are manually placed in `grammars/` and registered in `grammars/manifest.toml`. Adding a new language requires manual compilation and metadata generation, which is error-prone and tedious.

## Proposed Changes

### 1. New CLI Command: `circular grammars`
Implement a new command group to manage grammars:
- `circular grammars add <name> <url>`: Download, build, and register a new grammar.
- `circular grammars list`: Show installed grammars and their versions.
- `circular grammars remove <name>`: Delete a grammar and its registration.

### 2. Automatic Build Pipeline
Implement a Go-based build orchestrator that:
1. Clones the grammar repository into a temporary directory.
2. Runs `tree-sitter generate` (requires `tree-sitter` CLI to be installed on the host).
3. Compiles the `parser.c` (and optional `scanner.c`/`scanner.cc`) into a shared library using the platform's native compiler:
    - **Linux:** Produces `<name>.so` using `gcc` or `clang` with `-shared -fPIC`.
    - **macOS:** Produces `<name>.dylib` using `clang` with `-shared -dynamiclib -fPIC`.
    - **Windows:** Produces `<name>.dll` using `cl.exe` (MSVC) or `gcc` (MinGW) with `-shared`.
4. Extracts `src/node-types.json` from the repository.
5. Moves the binary and `node-types.json` to `grammars/<name>/`.
6. Cleans up the temporary source directory.

### 3. Automatic Manifest Update
Update `grammars/manifest.toml` automatically:
- Compute SHA-256 hashes for the new binary and `node-types.json`.
- Append a new `[[artifacts]]` entry to the manifest.
- Support platform-specific paths in the manifest (e.g., `linux_path`, `macos_path`, `windows_path`) or a template-based path resolution to ensure the correct binary is loaded on each OS.
- Set the `aib_version` to the current system default.

### 4. Cross-Platform Loading
Update the parser engine in `internal/engine/parser/` to:
- Detect the host OS at runtime.
- Select the appropriate library extension and path from the manifest.
- Use a cross-platform loading mechanism (like `purego` or platform-specific syscalls) to load the grammar.

### 5. Cross-Compilation Support (Optional)
If the user has cross-compilation toolchains installed (e.g., `mingw-w64` for Windows, `osxcross` for macOS), allow specifying a `--target` flag to build binaries for other platforms from a single host.

## Implementation Steps

### Phase 1: Build Orchestrator
1. Create `internal/engine/parser/builder.go` to handle the external command execution (`git`, `tree-sitter`, `cc`).
2. Implement logic to detect and handle C++ scanners (common in Tree-sitter).
3. Implement hash calculation utility.

### Phase 2: Manifest Manager
1. Create `internal/engine/parser/manifest.go` to handle reading/writing the TOML manifest.
2. Implement `AddArtifact` and `RemoveArtifact` methods that preserve comments and formatting where possible.

### Phase 3: CLI Integration
1. Update `internal/ui/cli/runtime.go` to include the `grammars` command.
2. Implement progress feedback (using Bubble Tea spinners) for the download and build phases.

## Verification Plan

### Automated Tests
- Unit test for manifest serialization/deserialization.
- Mock test for the builder ensuring it calls the correct external commands.

### Manual Verification
- Run `circular grammars add ruby https://github.com/tree-sitter/tree-sitter-ruby`.
- Verify `grammars/ruby/ruby.so` and `grammars/ruby/node-types.json` exist.
- Verify `grammars/manifest.toml` contains the Ruby entry.
- Run `circular --once` on a Ruby file and verify it parses correctly.
