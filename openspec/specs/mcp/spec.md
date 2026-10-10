# Mcp Specification

## Requirements

### Requirement: Server
Server answers MCP requests over Root's repo.

_Evidence: pkg/mcp/mcp.go:18_

### Requirement: Server.Serve
Serve reads newline-delimited JSON-RPC from in, writes to out.

_Evidence: pkg/mcp/mcp.go:41_

