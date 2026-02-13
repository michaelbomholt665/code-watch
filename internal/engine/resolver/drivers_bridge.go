package resolver

import "circular/internal/engine/resolver/drivers"

type GoResolver = drivers.GoResolver
type PythonResolver = drivers.PythonResolver
type JavaScriptResolver = drivers.JavaScriptResolver
type JavaResolver = drivers.JavaResolver
type RustResolver = drivers.RustResolver

func NewGoResolver() *GoResolver {
	return drivers.NewGoResolver()
}

func NewPythonResolver(projectRoot string) *PythonResolver {
	return drivers.NewPythonResolver(projectRoot)
}

func NewJavaScriptResolver() *JavaScriptResolver {
	return drivers.NewJavaScriptResolver()
}

func NewJavaResolver() *JavaResolver {
	return drivers.NewJavaResolver()
}

func NewRustResolver() *RustResolver {
	return drivers.NewRustResolver()
}
