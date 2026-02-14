package secrets

import (
	"testing"
)

func TestDetector_DetectsBuiltInPattern(t *testing.T) {
	d, err := NewDetector(Config{})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	content := []byte("package main\nconst key = \"AKIA1234567890ABCDEF\"\n")
	findings := d.Detect("main.go", content)
	if len(findings) == 0 {
		t.Fatal("expected at least one secret finding")
	}
	if findings[0].Kind != "aws-access-key-id" {
		t.Fatalf("expected aws-access-key-id finding, got %q", findings[0].Kind)
	}
}

func TestDetector_DetectsContextSensitiveAssignment(t *testing.T) {
	d, err := NewDetector(Config{EntropyThreshold: 3.5, MinTokenLength: 16})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	content := []byte("password = \"P4s$w0rdVeryLongToken99\"\n")
	findings := d.Detect("app.py", content)
	if len(findings) == 0 {
		t.Fatal("expected context finding")
	}

	found := false
	for _, finding := range findings {
		if finding.Kind == "sensitive-assignment" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected sensitive-assignment finding, got %#v", findings)
	}
}

func TestDetector_SkipsObviousPlaceholder(t *testing.T) {
	d, err := NewDetector(Config{EntropyThreshold: 3.0, MinTokenLength: 10})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	content := []byte("api_key = \"example_test_token_123456\"\n")
	findings := d.Detect("config.py", content)
	if len(findings) != 0 {
		t.Fatalf("expected no findings for placeholder token, got %#v", findings)
	}
}

func TestMaskValue(t *testing.T) {
	if got := MaskValue("ABCDEFGH"); got != "********" {
		t.Fatalf("unexpected short mask result: %q", got)
	}
	if got := MaskValue("ABCDEFGHIJKLMNOP"); got != "ABCD...MNOP" {
		t.Fatalf("unexpected long mask result: %q", got)
	}
}
