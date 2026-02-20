// # internal/parser/parser_test.go
package parser

import (
	"testing"
)

func TestGoExtraction_QualifiedTypes(t *testing.T) {
	loader, err := NewGrammarLoader("./grammars")
	if err != nil {
		t.Fatal(err)
	}

	p := NewParser(loader)
	p.RegisterExtractor("go", &GoExtractor{})

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

	foundPorts := false
	foundAnalysis := false
	for _, ref := range file.References {
		if ref.Name == "ports" {
			foundPorts = true
		}
		if ref.Name == "ports.AnalysisService" {
			foundAnalysis = true
		}
	}

	if !foundPorts {
		t.Error("Expected reference 'ports' not found")
	}
	if !foundAnalysis {
		t.Error("Expected reference 'ports.AnalysisService' not found")
	}
}

func TestPythonExtraction(t *testing.T) {
	loader, err := NewGrammarLoader("./grammars")
	if err != nil {
		t.Fatal(err)
	}

	p := NewParser(loader)
	p.RegisterExtractor("python", &PythonExtractor{})

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

def wrapped_params(a: int, b=1, *args, **kwargs):
    return a

class MyClass:
    @decorator
    def __init__(self):
        pass

def bridge():
    ctypes.CDLL("libdemo.so")
    subprocess.run(["worker"])
`
	file, err := p.ParseFile("test.py", []byte(code))
	if err != nil {
		t.Fatal(err)
	}

	if file.Language != "python" {
		t.Errorf("Expected python, got %s", file.Language)
	}

	// Check imports
	// 1. os
	// 2. sys
	// 3. auth.utils
	// 4. .
	// 5. ..parent
	if len(file.Imports) != 5 {
		t.Errorf("Expected 5 imports, got %d", len(file.Imports))
		for i, imp := range file.Imports {
			t.Logf("Import %d: %s", i, imp.Module)
		}
	}

	// Check definitions
	foundFunc := false
	foundClass := false
	foundWrapped := false
	var funcDef Definition
	var wrappedDef Definition
	for _, def := range file.Definitions {
		if def.Name == "my_func" && def.Kind == KindFunction {
			foundFunc = true
			funcDef = def
		}
		if def.Name == "wrapped_params" && def.Kind == KindFunction {
			foundWrapped = true
			wrappedDef = def
		}
		if def.Name == "MyClass" && def.Kind == KindClass {
			foundClass = true
		}
	}
	if !foundFunc {
		t.Fatal("my_func not found")
	}
	if !foundClass {
		t.Error("MyClass not found")
	}
	if !foundWrapped {
		t.Error("wrapped_params not found")
	}
	if funcDef.ParameterCount != 1 {
		t.Errorf("Expected my_func parameter count 1, got %d", funcDef.ParameterCount)
	}
	if len(funcDef.Decorators) != 1 || funcDef.Decorators[0] != "app.route(\"/health\")" {
		t.Errorf("Expected my_func decorator app.route(\"/health\"), got %v", funcDef.Decorators)
	}
	if funcDef.Visibility != "public" {
		t.Errorf("Expected my_func visibility public, got %s", funcDef.Visibility)
	}
	if funcDef.Scope != "global" {
		t.Errorf("Expected my_func scope global, got %s", funcDef.Scope)
	}
	if funcDef.TypeHint != "function" {
		t.Errorf("Expected my_func type hint function, got %s", funcDef.TypeHint)
	}
	if wrappedDef.ParameterCount != 4 {
		t.Errorf("Expected wrapped_params parameter count 4, got %d", wrappedDef.ParameterCount)
	}
	if funcDef.LOC < 2 {
		t.Errorf("Expected my_func LOC >= 2, got %d", funcDef.LOC)
	}

	foundBridgeFFI := false
	foundBridgeProcess := false
	for _, ref := range file.References {
		if ref.Name == "ctypes.CDLL" && ref.Context == RefContextFFI {
			foundBridgeFFI = true
		}
		if ref.Name == "subprocess.run" && ref.Context == RefContextProcess {
			foundBridgeProcess = true
		}
	}
	if !foundBridgeFFI {
		t.Errorf("Expected ctypes.CDLL to be tagged as %s", RefContextFFI)
	}
	if !foundBridgeProcess {
		t.Errorf("Expected subprocess.run to be tagged as %s", RefContextProcess)
	}

	// Check local symbols
	// my_func has parameter 'a'
	// MyClass.__init__ has parameter 'self'
	expected := []string{"a", "self"}
	for _, exp := range expected {
		found := false
		for _, sym := range file.LocalSymbols {
			if sym == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected local symbol %s not found in %v", exp, file.LocalSymbols)
		}
	}

	// Test assignments and for loops in Python
	code2 := `
def work(items):
    x = 10
    for item in items:
        y = item.val
        print(x, y)
`
	file2, err := p.ParseFile("work.py", []byte(code2))
	if err != nil {
		t.Fatal(err)
	}

	expected2 := []string{"items", "x", "item", "y"}
	for _, exp := range expected2 {
		found := false
		for _, sym := range file2.LocalSymbols {
			if sym == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected local symbol %s not found in %v", exp, file2.LocalSymbols)
		}
	}
}

func TestGoExtraction(t *testing.T) {
	loader, err := NewGrammarLoader("./grammars")
	if err != nil {
		t.Fatal(err)
	}

	p := NewParser(loader)
	p.RegisterExtractor("go", &GoExtractor{})

	code := `
package main

import (
	"fmt"
	"os"
)

func Main() {
	fmt.Println(os.Args)
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
		wantHint   string
		wantScope  string
		serviceRef string
	}{
		{
			name:       "javascript",
			path:       "svc.js",
			code:       "class GreeterService {} function run(){ grpc.connect() }",
			defName:    "GreeterService",
			wantHint:   "class",
			wantScope:  "global",
			serviceRef: "grpc.connect",
		},
		{
			name:       "typescript",
			path:       "svc.ts",
			code:       "interface UserService{}; function exec(){ fetch('/v1') }",
			defName:    "UserService",
			wantHint:   "interface",
			wantScope:  "global",
			serviceRef: "fetch",
		},
		{
			name:       "java",
			path:       "Svc.java",
			code:       "public class GreeterService { void run(){ io.grpc.ClientCalls.blockingUnaryCall(); } }",
			defName:    "GreeterService",
			wantHint:   "class",
			wantScope:  "global",
			serviceRef: "blockingUnaryCall",
		},
		{
			name:       "rust",
			path:       "svc.rs",
			code:       "pub struct GreeterService; fn call(){ tonic::transport::Channel::from_static(\"http://x\"); }",
			defName:    "GreeterService",
			wantHint:   "type",
			wantScope:  "global",
			serviceRef: "tonic::transport::Channel::from_static",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, err := p.ParseFile(tc.path, []byte(tc.code))
			if err != nil {
				t.Fatal(err)
			}

			var def Definition
			foundDef := false
			for _, d := range file.Definitions {
				if d.Name == tc.defName {
					foundDef = true
					def = d
					break
				}
			}
			if !foundDef {
				t.Fatalf("expected definition %s", tc.defName)
			}
			if def.Visibility == "" {
				t.Fatalf("expected non-empty visibility for %s", tc.defName)
			}
			if def.Scope != tc.wantScope {
				t.Fatalf("expected scope %s, got %s", tc.wantScope, def.Scope)
			}
			if def.Signature == "" {
				t.Fatalf("expected non-empty signature for %s", tc.defName)
			}
			if def.TypeHint != tc.wantHint {
				t.Fatalf("expected type hint %s, got %s", tc.wantHint, def.TypeHint)
			}

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
			code:     "use std::fmt; fn main(){ println!(\"hi\"); }",
			language: "rust",
		},
		{
			name:     "html",
			path:     "index.html",
			code:     "<html><head><script src=\"app.js\"></script></head><body class=\"hero\"></body></html>",
			language: "html",
		},
		{
			name:     "css",
			path:     "site.css",
			code:     "@import \"theme.css\"; .hero { color: red; }",
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

	foundMain := false
	foundStruct := false
	foundInterface := false
	foundSum := false
	var mainDef Definition
	var sumDef Definition
	for _, def := range file.Definitions {
		if def.Name == "Main" && def.Kind == KindFunction {
			foundMain = true
			mainDef = def
		}
		if def.Name == "MyStruct" && def.Kind == KindType {
			foundStruct = true
		}
		if def.Name == "MyInterface" && def.Kind == KindInterface {
			foundInterface = true
		}
		if def.Name == "Sum" && def.Kind == KindFunction {
			foundSum = true
			sumDef = def
		}
	}
	if !foundMain {
		t.Fatal("Main function not found")
	}
	if !foundStruct {
		t.Error("MyStruct type not found")
	}
	if !foundInterface {
		t.Error("MyInterface not found")
	}
	if !foundSum {
		t.Error("Sum function not found")
	}
	if mainDef.LOC < 2 {
		t.Errorf("Expected Main LOC >= 2, got %d", mainDef.LOC)
	}
	if mainDef.Visibility != "public" {
		t.Errorf("Expected Main visibility public, got %s", mainDef.Visibility)
	}
	if mainDef.Signature == "" {
		t.Error("Expected Main signature to be populated")
	}
	if mainDef.TypeHint != "function" {
		t.Errorf("Expected Main type hint function, got %s", mainDef.TypeHint)
	}
	if mainDef.ComplexityScore <= 0 {
		t.Errorf("Expected Main complexity score > 0, got %d", mainDef.ComplexityScore)
	}
	if sumDef.ParameterCount != 2 {
		t.Errorf("Expected Sum parameter count 2, got %d", sumDef.ParameterCount)
	}

	// Check local symbols in a more complex Go snippet
	code2 := `
package test
func Work(ctx Context, id int) {
	msg := "hello"
	var x = 10
	for i := range 5 {
		println(i, msg, x, ctx, id)
	}
}
`
	file2, err := p.ParseFile("work.go", []byte(code2))
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"ctx", "id", "msg", "x", "i"}
	for _, exp := range expected {
		found := false
		for _, got := range file2.LocalSymbols {
			if got == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected local symbol %s not found in %v", exp, file2.LocalSymbols)
		}
	}
}
