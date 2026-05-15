package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// IaC validate via /api/v1/cli/validate/iac.
//
// This is the heavier sibling of CheckIaC. It runs the full pipeline
// server-side: `terraform init` -> `terraform validate` -> tflint. Used by
// the Stop hook (once per Claude turn) rather than per-edit because of the
// init cost (5-30s for provider download).

// ValidateRequest mirrors what /api/v1/cli/validate/iac expects.
//
// Files map keys are relative paths under the working root; values are the
// file contents. Path traversal is rejected server-side.
type ValidateRequest struct {
	Files         map[string]string `json:"files"`
	IacType       string            `json:"iacType,omitempty"`       // terraform | opentofu | oxid
	CloudProvider string            `json:"cloudProvider,omitempty"` // aws | gcp | azure | oracle
}

// ValidateResponse is the JSON shape returned by the server.
type ValidateResponse struct {
	OK       bool             `json:"ok"`
	Validate ValidateSection  `json:"validate"`
	Tflint   *TflintScanResult `json:"tflint,omitempty"`
	Error    string           `json:"error,omitempty"`
}

type ValidateSection struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Info     []string `json:"info"`
}

type TflintFinding struct {
	RuleName  string `json:"ruleName"`
	Severity  string `json:"severity"` // error | warning | notice
	Message   string `json:"message"`
	FilePath  string `json:"filePath"`
	Ruleset   string `json:"ruleset"`
	LineRange struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"lineRange"`
}

type TflintSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Notices  int `json:"notices"`
}

type TflintScanResult struct {
	Success         bool            `json:"success"`
	Findings        []TflintFinding `json:"findings"`
	Summary         TflintSummary   `json:"summary"`
	ExecutionTimeMs int             `json:"executionTimeMs"`
	TflintVersion   string          `json:"tflintVersion,omitempty"`
	LimitedCoverage bool            `json:"limitedCoverage"`
	Error           string          `json:"error,omitempty"`
}

// ValidateIaC ships an HCL bundle to the backend and waits for the full
// init/validate/tflint pipeline to complete. The endpoint is synchronous;
// expect 10-30s of latency depending on whether providers need fetching.
func (c *Client) ValidateIaC(req *ValidateRequest) (*ValidateResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/cli/validate/iac", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out := &ValidateResponse{}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("decode validate response: %w", err)
	}
	return out, nil
}
