// # internal/parser/parser_test.go
package parser

import (
	"testing"
)

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

def my_func(a):
    print(a)
    return os.path.join(a, "b")

class MyClass:
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
	var funcDef Definition
	for _, def := range file.Definitions {
		if def.Name == "my_func" && def.Kind == KindFunction {
			foundFunc = true
			funcDef = def
		}
		if def.Name == "MyClass" && def.Kind == KindClass {
			foundClass = true
		}
	}
	if !foundFunc {
		t.Error("my_func not found")
	}
	if !foundClass {
		t.Error("MyClass not found")
	}
	if funcDef.ParameterCount != 1 {
		t.Errorf("Expected my_func parameter count 1, got %d", funcDef.ParameterCount)
	}
	if funcDef.LOC < 2 {
		t.Errorf("Expected my_func LOC >= 2, got %d", funcDef.LOC)
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

type MyStruct struct {
	ID int
}

type MyInterface interface {
	Run()
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
	var mainDef Definition
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
	}
	if !foundMain {
		t.Error("Main function not found")
	}
	if !foundStruct {
		t.Error("MyStruct type not found")
	}
	if !foundInterface {
		t.Error("MyInterface not found")
	}
	if mainDef.LOC < 2 {
		t.Errorf("Expected Main LOC >= 2, got %d", mainDef.LOC)
	}
	if mainDef.ComplexityScore <= 0 {
		t.Errorf("Expected Main complexity score > 0, got %d", mainDef.ComplexityScore)
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
