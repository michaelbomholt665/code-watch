package graph

import (
	"circular/internal/engine/parser"
	"strings"
	"unicode"
)

type SymbolRecord struct {
	Name       string
	FullName   string
	Module     string
	Language   string
	File       string
	Kind       parser.DefinitionKind
	Exported   bool
	Visibility string
	Scope      string
	Signature  string
	TypeHint   string
	Decorators []string
	IsService  bool
	Branches   int
	Parameters int
	Nesting    int
	LOC        int
	// v4 semantic tagging fields
	UsageTag   string
	Confidence float64
	Ancestry   string
}

type UniversalSymbolTable struct {
	symbols      []SymbolRecord
	byCanonical  map[string][]SymbolRecord
	byServiceKey map[string][]SymbolRecord
}

type SymbolLookupTable interface {
	Lookup(symbol string) []SymbolRecord
	LookupService(symbol string) []SymbolRecord
}

func (g *Graph) BuildUniversalSymbolTable() *UniversalSymbolTable {
	g.mu.RLock()
	defer g.mu.RUnlock()

	table := &UniversalSymbolTable{
		symbols:      make([]SymbolRecord, 0, 128),
		byCanonical:  make(map[string][]SymbolRecord),
		byServiceKey: make(map[string][]SymbolRecord),
	}

	for moduleName, defs := range g.definitions {
		for _, def := range defs {
			rec := SymbolRecord{
				Name:       def.Name,
				FullName:   def.FullName,
				Module:     moduleName,
				Language:   g.fileToLanguage[def.Location.File],
				File:       def.Location.File,
				Kind:       def.Kind,
				Exported:   def.Exported,
				Visibility: def.Visibility,
				Scope:      def.Scope,
				Signature:  def.Signature,
				TypeHint:   def.TypeHint,
				IsService:  isLikelyServiceDefinition(*def),
				Branches:   def.BranchCount,
				Parameters: def.ParameterCount,
				Nesting:    def.NestingDepth,
				LOC:        def.LOC,
			}
			if len(def.Decorators) > 0 {
				rec.Decorators = append([]string(nil), def.Decorators...)
			}

			table.symbols = append(table.symbols, rec)

			canonical := canonicalSymbol(def.Name)
			if canonical != "" {
				table.byCanonical[canonical] = append(table.byCanonical[canonical], rec)
			}
			if def.FullName != "" {
				fullCanonical := canonicalSymbol(def.FullName)
				if fullCanonical != "" && fullCanonical != canonical {
					table.byCanonical[fullCanonical] = append(table.byCanonical[fullCanonical], rec)
				}
			}

			if serviceKey := serviceSymbolKey(def.Name); serviceKey != "" && rec.IsService {
				table.byServiceKey[serviceKey] = append(table.byServiceKey[serviceKey], rec)
			}
		}
	}

	return table
}

func (t *UniversalSymbolTable) Lookup(symbol string) []SymbolRecord {
	if t == nil {
		return nil
	}
	key := canonicalSymbol(symbol)
	if key == "" {
		return nil
	}
	return cloneRecords(t.byCanonical[key])
}

func (t *UniversalSymbolTable) LookupService(symbol string) []SymbolRecord {
	if t == nil {
		return nil
	}
	key := serviceSymbolKey(symbol)
	if key == "" {
		return nil
	}
	return cloneRecords(t.byServiceKey[key])
}

func (t *UniversalSymbolTable) Symbols() []SymbolRecord {
	if t == nil {
		return nil
	}
	return cloneRecords(t.symbols)
}

func cloneRecords(records []SymbolRecord) []SymbolRecord {
	if len(records) == 0 {
		return nil
	}
	out := make([]SymbolRecord, len(records))
	copy(out, records)
	for i := range out {
		if len(out[i].Decorators) > 0 {
			out[i].Decorators = append([]string(nil), out[i].Decorators...)
		}
	}
	return out
}

func canonicalSymbol(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(symbol))
	for _, r := range symbol {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func serviceSymbolKey(symbol string) string {
	canonical := canonicalSymbol(symbol)
	if canonical == "" {
		return ""
	}
	suffixes := []string{
		"service",
		"servicer",
		"server",
		"client",
		"stub",
		"handler",
		"endpoint",
		"api",
		"rpc",
	}
	trimmed := canonical
	changed := true
	for changed {
		changed = false
		for _, suffix := range suffixes {
			if strings.HasSuffix(trimmed, suffix) && len(trimmed) > len(suffix)+2 {
				trimmed = strings.TrimSuffix(trimmed, suffix)
				changed = true
			}
		}
	}
	return trimmed
}

func isLikelyServiceDefinition(def parser.Definition) bool {
	name := strings.ToLower(def.Name)
	full := strings.ToLower(def.FullName)
	sig := strings.ToLower(def.Signature)
	hint := strings.ToLower(def.TypeHint)

	for _, token := range []string{"grpc", "thrift", "rpc", "route", "endpoint", "handler", "service", "servicer", "server"} {
		if strings.Contains(name, token) || strings.Contains(full, token) || strings.Contains(sig, token) || strings.Contains(hint, token) {
			return true
		}
	}

	for _, dec := range def.Decorators {
		dec = strings.ToLower(dec)
		if strings.Contains(dec, "grpc") || strings.Contains(dec, "thrift") || strings.Contains(dec, "route") || strings.Contains(dec, "api") {
			return true
		}
	}

	switch def.Kind {
	case parser.KindInterface, parser.KindClass:
		key := serviceSymbolKey(def.Name)
		return key != "" && key != canonicalSymbol(def.Name)
	default:
		return false
	}
}
