package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ops0-ai/ops0-cli/internal/api"
	"github.com/ops0-ai/ops0-cli/internal/config"
	"github.com/spf13/cobra"
)

// `ops0 validate` runs the full server-side IaC pipeline (init + validate +
// tflint) against the working directory's `.tf` / `.tofu` / `.hcl` files.
//
// This is heavier than `policies check` and is intended to run ONCE per
// Claude turn via the `Stop` hook, not per file edit.

var (
	validateFormat     string
	validateIacType    string
	validateProvider   string
	validateFailOnWarn bool
	validateScanFailOn string
)

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Run the full IaC validate + tflint pipeline server-side",
	Long: `Bundles .tf / .tofu / .hcl files at the given path (default: cwd) and
asks the ops0 platform to run:

  1. terraform / tofu / oxid init (downloads providers in a sandbox)
  2. terraform / tofu / oxid validate
  3. tflint (provider-aware lint)

Returns unified findings. Exit code is non-zero if validate failed or any
tflint error is present. tflint warnings/notices don't block by default.

Designed to be called from Claude Code's Stop hook so end-of-turn
validation runs automatically after the agent finishes writing IaC.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringVar(&validateFormat, "format", "pretty", "Output format: pretty | json")
	validateCmd.Flags().StringVar(&validateIacType, "iac-type", "terraform", "IaC flavor: terraform | opentofu | oxid")
	validateCmd.Flags().StringVar(&validateProvider, "cloud", "", "Cloud provider hint for tflint plugins: aws | gcp | azure | oracle")
	validateCmd.Flags().BoolVar(&validateFailOnWarn, "fail-on-warning", false, "Also exit non-zero on tflint warnings (default: errors only)")
	validateCmd.Flags().StringVar(&validateScanFailOn, "scan-fail-on", "high", "Severity threshold for security scan findings: critical | high | medium | low")
}

func runValidate(cmd *cobra.Command, args []string) error {
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

	// Bundle IaC files. We reuse `collectIacFiles` from policies.go so the
	// same ignore rules and size cap apply.
	checkFiles, err := collectIacFiles(target)
	if err != nil {
		return err
	}
	if len(checkFiles) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No .tf / .tofu / .hcl files under "+target+" — nothing to validate.")
		return nil
	}

	// Convert to the map shape /validate-files expects. The check endpoint
	// uses an array because Checkov wanted positional metadata; validate
	// just needs path -> content.
	files := make(map[string]string, len(checkFiles))
	for _, f := range checkFiles {
		files[f.Name] = f.Content
	}

	// Resolve the bound project ID by walking up from the target. The
	// server uses this to pull the per-project budget from BudgetSettings.
	projectID := ""
	if cfg, _, _ := config.FindRepo(target); cfg != nil {
		projectID = cfg.ProjectID
	}

	client := api.New(userCfg.APIBaseURL, userCfg.APIKey)
	start := time.Now()
	result, err := client.ValidateIaC(&api.ValidateRequest{
		Files:         files,
		IacType:       validateIacType,
		CloudProvider: validateProvider,
		ProjectID:     projectID,
	})
	if err != nil {
		return fmt.Errorf("validate failed: %w", err)
	}
	if result.Error != "" {
		return fmt.Errorf("validate error: %s", result.Error)
	}
	duration := time.Since(start)

	if validateFormat == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	} else {
		printValidateResult(cmd, result, target, len(checkFiles), duration)
	}

	// Telemetry — best-effort, never blocks. We only post if there's
	// something to record (validate failure, tflint findings, or scan
	// findings). A clean run doesn't produce a row.
	if userCfg.Telemetry && shouldReportValidate(result) {
		repoCfg, repoRoot, _ := config.FindRepo(target)
		_ = repoCfg
		hashSrc := repoRoot
		if hashSrc == "" {
			hashSrc, _ = os.Getwd()
		}
		hash := sha256.Sum256([]byte(hashSrc))
		_ = client.ReportValidate(&api.ValidateReport{
			Validate:   result.Validate,
			Tflint:     result.Tflint,
			Scan:       result.Scan,
			Cost:       result.Cost,
			Budget:     result.Budget,
			RepoHash:   hex.EncodeToString(hash[:]),
			CLIVersion: buildVersion,
		})
	}

	// Exit rules:
	//   - terraform validate failed              -> exit 1 (always)
	//   - tflint errors > 0                      -> exit 1
	//   - tflint warnings > 0 + --fail-on-warning -> exit 1
	//   - security scan finding at/above --scan-fail-on -> exit 1
	hardFail := !result.Validate.Valid
	if result.Tflint != nil {
		if result.Tflint.Summary.Errors > 0 {
			hardFail = true
		}
		if validateFailOnWarn && result.Tflint.Summary.Warnings > 0 {
			hardFail = true
		}
	}
	if result.Scan != nil && scanHasBlockingFinding(result.Scan, validateScanFailOn) {
		hardFail = true
	}
	// Budget enforcement: only gate when the server explicitly says
	// (Enforced && Exceeded && BlockOnExceed). Anything else is reported
	// but doesn't block the agent.
	if result.Budget != nil && result.Budget.Enforced && result.Budget.Exceeded && result.Budget.BlockOnExceed {
		hardFail = true
	}
	if hardFail {
		os.Exit(1)
	}
	return nil
}

// scanHasBlockingFinding returns true if any failed Checkov finding is at
// or above the configured severity threshold. Also blocks on parsing
// errors so that a syntactically broken file caught by Checkov (not just
// terraform validate) doesn't sneak through.
func scanHasBlockingFinding(s *api.ScanSection, threshold string) bool {
	if s == nil {
		return false
	}
	if s.Summary.ParsingErrors > 0 {
		return true
	}
	rank := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	min := rank[strings.ToLower(threshold)]
	if min == 0 {
		min = rank["high"]
	}
	for _, f := range s.Findings {
		if f.Status != "failed" {
			continue
		}
		if rank[strings.ToLower(f.Severity)] >= min {
			return true
		}
	}
	return false
}

// printValidateResult renders the validate + tflint sections in a compact
// human-readable form. Sent to stderr by callers so Claude Code's Stop
// hook surfaces the output to the model on a non-zero exit.
func printValidateResult(cmd *cobra.Command, r *api.ValidateResponse, target string, fileCount int, duration time.Duration) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "ops0 validate %s (%d files, %s)\n\n", target, fileCount, duration.Round(time.Millisecond))

	// terraform / tofu / oxid validate block
	if r.Validate.Valid {
		fmt.Fprintln(out, "✓ Configuration is valid")
	} else {
		fmt.Fprintln(out, "✗ Validation failed:")
		for _, e := range r.Validate.Errors {
			fmt.Fprintf(out, "  - %s\n", trimLines(e, 4))
		}
	}
	for _, w := range r.Validate.Warnings {
		fmt.Fprintf(out, "  ! %s\n", trimLines(w, 2))
	}

	// tflint block
	if r.Tflint == nil {
		fmt.Fprintln(out, "\ntflint: unavailable")
	} else {
		t := r.Tflint
		fmt.Fprintf(out, "\ntflint: %d error(s), %d warning(s), %d notice(s)\n",
			t.Summary.Errors, t.Summary.Warnings, t.Summary.Notices)

		if len(t.Findings) > 0 {
			max := 20
			if len(t.Findings) < max {
				max = len(t.Findings)
			}
			for i := 0; i < max; i++ {
				f := t.Findings[i]
				loc := f.FilePath
				if f.LineRange.Start > 0 {
					loc = fmt.Sprintf("%s:%d", f.FilePath, f.LineRange.Start)
				}
				fmt.Fprintf(out, "  [%s] %s: %s (%s)\n", strings.ToUpper(f.Severity), f.RuleName, f.Message, loc)
			}
			if len(t.Findings) > max {
				fmt.Fprintf(out, "  ...and %d more (use --format=json to see all)\n", len(t.Findings)-max)
			}
		}
	}

	// scan (Checkov) block — printed last because it tends to be the
	// noisiest and the agent should read validate + tflint first.
	if r.Scan == nil {
		fmt.Fprintln(out, "\nscan: unavailable")
		return
	}
	s := r.Scan
	fmt.Fprintf(out, "\nscan: %d passed, %d failed (%d parsing errors). Severity: %dC / %dH / %dM / %dL\n",
		s.Summary.Passed, s.Summary.Failed, s.Summary.ParsingErrors,
		s.SeverityDistribution.Critical, s.SeverityDistribution.High,
		s.SeverityDistribution.Medium, s.SeverityDistribution.Low)

	// Print up to ~30 failed findings ranked by severity so the agent sees
	// the worst issues first when output gets truncated by Claude Code.
	failed := make([]api.ScanFinding, 0, len(s.Findings))
	for _, f := range s.Findings {
		if f.Status == "failed" {
			failed = append(failed, f)
		}
	}
	rank := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3, "unknown": 4}
	sortByRank(failed, rank)

	max := 30
	if len(failed) < max {
		max = len(failed)
	}
	for i := 0; i < max; i++ {
		f := failed[i]
		loc := f.FilePath
		if f.LineRange.Start > 0 {
			loc = fmt.Sprintf("%s:%d", f.FilePath, f.LineRange.Start)
		}
		fmt.Fprintf(out, "  [%s] %s: %s (%s — %s)\n",
			strings.ToUpper(f.Severity), f.CheckID, f.CheckName, f.Resource, loc)
	}
	if len(failed) > max {
		fmt.Fprintf(out, "  ...and %d more (use --format=json to see all)\n", len(failed)-max)
	}

	// cost + budget block — printed last because it's the most "this is
	// going to cost the agent / your wallet" signal. Cost is informational
	// unless a budget is set AND exceeded AND blockOnExceed is true.
	if r.Cost != nil && r.Cost.OK {
		fmt.Fprintf(out, "\ncost: $%.2f / month across %d resource(s)\n", r.Cost.TotalMonthlyCost, len(r.Cost.Resources))
		top := r.Cost.Resources
		// Show the 5 most expensive resources so the agent sees which to
		// optimize. Server already rounded to 2dp.
		sortResourcesByCostDesc(top)
		if len(top) > 5 {
			top = top[:5]
		}
		for _, res := range top {
			label := res.ResourceType
			if label == "" {
				label = "resource"
			}
			fmt.Fprintf(out, "  $%-9.2f  %s (%s)\n", res.MonthlyCost, res.Name, label)
		}
	} else if r.Cost != nil && r.Cost.Error != "" {
		fmt.Fprintf(out, "\ncost: unavailable (%s)\n", r.Cost.Error)
	}

	if r.Budget != nil {
		b := r.Budget
		switch {
		case !b.Enforced:
			// Skip: enforcement off, nothing to surface.
		case b.Limit == 0 && b.Reason != "":
			fmt.Fprintf(out, "budget: %s\n", b.Reason)
		case b.Exceeded && b.BlockOnExceed:
			fmt.Fprintf(out, "\nbudget: ✗ BLOCKED — $%.2f/mo exceeds project limit of $%.2f/mo by $%.2f.\n",
				b.MonthlyCost, b.Limit, b.OverBy)
			fmt.Fprintln(out, "  Your organization has 'Block Deployments on Exceed' enabled.")
			fmt.Fprintln(out, "  Trim resources, downsize instances, or remove the over-budget components,")
			fmt.Fprintln(out, "  then ask Claude to suggest cheaper alternatives.")
		case b.Exceeded:
			fmt.Fprintf(out, "\nbudget: ⚠ $%.2f/mo exceeds project limit of $%.2f/mo by $%.2f (not blocked).\n",
				b.MonthlyCost, b.Limit, b.OverBy)
		default:
			fmt.Fprintf(out, "\nbudget: ✓ $%.2f/mo within project limit of $%.2f/mo.\n", b.MonthlyCost, b.Limit)
		}
	}
}

// sortResourcesByCostDesc sorts the slice in place, biggest monthly cost
// first. Same insertion-sort pattern as sortByRank — slice is small.
func sortResourcesByCostDesc(rs []api.CostResource) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && rs[j].MonthlyCost > rs[j-1].MonthlyCost; j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}

// sortByRank is an in-place insertion sort over the severity rank map.
// Small N (typically dozens of findings), so the simpler algorithm wins.
func sortByRank(findings []api.ScanFinding, rank map[string]int) {
	for i := 1; i < len(findings); i++ {
		for j := i; j > 0 && rank[strings.ToLower(findings[j].Severity)] < rank[strings.ToLower(findings[j-1].Severity)]; j-- {
			findings[j], findings[j-1] = findings[j-1], findings[j]
		}
	}
}

// shouldReportValidate returns true if the response contains anything worth
// putting in the audit trail: a failed validate, any tflint finding, any
// failed scan finding, or scan parsing errors. A clean pipeline produces
// no row.
func shouldReportValidate(r *api.ValidateResponse) bool {
	if r == nil {
		return false
	}
	if !r.Validate.Valid {
		return true
	}
	if r.Tflint != nil && len(r.Tflint.Findings) > 0 {
		return true
	}
	if r.Scan != nil {
		if r.Scan.Summary.ParsingErrors > 0 || r.Scan.Summary.Failed > 0 {
			return true
		}
	}
	// Budget violations are worth recording even when nothing else fails.
	if r.Budget != nil && r.Budget.Enforced && r.Budget.Exceeded {
		return true
	}
	return false
}

func trimLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + " ..."
}

// Used to deduplicate filepath import lint when no other consumer needs it.
var _ = filepath.Separator
