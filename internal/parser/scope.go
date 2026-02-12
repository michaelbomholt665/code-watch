// # internal/parser/scope.go
package parser

type Scope struct {
	Symbols []string
	Parent  *Scope
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		Symbols: []string{},
		Parent:  parent,
	}
}

func (s *Scope) Add(symbol string) {
	s.Symbols = append(s.Symbols, symbol)
}

func (s *Scope) Exists(symbol string) bool {
	for _, sym := range s.Symbols {
		if sym == symbol {
			return true
		}
	}
	if s.Parent != nil {
		return s.Parent.Exists(symbol)
	}
	return false
}
