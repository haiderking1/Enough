package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/opencode"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolCallTarget stores coordinates back to the original MCP server and tool.
type ToolCallTarget struct {
	ServerName       string
	OriginalToolName string
}

// Session represents an active connection to an MCP server.
type Session struct {
	mu            sync.Mutex
	name          string
	cfg           config.MCPServerConfig
	client        *mcp.Client
	clientSession *mcp.ClientSession
	tools         []opencode.Tool
	toolMap       map[string]string // sanitized model name -> original name
	unhealthy     bool
	lastErr       error
}

// Name returns the server name.
func (s *Session) Name() string {
	return s.name
}

// IsUnhealthy returns whether the session is unhealthy.
func (s *Session) IsUnhealthy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unhealthy
}

// LastError returns the last error encountered by the session.
func (s *Session) LastError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErr
}

// Tools returns the list of registered tools for the session.
func (s *Session) Tools() []opencode.Tool {
	return s.tools
}

// Manager manages all configured MCP server sessions.
type Manager struct {
	mu          sync.Mutex
	sessions    map[string]*Session
	toolMapping map[string]ToolCallTarget // sanitized tool name -> target
}

// NewManager creates a new Manager.
func NewManager() *Manager {
	return &Manager{
		sessions:    make(map[string]*Session),
		toolMapping: make(map[string]ToolCallTarget),
	}
}

// Tools returns all registered MCP tools across all healthy sessions.
func (m *Manager) Tools() []opencode.Tool {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []opencode.Tool
	for _, s := range m.sessions {
		s.mu.Lock()
		unhealthy := s.unhealthy
		s.mu.Unlock()
		if !unhealthy {
			list = append(list, s.tools...)
		}
	}
	return list
}

// Sessions returns all managed sessions.
func (m *Manager) Sessions() map[string]*Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy of the map to avoid concurrency issues
	copyMap := make(map[string]*Session)
	for k, v := range m.sessions {
		copyMap[k] = v
	}
	return copyMap
}

// Connect initializes and connects to a single MCP server.
func (m *Manager) Connect(ctx context.Context, name string, cfg config.MCPServerConfig) (*Session, error) {
	if cfg.Command != "" && cfg.URL != "" {
		return nil, fmt.Errorf("mutually exclusive transports; both command and url specified")
	}
	if cfg.Command == "" && cfg.URL == "" {
		return nil, fmt.Errorf("no command or url specified")
	}

	var transport mcp.Transport
	if cfg.Command != "" {
		// Use a long-lived context for the child process so it is not killed when a transient tool call is cancelled.
		cmdCtx := context.Background()
		cmd := exec.CommandContext(cmdCtx, cfg.Command, cfg.Args...)
		cmd.Dir = cfg.Cwd
		if len(cfg.Env) > 0 {
			cmd.Env = os.Environ()
			for k, v := range cfg.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}
		transport = &mcp.CommandTransport{Command: cmd}
	} else {
		hc := &http.Client{}
		if len(cfg.Headers) > 0 {
			hc.Transport = &headerTransport{
				underlying: http.DefaultTransport,
				headers:    cfg.Headers,
			}
		}
		transport = &mcp.SSEClientTransport{
			Endpoint:   cfg.URL,
			HTTPClient: hc,
		}
	}

	impl := &mcp.Implementation{
		Name:    "enough-mcp-client",
		Version: "1.0.0",
	}
	client := mcp.NewClient(impl, nil)

	connectCtx := ctx
	connTimeout := 30
	if cfg.ConnectTimeout > 0 {
		connTimeout = cfg.ConnectTimeout
	} else if cfg.Timeout > 0 {
		connTimeout = cfg.Timeout
	}
	if connTimeout > 0 {
		var cancel func()
		connectCtx, cancel = context.WithTimeout(ctx, time.Duration(connTimeout)*time.Second)
		defer cancel()
	}

	clientSession, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return nil, err
	}

	listCtx := ctx
	listTimeout := 30
	if cfg.Timeout > 0 {
		listTimeout = cfg.Timeout
	}
	if listTimeout > 0 {
		var cancel func()
		listCtx, cancel = context.WithTimeout(ctx, time.Duration(listTimeout)*time.Second)
		defer cancel()
	}

	toolsResult, err := clientSession.ListTools(listCtx, &mcp.ListToolsParams{})
	if err != nil {
		clientSession.Close()
		return nil, fmt.Errorf("list tools: %w", err)
	}

	var opencodeTools []opencode.Tool
	toolMap := make(map[string]string)

	for _, t := range toolsResult.Tools {
		if !isToolAllowed(t.Name, cfg.Tools) {
			continue
		}

		paramsJSON, err := json.Marshal(t.InputSchema)
		if err != nil {
			continue
		}

		sanitized := sanitizeName(t.Name)
		modelName := fmt.Sprintf("mcp_%s_%s", name, sanitized)
		toolMap[modelName] = t.Name

		opencodeTools = append(opencodeTools, opencode.Tool{
			Type: "function",
			Function: opencode.ToolFunction{
				Name:        modelName,
				Description: t.Description,
				Parameters:  json.RawMessage(paramsJSON),
			},
		})
	}

	return &Session{
		name:          name,
		cfg:           cfg,
		client:        client,
		clientSession: clientSession,
		tools:         opencodeTools,
		toolMap:       toolMap,
	}, nil
}

// Reload stops all current connections and connects to the updated servers list.
func (m *Manager) Reload(ctx context.Context, servers map[string]config.MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Close all existing sessions
	for _, s := range m.sessions {
		if s.clientSession != nil {
			_ = s.clientSession.Close()
		}
	}

	m.sessions = make(map[string]*Session)
	m.toolMapping = make(map[string]ToolCallTarget)

	var errs []string
	for name, cfg := range servers {
		enabled := true
		if cfg.Enabled != nil {
			enabled = *cfg.Enabled
		}
		if !enabled {
			continue
		}

		s, err := m.Connect(ctx, name, cfg)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			m.sessions[name] = &Session{
				name:      name,
				cfg:       cfg,
				unhealthy: true,
				lastErr:   err,
			}
			continue
		}

		m.sessions[name] = s
		for modelName, origName := range s.toolMap {
			m.toolMapping[modelName] = ToolCallTarget{
				ServerName:       name,
				OriginalToolName: origName,
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("MCP reload failures: %s", strings.Join(errs, "; "))
	}
	return nil
}

// CallTool routes a tool call to the appropriate MCP session.
func (m *Manager) CallTool(ctx context.Context, modelToolName string, argsJSON string) (opencode.ToolContentBlock, []opencode.ToolContentBlock, bool, error) {
	m.mu.Lock()
	target, ok := m.toolMapping[modelToolName]
	var s *Session
	if ok {
		s = m.sessions[target.ServerName]
	}
	m.mu.Unlock()

	if !ok || s == nil {
		return opencode.ToolContentBlock{}, nil, true, fmt.Errorf("tool %q not found", modelToolName)
	}

	s.mu.Lock()
	unhealthy := s.unhealthy
	lastErr := s.lastErr
	s.mu.Unlock()

	if unhealthy {
		return opencode.ToolContentBlock{}, nil, true, fmt.Errorf("server %q is unhealthy: %v", s.name, lastErr)
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return opencode.ToolContentBlock{}, nil, true, fmt.Errorf("invalid arguments JSON: %w", err)
	}

	params := &mcp.CallToolParams{
		Name:      target.OriginalToolName,
		Arguments: args,
	}

	callCtx := ctx
	timeout := 30
	if s.cfg.Timeout > 0 {
		timeout = s.cfg.Timeout
	}
	if timeout > 0 {
		var cancel func()
		callCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	res, err := s.clientSession.CallTool(callCtx, params)
	if err != nil {
		// Mark session unhealthy if context wasn't cancelled but connection failed
		if ctx.Err() == nil && (errors.Is(err, mcp.ErrConnectionClosed) || strings.Contains(err.Error(), "connection")) {
			s.mu.Lock()
			s.unhealthy = true
			s.lastErr = err
			s.mu.Unlock()
		}

		if errors.Is(err, context.Canceled) || (callCtx.Err() != nil && errors.Is(callCtx.Err(), context.Canceled)) {
			return opencode.ToolContentBlock{
				Type: "text",
				Text: "[interrupted]",
			}, nil, true, nil
		}

		return opencode.ToolContentBlock{}, nil, true, err
	}

	var textParts []string
	var contentBlocks []opencode.ToolContentBlock
	totalLen := 0
	truncated := false

	for _, c := range res.Content {
		switch val := c.(type) {
		case *mcp.TextContent:
			txt := val.Text
			if !truncated {
				if totalLen+len(txt) > 32000 {
					room := 32000 - totalLen
					if room > 0 {
						txt = txt[:room]
					} else {
						txt = ""
					}
					txt += "\n... truncated ..."
					truncated = true
				}
				totalLen += len(txt)
				if txt != "" {
					textParts = append(textParts, txt)
					contentBlocks = append(contentBlocks, opencode.ToolContentBlock{
						Type: "text",
						Text: txt,
					})
				}
			}
		case *mcp.ImageContent:
			encoded := base64.StdEncoding.EncodeToString(val.Data)
			contentBlocks = append(contentBlocks, opencode.ToolContentBlock{
				Type:     "image",
				Data:     encoded,
				MIMEType: val.MIMEType,
			})
		}
	}

	if len(textParts) == 0 && res.StructuredContent != nil {
		data, err := json.Marshal(res.StructuredContent)
		if err == nil {
			txt := string(data)
			if len(txt) > 32000 {
				txt = txt[:32000] + "\n... truncated ..."
			}
			textParts = append(textParts, txt)
			contentBlocks = append(contentBlocks, opencode.ToolContentBlock{
				Type: "text",
				Text: txt,
			})
		}
	}

	outputText := strings.Join(textParts, "\n")
	return opencode.ToolContentBlock{
		Type: "text",
		Text: outputText,
	}, contentBlocks, res.IsError, nil
}

// Close closes all active MCP client sessions.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.sessions {
		if s.clientSession != nil {
			_ = s.clientSession.Close()
		}
	}
	m.sessions = make(map[string]*Session)
	m.toolMapping = make(map[string]ToolCallTarget)
}

type headerTransport struct {
	underlying http.RoundTripper
	headers    map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())
	for k, v := range t.headers {
		reqCopy.Header.Set(k, v)
	}
	return t.underlying.RoundTrip(reqCopy)
}

func sanitizeName(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

func isToolAllowed(name string, filter *config.MCPServerToolsConfig) bool {
	if filter == nil {
		return true
	}
	if len(filter.Include) > 0 {
		found := false
		for _, inc := range filter.Include {
			if inc == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(filter.Exclude) > 0 {
		for _, exc := range filter.Exclude {
			if exc == name {
				return false
			}
		}
	}
	return true
}
