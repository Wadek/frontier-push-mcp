package mcpstdio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Minimal MCP subset over stdio (JSON-RPC 2.0, Content-Length framing optional:
// we also accept newline-delimited JSON for easy local debugging).

type Server struct {
	Name    string
	Version string
	Tools   []Tool
	Call    func(name string, args map[string]any) (string, error)
	In      io.Reader
	Out     io.Writer
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func (s *Server) Run() error {
	if s.In == nil {
		s.In = os.Stdin
	}
	if s.Out == nil {
		s.Out = os.Stdout
	}
	br := bufio.NewReader(s.In)
	for {
		msg, err := readMessage(br)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(msg) == 0 {
			continue
		}
		var req rpcReq
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if req.Method == "" {
			continue
		}
		// notifications have no response
		if req.ID == nil && (req.Method == "notifications/initialized" || req.Method == "initialized") {
			continue
		}
		resp := s.handle(req)
		if resp == nil {
			continue
		}
		if err := writeMessage(s.Out, resp); err != nil {
			return err
		}
	}
}

func (s *Server) handle(req rpcReq) map[string]any {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    s.Name,
					"version": s.Version,
				},
				"instructions": "Frontier AI-first push MCP. Start as observer. Elevate one step at a time. Gate before push. Local ledger records everything.",
			},
		}
	case "ping":
		return map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}}
	case "tools/list":
		tools := make([]map[string]any, 0, len(s.Tools))
		for _, t := range s.Tools {
			tools = append(tools, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": t.InputSchema,
			})
		}
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  map[string]any{"tools": tools},
		}
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &p)
		if p.Arguments == nil {
			p.Arguments = map[string]any{}
		}
		text, err := s.Call(p.Name, p.Arguments)
		if err != nil {
			return map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"content": []map[string]any{{"type": "text", "text": "error: " + err.Error()}},
					"isError": true,
				},
			}
		}
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"content": []map[string]any{{"type": "text", "text": text}},
			},
		}
	default:
		if req.ID == nil {
			return nil
		}
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error": map[string]any{
				"code":    -32601,
				"message": fmt.Sprintf("method not found: %s", req.Method),
			},
		}
	}
}

func readMessage(br *bufio.Reader) ([]byte, error) {
	// Try Content-Length framing first by peeking
	peek, err := br.Peek(1)
	if err != nil {
		return nil, err
	}
	if peek[0] == '{' {
		line, err := br.ReadBytes('\n')
		return trimCR(line), err
	}
	// header mode
	var contentLen int
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = string(trimCR([]byte(line)))
		if line == "" {
			break
		}
		var n int
		if _, err := fmt.Sscanf(line, "Content-Length: %d", &n); err == nil {
			contentLen = n
		}
	}
	if contentLen <= 0 {
		return nil, nil
	}
	buf := make([]byte, contentLen)
	_, err = io.ReadFull(br, buf)
	return buf, err
}

func writeMessage(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(b), b)
	return err
}

func trimCR(b []byte) []byte {
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	if len(b) > 0 && b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}
	return b
}
