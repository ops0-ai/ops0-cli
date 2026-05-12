package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/ops0-ai/ops0-cli/internal/api"
	"github.com/ops0-ai/ops0-cli/internal/config"
	"github.com/ops0-ai/ops0-cli/internal/opa"
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

// list ───────────────────────────────────────────────────────────────────────

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

	cwd, _ := os.Getwd()
	repoCfg, _ := config.LoadRepo(cwd)
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

// check ──────────────────────────────────────────────────────────────────────

var (
	checkFormat string
	checkFailOn string
)

var policiesCheckCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Evaluate policies locally against IaC files (default: cwd)",
	Long: `Runs OPA locally against the Terraform plan JSON or raw HCL in the
given path. Code never leaves your machine. If --report is enabled (default),
pass/fail counts and template IDs are sent to ops0 for audit telemetry.

Exit code is non-zero if any policy at or above --fail-on severity fails.
Default is --fail-on=error so warnings and info don't break CI.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPoliciesCheck,
}

func init() {
	policiesCheckCmd.Flags().StringVar(&checkFormat, "format", "pretty", "Output format: pretty | json")
	policiesCheckCmd.Flags().StringVar(&checkFailOn, "fail-on", "error", "Severity threshold for non-zero exit: error | warning | info")
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

	cwd, _ := os.Getwd()
	repoCfg, _ := config.LoadRepo(cwd)
	projectID := ""
	if repoCfg != nil {
		projectID = repoCfg.ProjectID
	}

	client := api.New(userCfg.APIBaseURL, userCfg.APIKey)
	policies, err := client.ListPolicies(projectID)
	if err != nil {
		return fmt.Errorf("fetch policies: %w", err)
	}

	start := time.Now()
	result, err := opa.Evaluate(target, policies)
	if err != nil {
		return fmt.Errorf("OPA evaluation failed: %w", err)
	}
	duration := time.Since(start)

	// Print human-readable result.
	if checkFormat == "json" {
		return printJSON(cmd, result)
	}
	printPretty(cmd, result)

	// Telemetry — best-effort, never blocks the user.
	if userCfg.Telemetry {
		hash := sha256.Sum256([]byte(cwd))
		_ = client.ReportCheck(&api.CheckReport{
			ProjectID:  projectID,
			RepoHash:   hex.EncodeToString(hash[:]),
			Total:      result.Total,
			Passed:     result.Passed,
			Failed:     result.Failed,
			Violations: result.AsViolations(),
			Duration:   int(duration.Milliseconds()),
			CLIVersion: buildVersion,
		})
	}

	if result.ExitNonZero(checkFailOn) {
		// Non-zero exit drives CI gates. Cobra's SilenceUsage=true on root
		// means we don't dump help spam on this failure.
		os.Exit(1)
	}
	return nil
}

// Small helpers ──────────────────────────────────────────────────────────────

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
