package parser

import (
	"fmt"
)

func DebugJava() {
	trueVal := true
	registry, err := BuildLanguageRegistry(map[string]LanguageOverride{
		"java": {Enabled: &trueVal},
	}, nil)
	if err != nil {
		panic(err)
	}
	loader, err := NewGrammarLoaderWithRegistry("../../grammars", registry, false)
	if err != nil {
		panic(err)
	}
	p := NewParser(loader)
	p.RegisterDefaultExtractors()
	code := []byte(`public class GreeterService { void run(){ io.grpc.ClientCalls.blockingUnaryCall(); } }`)
	f, err := p.ParseFile("Svc.java", code)
	if err != nil {
		panic(err)
	}
	fmt.Printf("File parsed: %s\n", f.Language)
	for _, ref := range f.References {
		fmt.Printf("Ref: '%s' (Context: %s) Ancestry: %s\n", ref.Name, ref.Context, ref.Context)
	}
	for _, def := range f.Definitions {
		fmt.Printf("Def: '%s' Kind: %v\n", def.Name, def.Kind)
	}
}
