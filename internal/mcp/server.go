// Package mcp implements a minimal Model Context Protocol server over stdio.
//
// We don't pull in a third-party MCP SDK — the protocol is small enough that
// 200 lines of hand-rolled JSON-RPC is cheaper than the dep, gives us tight
// control over what we expose, and avoids any abandonment risk while the
// spec is still maturing.
//
// What we implement (MCP 2024-11-05):
//   - initialize        (capabilities handshake)
//   - tools/list        (return our 3 tools)
//   - tools/call        (dispatch to handlers)
//
// What we don't (yet):
//   - resources         (we don't expose any read-only resources)
//   - prompts           (no canned prompts; agents already know what to ask)
//   - sampling          (we don't call the agent back)
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ops0-ai/ops0-cli/internal/api"
	"github.com/ops0-ai/ops0-cli/internal/config"
)

// Serve reads JSON-RPC messages from `in`, writes responses to `out`.
// Blocks until the input is closed (e.g. agent exits).
func Serve(in io.Reader, out io.Writer) error {
	s := &server{out: out}
	scanner := bufio.NewScanner(in)
	// MCP messages can be large — default 64KB buffer is too small for big
	// Terraform plans. Bump to 4MB.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			s.writeError(nil, -32700, "parse error: "+err.Error())
			continue
		}
		s.dispatch(&req)
	}
	return scanner.Err()
}

// JSON-RPC framing ────────────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // can be number, string, or absent (notification)
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type server struct {
	out io.Writer
}

func (s *server) write(resp *rpcResponse) {
	resp.JSONRPC = "2.0"
	data, err := json.Marshal(resp)
	if err != nil {
		// Defensive: if our own marshal fails we want to know.
		fmt.Fprintln(os.Stderr, "mcp marshal error:", err)
		return
	}
	// MCP over stdio uses newline-delimited JSON.
	fmt.Fprintln(s.out, string(data))
}

func (s *server) writeError(id json.RawMessage, code int, msg string) {
	s.write(&rpcResponse{ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func (s *server) dispatch(req *rpcRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "notifications/initialized", "notifications/cancelled":
		// Notifications have no ID and no response.
	default:
		// Method not found — but only respond if it's a request (has ID).
		if req.ID != nil {
			s.writeError(req.ID, -32601, "method not found: "+req.Method)
		}
	}
}

// initialize ─────────────────────────────────────────────────────────────────

func (s *server) handleInitialize(req *rpcRequest) {
	s.write(&rpcResponse{
		ID: req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]any{
				"name":    "ops0",
				"version": "0.1.0", // overridden at build time elsewhere
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		},
	})
}

// tools/list ─────────────────────────────────────────────────────────────────

var toolList = []map[string]any{
	{
		"name":        "list_policies",
		"description": "List ops0 policies that apply to the current repository. Call this BEFORE generating Infrastructure-as-Code so the rules are in context.",
		"inputSchema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		"name":        "check_compliance",
		"description": "Evaluate ops0 Rego policies against a Terraform plan JSON. Returns violations (or empty list if compliant).",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"planJsonPath": map[string]any{
					"type":        "string",
					"description": "Filesystem path to a Terraform plan JSON (output of `terraform show -json plan.out`).",
				},
			},
			"required": []string{"planJsonPath"},
		},
	},
	{
		"name":        "get_finding_details",
		"description": "Look up a discovery security check by templateId (e.g. aws-s3-public-access-block-disabled). Returns description + remediation.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"templateId": map[string]any{"type": "string"},
			},
			"required": []string{"templateId"},
		},
	},
}

func (s *server) handleToolsList(req *rpcRequest) {
	s.write(&rpcResponse{
		ID:     req.ID,
		Result: map[string]any{"tools": toolList},
	})
}

// tools/call ─────────────────────────────────────────────────────────────────

func (s *server) handleToolsCall(req *rpcRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(req.ID, -32602, "invalid params: "+err.Error())
		return
	}

	switch params.Name {
	case "list_policies":
		s.toolListPolicies(req.ID)
	case "check_compliance":
		path, _ := params.Arguments["planJsonPath"].(string)
		s.toolCheckCompliance(req.ID, path)
	case "get_finding_details":
		tid, _ := params.Arguments["templateId"].(string)
		s.toolGetFindingDetails(req.ID, tid)
	default:
		s.writeError(req.ID, -32601, "unknown tool: "+params.Name)
	}
}

// Tool handlers ──────────────────────────────────────────────────────────────

// toolListPolicies returns the policies in scope as MCP "content" blocks.
// MCP convention: tools return one or more typed content items; for now we
// stick to "text" so the agent gets a human-readable answer + a structured
// JSON payload it can parse if it wants.
func (s *server) toolListPolicies(id json.RawMessage) {
	client, projectID, err := bootstrapClient()
	if err != nil {
		s.toolError(id, err)
		return
	}
	policies, err := client.ListPolicies(projectID)
	if err != nil {
		s.toolError(id, err)
		return
	}

	payload, _ := json.MarshalIndent(policies, "", "  ")
	s.write(&rpcResponse{
		ID: id,
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": fmt.Sprintf("%d policies in scope:\n\n%s", len(policies), string(payload)),
				},
			},
		},
	})
}

// toolCheckCompliance is intentionally a thin shim — we keep the OPA-driving
// logic in the `opa` package and just delegate. Future: stream long checks
// as MCP progress notifications.
func (s *server) toolCheckCompliance(id json.RawMessage, planPath string) {
	if planPath == "" {
		s.toolError(id, fmt.Errorf("planJsonPath is required"))
		return
	}
	// Implementation deferred — once the `opa` package's Evaluate is invoked
	// directly from here we can return a structured result. For now we
	// surface a clear NotImplemented so it's testable end-to-end.
	s.write(&rpcResponse{
		ID: id,
		Result: map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "check_compliance: not implemented in this build. Run `ops0 policies check " + planPath + "` directly."},
			},
		},
	})
}

func (s *server) toolGetFindingDetails(id json.RawMessage, _ string) {
	// Stub — wire up to /api/v1/cli/findings/:templateId once the backend
	// exposes it. Leaving the contract in place so agents start exercising it.
	s.write(&rpcResponse{
		ID: id,
		Result: map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "get_finding_details: not implemented in this build."},
			},
		},
	})
}

func (s *server) toolError(id json.RawMessage, err error) {
	s.write(&rpcResponse{
		ID: id,
		Result: map[string]any{
			"isError": true,
			"content": []map[string]any{
				{"type": "text", "text": err.Error()},
			},
		},
	})
}

// bootstrapClient reads user + repo config and returns a ready API client +
// the project ID to filter policies by (may be empty for org-wide policies).
func bootstrapClient() (*api.Client, string, error) {
	userCfg, err := config.LoadUser()
	if err != nil {
		return nil, "", err
	}
	if userCfg.APIKey == "" {
		return nil, "", fmt.Errorf("ops0 CLI not authenticated — run `ops0 login` in a shell first")
	}
	cwd, _ := os.Getwd()
	repoCfg, _ := config.LoadRepo(cwd)
	projectID := ""
	if repoCfg != nil {
		projectID = repoCfg.ProjectID
	}
	return api.New(userCfg.APIBaseURL, userCfg.APIKey), projectID, nil
}
