// # internal/resolver/heuristics.go
package resolver

import (
	"circular/internal/engine/parser"
	"strings"
)

func IsKnownNonModule(name string, excluded []string) bool {
	parts := strings.Split(name, ".")
	prefix := parts[0]

	for _, sym := range excluded {
		if sym == prefix || sym == prefix+"." {
			return true
		}
	}

	return false
}

func IsCrossLanguageBridgeReference(language string, ref parser.Reference) bool {
	if ref.Context == parser.RefContextFFI || ref.Context == parser.RefContextProcess || ref.Context == parser.RefContextService {
		return true
	}

	name := strings.TrimSpace(ref.Name)
	if name == "" {
		return false
	}

	switch language {
	case "python":
		return strings.HasPrefix(name, "ctypes.") ||
			strings.HasPrefix(name, "cffi.") ||
			strings.HasPrefix(name, "subprocess.") ||
			strings.HasPrefix(name, "grpc.") ||
			strings.HasPrefix(name, "thrift.")
	case "go":
		return strings.HasPrefix(name, "C.") ||
			strings.HasPrefix(name, "exec.") ||
			strings.HasPrefix(name, "grpc.")
	case "javascript", "typescript", "tsx":
		return strings.HasPrefix(name, "ffi.") ||
			strings.HasPrefix(name, "nodeffi.") ||
			strings.HasPrefix(name, "child_process.") ||
			strings.HasPrefix(name, "grpc.") ||
			strings.HasPrefix(name, "thrift.")
	case "java":
		return strings.HasPrefix(name, "jni.") ||
			strings.HasPrefix(name, "io.grpc.") ||
			strings.HasPrefix(name, "thrift.") ||
			strings.HasPrefix(name, "processbuilder.")
	case "rust":
		return strings.HasPrefix(name, "libloading.") ||
			strings.HasPrefix(name, "tonic::") ||
			strings.HasPrefix(name, "grpc::") ||
			strings.HasPrefix(name, "thrift::")
	default:
		return false
	}
}

func IsCrossLanguageBridgeImportHint(language, module, base string) bool {
	module = strings.ToLower(strings.TrimSpace(module))
	base = strings.ToLower(strings.TrimSpace(base))
	if module == "" && base == "" {
		return false
	}

	hasAnyPrefix := func(prefixes ...string) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(module, prefix) || strings.HasPrefix(base, prefix) {
				return true
			}
		}
		return false
	}

	switch strings.ToLower(strings.TrimSpace(language)) {
	case "python":
		return hasAnyPrefix("ctypes", "cffi", "subprocess", "grpc", "thrift")
	case "go":
		return hasAnyPrefix("c", "os/exec", "exec", "google.golang.org/grpc", "grpc")
	case "javascript", "typescript", "tsx":
		return hasAnyPrefix("ffi", "nodeffi", "child_process", "grpc", "thrift")
	case "java":
		return hasAnyPrefix("jni", "io.grpc", "thrift", "processbuilder")
	case "rust":
		return hasAnyPrefix("libloading", "tonic", "grpc", "thrift")
	default:
		return false
	}
}
