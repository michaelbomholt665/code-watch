// # internal/resolver/stdlib.go
package resolver

import (
	_ "embed"
	"strings"
)

//go:embed stdlib/python.txt
var pythonStdlibData string

//go:embed stdlib/go.txt
var goStdlibData string

//go:embed stdlib/javascript.txt
var javascriptStdlibData string

//go:embed stdlib/java.txt
var javaStdlibData string

//go:embed stdlib/rust.txt
var rustStdlibData string

var pythonStdlib = map[string]bool{}
var goStdlib = map[string]bool{}
var javascriptStdlib = map[string]bool{}
var javaStdlib = map[string]bool{}
var rustStdlib = map[string]bool{}

func init() {
	for _, line := range strings.Split(pythonStdlibData, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			pythonStdlib[line] = true
			// Add base name: e.g. urllib.request -> urllib
			parts := strings.Split(line, ".")
			pythonStdlib[parts[0]] = true
		}
	}

	for _, line := range strings.Split(goStdlibData, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			goStdlib[line] = true
			// Add base name: e.g. log/slog -> slog
			parts := strings.Split(line, "/")
			goStdlib[parts[len(parts)-1]] = true
		}
	}

	for _, line := range strings.Split(javascriptStdlibData, "\n") {
		registerStdlibLine(javascriptStdlib, line)
	}
	for _, line := range strings.Split(javaStdlibData, "\n") {
		registerStdlibLine(javaStdlib, line)
	}
	for _, line := range strings.Split(rustStdlibData, "\n") {
		registerStdlibLine(rustStdlib, line)
	}
}

func registerStdlibLine(dst map[string]bool, line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}
	dst[line] = true
	for _, part := range strings.FieldsFunc(line, func(r rune) bool {
		return r == '/' || r == '.' || r == ':'
	}) {
		part = strings.TrimSpace(part)
		if part != "" {
			dst[part] = true
		}
	}
}

func getStdlibByLanguage() map[string]map[string]bool {
	return map[string]map[string]bool{
		"go":         goStdlib,
		"python":     pythonStdlib,
		"javascript": javascriptStdlib,
		"typescript": javascriptStdlib,
		"tsx":        javascriptStdlib,
		"java":       javaStdlib,
		"rust":       rustStdlib,
	}
}

var pythonBuiltins = map[string]bool{
	"abs": true, "aiter": true, "all": true, "anext": true, "any": true,
	"ascii": true, "bin": true, "bool": true, "breakpoint": true, "bytearray": true,
	"bytes": true, "callable": true, "chr": true, "classmethod": true, "compile": true,
	"complex": true, "delattr": true, "dict": true, "dir": true, "divmod": true,
	"enumerate": true, "eval": true, "exec": true, "filter": true, "float": true,
	"format": true, "frozenset": true, "getattr": true, "globals": true, "hasattr": true,
	"hash": true, "help": true, "hex": true, "id": true, "input": true,
	"int": true, "isinstance": true, "issubclass": true, "iter": true, "len": true,
	"list": true, "locals": true, "map": true, "max": true, "memoryview": true,
	"min": true, "next": true, "object": true, "oct": true, "open": true,
	"ord": true, "pow": true, "print": true, "property": true, "range": true,
	"repr": true, "reversed": true, "round": true, "set": true, "setattr": true,
	"slice": true, "sorted": true, "staticmethod": true, "str": true, "sum": true,
	"super": true, "tuple": true, "type": true, "vars": true, "zip": true,
	"__import__": true,
}

var goBuiltins = map[string]bool{
	"append": true, "cap": true, "close": true, "complex": true, "copy": true,
	"delete": true, "imag": true, "len": true, "make": true, "new": true,
	"panic": true, "print": true, "println": true, "real": true, "recover": true,
	"bool": true, "byte": true, "complex64": true, "complex128": true, "error": true,
	"float32": true, "float64": true, "int": true, "int8": true, "int16": true,
	"int32": true, "int64": true, "rune": true, "string": true, "uint": true,
	"uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"nil": true, "true": true, "false": true, "iota": true,
}
