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
	Secrets      []Secret
	LocalSymbols []string // Variables defined in local scope (vars, params, self)
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
	Name       string
	FullName   string // module.function or package.Type
	Kind       DefinitionKind
	Location   Location
	Exported   bool   // Is public/exported?
	Visibility string // public, private, or internal
	Scope      string // global, class, method, nested
	Signature  string // Lightweight declaration signature for cross-language comparisons
	TypeHint   string // Normalized type category/signature hint
	Decorators []string
	// Heuristic complexity metrics used for hotspot ranking.
	BranchCount     int
	ParameterCount  int
	NestingDepth    int
	LOC             int
	ComplexityScore int
}

type Reference struct {
	Name     string
	FullName string // Resolved if possible
	Location Location
	Context  string // Where this reference occurs
	Resolved bool   // Did we find the definition?
}

type Secret struct {
	Kind       string
	Severity   string
	Value      string
	Entropy    float64
	Confidence float64
	Location   Location
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

const (
	RefContextDefault = ""
	RefContextFFI     = "ffi_bridge"
	RefContextProcess = "process_bridge"
	RefContextService = "service_bridge"
)

type Location struct {
	File   string
	Line   int
	Column int
}
