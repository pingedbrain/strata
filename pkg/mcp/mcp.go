// Package mcp exposes the traceability graph over the Model Context
// Protocol (stdio, newline-delimited JSON-RPC 2.0) so agents can query
// requirements while they code. Zero deps — hand-rolled transport.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/pingedbrain/strata/pkg/check"
)

// Server answers MCP requests over Root's repo.
type Server struct {
	Root fs.FS
}

type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcErr         `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Serve reads newline-delimited JSON-RPC from in, writes to out.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	w := bufio.NewWriter(out)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue // ignore malformed input
		}
		resp, ok := s.handle(req)
		if !ok { // notification — no response
			continue
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
		w.WriteByte('\n')
		w.Flush()
	}
	return sc.Err()
}

func (s *Server) handle(req request) (response, bool) {
	r := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		r.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]string{"name": "spec-blame", "version": "0.2.0"},
		}
	case "ping":
		r.Result = map[string]any{}
	case "notifications/initialized", "notifications/cancelled":
		return r, false
	case "tools/list":
		r.Result = map[string]any{"tools": toolList()}
	case "tools/call":
		res, err := s.callTool(req.Params)
		if err != nil {
			r.Error = &rpcErr{Code: -32602, Message: err.Error()}
		} else {
			r.Result = res
		}
	default:
		r.Error = &rpcErr{Code: -32601, Message: "method not found: " + req.Method}
	}
	return r, true
}

func toolList() []map[string]any {
	prop := func(name, desc string) map[string]any {
		return map[string]any{"type": "object",
			"properties": map[string]any{name: map[string]string{"type": "string", "description": desc}},
			"required":   []string{name}}
	}
	empty := map[string]any{"type": "object", "properties": map[string]any{}}
	return []map[string]any{
		{"name": "strata_reqs_for_file", "description": "Requirements implemented by a source file",
			"inputSchema": prop("file", "repo-relative path or basename")},
		{"name": "strata_files_for_req", "description": "Files implementing a requirement",
			"inputSchema": prop("id", "requirement id, e.g. auth/token-expiry")},
		{"name": "strata_coverage", "description": "Requirement coverage summary", "inputSchema": empty},
		{"name": "strata_stale", "description": "Markers whose bound spec changed since annotation", "inputSchema": empty},
		{"name": "strata_dangling", "description": "Markers pointing at nonexistent requirements", "inputSchema": empty},
		{"name": "strata_req_for_symbol", "description": "Requirements linked to a code symbol (func/type/method name)",
			"inputSchema": prop("symbol", "symbol name, e.g. Validate or Client.Validate")},
		{"name": "strata_symbols_for_req", "description": "Code symbols implementing a requirement",
			"inputSchema": prop("id", "requirement id, e.g. auth/token-expiry")},
		{"name": "strata_unlinked", "description": "Exported symbols with no requirement marker (dead-code radar)", "inputSchema": empty},
	}
}

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func textResult(s string) map[string]any {
	return map[string]any{"content": []map[string]string{{"type": "text", "text": s}}}
}

// matchSymbol compares a bound symbol name to a query: exact match, or
// unqualified tail for dotted method names (Client.Validate matches
// "Validate" and "Client.Validate").
func matchSymbol(bound, query string) bool {
	if bound == "" || query == "" {
		return false
	}
	return bound == query || strings.HasSuffix(bound, "."+query)
}

func (s *Server) callTool(raw json.RawMessage) (map[string]any, error) {
	var p callParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	var args struct {
		File   string `json:"file"`
		ID     string `json:"id"`
		Symbol string `json:"symbol"`
	}
	json.Unmarshal(p.Arguments, &args)

	rep, err := check.Run(s.Root, check.Options{})
	if err != nil {
		return nil, err
	}

	switch p.Name {
	case "strata_reqs_for_file":
		var b strings.Builder
		for _, e := range rep.Result.EdgesIn(args.File) {
			status := "ok"
			if e.Stale {
				status = "stale"
			}
			r := rep.Graph.Requirements[e.ReqID]
			fmt.Fprintf(&b, "%s → %s [%s] %s\n", e.File, e.ReqID, status, r.Title)
		}
		for _, m := range rep.Result.DanglingIn(args.File) {
			fmt.Fprintf(&b, "%s → %s [dangling]\n", m.File, m.ReqID)
		}
		if b.Len() == 0 {
			b.WriteString("no markers in " + args.File)
		}
		return textResult(b.String()), nil
	case "strata_files_for_req":
		r, ok := rep.Graph.Requirements[args.ID]
		if !ok {
			return nil, fmt.Errorf("no such requirement: %s", args.ID)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%s — %s (%s)\n", r.ID, r.Title, r.Source)
		for _, e := range rep.Result.EdgesFor(args.ID) {
			fmt.Fprintf(&b, "  %s:%d (%s)\n", e.File, e.Line, e.Kind)
		}
		return textResult(b.String()), nil
	case "strata_coverage":
		cov, tot := rep.Result.Coverage()
		return textResult(fmt.Sprintf("%d/%d requirements linked", cov, tot)), nil
	case "strata_stale":
		var b strings.Builder
		for _, e := range rep.Result.Edges {
			if e.Stale {
				fmt.Fprintf(&b, "%s:%d → %s\n", e.File, e.Line, e.ReqID)
			}
		}
		if b.Len() == 0 {
			b.WriteString("no stale markers")
		}
		return textResult(b.String()), nil
	case "strata_dangling":
		var b strings.Builder
		for _, m := range rep.Result.Dangling {
			fmt.Fprintf(&b, "%s:%d → %s\n", m.File, m.Line, m.ReqID)
		}
		if b.Len() == 0 {
			b.WriteString("no dangling markers")
		}
		return textResult(b.String()), nil
	case "strata_req_for_symbol":
		var b strings.Builder
		for _, e := range rep.Result.Edges {
			if !matchSymbol(e.Symbol, args.Symbol) {
				continue
			}
			status := "ok"
			if e.Stale {
				status = "stale"
			}
			r := rep.Graph.Requirements[e.ReqID]
			fmt.Fprintf(&b, "%s %s:%d → %s [%s] %s\n", e.Symbol, e.File, e.Line, e.ReqID, status, r.Title)
		}
		if b.Len() == 0 {
			b.WriteString("no requirement links to symbol " + args.Symbol)
		}
		return textResult(b.String()), nil
	case "strata_symbols_for_req":
		r, ok := rep.Graph.Requirements[args.ID]
		if !ok {
			return nil, fmt.Errorf("no such requirement: %s", args.ID)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%s — %s (%s)\n", r.ID, r.Title, r.Source)
		for _, e := range rep.Result.EdgesFor(args.ID) {
			sym := e.Symbol
			if sym == "" {
				sym = "(file-level marker)"
			}
			fmt.Fprintf(&b, "  %s:%d %s (%s)\n", e.File, e.Line, sym, e.Kind)
		}
		return textResult(b.String()), nil
	case "strata_unlinked":
		var b strings.Builder
		for _, s := range rep.Result.Unlinked {
			fmt.Fprintf(&b, "%s:%d %s (%s)\n", s.File, s.Line, s.Name, s.Kind)
		}
		if b.Len() == 0 {
			b.WriteString("no unlinked exported symbols")
		}
		return textResult(b.String()), nil
	}
	return nil, fmt.Errorf("unknown tool: %s", p.Name)
}
