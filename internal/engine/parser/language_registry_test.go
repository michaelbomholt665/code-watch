package parser

import "testing"

func TestBuildLanguageRegistry_Defaults(t *testing.T) {
	registry, err := BuildLanguageRegistry(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !registry["go"].Enabled {
		t.Fatal("expected go to be enabled by default")
	}
	if !registry["python"].Enabled {
		t.Fatal("expected python to be enabled by default")
	}
	if registry["javascript"].Enabled {
		t.Fatal("expected javascript to be disabled by default")
	}
}

func TestBuildLanguageRegistry_RejectsDuplicateExtensions(t *testing.T) {
	enabled := true
	_, err := BuildLanguageRegistry(map[string]LanguageOverride{
		"javascript": {Enabled: &enabled, Extensions: []string{".go"}},
	}, nil)
	if err == nil {
		t.Fatal("expected duplicate extension validation error")
	}
}

func TestBuildLanguageRegistry_RejectsUnknownLanguage(t *testing.T) {
	_, err := BuildLanguageRegistry(map[string]LanguageOverride{
		"kotlin": {Extensions: []string{".kt"}},
	}, nil)
	if err == nil {
		t.Fatal("expected unknown language override error")
	}
}
