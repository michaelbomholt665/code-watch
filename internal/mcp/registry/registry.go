package registry

import (
	"context"
	"fmt"
	"sync"
)

type Handler func(ctx context.Context, input any) (any, error)

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	order    []string
}

func New() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
		order:    make([]string, 0),
	}
}

func (r *Registry) Register(tool string, handler Handler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if tool == "" {
		return fmt.Errorf("tool name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[tool]; exists {
		return fmt.Errorf("tool already registered: %s", tool)
	}
	r.handlers[tool] = handler
	r.order = append(r.order, tool)
	return nil
}

func (r *Registry) HandlerFor(tool string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	h, ok := r.handlers[tool]
	return h, ok
}

func (r *Registry) Tools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}
