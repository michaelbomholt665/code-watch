package secrets

import "testing"

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
