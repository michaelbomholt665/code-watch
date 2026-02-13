package parser

func cloneLanguageRegistry(in map[string]LanguageSpec) map[string]LanguageSpec {
	out := make(map[string]LanguageSpec, len(in))
	for id, spec := range in {
		copySpec := spec
		copySpec.Extensions = append([]string(nil), spec.Extensions...)
		copySpec.Filenames = append([]string(nil), spec.Filenames...)
		copySpec.TestFileSuffixes = append([]string(nil), spec.TestFileSuffixes...)
		out[id] = copySpec
	}
	return out
}
