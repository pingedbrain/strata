package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

var fixture = fstest.MapFS{
	"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# Auth

## Requirements

### Requirement: Token expiry
Tokens expire.

#### Scenario: Expired rejected
- **WHEN** token is old
- **THEN** reject it
`)},
	"auth.go": &fstest.MapFile{Data: []byte(`package auth

// @spec auth/token-expiry
func Validate() {}
`)},
}

func roundTrip(t *testing.T, srv *Server, lines ...string) []json.RawMessage {
	t.Helper()
	var in bytes.Buffer
	for _, l := range lines {
		in.WriteString(l + "\n")
	}
	var out bytes.Buffer
	if err := srv.Serve(&in, &out); err != nil {
		t.Fatal(err)
	}
	var resps []json.RawMessage
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l != "" {
			resps = append(resps, json.RawMessage(l))
		}
	}
	return resps
}

func TestInitializeAndToolsList(t *testing.T) {
	srv := &Server{Root: fixture}
	resps := roundTrip(t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	if len(resps) != 2 {
		t.Fatalf("want 2 responses (init + tools/list), got %d", len(resps))
	}
	var init struct {
		Result struct {
			ServerInfo struct{ Name string } `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resps[0], &init); err != nil {
		t.Fatal(err)
	}
	if init.Result.ServerInfo.Name != "spec-blame" {
		t.Fatalf("serverInfo: %+v", init.Result.ServerInfo)
	}
	var tl struct {
		Result struct {
			Tools []struct{ Name string } `json:"tools"`
		} `json:"result"`
	}
	json.Unmarshal(resps[1], &tl)
	if len(tl.Result.Tools) < 3 {
		t.Fatalf("want ≥3 tools, got %+v", tl.Result.Tools)
	}
}

func TestToolsCall(t *testing.T) {
	srv := &Server{Root: fixture}
	resps := roundTrip(t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"strata_reqs_for_file","arguments":{"file":"auth.go"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"strata_coverage","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"nope","arguments":{}}}`,
	)
	var call struct {
		Result struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"result"`
		Error *struct{ Message string } `json:"error"`
	}
	json.Unmarshal(resps[0], &call)
	if !strings.Contains(call.Result.Content[0].Text, "auth/token-expiry") {
		t.Fatalf("reqs_for_file: %+v", call.Result)
	}
	json.Unmarshal(resps[1], &call)
	if !strings.Contains(call.Result.Content[0].Text, "1/1") {
		t.Fatalf("coverage: %+v", call.Result)
	}
	json.Unmarshal(resps[2], &call)
	if call.Error == nil {
		t.Fatal("unknown tool must error")
	}
}

func TestSymbolTools(t *testing.T) {
	srv := &Server{Root: fixture}
	resps := roundTrip(t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"strata_req_for_symbol","arguments":{"symbol":"Validate"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"strata_symbols_for_req","arguments":{"id":"auth/token-expiry"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"strata_unlinked","arguments":{}}}`,
	)
	var call struct {
		Result struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"result"`
	}
	json.Unmarshal(resps[0], &call)
	if !strings.Contains(call.Result.Content[0].Text, "Validate") ||
		!strings.Contains(call.Result.Content[0].Text, "auth/token-expiry") {
		t.Fatalf("req_for_symbol: %+v", call.Result)
	}
	json.Unmarshal(resps[1], &call)
	if !strings.Contains(call.Result.Content[0].Text, "Validate") {
		t.Fatalf("symbols_for_req should list Validate: %+v", call.Result)
	}
	json.Unmarshal(resps[2], &call)
	if !strings.Contains(call.Result.Content[0].Text, "no unlinked") {
		t.Fatalf("fixture has no unlinked exports: %+v", call.Result)
	}
}
