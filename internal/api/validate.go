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
//
// `ProjectID` lets the server resolve the org's per-project budget so a
// cost overrun against that project becomes a hard exit. If omitted, the
// server falls back to the org's global budget (if any) and otherwise
// reports cost as informational.
type ValidateRequest struct {
	Files         map[string]string `json:"files"`
	IacType       string            `json:"iacType,omitempty"`       // terraform | opentofu | oxid
	CloudProvider string            `json:"cloudProvider,omitempty"` // aws | gcp | azure | oracle
	ProjectID     string            `json:"projectId,omitempty"`
}

// ValidateResponse is the JSON shape returned by the server.
//
// `Scan` is the Checkov-style security scan that the server now runs as
// part of the pipeline. May be nil if the scan step failed — we don't
// fail the whole call when only scan fails, so the CLI gets validate +
// tflint results regardless.
type ValidateResponse struct {
	OK       bool              `json:"ok"`
	Validate ValidateSection   `json:"validate"`
	Tflint   *TflintScanResult `json:"tflint,omitempty"`
	Scan     *ScanSection      `json:"scan,omitempty"`
	Cost     *CostSection      `json:"cost,omitempty"`
	Budget   *BudgetSection    `json:"budget,omitempty"`
	Error    string            `json:"error,omitempty"`
}

// CostSection holds Infracost output. nil/ok=false means cost stage
// failed (no API key, network blip, unsupported provider). The CLI
// treats absent cost as informational, not blocking.
type CostSection struct {
	OK               bool           `json:"ok"`
	TotalMonthlyCost float64        `json:"totalMonthlyCost"`
	Resources        []CostResource `json:"resources"`
	Error            string         `json:"error,omitempty"`
}

type CostResource struct {
	Name         string  `json:"name"`
	ResourceType string  `json:"resourceType,omitempty"`
	MonthlyCost  float64 `json:"monthlyCost"`
	HourlyCost   float64 `json:"hourlyCost,omitempty"`
}

// BudgetSection is the result of evaluating the org's BudgetSettings
// against the cost estimate. Present only when the server computed it
// (cost available + org has settings). The CLI gates on (Enforced &&
// Exceeded && BlockOnExceed); other combinations are info-only.
type BudgetSection struct {
	Enforced      bool    `json:"enforced"`
	Limit         float64 `json:"limit,omitempty"`
	MonthlyCost   float64 `json:"monthlyCost,omitempty"`
	OverBy        float64 `json:"overBy,omitempty"`
	Exceeded      bool    `json:"exceeded,omitempty"`
	BlockOnExceed bool    `json:"blockOnExceed,omitempty"`
	Reason        string  `json:"reason,omitempty"`
}

// ScanSection mirrors the shape returned by iac-service /internal/scan-files
// (which is what the server fans out to). We only model the fields the CLI
// renders; the rest is parsed lazily as `any` when needed.
type ScanSection struct {
	Findings             []ScanFinding `json:"findings"`
	Summary              ScanSummary   `json:"summary"`
	SeverityDistribution struct {
		Critical int `json:"critical"`
		High     int `json:"high"`
		Medium   int `json:"medium"`
		Low      int `json:"low"`
		Unknown  int `json:"unknown"`
	} `json:"severityDistribution"`
	CheckovVersion string `json:"checkovVersion,omitempty"`
}

type ScanSummary struct {
	Passed        int `json:"passed"`
	Failed        int `json:"failed"`
	Skipped       int `json:"skipped"`
	ParsingErrors int `json:"parsingErrors"`
}

type ScanFinding struct {
	CheckID         string `json:"checkId"`
	CheckName       string `json:"checkName"`
	Severity        string `json:"severity"`
	Status          string `json:"status"`
	Resource        string `json:"resource"`
	ResourceAddress string `json:"resourceAddress,omitempty"`
	FilePath        string `json:"filePath"`
	LineRange       struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"lineRange"`
	Guideline string `json:"guideline,omitempty"`
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
