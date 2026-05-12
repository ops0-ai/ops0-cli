// Package opa wraps the local OPA binary for offline Rego evaluation.
//
// Why shell out instead of embedding the OPA library directly?
//   - OPA's Go module is heavy (~30MB+ bundled) and dramatically slows builds.
//   - Shelling out keeps the CLI small (target: <20MB binary) and lets users
//     pin their own OPA version for compliance/audit if they care.
//   - For the rare offline / restricted environment, users can sideload opa.
//
// We do soft-validation: if `opa` isn't on PATH we print a one-liner pointing
// at the install URL and fall back to a stub evaluator that still reports
// which policies *would* run.
package opa

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ops0-ai/ops0-cli/internal/api"
)

// Result is the aggregated outcome of evaluating all policies against a target.
type Result struct {
	Target     string
	Total      int
	Passed     int
	Failed     int
	Violations []Violation
}

// Violation captures a single policy failure with enough context to point
// the user (or the agent) at the offending resource.
type Violation struct {
	PolicyID   string
	PolicyName string
	Severity   string
	Message    string
	Resource   string // resource address from the Terraform plan, if available
}

// AsViolations adapts to the telemetry payload shape without exposing the
// internal Violation struct outside this package.
func (r *Result) AsViolations() []api.CheckViolation {
	out := make([]api.CheckViolation, 0, len(r.Violations))
	for _, v := range r.Violations {
		out = append(out, api.CheckViolation{
			PolicyID: v.PolicyID,
			Severity: v.Severity,
			Message:  v.Message,
		})
	}
	return out
}

// ExitNonZero returns true if any violation is at or above the configured
// fail threshold. The severity ladder is info < warning < error.
func (r *Result) ExitNonZero(failOn string) bool {
	rank := map[string]int{"info": 1, "warning": 2, "error": 3}
	threshold := rank[strings.ToLower(failOn)]
	if threshold == 0 {
		threshold = rank["error"]
	}
	for _, v := range r.Violations {
		if rank[strings.ToLower(v.Severity)] >= threshold {
			return true
		}
	}
	return false
}

// Evaluate runs every enabled policy against the target path.
//
// For now we expect the target to either:
//   - be a *.tfplan.json file (produced by `terraform show -json plan.out`), OR
//   - contain raw .tf files (in which case we don't run a plan — we report
//     that planning is required for a full check).
//
// Producing the plan ourselves would require a Terraform binary and provider
// init, both of which are heavy and break offline. Better UX: tell the user
// to wire `terraform plan -out=plan.out && terraform show -json plan.out > plan.json`
// into their workflow, and we check that.
func Evaluate(target string, policies []api.Policy) (*Result, error) {
	res := &Result{Target: target}

	planJSON, err := resolvePlanJSON(target)
	if err != nil {
		return res, err
	}

	if !opaInstalled() {
		fmt.Fprintln(os.Stderr, "warning: `opa` not found on PATH — skipping evaluation.")
		fmt.Fprintln(os.Stderr, "         Install via: brew install opa  (or https://www.openpolicyagent.org/docs/latest/#running-opa)")
		// Still list policies that would have run so the user knows what's
		// in scope. Avoids silently passing.
		for _, p := range policies {
			if !p.IsEnabled {
				continue
			}
			res.Total++
			res.Failed++
			res.Violations = append(res.Violations, Violation{
				PolicyID:   p.ID,
				PolicyName: p.Name,
				Severity:   "info",
				Message:    "opa binary missing — policy could not be evaluated",
			})
		}
		return res, nil
	}

	for _, p := range policies {
		if !p.IsEnabled || p.PolicyType != "iac" {
			continue
		}
		res.Total++

		passed, msgs, err := runOPA(p, planJSON)
		if err != nil {
			res.Failed++
			res.Violations = append(res.Violations, Violation{
				PolicyID:   p.ID,
				PolicyName: p.Name,
				Severity:   p.Severity,
				Message:    fmt.Sprintf("evaluation error: %v", err),
			})
			continue
		}
		if passed {
			res.Passed++
			continue
		}
		res.Failed++
		for _, m := range msgs {
			res.Violations = append(res.Violations, Violation{
				PolicyID:   p.ID,
				PolicyName: p.Name,
				Severity:   p.Severity,
				Message:    m,
			})
		}
	}
	return res, nil
}

// runOPA writes the Rego to a tempfile and shells out:
//   opa eval -d <regofile> -i <planfile> "data.ops0.deny"
// We expect policies to follow ops0's convention of producing a `deny`
// rule set under `package ops0` — convention is documented in the policy
// authoring guide.
func runOPA(policy api.Policy, planPath string) (bool, []string, error) {
	tmp, err := os.CreateTemp("", "ops0-policy-*.rego")
	if err != nil {
		return false, nil, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(policy.Rego); err != nil {
		return false, nil, err
	}
	tmp.Close()

	cmd := exec.Command("opa", "eval", "-d", tmp.Name(), "-i", planPath, "data.ops0.deny", "-f", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil, fmt.Errorf("opa: %w: %s", err, string(out))
	}

	// `opa eval -f json` returns {"result":[{"expressions":[{"value":[...]}]}]}
	var parsed struct {
		Result []struct {
			Expressions []struct {
				Value []string `json:"value"`
			} `json:"expressions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return false, nil, fmt.Errorf("parse opa output: %w", err)
	}
	if len(parsed.Result) == 0 || len(parsed.Result[0].Expressions) == 0 {
		return true, nil, nil
	}
	msgs := parsed.Result[0].Expressions[0].Value
	return len(msgs) == 0, msgs, nil
}

// resolvePlanJSON finds a Terraform plan JSON to feed OPA. Looks for any
// file ending in .tfplan.json (most common convention), falling back to
// plan.json. If nothing is found we return a clear error rather than silently
// passing — silent passes are the worst failure mode for a policy tool.
func resolvePlanJSON(target string) (string, error) {
	info, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return target, nil
	}
	candidates := []string{}
	_ = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".tfplan.json") || filepath.Base(path) == "plan.json" {
			candidates = append(candidates, path)
		}
		return nil
	})
	if len(candidates) == 0 {
		return "", fmt.Errorf("no Terraform plan JSON found under %s — generate one with `terraform show -json plan.out > plan.json`", target)
	}
	return candidates[0], nil
}

func opaInstalled() bool {
	_, err := exec.LookPath("opa")
	return err == nil
}
