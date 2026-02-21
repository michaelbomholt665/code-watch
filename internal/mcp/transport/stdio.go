package transport

import (
	"bufio"
	"circular/internal/core/config"
	"circular/internal/mcp/contracts"
	"circular/internal/mcp/schema"
	"circular/internal/shared/util"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

type Handler func(ctx context.Context, tool string, raw map[string]any) (any, error)

type Adapter interface {
	Start(ctx context.Context, handler Handler) error
	Stop() error
}

type Stdio struct {
	cfg     config.MCPRateLimit
	limiter *util.Limiter

	mu      sync.Mutex
	running bool
}

func NewStdio(cfg config.MCPRateLimit) (Adapter, error) {
	s := &Stdio{cfg: cfg}
	if cfg.Enabled {
		rate := float64(cfg.RequestsPerMinute) / 60.0
		s.limiter = util.NewLimiter(rate, cfg.Burst)
	}
	return s, nil
}

func (s *Stdio) Start(ctx context.Context, handler Handler) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		<-ctx.Done()
		return ctx.Err()
	}
	s.running = true
	s.mu.Unlock()

	if err := s.serve(ctx, handler); err != nil && !errors.Is(err, context.Canceled) {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	return ctx.Err()
}

func (s *Stdio) Stop() error {
	return nil
}

type toolRequest struct {
	ID   any            `json:"id,omitempty"`
	Tool string         `json:"tool"`
	Args map[string]any `json:"args,omitempty"`
}

type toolResponse struct {
	ID     any                  `json:"id,omitempty"`
	OK     bool                 `json:"ok"`
	Result any                  `json:"result,omitempty"`
	Error  *contracts.ToolError `json:"error,omitempty"`
}

type rpcRequest struct {
	JSONRPC string         `json:"jsonrpc,omitempty"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func (s *Stdio) serve(ctx context.Context, handler Handler) error {
	if handler == nil {
		return contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "stdio handler is required"}
	}

	decoder := json.NewDecoder(bufio.NewReader(os.Stdin))
	writer := bufio.NewWriter(os.Stdout)
	encoder := json.NewEncoder(writer)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		var raw map[string]any
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if s.cfg.Enabled && s.limiter != nil {
			if !s.limiter.Allow(1) {
				// For stdio, we can't easily return 429, but we can return an error response
				// However, if we don't know the ID yet, it's hard.
				// For now, we'll just log and continue, or we can send a generic error if possible.
				// Better to try to parse ID if possible.
				reqID, _ := raw["id"]
				resp := rpcResponse{
					JSONRPC: "2.0",
					ID:      reqID,
					Error: &rpcError{
						Code:    -32005, // Rate limit exceeded (JSON-RPC reserved range)
						Message: "Rate limit exceeded",
					},
				}
				if err := encoder.Encode(resp); err != nil {
					return err
				}
				if err := writer.Flush(); err != nil {
					return err
				}
				continue
			}
		}

		handled, err := s.handleRPCMessage(ctx, handler, raw, encoder, writer)
		if err != nil {
			return err
		}
		if handled {
			continue
		}

		req := parseLegacyToolRequest(raw)
		if req.Args == nil {
			req.Args = map[string]any{}
		}

		result, callErr := handler(ctx, req.Tool, req.Args)
		resp := toolResponse{ID: req.ID}
		if callErr != nil {
			toolErr := normalizeToolError(callErr)
			resp.OK = false
			resp.Error = &toolErr
		} else {
			resp.OK = true
			resp.Result = result
		}

		if err := encoder.Encode(resp); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
}

func parseLegacyToolRequest(raw map[string]any) toolRequest {
	req := toolRequest{}
	if id, ok := raw["id"]; ok {
		req.ID = id
	}
	if tool, ok := raw["tool"].(string); ok {
		req.Tool = tool
	}
	if args, ok := raw["args"].(map[string]any); ok {
		req.Args = args
	}
	return req
}

func (s *Stdio) handleRPCMessage(ctx context.Context, handler Handler, raw map[string]any, encoder *json.Encoder, writer *bufio.Writer) (bool, error) {
	method, hasMethod := raw["method"].(string)
	if !hasMethod || method == "" {
		return false, nil
	}
	jsonrpc, _ := raw["jsonrpc"].(string)
	if jsonrpc == "" {
		return false, nil
	}

	req := rpcRequest{
		JSONRPC: jsonrpc,
		Method:  method,
		Params:  map[string]any{},
	}
	if id, ok := raw["id"]; ok {
		req.ID = id
	}
	if params, ok := raw["params"].(map[string]any); ok {
		req.Params = params
	}

	if req.Method == "notifications/initialized" {
		return true, nil
	}

	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    contracts.ToolNameCircular,
				"version": contracts.ContractVersion,
			},
		}
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		toolDefs := schema.BuildToolDefinitions()
		tools := make([]map[string]any, 0, len(toolDefs))
		for _, def := range toolDefs {
			tools = append(tools, map[string]any{
				"name":        def.Name,
				"description": def.Description,
				"inputSchema": def.InputSchema,
			})
		}
		resp.Result = map[string]any{"tools": tools}
	case "tools/call":
		name, _ := req.Params["name"].(string)
		args, _ := req.Params["arguments"].(map[string]any)
		if args == nil {
			args = map[string]any{}
		}
		result, err := handler(ctx, name, args)
		if err != nil {
			toolErr := normalizeToolError(err)
			resp.Result = map[string]any{
				"isError": true,
				"content": []map[string]any{
					{
						"type": "text",
						"text": fmt.Sprintf("%s: %s", toolErr.Code, toolErr.Message),
					},
				},
			}
		} else {
			text := mustJSONText(result)
			resp.Result = map[string]any{
				"isError":           false,
				"structuredContent": result,
				"content": []map[string]any{
					{
						"type": "text",
						"text": text,
					},
				},
			}
		}
	default:
		resp.Error = &rpcError{
			Code:    -32601,
			Message: "Method not found",
		}
	}

	if err := encoder.Encode(resp); err != nil {
		return true, err
	}
	if err := writer.Flush(); err != nil {
		return true, err
	}
	return true, nil
}

func mustJSONText(v any) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func normalizeToolError(err error) contracts.ToolError {
	var toolErr contracts.ToolError
	if errors.As(err, &toolErr) {
		return toolErr
	}
	return contracts.ToolError{Code: contracts.ErrorInternal, Message: err.Error()}
}
