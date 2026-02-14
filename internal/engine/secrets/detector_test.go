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

func TestDetector_DetectInRanges(t *testing.T) {
	d, err := NewDetector(Config{})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	content := []byte("const ok = \"hello\"\nconst key = \"AKIA1234567890ABCDEF\"\n")
	findings := d.DetectInRanges("main.go", content, []LineRange{{Start: 1, End: 1}})
	if len(findings) != 0 {
		t.Fatalf("expected no findings outside selected line range, got %#v", findings)
	}

	findings = d.DetectInRanges("main.go", content, []LineRange{{Start: 2, End: 2}})
	if len(findings) == 0 {
		t.Fatal("expected finding in changed line range")
	}
}

func TestDetector_EntropyGatedByHighRiskExtensions(t *testing.T) {
	d, err := NewDetector(Config{EntropyThreshold: 4.0, MinTokenLength: 12})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	content := []byte("value = \"A1b2C3d4E5f6G7h8I9j0\"\n")
	if findings := d.Detect("main.go", content); len(findings) != 0 {
		t.Fatalf("expected entropy finding to be skipped for non high-risk extension, got %#v", findings)
	}
	if findings := d.Detect(".env", content); len(findings) == 0 {
		t.Fatal("expected entropy finding for high-risk extension")
	}
}

func TestChangedLineRanges(t *testing.T) {
	prev := []byte("a\nb\nc\n")
	curr := []byte("a\nbx\nc\n")
	ranges := ChangedLineRanges(prev, curr)
	if len(ranges) != 1 {
		t.Fatalf("expected one changed range, got %d", len(ranges))
	}
	if ranges[0].Start != 2 || ranges[0].End != 2 {
		t.Fatalf("expected changed range 2..2, got %d..%d", ranges[0].Start, ranges[0].End)
	}
}
