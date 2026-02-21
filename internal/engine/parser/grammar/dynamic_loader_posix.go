//go:build !windows

package grammar

/*
#include <dlfcn.h>
#include <stdlib.h>

void* load_ts_lang(const char* path, const char* name) {
    void* handle = dlopen(path, RTLD_LAZY);
    if (!handle) return NULL;
    return dlsym(handle, name);
}
*/
import "C"
import (
	"fmt"
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// LoadDynamic loads a Tree-sitter language from a shared object file.
func LoadDynamic(path, langName string) (*sitter.Language, error) {
	symbol := "tree_sitter_" + langName
	cPath := C.CString(path)
	cSymbol := C.CString(symbol)
	defer C.free(unsafe.Pointer(cPath))
	defer C.free(unsafe.Pointer(cSymbol))

	ptr := C.load_ts_lang(cPath, cSymbol)
	if ptr == nil {
		return nil, fmt.Errorf("failed to load %s from %s", symbol, path)
	}
	return sitter.NewLanguage(ptr), nil
}
