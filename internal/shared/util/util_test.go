package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePatternPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "Empty", input: "", expected: ""},
		{name: "Dot", input: ".", expected: ""},
		{name: "Trim", input: "  ./foo/bar  ", expected: "foo/bar"},
		{name: "Relative", input: "foo/../bar", expected: "bar"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizePatternPath(tc.input); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestHasPathPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		path     string
		prefix   string
		expected bool
	}{
		{name: "Exact", path: "foo/bar", prefix: "foo/bar", expected: true},
		{name: "Nested", path: "foo/bar/baz", prefix: "foo/bar", expected: true},
		{name: "Neighbor", path: "foo/barista", prefix: "foo/bar", expected: false},
		{name: "Shorter", path: "foo", prefix: "foo/bar", expected: false},
		{name: "MixedSeparators", path: `foo\bar\baz`, prefix: "foo/bar", expected: true},
		{name: "RelativePrefix", path: "./foo/bar/baz", prefix: "foo/bar", expected: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HasPathPrefix(tc.path, tc.prefix); got != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestContainsPathSeparator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		value    string
		expected bool
	}{
		{name: "Unix", value: "foo/bar", expected: true},
		{name: "Windows", value: `foo\bar`, expected: true},
		{name: "Flat", value: "graph.mmd", expected: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ContainsPathSeparator(tc.value); got != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestSortedStringKeys(t *testing.T) {
	t.Parallel()

	m := map[string]int{"b": 2, "a": 1, "c": 3}
	keys := SortedStringKeys(m)
	expected := []string{"a", "b", "c"}
	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}
	for i, key := range expected {
		if keys[i] != key {
			t.Fatalf("expected %q at %d, got %q", key, i, keys[i])
		}
	}
}

func TestWriteFileWithDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "file.txt")
	content := []byte("hello")

	if err := WriteFileWithDirs(path, content, 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("expected %q, got %q", string(content), string(got))
	}
}

func TestWriteStringWithDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "file.txt")

	if err := WriteStringWithDirs(path, "hello", 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("expected %q, got %q", "hello", string(got))
	}
}
