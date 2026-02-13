// # internal/parser/types.go
package parser

import (
	"time"
)

type File struct {
	Path         string
	Language     string
	Module       string // Fully qualified module name
	PackageName  string // Local package/module name
	Imports      []Import
	Definitions  []Definition
	References   []Reference // Function/symbol calls
	LocalSymbols []string    // Variables defined in local scope (vars, params, self)
	ParsedAt     time.Time
}

type Import struct {
	Module     string   // Imported module (resolved to FQN)
	RawImport  string   // Original import string
	Alias      string   // Optional alias
	Items      []string // For "from X import Y, Z"
	IsRelative bool     // For Python relative imports
	Used       bool     // Set by analysis stages when usage is detected
	UsageCount int      // Number of detected reference hits for this import
	Location   Location
}

type Definition struct {
	Name     string
	FullName string // module.function or package.Type
	Kind     DefinitionKind
	Location Location
	Exported bool   // Is public/exported?
	Scope    string // Global, class method, nested function
}

type Reference struct {
	Name     string
	FullName string // Resolved if possible
	Location Location
	Context  string // Where this reference occurs
	Resolved bool   // Did we find the definition?
}

type DefinitionKind int

const (
	KindFunction DefinitionKind = iota
	KindClass
	KindMethod
	KindVariable
	KindConstant
	KindType
	KindInterface
)

type Location struct {
	File   string
	Line   int
	Column int
}
