package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// IaC check via /api/v1/cli/check/iac.
//
// We push the raw HCL up to ops0; the server-side Checkov + ops0 policy
// engine evaluates and returns unified findings. This removes the local
// OPA + Terraform plan JSON dependency the old `policies check` had.

// CheckFile is one Terraform/HCL file in the request bundle. The server
// caps the total payload size; respect that on the client too if you
// extend this in the future.
type CheckFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// CheckRequest mirrors what /api/v1/cli/check/iac expects.
type CheckRequest struct {
	Files     []CheckFile `json:"files"`
	Framework string      `json:"framework,omitempty"` // "terraform" | "cloudformation"
}

// CheckFinding is one violation from Checkov (or, later, ops0 Rego rules).
// We keep the field shape identical to the Checkov scan result so we can
// surface it without remapping in the CLI's pretty printer.
type CheckFinding struct {
	CheckID         string `json:"checkId"`
	CheckName       string `json:"checkName"`
	CheckType       string `json:"checkType"`
	Severity        string `json:"severity"` // critical | high | medium | low | unknown
	Status          string `json:"status"`   // passed | failed | skipped
	Resource        string `json:"resource"`
	ResourceAddress string `json:"resourceAddress,omitempty"`
	FilePath        string `json:"filePath"`
	LineRange       struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"lineRange"`
	Guideline   string `json:"guideline,omitempty"`
	Description string `json:"description,omitempty"`
}

type CheckSummary struct {
	Passed         int `json:"passed"`
	Failed         int `json:"failed"`
	Skipped        int `json:"skipped"`
	ParsingErrors  int `json:"parsingErrors"`
}

type SeverityDistribution struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Unknown  int `json:"unknown"`
}

type CheckResponse struct {
	Success              bool                 `json:"success"`
	Findings             []CheckFinding       `json:"findings"`
	Summary              CheckSummary         `json:"summary"`
	SeverityDistribution SeverityDistribution `json:"severityDistribution"`
	ExecutionTimeMs      int                  `json:"executionTimeMs"`
	CheckovVersion       string               `json:"checkovVersion,omitempty"`
	Error                string               `json:"error,omitempty"`
}

// CheckIaC ships an HCL bundle to the backend for evaluation and returns
// unified findings. The endpoint never sees source code on disk — only
// the in-memory contents we send.
func (c *Client) CheckIaC(req *CheckRequest) (*CheckResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/cli/check/iac", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out := &CheckResponse{}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("decode check response: %w", err)
	}
	return out, nil
}
