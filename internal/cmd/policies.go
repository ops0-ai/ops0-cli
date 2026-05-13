package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ops0-ai/ops0-cli/internal/api"
	"github.com/ops0-ai/ops0-cli/internal/config"
	"github.com/spf13/cobra"
)

var policiesCmd = &cobra.Command{
	Use:   "policies",
	Short: "List or check policies that apply to this repo",
}

func init() {
	policiesCmd.AddCommand(policiesListCmd)
	policiesCmd.AddCommand(policiesCheckCmd)
}

// ─── list ──────────────────────────────────────────────────────────────────

var policiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List policies in scope for the current project",
	RunE:  runPoliciesList,
}

func runPoliciesList(cmd *cobra.Command, _ []string) error {
	userCfg, err := config.LoadUser()
	if err != nil {
		return err
	}
	if userCfg.APIKey == "" {
		return fmt.Errorf("not logged in — run `ops0 login` first")
	}

	// Walk up to find the nearest .ops0/config.json. Lets the user run
	// `ops0 policies list` from anywhere inside a monorepo subdir and still
	// get that subproject's policies, not the parent's.
	cwd, _ := os.Getwd()
	repoCfg, _, _ := config.FindRepo(cwd)
	projectID := ""
	if repoCfg != nil {
		projectID = repoCfg.ProjectID
	}

	client := api.New(userCfg.APIBaseURL, userCfg.APIKey)
	policies, err := client.ListPolicies(projectID)
	if err != nil {
		return err
	}

	if len(policies) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No policies in scope.")
		if projectID == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "Tip: run `ops0 init --project=<id>` to bind this repo to an IaC project.")
		}
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%-32s  %-12s  %-8s  %s\n", "NAME", "CATEGORY", "SEVERITY", "DESCRIPTION")
	for _, p := range policies {
		desc := p.Description
		if len(desc) > 64 {
			desc = desc[:61] + "..."
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-32s  %-12s  %-8s  %s\n", trunc(p.Name, 32), p.Category, p.Severity, desc)
	}
	return nil
}

// ─── check ─────────────────────────────────────────────────────────────────

var (
	checkFormat string
	checkFailOn string
)

var policiesCheckCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Scan IaC files for security violations and policy non-compliance",
	Long: `Scans Terraform / HCL files at the given path (default: cwd) by uploading
their contents to the ops0 platform, which runs Checkov (security rules) and
your organization's Rego policies, then returns unified findings.

Why an API call and not local? Checkov requires a Python install and a
several-hundred-MB rule set; running it server-side keeps the CLI tiny and
ensures everyone hits the same rule version your dashboard uses. Source
files are sent over HTTPS but not persisted — they live in a tempdir on the
scanner pod for the duration of the scan.

Exit code is non-zero if any finding at or above --fail-on severity fails.
Default is --fail-on=high so 'medium' or 'low' don't break CI.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPoliciesCheck,
}

func init() {
	policiesCheckCmd.Flags().StringVar(&checkFormat, "format", "pretty", "Output format: pretty | json")
	policiesCheckCmd.Flags().StringVar(&checkFailOn, "fail-on", "high", "Severity threshold for non-zero exit: critical | high | medium | low")
}

// Collect candidate files to scan. We send only Terraform / OpenTofu / HCL
// to keep the payload small; CloudFormation can be added later via a flag.
// Skips common ignore dirs to avoid uploading vendor trees and lockfiles.
func collectIacFiles(root string) ([]api.CheckFile, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	// Single file: send just that one — useful for hook callbacks.
	if !info.IsDir() {
		data, err := os.ReadFile(root)
		if err != nil {
			return nil, err
		}
		return []api.CheckFile{{
			Name:    filepath.Base(root),
			Content: string(data),
		}}, nil
	}

	skipDirs := map[string]struct{}{
		".git": {}, ".terraform": {}, "node_modules": {}, ".idea": {}, ".vscode": {},
	}

	var files []api.CheckFile
	const maxBytes = 4 * 1024 * 1024 // server caps at 5MB; leave headroom
	var total int

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return fs.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if !(strings.HasSuffix(name, ".tf") || strings.HasSuffix(name, ".tofu") || strings.HasSuffix(name, ".hcl") || strings.HasSuffix(name, ".tf.json")) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		total += len(body)
		if total > maxBytes {
			return fmt.Errorf("scan target exceeds %d bytes — narrow the path or split the scan", maxBytes)
		}
		files = append(files, api.CheckFile{Name: rel, Content: string(body)})
		return nil
	})
	return files, err
}

func runPoliciesCheck(cmd *cobra.Command, args []string) error {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	userCfg, err := config.LoadUser()
	if err != nil {
		return err
	}
	if userCfg.APIKey == "" {
		return fmt.Errorf("not logged in — run `ops0 login` first")
	}

	files, err := collectIacFiles(target)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No .tf / .tofu / .hcl files under "+target+" — nothing to scan.")
		return nil
	}

	client := api.New(userCfg.APIBaseURL, userCfg.APIKey)

	start := time.Now()
	result, err := client.CheckIaC(&api.CheckRequest{
		Files:     files,
		Framework: "terraform",
	})
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}
	if result.Error != "" {
		return fmt.Errorf("scan error: %s", result.Error)
	}
	duration := time.Since(start)

	if checkFormat == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	} else {
		printCheckResult(cmd, result, target, len(files))
	}

	// Telemetry — best-effort, never blocks.
	if userCfg.Telemetry {
		// Resolve the project from the SCAN TARGET, not CWD. In monorepo
		// setups (one repo, many ops0 projects), the hook may invoke
		// `ops0 policies check sub/dir/main.tf` from the parent's CWD —
		// walking up from the target gets us the right binding.
		repoCfg, repoRoot, _ := config.FindRepo(target)
		projectID := ""
		if repoCfg != nil {
			projectID = repoCfg.ProjectID
		}
		// Hash whichever path is most stable: the bound repo root if we found
		// one, else the CWD (legacy fallback for unbound runs).
		hashSrc := repoRoot
		if hashSrc == "" {
			hashSrc, _ = os.Getwd()
		}
		hash := sha256.Sum256([]byte(hashSrc))
		violations := make([]api.CheckViolation, 0, len(result.Findings))
		for _, f := range result.Findings {
			if f.Status != "failed" {
				continue
			}
			violations = append(violations, api.CheckViolation{
				PolicyID:    f.CheckID,
				Severity:    f.Severity,
				Message:     f.CheckName,
				FilePath:    f.FilePath,
				LineStart:   f.LineRange.Start,
				Resource:    f.Resource,
				Remediation: f.Guideline,
			})
		}
		_ = client.ReportCheck(&api.CheckReport{
			ProjectID:  projectID,
			RepoHash:   hex.EncodeToString(hash[:]),
			Total:      result.Summary.Passed + result.Summary.Failed,
			Passed:     result.Summary.Passed,
			Failed:     result.Summary.Failed,
			Violations: violations,
			Duration:   int(duration.Milliseconds()),
			CLIVersion: buildVersion,
		})
	}

	if shouldExitNonZero(result, checkFailOn) {
		// Non-zero exit drives CI gates and the Claude Code PostToolUse hook.
		os.Exit(1)
	}
	return nil
}

// shouldExitNonZero returns true if any failed finding at or above the
// configured threshold severity is present. Severities are ordered:
// critical (highest) > high > medium > low > unknown.
func shouldExitNonZero(r *api.CheckResponse, threshold string) bool {
	rank := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	min := rank[strings.ToLower(threshold)]
	if min == 0 {
		min = rank["high"]
	}
	for _, f := range r.Findings {
		if f.Status != "failed" {
			continue
		}
		if rank[strings.ToLower(f.Severity)] >= min {
			return true
		}
	}
	return false
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
