// # internal/parser/parser_test.go
package parser

import (
	"testing"
)

// newDefaultParser creates a Parser with all default extractors registered.
// Since DefaultExtractorForLanguage now returns NewUniversalExtractor() for all
// tree-sitter languages (go, python, etc.), these tests exercise the universal
// extractor end-to-end through the normal Parser API.
func newDefaultParser(t *testing.T) *Parser {
	t.Helper()
	loader, err := NewGrammarLoader("./grammars")
	if err != nil {
		t.Fatal(err)
	}
	p := NewParser(loader)
	if err := p.RegisterDefaultExtractors(); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGoExtraction_QualifiedTypes(t *testing.T) {
	p := newDefaultParser(t)

	code := `
package test
import "circular/internal/core/ports"
type Deps struct {
	Analysis ports.AnalysisService
}
`
	file, err := p.ParseFile("test.go", []byte(code))
	if err != nil {
		t.Fatal(err)
	}

	// Universal extractor surfaces the import for the ports package.
	if len(file.Imports) == 0 {
		t.Error("Expected at least one import for 'ports'")
	}
}

func TestPythonExtraction(t *testing.T) {
	p := newDefaultParser(t)

	code := `
import os
import sys as system
from auth.utils import login as auth_login
from . import local_mod
from ..parent import parent_mod

@app.route("/health")
def my_func(a):
    print(a)
    return os.path.join(a, "b")

class MyClass:
    @decorator
    def __init__(self):
        pass
`
	file, err := p.ParseFile("test.py", []byte(code))
	if err != nil {
		t.Fatal(err)
	}

	if file.Language != "python" {
		t.Errorf("Expected python, got %s", file.Language)
	}

	// Universal extractor detects function and class definitions.
	foundFunc := false
	foundClass := false
	for _, def := range file.Definitions {
		if def.Name == "my_func" {
			foundFunc = true
		}
		if def.Name == "MyClass" {
			foundClass = true
		}
	}
	if !foundFunc {
		t.Error("my_func not found")
	}
	if !foundClass {
		t.Error("MyClass not found")
	}
}

func TestGoExtraction(t *testing.T) {
	p := newDefaultParser(t)

	code := `
package main

import (
	"fmt"
	"os"
)

func Main() {
	fmt.Println(os.Args)
}

type MyStruct struct {
	ID int
}

type MyInterface interface {
	Run()
}

func Sum(base int, extra ...int) int {
	return base
}
`
	file, err := p.ParseFile("main.go", []byte(code))
	if err != nil {
		t.Fatal(err)
	}

	if file.PackageName != "main" {
		t.Errorf("Expected package main, got %s", file.PackageName)
	}

	if len(file.Imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(file.Imports))
	}

	// Universal extractor detects top-level function definitions.
	foundMain := false
	for _, def := range file.Definitions {
		if def.Name == "Main" {
			foundMain = true
			break
		}
	}
	if !foundMain {
		t.Error("Main function not found")
	}
}

func TestGoExtraction_ComplexityMetrics(t *testing.T) {
	p := newDefaultParser(t)

	code := `
package main

func Sum(a, b int) int {
	if a > b {
		return a
	}
	for i := 0; i < b; i++ {
		a += i
	}
	return a
}
`
	file, err := p.ParseFile("main.go", []byte(code))
	if err != nil {
		t.Fatal(err)
	}

	var sumDef *Definition
	for i := range file.Definitions {
		if file.Definitions[i].Name == "Sum" {
			sumDef = &file.Definitions[i]
			break
		}
	}
	if sumDef == nil {
		t.Fatal("Sum function not found")
	}
	if sumDef.Kind != KindFunction {
		t.Fatalf("expected KindFunction, got %v", sumDef.Kind)
	}
	if sumDef.ParameterCount != 2 {
		t.Fatalf("expected 2 parameters, got %d", sumDef.ParameterCount)
	}
	if sumDef.BranchCount < 2 {
		t.Fatalf("expected at least 2 branches, got %d", sumDef.BranchCount)
	}
	if sumDef.NestingDepth < 1 {
		t.Fatalf("expected nesting depth >= 1, got %d", sumDef.NestingDepth)
	}
	if sumDef.LOC < 3 {
		t.Fatalf("expected LOC >= 3, got %d", sumDef.LOC)
	}
}

func TestProfileExtractor_MetadataParityAndBridgeContexts(t *testing.T) {
	trueVal := true
	registry, err := BuildLanguageRegistry(map[string]LanguageOverride{
		"javascript": {Enabled: &trueVal},
		"typescript": {Enabled: &trueVal},
		"java":       {Enabled: &trueVal},
		"rust":       {Enabled: &trueVal},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	loader, err := NewGrammarLoaderWithRegistry("./grammars", registry, false)
	if err != nil {
		t.Fatal(err)
	}
	p := NewParser(loader)
	if err := p.RegisterDefaultExtractors(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		code       string
		defName    string
		serviceRef string
	}{
		{
			name:       "javascript",
			path:       "svc.js",
			code:       "class GreeterService {} function run(){ grpc.connect() }",
			defName:    "GreeterService",
			serviceRef: "grpc.connect",
		},
		{
			name:       "typescript",
			path:       "svc.ts",
			code:       "interface UserService{}; function exec(){ fetch('/v1') }",
			defName:    "UserService",
			serviceRef: "fetch",
		},
		{
			name:       "java",
			path:       "Svc.java",
			code:       "public class GreeterService { void run(){ io.grpc.ClientCalls.blockingUnaryCall(); } }",
			defName:    "GreeterService",
			serviceRef: "blockingUnaryCall",
		},
		{
			name:       "rust",
			path:       "svc.rs",
			code:       `pub struct GreeterService; fn call(){ tonic::transport::Channel::from_static("http://x"); }`,
			defName:    "GreeterService",
			serviceRef: "tonic::transport::Channel::from_static",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, err := p.ParseFile(tc.path, []byte(tc.code))
			if err != nil {
				t.Fatal(err)
			}

			// Verify the key definition is detected.
			foundDef := false
			for _, d := range file.Definitions {
				if d.Name == tc.defName {
					foundDef = true
					break
				}
			}
			if !foundDef {
				t.Fatalf("expected definition %s in %v", tc.defName, file.Definitions)
			}

			// Verify the service bridge reference is tagged.
			foundBridgeRef := false
			for _, ref := range file.References {
				if ref.Name == tc.serviceRef && ref.Context == RefContextService {
					foundBridgeRef = true
					break
				}
			}
			if !foundBridgeRef {
				t.Fatalf("expected %s to be tagged as %s", tc.serviceRef, RefContextService)
			}
		})
	}
}

func TestExtraction_MultiLanguage(t *testing.T) {
	trueVal := true
	registry, err := BuildLanguageRegistry(map[string]LanguageOverride{
		"javascript": {Enabled: &trueVal},
		"typescript": {Enabled: &trueVal},
		"tsx":        {Enabled: &trueVal},
		"java":       {Enabled: &trueVal},
		"rust":       {Enabled: &trueVal},
		"html":       {Enabled: &trueVal},
		"css":        {Enabled: &trueVal},
		"gomod":      {Enabled: &trueVal},
		"gosum":      {Enabled: &trueVal},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	loader, err := NewGrammarLoaderWithRegistry("./grammars", registry, false)
	if err != nil {
		t.Fatal(err)
	}
	p := NewParser(loader)
	if err := p.RegisterDefaultExtractors(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		path     string
		code     string
		language string
	}{
		{
			name:     "javascript",
			path:     "main.js",
			code:     "import foo from 'pkg'; function run(a){ return foo(a) }",
			language: "javascript",
		},
		{
			name:     "typescript",
			path:     "main.ts",
			code:     "import {x} from 'pkg'; interface Runner{}; function run(a:number){ return x(a) }",
			language: "typescript",
		},
		{
			name:     "tsx",
			path:     "main.tsx",
			code:     "import React from 'react'; export function App(){ return <div/> }",
			language: "tsx",
		},
		{
			name:     "java",
			path:     "Main.java",
			code:     "package p; import java.util.List; class Main { void run(){} }",
			language: "java",
		},
		{
			name:     "rust",
			path:     "main.rs",
			code:     `use std::fmt; fn main(){ println!("hi"); }`,
			language: "rust",
		},
		{
			name:     "html",
			path:     "index.html",
			code:     `<html><head><script src="app.js"></script></head><body class="hero"></body></html>`,
			language: "html",
		},
		{
			name:     "css",
			path:     "site.css",
			code:     `@import "theme.css"; .hero { color: red; }`,
			language: "css",
		},
		{
			name:     "gomod",
			path:     "go.mod",
			code:     "module example.com/x\n\nrequire github.com/pkg/errors v0.9.1",
			language: "gomod",
		},
		{
			name:     "gosum",
			path:     "go.sum",
			code:     "github.com/pkg/errors v0.9.1 h1:abc\n",
			language: "gosum",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, err := p.ParseFile(tc.path, []byte(tc.code))
			if err != nil {
				t.Fatal(err)
			}
			if file.Language != tc.language {
				t.Fatalf("expected language %q, got %q", tc.language, file.Language)
			}
		})
	}
}
