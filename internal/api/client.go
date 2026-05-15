// Package api is the HTTP client for the ops0 backend.
//
// Endpoint shape (all under <base>/api/v1/cli/...) is intentionally separate
// from the dashboard's internal API so we can version it independently and
// keep a stable contract for the CLI.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps an authenticated *http.Client pinned to one ops0 instance.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// New creates a client with sensible defaults. baseURL should be the full
// origin (e.g. "https://brew.ops0.ai") — we append /api/v1/... internally so
// callers don't have to remember the prefix.
func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// do performs the request with auth + JSON headers. The caller owns the
// returned response body and must Close it.
func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		// We use a custom header rather than Authorization: Bearer so the
		// server can distinguish CLI keys from session JWTs cleanly.
		req.Header.Set("X-Ops0-API-Key", c.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ops0-cli")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		// Read & close so the caller doesn't have to handle the error response
		// body; we surface it as part of the error message.
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &APIError{
			Status: resp.StatusCode,
			Body:   string(body),
			Path:   path,
		}
	}
	return resp, nil
}

// APIError is returned for any non-2xx response, with the body inlined so the
// caller can show it.
type APIError struct {
	Status int
	Body   string
	Path   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ops0 API %s returned %d: %s", e.Path, e.Status, e.Body)
}

// Whoami verifies the API key is valid and returns the org/user it's bound to.
// First call after `ops0 login` so we fail fast on bad keys.
func (c *Client) Whoami() (*WhoamiResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/cli/whoami", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out := &WhoamiResponse{}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("decode whoami: %w", err)
	}
	return out, nil
}

// ListPolicies returns the policies in scope for a given project (or org-wide
// if projectID is empty).
func (c *Client) ListPolicies(projectID string) ([]Policy, error) {
	path := "/api/v1/cli/policies"
	if projectID != "" {
		path += "?projectId=" + projectID
	}
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out := &policiesResponse{}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, err
	}
	return out.Policies, nil
}

// ReportCheck sends anonymized check results to the backend for audit
// telemetry. Best-effort: callers should ignore errors so a network blip
// never blocks a local check.
func (c *Client) ReportCheck(req *CheckReport) error {
	resp, err := c.do(http.MethodPost, "/api/v1/cli/telemetry/checks", req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// BlockedCommand is the payload for /telemetry/blocked-command — fired by
// the PreToolUse hook just before it exits 2 to capture which command was
// rejected.
type BlockedCommand struct {
	Command        string `json:"command"`
	MatchedPattern string `json:"matchedPattern,omitempty"`
	Title          string `json:"title,omitempty"`
	RepoHash       string `json:"repoHash,omitempty"`
	CLIVersion     string `json:"cliVersion,omitempty"`
}

// ReportBlockedCommand records a destroy/apply block in the org's audit
// trail. Like ReportCheck, this is best-effort — the hook still exits 2
// to block the agent regardless of whether this POST succeeds.
func (c *Client) ReportBlockedCommand(req *BlockedCommand) error {
	resp, err := c.do(http.MethodPost, "/api/v1/cli/telemetry/blocked-command", req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ValidateReport is what /telemetry/validate expects.
// Mirrors ValidateResponse plus repo/version metadata. tflint / scan may
// be nil if those stages were unavailable / errored.
type ValidateReport struct {
	Validate   ValidateSection   `json:"validate"`
	Tflint     *TflintScanResult `json:"tflint,omitempty"`
	Scan       *ScanSection      `json:"scan,omitempty"`
	Cost       *CostSection      `json:"cost,omitempty"`
	Budget     *BudgetSection    `json:"budget,omitempty"`
	RepoHash   string            `json:"repoHash,omitempty"`
	CLIVersion string            `json:"cliVersion,omitempty"`
	// ProjectID identifies the ops0 IaC project the scanned files belong
	// to. Resolved via FindRepo on the file path. Persisted server-side
	// so the Activity tab can link audit rows back to the project.
	ProjectID  string            `json:"projectId,omitempty"`
}

// ReportValidate records validate + tflint findings against the user's API
// key so they show up in Settings -> Activity. Best-effort like the other
// telemetry calls; the CLI still surfaces the failure to the agent via
// non-zero exit regardless of what this returns.
func (c *Client) ReportValidate(req *ValidateReport) error {
	resp, err := c.do(http.MethodPost, "/api/v1/cli/telemetry/validate", req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
