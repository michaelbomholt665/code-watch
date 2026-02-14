package openapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

const maxSpecSizeBytes = 8 << 20 // 8 MiB

func LoadSpec(path string) (*openapi3.T, error) {
	source := strings.TrimSpace(path)
	if source == "" {
		return nil, fmt.Errorf("openapi source is required")
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	var (
		doc *openapi3.T
		err error
	)
	if isHTTPSource(source) {
		doc, err = loadSpecFromURL(loader, source)
	} else {
		if _, statErr := os.Stat(source); statErr != nil {
			return nil, fmt.Errorf("openapi spec path %q: %w", source, statErr)
		}
		doc, err = loader.LoadFromFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("load openapi spec from %q: %w", source, err)
	}
	if doc == nil {
		return nil, fmt.Errorf("openapi spec %q resolved to nil document", source)
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate openapi spec %q: %w", source, err)
	}
	return doc, nil
}

func isHTTPSource(source string) bool {
	lower := strings.ToLower(source)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func loadSpecFromURL(loader *openapi3.Loader, source string) (*openapi3.T, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(source)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSpecSizeBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSpecSizeBytes {
		return nil, fmt.Errorf("spec exceeds %d bytes", maxSpecSizeBytes)
	}
	return loader.LoadFromData(data)
}
