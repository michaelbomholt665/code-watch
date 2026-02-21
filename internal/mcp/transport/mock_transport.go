package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// MockAdapter implements transport.Adapter for end-to-end testing.
type MockAdapter struct {
	mu       sync.Mutex
	started  bool
	handler  Handler
	requests chan mockRequest
}

type mockRequest struct {
	tool string
	args map[string]any
	res  chan mockResponse
}

type mockResponse struct {
	result any
	err    error
}

func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		requests: make(chan mockRequest),
	}
}

func (m *MockAdapter) Start(ctx context.Context, handler Handler) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return fmt.Errorf("mock adapter already started")
	}
	m.started = true
	m.handler = handler
	m.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req := <-m.requests:
			res, err := m.handler(ctx, req.tool, req.args)
			req.res <- mockResponse{result: res, err: err}
		}
	}
}

func (m *MockAdapter) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
	return nil
}

// Call simulates a tool call from a client.
func (m *MockAdapter) Call(tool string, args map[string]any) (any, error) {
	resChan := make(chan mockResponse)
	m.requests <- mockRequest{
		tool: tool,
		args: args,
		res:  resChan,
	}
	resp := <-resChan
	return resp.result, resp.err
}

// Helper to marshal/unmarshal for realism if needed.
func (m *MockAdapter) CallJSON(tool string, args map[string]any) (any, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	var mapArgs map[string]any
	if err := json.Unmarshal(data, &mapArgs); err != nil {
		return nil, err
	}
	return m.Call(tool, mapArgs)
}
