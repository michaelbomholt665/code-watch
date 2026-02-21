package transport

import (
	"circular/internal/core/config"
	"circular/internal/mcp/contracts"
	"circular/internal/mcp/schema"
	"circular/internal/shared/util"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SSE struct {
	address string
	server  *http.Server
	handler Handler
	cfg     config.MCPRateLimit

	sessions   map[string]*sseSession
	sessionsMu sync.RWMutex

	requestLimiter    *util.LimiterRegistry
	connectionLimiter *util.LimiterRegistry
}

type sseSession struct {
	id        string
	messages  chan any
	createdAt time.Time
}

func NewSSE(address string, cfg config.MCPRateLimit) (Adapter, error) {
	s := &SSE{
		address:  address,
		cfg:      cfg,
		sessions: make(map[string]*sseSession),
	}

	if cfg.Enabled {
		// Convert RPM to tokens per second
		reqRate := float64(cfg.SSERequestsPerMinute) / 60.0
		connRate := float64(cfg.SSEConnectionsPerMinute) / 60.0

		s.requestLimiter = util.NewLimiterRegistry(reqRate, cfg.Burst, 10*time.Minute)
		s.connectionLimiter = util.NewLimiterRegistry(connRate, 5, 10*time.Minute)
	}

	return s, nil
}

func (s *SSE) Start(ctx context.Context, handler Handler) error {
	s.handler = handler

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)

	s.server = &http.Server{
		Addr:    s.address,
		Handler: mux,
	}

	errChan := make(chan error, 1)
	go func() {
		log.Printf("MCP SSE server listening on %s", s.address)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return s.Stop()
	}
}

func (s *SSE) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *SSE) handleSSE(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Enabled && s.connectionLimiter != nil {
		ip := util.GetClientIP(r)
		if !s.connectionLimiter.Get(ip).Allow(1) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many connections", http.StatusTooManyRequests)
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	sessionID := uuid.New().String()
	session := &sseSession{
		id:        sessionID,
		messages:  make(chan any, 32),
		createdAt: time.Now(),
	}

	s.sessionsMu.Lock()
	s.sessions[sessionID] = session
	s.sessionsMu.Unlock()

	defer func() {
		s.sessionsMu.Lock()
		delete(s.sessions, sessionID)
		s.sessionsMu.Unlock()
	}()

	// Send endpoint event
	fmt.Fprintf(w, "event: endpoint\ndata: /message?session_id=%s\n\n", sessionID)
	flusher.Flush()

	for {
		select {
		case msg := <-session.messages:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
			flusher.Flush()
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Second):
			// Keep-alive
			fmt.Fprintf(w, ":\n\n")
			flusher.Flush()
		}
	}
}

func (s *SSE) handleMessage(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id", http.StatusBadRequest)
		return
	}

	s.sessionsMu.RLock()
	session, ok := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !ok {
		http.Error(w, "Invalid session_id", http.StatusNotFound)
		return
	}

	if s.cfg.Enabled && s.requestLimiter != nil {
		ip := util.GetClientIP(r)
		if !s.requestLimiter.Get(ip).Allow(1) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Process message asynchronously and send response back through SSE channel
	go func() {
		resp := s.processRPC(context.Background(), raw)
		if resp != nil {
			session.messages <- resp
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (s *SSE) processRPC(ctx context.Context, raw map[string]any) any {
	method, hasMethod := raw["method"].(string)
	jsonrpc, _ := raw["jsonrpc"].(string)

	if !hasMethod || method == "" || jsonrpc == "" {
		// Legacy or malformed
		req := parseLegacyToolRequest(raw)
		if req.Tool == "" {
			return nil
		}
		result, err := s.handler(ctx, req.Tool, req.Args)
		resp := toolResponse{ID: req.ID}
		if err != nil {
			toolErr := normalizeToolError(err)
			resp.OK = false
			resp.Error = &toolErr
		} else {
			resp.OK = true
			resp.Result = result
		}
		return resp
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
		return nil
	}

	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2024-11-05",
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
		result, err := s.handler(ctx, name, args)
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

	return resp
}
