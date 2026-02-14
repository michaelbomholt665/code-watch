package secrets

import (
	"circular/internal/core/ports"
	"testing"
)

func TestAdapter_Detect(t *testing.T) {
	adapter, err := NewAdapter(Config{})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	findings := adapter.Detect("main.go", []byte(`package main
const token = "AKIA1234567890ABCDEF"
`))
	if len(findings) == 0 {
		t.Fatal("expected secret findings")
	}
}

func TestAdapter_DetectInRanges(t *testing.T) {
	adapter, err := NewAdapter(Config{})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	findings := adapter.DetectInRanges("main.go", []byte("const a = 1\nconst token = \"AKIA1234567890ABCDEF\"\n"), []ports.LineRange{
		{Start: 1, End: 1},
	})
	if len(findings) != 0 {
		t.Fatalf("expected no findings in unchanged range, got %#v", findings)
	}
}
